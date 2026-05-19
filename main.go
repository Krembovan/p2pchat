package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
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
	ch := make(chan Event, 512)
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
	http.HandleFunc("/typing", handleTyping)
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

	if strings.HasPrefix(text, "/connect ") {
		go dial(strings.TrimPrefix(text, "/connect "))
		return
	}

	if strings.HasPrefix(text, "/") {
		if result := execCmd(text); result != "" {
			send(result)
			hub.Broadcast(Event{Type: "self", Name: getSelf(), Text: result})
			return
		}
	}

	send(text)
	hub.Broadcast(Event{Type: "self", Name: getSelf(), Text: text})
}

func handleTyping(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	sendTyping()
	hub.Broadcast(Event{Type: "typing", Name: getSelf()})
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

func execCmd(cmd string) string {
	parts := strings.SplitN(cmd, " ", 2)
	switch parts[0] {
	case "/help":
		return "Commands: /coinflip, /roll [N], /8ball Q, /shrug, /lenny, /flip, /tableflip, /unflip"
	case "/coinflip":
		if rand.Intn(2) == 0 {
			return "🪙 Орёл"
		}
		return "🪙 Решка"
	case "/roll":
		max := 100
		if len(parts) > 1 {
			if n, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil && n > 0 && n <= 100000 {
				max = n
			}
		}
		return fmt.Sprintf("🎲 %d", rand.Intn(max)+1)
	case "/8ball":
		answers := []string{
			"🎱 Бесспорно", "🎱 Предрешено", "🎱 Никаких сомнений",
			"🎱 Определённо да", "🎱 Можешь быть уверен",
			"🎱 Мне кажется — да", "🎱 Вероятнее всего", "🎱 Хорошие перспективы",
			"🎱 Знаки говорят — да", "🎱 Да",
			"🎱 Пока не ясно, попробуй снова", "🎱 Спроси позже",
			"🎱 Лучше не рассказывать", "🎱 Сейчас нельзя предсказать",
			"🎱 Сконцентрируйся и спроси опять",
			"🎱 Даже не думай", "🎱 Мой ответ — нет",
			"🎱 По моим данным — нет", "🎱 Перспективы не очень хорошие",
			"🎱 Весьма сомнительно",
		}
		return answers[rand.Intn(len(answers))]
	case "/shrug":
		return "¯\\_(ツ)_/¯"
	case "/lenny":
		return "( ͡° ͜ʖ ͡°)"
	case "/flip":
		return "(╯°□°)╯︵ ┻━┻"
	case "/tableflip":
		return "(╯°□°)╯︵ ┻━┻"
	case "/unflip":
		return "┬──┬ ノ( ゜-゜ノ)"
	case "/date":
		return time.Now().Format("📅 Mon Jan 2 15:04:05")
	case "/time":
		return time.Now().Format("🕐 15:04:05")
	case "/moon":
		return "🌙"
	case "/say":
		if len(parts) > 1 {
			return "💬 " + parts[1]
		}
		return ""
	case "/reverse":
		if len(parts) > 1 {
			runes := []rune(parts[1])
			for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
				runes[i], runes[j] = runes[j], runes[i]
			}
			return string(runes)
		}
		return ""
	}
	return ""
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
