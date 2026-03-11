package youtube

import (
	"context"
	"testing"
)

func TestNext(t *testing.T) {
	t.Run("200 response", func(t *testing.T) {
		//server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//	http.Error(w, "nope", http.StatusInternalServerError)
		//}))
		//t.Cleanup(server.Close)

		client := NewClient()
		//client.http = server.Client()

		results, err := client.GetNextSongs(context.Background(), "AR8D2yqgQ1U")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(results) == 0 {
			t.Fatalf("expected results, got %d", len(results))
		}
	})
}
