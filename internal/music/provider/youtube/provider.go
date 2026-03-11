package youtube

import (
	"Jabba_The_Bot/internal/music/domain"
	"Jabba_The_Bot/internal/music/playback"
	"Jabba_The_Bot/pkg/youtube"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
)

// validate Provider satisfies AudioProvider interface
var _ playback.AudioProvider = (*Provider)(nil)

type Provider struct {
	cache    *cache
	pipeline *pipeline
	client   *youtube.Client

	recommendationLogLevel  RecommendationLogLevel
	recommendationLogLogger *slog.Logger
	demuxerLogLogger        *slog.Logger
}

type ProviderOption func(*Provider)

func WithRecommendationLogLevel(level RecommendationLogLevel) ProviderOption {
	return func(p *Provider) {
		p.recommendationLogLevel = level
	}
}

func WithRecommendationLogger(logger *slog.Logger) ProviderOption {
	return func(p *Provider) {
		p.recommendationLogLogger = logger
	}
}

func WithDemuxerLogger(logger *slog.Logger) ProviderOption {
	return func(p *Provider) {
		p.demuxerLogLogger = logger
	}
}

func NewProvider(cacheDir string, cookiesPath string, ytClient *youtube.Client, options ...ProviderOption) (*Provider, error) {
	ytDlp, err := resolveYtDlpPath()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve yt-dlp: %w", err)
	}

	p := &Provider{
		cache:                   newCache(cacheDir),
		pipeline:                newPipeline(ytDlp, cookiesPath, cacheDir),
		client:                  ytClient,
		recommendationLogLevel:  RecommendationLogOff,
		recommendationLogLogger: slog.Default(),
		demuxerLogLogger:        nil,
	}
	for _, option := range options {
		if option == nil {
			continue
		}
		option(p)
	}
	return p, nil
}

func (p *Provider) Provide(ctx context.Context, track *domain.Track) (playback.AudioStream, error) {
	if track == nil {
		return nil, fmt.Errorf("track is nil")
	}

	if r, ok := p.cache.Reader(track.VideoID); ok {
		return newStream(r, p.demuxerLogLogger)
	}

	r, err := p.pipeline.Start(ctx, track)
	if err != nil {
		return nil, err
	}

	return newStream(r, p.demuxerLogLogger)
}

func resolveYtDlpPath() (string, error) {
	return exec.LookPath("yt-dlp")
}
