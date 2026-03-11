package playback

import (
	"time"
)

type AudioStream interface {
	ProvideFrame() ([]byte, error)
}

type TimedAudioStream interface {
	AudioStream
	Duration() time.Duration
	Position() time.Duration
}

type SeekableAudioStream interface {
	TimedAudioStream
	Seek(target time.Duration) error
}
