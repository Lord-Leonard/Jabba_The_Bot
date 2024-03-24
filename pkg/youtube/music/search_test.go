package music

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearch(t *testing.T) {
	t.Run("empty query", func(t *testing.T) {
		client := NewClient()

		_, err := client.Search(context.Background(), "")
		if !errors.Is(err, ErrEmptyQuery) {
			t.Fatalf("expected ErrEmptyQuery, got %v", err)
		}
	})

	t.Run("200 response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "nope", http.StatusInternalServerError)
		}))
		t.Cleanup(server.Close)

		client := NewClient()
		client.http = server.Client()

		results, err := client.Search(context.Background(), "dire straits")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(results) == 0 {
			t.Fatalf("expected results, got %d", len(results))
		}
	})
}
