package http

import (
	"errors"
	"fmt"
	"time"

	"Jabba_The_Bot/internal/core/app/ports"
	"Jabba_The_Bot/internal/music"
)

type LegacyManagerBridge struct {
	manager *music.Manager
}

func NewLegacyManagerBridge(manager *music.Manager) *LegacyManagerBridge {
	return &LegacyManagerBridge{manager: manager}
}

func (b *LegacyManagerBridge) Enqueue(query string, _ ports.RequestSource) (ports.QueuePosition, error) {
	position, err := b.manager.Enqueue(query)
	if err != nil {
		return 0, err
	}
	return ports.QueuePosition(position), nil
}

func (b *LegacyManagerBridge) Skip() error {
	if !b.manager.Skip() {
		return errors.New("nothing to skip")
	}
	return nil
}

func (b *LegacyManagerBridge) Seek(position time.Duration) error {
	return b.manager.Seek(position)
}

func (b *LegacyManagerBridge) Pause() error {
	return errors.New("pause not implemented in current manager")
}

func (b *LegacyManagerBridge) Resume() error {
	return errors.New("resume not implemented in current manager")
}

func (b *LegacyManagerBridge) Stop(clearQueue bool) error {
	if !clearQueue {
		return errors.New("stop with clearQueue=false not implemented in current manager")
	}

	stopped, cleared := b.manager.Stop()
	if !stopped && cleared == 0 {
		return errors.New("nothing to stop")
	}
	return nil
}

func (b *LegacyManagerBridge) Remove(itemID string) error {
	return fmt.Errorf("remove not implemented in current manager: %s", itemID)
}

func (b *LegacyManagerBridge) Move(itemID string, toIndex int) error {
	return fmt.Errorf("move not implemented in current manager: %s -> %d", itemID, toIndex)
}

func (b *LegacyManagerBridge) GetState() ports.StateView {
	return ports.StateView{
		PlaybackState: ports.PlaybackState(b.manager.PlaybackState()),
		IsConnected:   true,
	}
}

func (b *LegacyManagerBridge) GetQueue() []ports.QueueItemView {
	snapshot := b.manager.QueueSnapshot()
	items := make([]ports.QueueItemView, 0, len(snapshot))
	for _, item := range snapshot {
		title := item.Query
		if item.TrackTitle != "" {
			title = item.TrackTitle
		}
		items = append(items, ports.QueueItemView{
			ID:          item.VideoID,
			Title:       title,
			Source:      item.URL,
			CoverURL:    youtubeCoverURL(item.VideoID),
			RequestedAt: item.Requested,
		})
	}
	return items
}

func (b *LegacyManagerBridge) GetNowPlaying() *ports.NowPlayingView {
	now := b.manager.NowPlayingSnapshot()
	if now == nil {
		return nil
	}
	return &ports.NowPlayingView{
		ID:          now.VideoID,
		Title:       now.Title,
		Source:      now.URL,
		CoverURL:    youtubeCoverURL(now.VideoID),
		PositionSec: int64(now.Position.Seconds()),
		DurationSec: int64(now.Duration.Seconds()),
	}
}
