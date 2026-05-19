package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const discPort = 42069

type Msg struct {
	Name string `json:"n"`
	Text string `json:"t"`
}

type Peer struct {
	conn net.Conn
	enc  *json.Encoder
	name string
	addr string
}

var (
	pmu         sync.Mutex
	peers       = map[string]*Peer{}
	selfMu      sync.RWMutex
	self        string
	tcp         string
	dialMu      sync.Mutex
	dialSet     = map[string]struct{}{}
	cooldownMu  sync.Mutex
	cooldownMap = map[string]time.Time{}
)

func canAnnounce(name string) bool {
	cooldownMu.Lock()
	defer cooldownMu.Unlock()
	t, ok := cooldownMap[name]
	if ok && time.Since(t) < 3*time.Second {
		return false
	}
	cooldownMap[name] = time.Now()
	return true
}

func getSelf() string {
	selfMu.RLock()
	defer selfMu.RUnlock()
	return self
}

func setSelf(s string) {
	selfMu.Lock()
	defer selfMu.Unlock()
	self = s
}

func startP2P() {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		fmt.Println("  p2p error:", err)
		return
	}
	_, p, _ := net.SplitHostPort(ln.Addr().String())
	tcp = p
	fmt.Printf("  p2p :%s\n", tcp)

	go discover()

	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		go handleConn(conn)
	}
}

func discover() {
	go func() {
		c, err := net.ListenUDP("udp", nil)
		if err != nil {
			return
		}
		defer c.Close()

		b := []byte(fmt.Sprintf(`{"n":"%s","p":"%s"}`, getSelf(), tcp))

		broadcast, _ := net.ResolveUDPAddr("udp", "255.255.255.255:42069")
		loopback, _ := net.ResolveUDPAddr("udp", "127.0.0.1:42069")

		for {
			if broadcast != nil {
				c.WriteTo(b, broadcast)
			}
			if loopback != nil {
				c.WriteTo(b, loopback)
			}
			time.Sleep(2 * time.Second)
		}
	}()

	c, err := listenUDPReuse(fmt.Sprintf(":%d", discPort))
	if err != nil {
		fmt.Println("  discovery unavailable:", err)
		return
	}
	defer c.Close()

	buf := make([]byte, 512)
	for {
		n, r, err := c.ReadFromUDP(buf)
		if err != nil || n == 0 {
			continue
		}

		var d struct {
			Name string `json:"n"`
			Port string `json:"p"`
		}
		if json.Unmarshal(buf[:n], &d) != nil || d.Name == "" || d.Name == getSelf() {
			continue
		}

		addr := net.JoinHostPort(r.IP.String(), d.Port)
		if r.IP.IsLoopback() && d.Port == tcp {
			continue
		}

		pmu.Lock()
		_, exists := peers[addr]
		if !exists {
			for _, p := range peers {
				if p.name == d.Name {
					exists = true
					break
				}
			}
		}
		pmu.Unlock()
		if exists {
			continue
		}

		name := d.Name
		go func(a string, n string) {
			if getSelf() > n {
				dial(a)
			}
		}(addr, name)
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()

	addr := conn.RemoteAddr().String()
	enc, dec := json.NewEncoder(conn), json.NewDecoder(conn)

	enc.Encode(Msg{Name: getSelf()})
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	var h Msg
	if err := dec.Decode(&h); err != nil {
		return
	}
	conn.SetDeadline(time.Time{})

	p := &Peer{conn, enc, h.Name, addr}

	pmu.Lock()
	if _, ok := peers[addr]; ok {
		pmu.Unlock()
		return
	}
	for _, old := range peers {
		if old.name == h.Name {
			old.conn.Close()
			delete(peers, old.addr)
			break
		}
	}
	peers[addr] = p
	pmu.Unlock()

	if canAnnounce(h.Name) {
		join := fmt.Sprintf("%s joined (%d)", h.Name, len(peers))
		pmu.Lock()
		for _, op := range peers {
			if op.addr != addr {
				op.enc.Encode(Msg{"*", join})
			}
		}
		pmu.Unlock()
		hub.Broadcast(Event{Type: "system", Text: join})
	}
	hub.Broadcast(Event{Type: "peers", Names: peerNames()})

	dec = json.NewDecoder(conn)
	for {
		var msg Msg
		if err := dec.Decode(&msg); err != nil {
			break
		}
		if msg.Name == "*" {
			hub.Broadcast(Event{Type: "system", Text: msg.Text})
		} else {
			hub.Broadcast(Event{Type: "msg", Name: msg.Name, Text: msg.Text})
		}
	}

	pmu.Lock()
	delete(peers, addr)
	list := make([]*Peer, 0, len(peers))
	for _, op := range peers {
		list = append(list, op)
	}
	pmu.Unlock()

	if canAnnounce(p.name) {
		leave := fmt.Sprintf("%s left", p.name)
		for _, op := range list {
			op.enc.Encode(Msg{"*", leave})
		}
		hub.Broadcast(Event{Type: "system", Text: leave})
	}
	hub.Broadcast(Event{Type: "peers", Names: peerNames()})
}

func tryDial(addr string) bool {
	dialMu.Lock()
	if _, ok := dialSet[addr]; ok {
		dialMu.Unlock()
		return false
	}
	dialSet[addr] = struct{}{}
	dialMu.Unlock()
	return true
}

func doneDial(addr string) {
	dialMu.Lock()
	delete(dialSet, addr)
	dialMu.Unlock()
}

func dial(addr string) {
	if !strings.Contains(addr, ":") || !tryDial(addr) {
		return
	}
	defer doneDial(addr)

	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return
	}
	handleConn(conn)
}

func send(text string) {
	msg := Msg{Name: getSelf(), Text: text}
	pmu.Lock()
	for _, p := range peers {
		p.enc.Encode(msg)
	}
	pmu.Unlock()
}

func peerNames() []string {
	pmu.Lock()
	defer pmu.Unlock()
	names := make([]string, 0, len(peers))
	for _, p := range peers {
		names = append(names, p.name)
	}
	return names
}

func hostname() string {
	h, _ := os.Hostname()
	return strings.SplitN(h, ".", 2)[0]
}
