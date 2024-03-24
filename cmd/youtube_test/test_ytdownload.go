package main

import (
	"Jabba_The_Bot/pkg/audio/webm"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"time"
)

func main() {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(handler))

	videoID := "rlpqflI0ghI"
	videoURL := "https://www.youtube.com/watch?v=" + videoID
	outputPath := "library/" + videoID + ".webm"

	cmd := exec.Command(
		"yt-dlp",
		"--no-playlist",
		"--no-progress",
		"--no-part",
		"-f", "251/250/249/ba[acodec=opus]",
		"-o", "-",
		"--cookies", "cookies.txt",
		videoURL,
	)

	if err := os.MkdirAll("library", 0o755); err != nil {
		slog.Error("Error creating library folder", "error", err)
		return
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		slog.Error("Error creating output file", "error", err, "path", outputPath)
		return
	}
	defer outFile.Close()

	cmd.Stdout = outFile
	cmd.Stderr = os.Stderr

	inFile, err := os.Open(outputPath)
	if err != nil {
		slog.Error("Error opening output file", "error", err, "path", outputPath)
		return
	}
	defer inFile.Close()

	done := make(chan error, 1)
	go func() {
		runErr := cmd.Run()
		done <- runErr
		close(done)
		if runErr != nil {
			slog.Error("yt-dlp exited with error", "error", runErr)
			return
		}
		slog.Info("Download finished", "path", outputPath)
	}()

	growingReader := newGrowingFileReader(inFile, done)
	demuxer, err := webm.NewDemuxer(growingReader)
	if err != nil {
		slog.Error("Error creating demuxer", "error", err)
		return
	}
	defer demuxer.Close()

	frameCount := 0
	for {
		_, err := demuxer.ProvideFrame()
		if err == io.EOF {
			slog.Info("Demuxing completed", "frames", frameCount, "duration", demuxer.Duration())
			return
		}
		if err != nil {
			slog.Error("Demuxing failed", "error", err, "frame", frameCount)
			return
		}
		frameCount++

		if frameCount%100 == 0 {
			slog.Info("Demuxing progress", "frames", frameCount, "position", demuxer.Position())
		}

		if frameCount == 1000 {
			err := demuxer.Seek(10 * time.Minute)
			if err != nil {
				slog.Error("Seek failed", "error", err)
			}
		}
	}
}

type growingFileReader struct {
	file *os.File
	done <-chan error

	mu        sync.Mutex
	closed    bool
	writerErr error
}

func newGrowingFileReader(file *os.File, done <-chan error) *growingFileReader {
	return &growingFileReader{
		file: file,
		done: done,
	}
}

func (r *growingFileReader) Read(p []byte) (int, error) {
	for {
		r.mu.Lock()
		n, err := r.file.Read(p)
		r.mu.Unlock()
		if n > 0 {
			return n, nil
		}

		if err == nil {
			continue
		}
		if !errors.Is(err, io.EOF) {
			return 0, err
		}

		r.mu.Lock()
		closed := r.closed
		writerErr := r.writerErr
		r.mu.Unlock()
		if closed {
			if writerErr != nil {
				return 0, writerErr
			}
			return 0, io.EOF
		}

		select {
		case writerErr, ok := <-r.done:
			r.mu.Lock()
			if ok {
				r.writerErr = writerErr
			}
			r.closed = true
			r.mu.Unlock()
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func (r *growingFileReader) Seek(offset int64, whence int) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.file.Seek(offset, whence)
}
