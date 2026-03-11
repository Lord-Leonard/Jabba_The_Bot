package provider

import (
	"Jabba_The_Bot/internal/music/domain"
	searchadapter "Jabba_The_Bot/internal/music/provider/youtube"
	"Jabba_The_Bot/internal/music/stream"
	youtubemusic "Jabba_The_Bot/pkg/youtube"
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type YouTubeProvider struct {
	client         *youtubemusic.Client
	searchProvider *searchadapter.YouTubeSearchProvider

	callbackMu   sync.RWMutex
	onFirstBytes func(videoID, title string, duration time.Duration, cacheHit bool)
	ytDlpPath    string
}

var (
	toolCacheMu      sync.Mutex
	cachedYtDlpPath  string
	cachedFFmpegPath string
	prefetchMu       sync.Mutex
	prefetchInFlight = make(map[string]struct{})
)

// TODO: Interface for youtubemusic.Client?
func NewYouTubeProvider(client *youtubemusic.Client) (*YouTubeProvider, error) {
	wd, _ := os.Getwd()
	ytDlpPath, err := resolveYtDlpPath(wd)
	if err != nil {
		return nil, err
	}

	return &YouTubeProvider{
		client:         client,
		searchProvider: searchadapter.NewYouTubeSearcher(client),
		ytDlpPath:      ytDlpPath,
	}, nil
}

func (p *YouTubeProvider) Search(ctx context.Context, query string) ([]domain.Track, error) {
	results, err := p.searchProvider.Search(ctx, query, 0)
	if err != nil {
		return nil, err
	}

	tracks := make([]domain.Track, len(results))
	for i, res := range results {
		tracks[i] = domain.Track{
			Title:    res.Title,
			VideoID:  res.VideoID,
			URL:      "https://www.youtube.com/watch?v=" + res.VideoID,
			provider: p,
		}
	}
	return tracks, nil
}

func (p *YouTubeProvider) Download(track *Track) (io.Reader, error) {
	if track == nil {
		return nil, fmt.Errorf("track is nil")
	}
	if strings.TrimSpace(track.VideoID) == "" {
		return nil, fmt.Errorf("track has empty video ID")
	}
	if strings.TrimSpace(track.URL) == "" {
		return nil, fmt.Errorf("track has empty URL")
	}

	started := time.Now()
	wd, _ := os.Getwd()
	slog.Info("download pipeline init", "videoID", track.VideoID, "title", track.Title)

	cachePath, err := resolveCachePath(wd, track.VideoID)
	if err == nil && cachePath != "" && isCacheReady(cachePath) {
		if cachedFile, openErr := os.Open(cachePath); openErr == nil {
			slog.Info("cache hit", "videoID", track.VideoID, "path", cachePath)
			return &firstReadCloser{
				r: cachedFile,
				firstRead: firstReadMeta{
					videoID:  track.VideoID,
					title:    track.Title,
					started:  started,
					cacheHit: true,
					notify:   p.emitFirstBytes,
				},
			}, nil
		}
	}

	ytDlpPath, err := resolveYtDlpPath(wd)
	if err != nil {
		return nil, err
	}
	bypassFFmpeg := bypassFFmpegDownloadEnabled()
	ffmpegPath := ""
	if !bypassFFmpeg {
		ffmpegPath, err = resolveFFmpegPath()
		if err != nil {
			return nil, err
		}
	}
	if bypassFFmpeg {
		slog.Info("download tools resolved", "videoID", track.VideoID, "ytDlp", ytDlpPath, "ffmpeg", "bypassed", "duration", time.Since(started))
	} else {
		slog.Info("download tools resolved", "videoID", track.VideoID, "ytDlp", ytDlpPath, "ffmpeg", ffmpegPath, "duration", time.Since(started))
	}

	ctx, cancel := context.WithCancel(context.Background())
	ytCmd := exec.CommandContext(ctx, ytDlpPath)
	ytFormat := "ba/b"
	if bypassFFmpeg {
		ytFormat = bypassFFmpegFormatSelector()
	}
	ytCmd.Args = append(ytCmd.Args,
		"--no-playlist",
		"--no-progress",
		"--no-part",
		"-f", ytFormat,
		"-o", "-",
	)
	if extractorArgs := youtubeExtractorArgs(); extractorArgs != "" {
		ytCmd.Args = append(ytCmd.Args, "--extractor-args", extractorArgs)
	}
	ytCmd.Args = append(ytCmd.Args, "--cookies", "cookies.txt", track.URL)
	ytCmd.Dir = wd
	ytCmd.Stderr = os.Stderr

	var ffmpegCmd *exec.Cmd

	streamPath, writerFile, removeOnClose, err := prepareStreamingFile(cachePath)
	if err != nil {
		cancel()
		return nil, err
	}

	readerFile, err := os.Open(streamPath)
	if err != nil {
		_ = writerFile.Close()
		cancel()
		return nil, err
	}

	done := make(chan error, 1)
	if bypassFFmpeg {
		ytCmd.Stdout = writerFile
	} else {
		ytStdout, pipeErr := ytCmd.StdoutPipe()
		if pipeErr != nil {
			_ = writerFile.Close()
			cancel()
			return nil, fmt.Errorf("failed to create yt-dlp stdout pipe: %w", pipeErr)
		}

		ffmpegCmd = exec.CommandContext(ctx, ffmpegPath,
			"-hide_banner",
			"-loglevel", "fatal",
			"-fflags", "+discardcorrupt",
			"-err_detect", "ignore_err",
			"-i", "pipe:0",
			"-vn",
			"-ac", "2",
			"-ar", "48000",
			"-c:a", "libopus",
			"-application", "audio",
			"-frame_duration", "20",
			"-vbr", "constrained",
			"-b:a", "96k",
			"-f", "webm",
			"pipe:1",
		)
		ffmpegCmd.Dir = wd
		ffmpegCmd.Stdin = ytStdout
		ffmpegCmd.Stdout = writerFile
		ffmpegCmd.Stderr = os.Stderr
	}

	go func() {
		cleanupFailed := func(cause error) {
			_ = writerFile.Close()
			if cachePath != "" {
				_ = os.Remove(streamPath)
			}
			done <- cause
			close(done)
		}

		if ffmpegCmd != nil {
			if err := ffmpegCmd.Start(); err != nil {
				slog.Error("ffmpeg failed to start", "error", err, "path", ffmpegPath, "videoID", track.VideoID)
				cleanupFailed(err)
				return
			}
			slog.Info("ffmpeg started", "videoID", track.VideoID, "duration", time.Since(started))
		}

		if err := ytCmd.Start(); err != nil {
			slog.Error("yt-dlp failed to start", "error", err, "path", ytDlpPath, "videoID", track.VideoID)
			if ffmpegCmd != nil && ffmpegCmd.Process != nil {
				_ = ffmpegCmd.Process.Kill()
				_ = ffmpegCmd.Wait()
			}
			cleanupFailed(err)
			return
		}
		slog.Info("yt-dlp started", "videoID", track.VideoID, "duration", time.Since(started))

		ytErr := ytCmd.Wait()
		if ffmpegCmd != nil {
			ffmpegErr := ffmpegCmd.Wait()
			if ffmpegErr != nil {
				slog.Error("ffmpeg exited with error", "Error", ffmpegErr, "VideoID", track.VideoID, "Path", ffmpegPath)
				cleanupFailed(ffmpegErr)
				return
			}
		}
		if ytErr != nil && !isIgnorableYtDlpExit(ytErr, streamPath) {
			slog.Error("yt-dlp exited with error", "Error", ytErr, "VideoID", track.VideoID, "Path", ytDlpPath)
			cleanupFailed(ytErr)
			return
		}
		if ytErr != nil {
			slog.Warn("yt-dlp exited non-zero but ffmpeg output looks valid; continuing", "videoID", track.VideoID, "error", ytErr)
		}

		_ = writerFile.Close()

		if cachePath != "" {
			slog.Info("cache stored", "videoID", track.VideoID, "path", cachePath)
		}

		slog.Info("download pipeline completed", "videoID", track.VideoID, "duration", time.Since(started))
		done <- nil
		close(done)
	}()

	growingReader := stream.NewGrowingFileReader(readerFile, done)

	return &asyncPipelineReader{
		reader: growingReader,
		cancel: cancel,
		onClose: func() {
			_ = growingReader.Close()
			if removeOnClose {
				_ = os.Remove(streamPath)
			}
		},
		firstRead: firstReadMeta{
			videoID:  track.VideoID,
			title:    track.Title,
			started:  started,
			cacheHit: false,
			notify:   p.emitFirstBytes,
		},
	}, nil
}

func (p *YouTubeProvider) HasCachedTrack(videoID string) bool {
	videoID = strings.TrimSpace(videoID)
	if videoID == "" {
		return false
	}
	wd, _ := os.Getwd()
	cachePath, err := resolveCachePath(wd, videoID)
	if err != nil || cachePath == "" {
		return false
	}
	return isCacheReady(cachePath)
}

func (p *YouTubeProvider) Prefetch(ctx context.Context, track domain.Track) error {
	if strings.TrimSpace(track.VideoID) == "" {
		return fmt.Errorf("prefetch requires a valid video ID")
	}

	wd, _ := os.Getwd()
	cachePath, err := resolveCachePath(wd, track.VideoID)
	if err != nil || cachePath == "" {
		return err
	}
	if isCacheReady(cachePath) {
		return nil
	}

	prefetchMu.Lock()
	if _, busy := prefetchInFlight[track.VideoID]; busy {
		prefetchMu.Unlock()
		return nil
	}
	prefetchInFlight[track.VideoID] = struct{}{}
	prefetchMu.Unlock()
	defer func() {
		prefetchMu.Lock()
		delete(prefetchInFlight, track.VideoID)
		prefetchMu.Unlock()
	}()

	ytDlpPath, err := resolveYtDlpPath(wd)
	if err != nil {
		return err
	}
	bypassFFmpeg := bypassFFmpegDownloadEnabled()

	tempPath := cachePath + ".part"
	_ = os.Remove(tempPath)
	ytFormat := "ba/b"
	if bypassFFmpeg {
		ytFormat = bypassFFmpegFormatSelector()
	}

	ytArgs := []string{
		"--no-playlist",
		"--no-progress",
		"--no-part",
		"-f", ytFormat,
	}
	if extractorArgs := youtubeExtractorArgs(); extractorArgs != "" {
		ytArgs = append(ytArgs, "--extractor-args", extractorArgs)
	}
	ytArgs = append(ytArgs, "--cookies", "cookies.txt")

	started := time.Now()
	slog.Info("prefetch started", "videoID", track.VideoID, "title", track.Title, "bypassFFmpeg", bypassFFmpeg)

	if bypassFFmpeg {
		ytArgs = append(ytArgs, "-o", tempPath, track.URL)
		ytCmd := exec.CommandContext(ctx, ytDlpPath, ytArgs...)
		ytCmd.Dir = wd
		ytCmd.Stderr = io.Discard

		ytErr := ytCmd.Run()
		if ytErr != nil && !isIgnorableYtDlpExit(ytErr, tempPath) {
			_ = os.Remove(tempPath)
			return fmt.Errorf("yt-dlp prefetch error: %w", ytErr)
		}
		if ytErr != nil {
			slog.Warn("yt-dlp prefetch exited non-zero but output looks valid; continuing", "videoID", track.VideoID, "error", ytErr)
		}

		if !isCacheReady(tempPath) {
			_ = os.Remove(tempPath)
			return fmt.Errorf("yt-dlp prefetch produced empty output")
		}
		if err := os.Rename(tempPath, cachePath); err != nil {
			_ = os.Remove(tempPath)
			return err
		}

		slog.Info("prefetch completed", "videoID", track.VideoID, "duration", time.Since(started), "path", cachePath)
		return nil
	}

	ffmpegPath, err := resolveFFmpegPath()
	if err != nil {
		return err
	}
	tempPath, cacheFile, err := prepareCacheFile(cachePath)
	if err != nil {
		return err
	}
	defer func() {
		_ = cacheFile.Close()
	}()

	ytPipeArgs := append(append([]string{}, ytArgs...), "-o", "-", track.URL)

	ytCmd := exec.CommandContext(ctx, ytDlpPath, ytPipeArgs...)
	ytCmd.Dir = wd
	ytStdout, err := ytCmd.StdoutPipe()
	if err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to create yt-dlp stdout pipe: %w", err)
	}
	ytCmd.Stderr = io.Discard

	ffmpegCmd := exec.CommandContext(
		ctx,
		ffmpegPath,
		"-hide_banner",
		"-loglevel", "fatal",
		"-fflags", "+discardcorrupt",
		"-err_detect", "ignore_err",
		"-i", "pipe:0",
		"-vn",
		"-ac", "2",
		"-ar", "48000",
		"-c:a", "libopus",
		"-application", "audio",
		"-frame_duration", "20",
		"-vbr", "constrained",
		"-b:a", "96k",
		"-f", "webm",
		"pipe:1",
	)
	ffmpegCmd.Dir = wd
	ffmpegCmd.Stdin = ytStdout
	ffmpegCmd.Stdout = cacheFile
	ffmpegCmd.Stderr = io.Discard

	if err := ffmpegCmd.Start(); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("ffmpeg failed to start: %w", err)
	}
	if err := ytCmd.Start(); err != nil {
		_ = ffmpegCmd.Process.Kill()
		_ = os.Remove(tempPath)
		return fmt.Errorf("yt-dlp failed to start: %w", err)
	}

	ytErr := ytCmd.Wait()
	ffmpegErr := ffmpegCmd.Wait()
	if ffmpegErr != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("ffmpeg prefetch error: %w", ffmpegErr)
	}
	if ytErr != nil && !isIgnorableYtDlpExit(ytErr, tempPath) {
		_ = os.Remove(tempPath)
		return fmt.Errorf("yt-dlp prefetch error: %w", ytErr)
	}
	if ytErr != nil {
		slog.Warn("yt-dlp prefetch exited non-zero but ffmpeg output looks valid; continuing", "videoID", track.VideoID, "error", ytErr)
	}

	if err := cacheFile.Close(); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	if err := os.Rename(tempPath, cachePath); err != nil {
		_ = os.Remove(tempPath)
		return err
	}

	slog.Info("prefetch completed", "videoID", track.VideoID, "duration", time.Since(started), "path", cachePath)
	return nil
}

