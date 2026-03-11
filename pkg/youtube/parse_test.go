package youtube

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
			filename:  filepath.FromSlash("../../requests/music/youtube/search-response.json"),
			wantID:    "dQw4w9WgXcQ",
			wantTitle: "Never Gonna Give You Up",
		},
		{
			name:      "search-response2",
			filename:  filepath.FromSlash("../../requests/music/youtube/search-response2.json"),
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

func TestParseNextResult_Example(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile(filepath.FromSlash("../../requests/music/youtube/next-response.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	results, err := parseNextResult(raw)
	if err != nil {
		t.Fatalf("parseNextResult error: %v", err)
	}
	if len(results) == 0 {
		t.Fatalf("expected results, got %d", len(results))
	}

	first := findResultByVideoID(t, results, "dQw4w9WgXcQ")
	if first.Title != "Never Gonna Give You Up" {
		t.Fatalf("unexpected title for first recommendation: %q", first.Title)
	}
	if first.Artist != "Rick Astley" {
		t.Fatalf("unexpected artist for first recommendation: %q", first.Artist)
	}
	if first.CoverArtUrl == "" {
		t.Fatalf("expected cover art URL to be present")
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

func findResultByVideoID(t *testing.T, results []SearchResult, videoID string) SearchResult {
	t.Helper()
	for _, result := range results {
		if result.VideoID == videoID {
			return result
		}
	}
	t.Fatalf("expected result with video ID %q", videoID)
	return SearchResult{}
}
