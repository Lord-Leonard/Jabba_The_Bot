package http

import (
	"Jabba_The_Bot/internal/music"
	"encoding/json"
	"net/http"
)

type Server struct {
	svc    *music.Service
	broker *SSEBroker
	ui     http.Handler
}

func NewServer(httpAddr string, svc *music.Service, broker *SSEBroker, ui http.Handler) *http.Server {
	if broker == nil {
		broker = NewSSEBroker()
	}

	server := Server{
		svc:    svc,
		broker: broker,
		ui:     ui,
	}

	return &http.Server{
		Addr:    httpAddr,
		Handler: server.Routes(),
	}
}

func (s *Server) Routes() http.Handler {
	api := http.NewServeMux()
	api.HandleFunc("/api/commands", s.handleCommands)
	api.HandleFunc("/api/state", s.handleState)
	api.HandleFunc("/api/queue", s.handleQueue)
	api.HandleFunc("/api/now-playing", s.handleNowPlaying)
	api.HandleFunc("/api/cover", s.handleCoverArt)
	api.HandleFunc("/api/search", s.handleSearch)
	api.HandleFunc("/api/events", s.handleEvents)

	if s.ui == nil {
		return withCORS(api)
	}

	root := http.NewServeMux()
	root.Handle("/api/", withCORS(api))
	root.Handle("/", s.ui)
	return root
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