func (p *YouTubeProvider) SetOnFirstBytes(fn func(videoID, title string, duration time.Duration, cacheHit bool)) {
	p.callbackMu.Lock()
	defer p.callbackMu.Unlock()
	p.onFirstBytes = fn
}

func (p *YouTubeProvider) emitFirstBytes(videoID, title string, duration time.Duration, cacheHit bool) {
	p.callbackMu.RLock()
	cb := p.onFirstBytes
	p.callbackMu.RUnlock()
	if cb != nil {
		cb(videoID, title, duration, cacheHit)
	}
}

func resolveYtDlpPath(workdir string) (string, error) {
	toolCacheMu.Lock()
	if cachedYtDlpPath != "" {
		p := cachedYtDlpPath
		toolCacheMu.Unlock()
		return p, nil
	}
	toolCacheMu.Unlock()

	if p := os.Getenv("YTDLP_PATH"); p != "" {
		if !isExecutableFile(p) {
			return "", fmt.Errorf("YTDLP_PATH is set but not executable on this system: %s", p)
		}
		toolCacheMu.Lock()
		cachedYtDlpPath = p
		toolCacheMu.Unlock()
		return p, nil
	}

	candidates := []string{
		"./yt-dlp_macos",
		"./yt-dlp",
		"./yt-dlp_linux",
		"./yt-dlp.exe",
	}
	for _, candidate := range candidates {
		full := candidate
		if !filepath.IsAbs(candidate) {
			full = filepath.Join(workdir, candidate)
		}

		info, err := os.Stat(full)
		if err != nil || info.IsDir() {
			continue
		}

		if isExecutableFile(full) {
			toolCacheMu.Lock()
			cachedYtDlpPath = full
			toolCacheMu.Unlock()
			return full, nil
		}
	}

	if fromPath, err := exec.LookPath("yt-dlp"); err == nil {
		toolCacheMu.Lock()
		cachedYtDlpPath = fromPath
		toolCacheMu.Unlock()
		return fromPath, nil
	}

	return "", fmt.Errorf("no yt-dlp executable found; set YTDLP_PATH to a compatible binary (macOS arm64 needs a Darwin/arm64 build)")
}

