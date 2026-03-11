package http

import (
	"context"
	"net/http"
	"strings"
	"time"
)

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		query = strings.TrimSpace(r.URL.Query().Get("query"))
	}
	if query == "" {
		http.Error(w, "missing query param: q", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	results, err := s.svc.Search(ctx, query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	for i := range results {
		results[i].CoverUrl = coverProxyURL(results[i].CoverUrl, results[i].VideoID)
	}

	writeJSON(w, http.StatusOK, results)
}
