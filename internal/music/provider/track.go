package provider

import (
	"Jabba_The_Bot/pkg/audio/webm"
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

type Track struct {
	Title   string
	VideoID string
	URL     string
	Stream  AudioStream

	provider *YouTubeProvider
}

func NewTrack(title, videoID, url string) *Track {
	return &Track{
		Title:    title,
		VideoID:  videoID,
		URL:      url,
		provider: NewYouTubeProvider(),
	}
}

func (t *Track) StartStream() error {
	downloadBuffer, err := t.provider.Download(t)
	if err != nil {
		return err
	}

	demuxer, err := webm.NewDemuxer(downloadBuffer)
	if err != nil {
		return err
	}
	t.Stream = demuxer
	return nil
}