func resolveFFmpegPath() (string, error) {
	toolCacheMu.Lock()
	if cachedFFmpegPath != "" {
		p := cachedFFmpegPath
		toolCacheMu.Unlock()
		return p, nil
	}
	toolCacheMu.Unlock()

	if p := os.Getenv("FFMPEG_PATH"); p != "" {
		if !isExecutableFile(p) {
			return "", fmt.Errorf("FFMPEG_PATH is set but not executable on this system: %s", p)
		}
		toolCacheMu.Lock()
		cachedFFmpegPath = p
		toolCacheMu.Unlock()
		return p, nil
	}

	fromPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return "", fmt.Errorf("ffmpeg not found; install ffmpeg or set FFMPEG_PATH")
	}
	toolCacheMu.Lock()
	cachedFFmpegPath = fromPath
	toolCacheMu.Unlock()
	return fromPath, nil
}

func isExecutableFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		return strings.HasSuffix(strings.ToLower(path), ".exe")
	}
	return info.Mode()&0o111 != 0
}

type firstReadMeta struct {
	videoID  string
	title    string
	started  time.Time
	cacheHit bool
	once     sync.Once
	notify   func(videoID, title string, duration time.Duration, cacheHit bool)
}

func (m *firstReadMeta) mark() {
	m.once.Do(func() {
		duration := time.Since(m.started)
		slog.Info("first encoded bytes received", "videoID", m.videoID, "duration", duration, "cacheHit", m.cacheHit)
		if m.notify != nil {
			m.notify(m.videoID, m.title, duration, m.cacheHit)
		}
	})
}

