package ports

import (
	"Jabba_The_Bot/internal/core/app/dto"
	"context"
	"time"
)

type PlaybackState string

const (
	PlaybackStateStopped PlaybackState = "stopped"
	PlaybackStateLoading PlaybackState = "loading"
	PlaybackStatePlaying PlaybackState = "playing"
	PlaybackStatePaused  PlaybackState = "paused"
)

type RequestSource struct {
	Platform string
	UserID   string
}

type QueuePosition int

type QueueItemView struct {
	ID          string    `json:"id,omitempty"`
	Title       string    `json:"title,omitempty"`
	Source      string    `json:"source,omitempty"`
	CoverURL    string    `json:"coverUrl,omitempty"`
	DurationSec int64     `json:"duration,omitempty"`
	RequestedBy string    `json:"requestedBy,omitempty"`
	RequestedAt time.Time `json:"requestedAt,omitempty"`
}

type NowPlayingView struct {
	ID          string `json:"id,omitempty"`
	Title       string `json:"title,omitempty"`
	Source      string `json:"source,omitempty"`
	CoverURL    string `json:"coverUrl,omitempty"`
	PositionSec int64  `json:"position,omitempty"`
	DurationSec int64  `json:"duration,omitempty"`
}

type StateView struct {
	PlaybackState PlaybackState `json:"playbackState"`
	IsConnected   bool          `json:"isConnected"`
	Volume        int           `json:"volume,omitempty"`
}

type MusicCommands interface {
	Enqueue(query string, source RequestSource) (QueuePosition, error)
	Skip() error
	Seek(position time.Duration) error
	Pause() error
	Resume() error
	Stop(clearQueue bool) error
	Remove(itemID string) error
	Move(itemID string, toIndex int) error
}

type MusicQueries interface {
	GetState() StateView
	GetQueue() []QueueItemView
	GetNowPlaying() *NowPlayingView
}

type MusicSearchQueries interface {
	Search(ctx context.Context, query string, limit int) ([]dto.SearchResult, error)
}
