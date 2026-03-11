package youtube

import (
	"Jabba_The_Bot/internal/music/domain"
	stream2 "Jabba_The_Bot/internal/music/stream"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

type pipeline struct {
	ytDlpPath string
	cookies   string
	cacheDir  string
}

func newPipeline(ytDlpPath string, cookiesPaht string, cacheDir string) *pipeline {
	return &pipeline{
		ytDlpPath: ytDlpPath,
		cookies:   cookiesPaht,
		cacheDir:  cacheDir,
	}
}

func (p *pipeline) Start(ctx context.Context, track *domain.Track) (io.ReadCloser, error) {
	cachePath := filepath.Join(p.cacheDir, track.VideoID+".webm")
	streamPath, writer, err := prepareStreamingFile(cachePath)
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, p.ytDlpPath,
		"--no-playlist", "--no-progress", "--no-part",
		//"-f", "ba[acodec*=opus][ext=webm]/ba[ext=webm]/ba",
		"-f", "249",
		"--cookies", p.cookies,
		"-o", "-",
		buildUrl(track.VideoID),
	)
	cmd.Stdout = writer
	cmd.Stderr = os.Stderr

	done := make(chan error, 1)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start yt-dlp: %w", err)
	}

	go func() {
		err := cmd.Wait()
		_ = writer.Close()
		done <- err
		close(done)
	}()

	file, err := os.Open(streamPath)
	if err != nil {
		return nil, err
	}

	return stream2.NewGrowingFileReader(file, done), nil
}

func buildUrl(videoID string) string {
	return fmt.Sprintf("https://music.youtube.com/watch?v=%s", videoID)
}

func prepareStreamingFile(cachePath string) (path string, f *os.File, err error) {
	f, err = os.Create(cachePath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create stream cache file: %w", err)
	}
	return cachePath, f, nil
}