type firstReadCloser struct {
	r         io.ReadCloser
	firstRead firstReadMeta
	closeOnce sync.Once
}

func (r *firstReadCloser) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	if n > 0 {
		r.firstRead.mark()
	}
	if err == io.EOF {
		_ = r.Close()
	}
	return n, err
}

func (r *firstReadCloser) Close() error {
	var closeErr error
	r.closeOnce.Do(func() {
		closeErr = r.r.Close()
	})
	return closeErr
}

func (r *firstReadCloser) Seek(offset int64, whence int) (int64, error) {
	s, ok := r.r.(io.Seeker)
	if !ok {
		return 0, fmt.Errorf("underlying stream is not seekable")
	}
	return s.Seek(offset, whence)
}

type asyncPipelineReader struct {
	reader    io.Reader
	cancel    context.CancelFunc
	onClose   func()
	firstRead firstReadMeta
	closeOnce sync.Once
}

func (r *asyncPipelineReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 {
		r.firstRead.mark()
	}
	return n, err
}

func (r *asyncPipelineReader) Close() error {
	r.closeOnce.Do(func() {
		r.cancel()
		if r.onClose != nil {
			r.onClose()
		}
		if c, ok := r.reader.(io.Closer); ok {
			_ = c.Close()
		}
	})
	return nil
}

