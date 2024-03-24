package music

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSearchResult_Examples(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		filename  string
		wantID    string
		wantTitle string
	}{
		{
			name:      "search-response",
			filename:  filepath.FromSlash("../../../requests/music/youtube/search-response.json"),
			wantID:    "dQw4w9WgXcQ",
			wantTitle: "Never Gonna Give You Up",
		},
		{
			name:      "search-response2",
			filename:  filepath.FromSlash("../../../requests/music/youtube/search-response2.json"),
			wantID:    "cF3OWCYLLVQ",
			wantTitle: "Sultans Of Swing",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			raw, err := os.ReadFile(tt.filename)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}

			results, err := parseSearchResult(raw)
			if err != nil {
				t.Fatalf("parseSearchResult error: %v", err)
			}
			if len(results) == 0 {
				t.Fatalf("expected results, got %d", len(results))
			}

			assertContainsResult(t, results, tt.wantID, tt.wantTitle)
		})
	}
}

func assertContainsResult(t *testing.T, results []SearchResult, wantID, wantTitle string) {
	t.Helper()

	for _, result := range results {
		if result.VideoID == wantID && result.Title == wantTitle {
			return
		}
	}

	t.Fatalf("expected result with video ID %q and title %q", wantID, wantTitle)
}
