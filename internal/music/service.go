package music

import (
	"Jabba_The_Bot/internal/music/domain"
	"Jabba_The_Bot/internal/music/playback"
	"Jabba_The_Bot/internal/music/queue"
	"context"
	"time"
)

type Resolver interface {
	Resolve(ctx context.Context, querry string) (domain.Track, error)
}

type SearchProvider interface {
	Search(ctx context.Context, query string) ([]domain.Track, error)
}

type Service struct {
	resolver Resolver
	searcher SearchProvider
	queue    *queue.Queue
	engine   *playback.Engine
}

func NewMusicService(
	resolver Resolver,
	searcher SearchProvider,
	queue *queue.Queue,
	engine *playback.Engine,
) *Service {
	return &Service{
		resolver: resolver,
		searcher: searcher,
		queue:    queue,
		engine:   engine,
	}
}

func (s *Service) Resolve(ctx context.Context, querry string) (domain.Track, error) {
	return s.resolver.Resolve(ctx, querry)
}

func (s *Service) Play(ctx context.Context, querry string) error {
	track, err := s.resolver.Resolve(ctx, querry)
	if err != nil {
		return err
	}
	return s.queue.Enqueue(track)
}

func (s *Service) Search(ctx context.Context, querry string) ([]domain.Track, error) {
	return s.searcher.Search(ctx, querry)
}

// Not sure if I want to enqueue an entire Track! When coming from Teamspeak I only have a querry, and from UI i have a ISRC or videoID I guess
func (s *Service) Enqueue(track domain.Track) error {
	// Maybe resolve the track?
	return s.queue.Enqueue(track)
}

func (s *Service) Skip()                             { s.engine.Skip() }
func (s *Service) Stop()                             { s.engine.Stop() }
func (s *Service) Pause()                            { s.engine.Pause() }
func (s *Service) Resume()                           { s.engine.Resume() }
func (s *Service) Seek(position time.Duration) error { return s.engine.Seek(position) }
func (s *Service) SetVolume(vol int)                 { s.engine.Volume(vol) }

func (s *Service) NowPlaying() *NowPlayingSnapshot {
	track := s.engine.CurrentTrack()
	if track == nil {
		return nil
	}

	currentStream := s.engine.CurrentStream()
	var pos, dur time.Duration
	if timed, ok := currentStream.(playback.TimedAudioStream); ok {
		pos = timed.Position()
		dur = timed.Duration()
	}
	return &NowPlayingSnapshot{
		Title:       track.Title,
		VideoID:     track.VideoID,
		URL:         track.VideoID,
		CoverArtURL: track.CoverUrl,
		Position:    pos,
		Duration:    dur,
	}
}

func (s *Service) State() playback.EngineState {
	return s.engine.State()
}

func (s *Service) Queue() []domain.Track {
	return s.queue.Queue()
}

type NowPlayingSnapshot struct {
	Title       string
	VideoID     string
	URL         string
	CoverArtURL string
	Position    time.Duration
	Duration    time.Duration
}