func (r *asyncPipelineReader) Seek(offset int64, whence int) (int64, error) {
	s, ok := r.reader.(io.Seeker)
	if !ok {
		return 0, fmt.Errorf("underlying stream is not seekable")
	}
	return s.Seek(offset, whence)
}

func resolveCachePath(workdir, videoID string) (string, error) {
	if !cacheEnabled() {
		return "", nil
	}

	cacheDir := os.Getenv("MUSIC_CACHE_DIR")
	if cacheDir == "" {
		cacheDir = filepath.Join(workdir, "library", "cache", "webm")
	}
	cacheDir = filepath.Join(cacheDir, cacheProfileKey())
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create cache dir %s: %w", cacheDir, err)
	}
	return filepath.Join(cacheDir, videoID+".webm"), nil
}

func cacheProfileKey() string {
	if !bypassFFmpegDownloadEnabled() {
		// Matches the hard-coded ffmpeg settings in this provider.
		return "ffmpeg_opus_96k_cbr20ms"
	}
	selector := bypassFFmpegFormatSelector()
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(selector))
	return fmt.Sprintf("yt_bypass_%08x", hasher.Sum32())
}

func prepareCacheFile(finalPath string) (string, *os.File, error) {
	tempPath := finalPath + ".part"
	f, err := os.Create(tempPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create cache temp file: %w", err)
	}
	return tempPath, f, nil
}

