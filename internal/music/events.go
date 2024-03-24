package music

import "time"

type Event interface {
	Name() string
}

type EventSink interface {
	HandleEvent(event Event)
}

type EventSinkFunc func(event Event)

func (f EventSinkFunc) HandleEvent(event Event) {
	f(event)
}

type QueueUpdatedEvent struct {
	QueueSize int
}

func (QueueUpdatedEvent) Name() string { return "queue_updated" }

type PlaybackStateChangedEvent struct {
	PlaybackState string
}

func (PlaybackStateChangedEvent) Name() string { return "playback_state_changed" }

type NowPlayingEvent struct {
	TrackID string
	Title   string
	Source  string
	Cleared bool
}

func (NowPlayingEvent) Name() string { return "now_playing" }

type ProgressEvent struct {
	TrackID     string
	PositionSec int64
	DurationSec int64
	ServerTime  time.Time
}

func (ProgressEvent) Name() string { return "progress" }
