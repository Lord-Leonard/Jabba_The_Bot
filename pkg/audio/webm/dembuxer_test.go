package webm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDemuxer(t *testing.T) {
	candidates := []string{
		filepath.Join("library", "*.webm"),
		filepath.Join("..", "..", "..", "library", "*.webm"),
	}

	var matches []string
	for _, pattern := range candidates {
		found, globErr := filepath.Glob(pattern)
		if globErr != nil {
			t.Fatalf("invalid glob pattern %q: %v", pattern, globErr)
		}
		if len(found) > 0 {
			matches = found
			break
		}
	}
	if len(matches) == 0 {
		t.Skip("no .webm files found in library directory for demuxer test")
	}

	file, err := os.Open(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	demuxer, err := NewDemuxer(file)
	if err != nil {
		t.Fatal(err)
	}

	frame, err := demuxer.ProvideFrame()
	if err != nil {
		t.Fatalf("ProvideFrame failed: %v", err)
	}
	if len(frame) == 0 {
		t.Fatalf("expected non-empty frame from %s", matches[0])
	}
}