func prepareStreamingFile(cachePath string) (path string, f *os.File, removeOnClose bool, err error) {
	if cachePath != "" {
		f, err = os.Create(cachePath)
		if err != nil {
			return "", nil, false, fmt.Errorf("failed to create stream cache file: %w", err)
		}
		return cachePath, f, false, nil
	}

	f, err = os.CreateTemp("", "jabba-stream-*.webm")
	if err != nil {
		return "", nil, false, fmt.Errorf("failed to create temp stream file: %w", err)
	}
	return f.Name(), f, true, nil
}

func isCacheReady(cachePath string) bool {
	if cachePath == "" {
		return false
	}
	info, err := os.Stat(cachePath)
	if err != nil || info.IsDir() || info.Size() == 0 {
		return false
	}
	return true
}

func cacheEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("MUSIC_CACHE_ENABLED")))
	switch v {
	case "0", "false", "off", "no":
		return false
	default:
		return true
	}
}

func bypassFFmpegDownloadEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("MUSIC_BYPASS_FFMPEG")))
	switch v {
	case "1", "true", "on", "yes":
		return true
	default:
		return false
	}
}

func bypassFFmpegFormatSelector() string {
	if raw := strings.TrimSpace(os.Getenv("MUSIC_BYPASS_FFMPEG_FORMAT")); raw != "" {
		return raw
	}
	// TeamSpeak has a strict voice payload limit (~484 bytes in this client).
	// Prefer lower bitrate Opus variants first to reduce packet-size spikes.
	// YouTube Opus itags (typical): 249 ~50k, 250 ~70k, 251 ~160k.
	return "250/249/ba[acodec=opus]"
}

func youtubeExtractorArgs() string {
	// Default tuned profile; override with YTDLP_EXTRACTOR_ARGS if needed.
	if raw := strings.TrimSpace(os.Getenv("YTDLP_EXTRACTOR_ARGS")); raw != "" {
		return raw
	}
	return "youtube:player_client=tv,web"
}

func isIgnorableYtDlpExit(err error, outputPath string) bool {
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return false
	}
	if exitErr.ExitCode() != 1 {
		return false
	}
	info, statErr := os.Stat(outputPath)
	if statErr != nil || info.IsDir() || info.Size() == 0 {
		return false
	}
	return true
}
