package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type SSEEvent struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

type SSEBroker struct {
	mu          sync.RWMutex
	subscribers map[chan SSEEvent]struct{}
}

func NewSSEBroker() *SSEBroker {
	return &SSEBroker{subscribers: make(map[chan SSEEvent]struct{})}
}

func (b *SSEBroker) Subscribe() (<-chan SSEEvent, func()) {
	ch := make(chan SSEEvent, 16)

	b.mu.Lock()
	b.subscribers[ch] = struct{}{}
	b.mu.Unlock()

	cancel := func() {
		b.mu.Lock()
		if _, ok := b.subscribers[ch]; ok {
			delete(b.subscribers, ch)
			close(ch)
		}
		b.mu.Unlock()
	}

	return ch, cancel
}

func (b *SSEBroker) Publish(event SSEEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.subscribers {
		select {
		case ch <- event:
		default:
		}
	}
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	if s.queries != nil {
		sendSSE(w, "snapshot", map[string]any{
			"state":      s.queries.GetState(),
			"queue":      s.queries.GetQueue(),
			"nowPlaying": s.queries.GetNowPlaying(),
		})
		flusher.Flush()
	}

	if s.broker == nil {
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-r.Context().Done():
				return
			case <-ticker.C:
				sendSSE(w, "heartbeat", map[string]any{"ts": time.Now().Unix()})
				flusher.Flush()
			}
		}
	}

	events, cancel := s.broker.Subscribe()
	defer cancel()

	heartbeat := time.NewTicker(20 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			sendSSE(w, "heartbeat", map[string]any{"ts": time.Now().Unix()})
			flusher.Flush()
		case event, ok := <-events:
			if !ok {
				return
			}
			sendSSE(w, event.Type, event.Data)
			flusher.Flush()
		}
	}
}

func sendSSE(w http.ResponseWriter, event string, data any) {
	payload, _ := json.Marshal(data)
	_, _ = fmt.Fprintf(w, "event: %s\n", event)
	_, _ = fmt.Fprintf(w, "data: %s\n\n", payload)
}
