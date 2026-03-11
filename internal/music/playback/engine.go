package playback

import (
	"Jabba_The_Bot/internal/music/domain"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

type EngineState int

const (
	EngineStateStopped EngineState = iota
	EngineStatePlaying
	EngineStatePaused
)

type Engine struct {
	trackProvider TrackProvider
	audioProvider AudioProvider
	sink          AudioSink

	currentStream AudioStream
	currentTrack  *domain.Track
	state         EngineState

	pauseCh chan struct{}
	skipCh  chan struct{}
	stopCh  chan struct{}
}

func NewEngine(tp TrackProvider, sink AudioSink, ap AudioProvider) *Engine {
	return &Engine{
		state: EngineStateStopped,

		trackProvider: tp,
		audioProvider: ap,
		sink:          sink,

		pauseCh: make(chan struct{}),
		skipCh:  make(chan struct{}, 1),
		stopCh:  make(chan struct{}),
	}
}

func (e *Engine) loop(ctx context.Context) {
	for {
		track, err := e.trackProvider.Next(ctx)
		if err != nil {
			if errors.Is(err, ErrQueueEmpty) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				e.currentTrack = nil
				e.currentStream = nil
				e.state = EngineStateStopped
				return
			}
			slog.Error("Error loading next track", "error", err)
			continue
		}
		if track == nil {
			slog.Error("Track provider returned nil track")
			e.currentTrack = nil
			e.currentStream = nil
			e.state = EngineStateStopped
			continue
		}
		e.currentTrack = track

		source, err := e.audioProvider.Provide(ctx, track)
		if err != nil {
			slog.Error("Error creating audio stream", "error", err, "videoID", track.VideoID)
			e.currentTrack = nil
			e.currentStream = nil
			e.state = EngineStateStopped
			continue
		}
		if source == nil {
			slog.Error("Audio provider returned nil stream", "videoID", track.VideoID)
			e.currentTrack = nil
			e.currentStream = nil
			e.state = EngineStateStopped
			continue
		}
		e.currentStream = source

		wrapped := newInterruptibleStream(source, e.skipCh)

		e.state = EngineStatePlaying

		err = e.sink.Consume(wrapped)
		if err != nil {
			slog.Error("Error playing opus", "error", err, "videoID", track.VideoID)
		}

		e.currentTrack = nil
		e.currentStream = nil
		e.state = EngineStateStopped
		drainSignal(e.skipCh)
	}
}

func (e *Engine) Start(ctx context.Context) {
	go e.loop(ctx)
}

func (e *Engine) Stop() {
	// Stop the playback loop and empty the queue?
	// How do I actually clear the playback loop? Or do I just stop the Engine Loop?
}

func (e *Engine) Pause() {
	e.state = EngineStatePaused
	// Pause the playback loop until further notice?
}

func (e *Engine) Resume() {
	e.state = EngineStatePlaying
	// Resume the engine
	// No shit sherlok
}

func (e *Engine) Skip() {
	if e.state != EngineStatePlaying {
		return
	}
	select {
	case e.skipCh <- struct{}{}:
	default:
	}
}

func (e *Engine) Seek(target time.Duration) error {
	if seekable, ok := e.currentStream.(SeekableAudioStream); ok {
		return seekable.Seek(target)
	}
	return fmt.Errorf("current source is not seekable")
}

func (e *Engine) Volume(vol int) {
	// Set the gain so the OPUS Transcoder can apply it
}

func (e *Engine) State() EngineState {
	return e.state
}

func (e *Engine) CurrentTrack() *domain.Track {
	if e.state == EngineStateStopped {
		return nil
	}
	return e.currentTrack
}

func (e *Engine) CurrentStream() AudioStream {
	if e.state == EngineStateStopped {
		return nil
	}
	return e.currentStream
}

func drainSignal(ch chan struct{}) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}
