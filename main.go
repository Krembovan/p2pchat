package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
)

//go:embed static/*
var staticFiles embed.FS

type Event struct {
	Type  string   `json:"type"`
	Name  string   `json:"name,omitempty"`
	Text  string   `json:"text,omitempty"`
	Names []string `json:"names,omitempty"`
}

type Hub struct {
	mu   sync.Mutex
	subs map[chan Event]struct{}
}

func NewHub() *Hub {
	return &Hub{subs: make(map[chan Event]struct{})}
}

func (h *Hub) Subscribe() chan Event {
	ch := make(chan Event, 256)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *Hub) Unsubscribe(ch chan Event) {
	h.mu.Lock()
	delete(h.subs, ch)
	h.mu.Unlock()
}

func (h *Hub) Broadcast(ev Event) {
	h.mu.Lock()
	for ch := range h.subs {
		select {
		case ch <- ev:
		default:
		}
	}
	h.mu.Unlock()
}

var hub = NewHub()

func main() {
	setSelf(hostname())
	if len(os.Args) > 1 {
		setSelf(os.Args[1])
	}

	go startP2P()

	sub, _ := fs.Sub(staticFiles, "static")
	http.Handle("/", http.FileServer(http.FS(sub)))
	http.HandleFunc("/send", handleSend)
	http.HandleFunc("/events", handleEvents)
	http.HandleFunc("/info", handleInfo)

	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		ln, err = net.Listen("tcp", ":0")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	_, p, _ := net.SplitHostPort(ln.Addr().String())

	fmt.Printf("  ✦ p2pchat — %s\n", getSelf())
	fmt.Printf("  → http://localhost:%s\n\n", p)

	http.Serve(ln, nil)
}

func handleSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	text := strings.TrimSpace(r.FormValue("text"))
	if text == "" {
		return
	}

	if strings.HasPrefix(text, "/name ") {
		setSelf(strings.TrimPrefix(text, "/name "))
		hub.Broadcast(Event{Type: "peers", Names: peerNames()})
		return
	}

	send(text)
	hub.Broadcast(Event{Type: "self", Name: getSelf(), Text: text})
}

func handleEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := hub.Subscribe()
	defer hub.Unsubscribe(ch)

	data, _ := json.Marshal(Event{Type: "peers", Names: peerNames()})
	fmt.Fprintf(w, "data: %s\n\n", data)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	for {
		select {
		case ev := <-ch:
			data, _ := json.Marshal(ev)
			fmt.Fprintf(w, "data: %s\n\n", data)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		case <-r.Context().Done():
			return
		}
	}
}

func handleInfo(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]any{
		"name":  getSelf(),
		"peers": peerNames(),
	})
}
