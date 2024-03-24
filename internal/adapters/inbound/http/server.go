package http

import (
	"encoding/json"
	"net/http"

	"Jabba_The_Bot/internal/core/app/ports"
)

type Server struct {
	commands ports.MusicCommands
	queries  ports.MusicQueries
	search   ports.MusicSearchQueries
	broker   *SSEBroker
	ui       http.Handler
}

func NewServer(commands ports.MusicCommands, queries ports.MusicQueries, search ports.MusicSearchQueries, broker *SSEBroker) *Server {
	return NewServerWithUI(commands, queries, search, broker, nil)
}

func NewServerWithUI(
	commands ports.MusicCommands,
	queries ports.MusicQueries,
	search ports.MusicSearchQueries,
	broker *SSEBroker,
	ui http.Handler,
) *Server {
	if broker == nil {
		broker = NewSSEBroker()
	}

	return &Server{
		commands: commands,
		queries:  queries,
		search:   search,
		broker:   broker,
		ui:       ui,
	}
}

func (s *Server) Routes() http.Handler {
	api := http.NewServeMux()
	api.HandleFunc("/api/commands", s.handleCommands)
	api.HandleFunc("/api/state", s.handleState)
	api.HandleFunc("/api/queue", s.handleQueue)
	api.HandleFunc("/api/now-playing", s.handleNowPlaying)
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
