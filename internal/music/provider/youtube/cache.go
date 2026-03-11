package youtube

import (
	"fmt"
	"os"
	"path/filepath"
)

type cache struct {
	dir string
}

func newCache(dir string) *cache {
	return &cache{dir: dir}
}

func (c *cache) Writer(videoID string) (path string, f *os.File, err error) {
	path = filepath.Join(c.dir, videoID+".webm")
	f, err = os.Create(path)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create cache file: %w", err)
	}
	return path, f, err
}

func (c *cache) Reader(videoID string) (*os.File, bool) {
	path := filepath.Join(c.dir, videoID+".webm")
	f, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	return f, true
}

func (c *cache) Has(videoID string) bool {
	path := filepath.Join(c.dir, videoID+".webm")
	info, err := os.Stat(path)
	return err == nil && info.Size() > 0
}
