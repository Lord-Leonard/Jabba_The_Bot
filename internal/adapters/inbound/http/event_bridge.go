package http

import (
	"context"
	"log/slog"

	eventbusadapter "Jabba_The_Bot/internal/adapters/outbound/eventbus"
	"Jabba_The_Bot/internal/music"
)

func BridgeAppEventsToSSE(ctx context.Context, sub eventbusadapter.Subscriber, broker *SSEBroker) {
	if sub == nil || broker == nil {
		return
	}

	events, cancel := sub.Subscribe(64)
	go func() {
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-events:
				if !ok {
					return
				}
				sseEvent, ok := mapMusicEventToSSE(event)
				if !ok {
					continue
				}
				broker.Publish(sseEvent)
			}
		}
	}()

	slog.Debug("SSE bridge subscribed to app event bus")
}

func mapMusicEventToSSE(event music.Event) (SSEEvent, bool) {
	switch e := event.(type) {
	case music.QueueUpdatedEvent:
		return SSEEvent{
			Type: e.Name(),
			Data: map[string]any{"queueSize": e.QueueSize},
		}, true
	case music.PlaybackStateChangedEvent:
		return SSEEvent{
			Type: e.Name(),
			Data: map[string]any{"playbackState": e.PlaybackState},
		}, true
	case music.NowPlayingEvent:
		if e.Cleared {
			return SSEEvent{
				Type: e.Name(),
				Data: map[string]any{"nowPlaying": nil},
			}, true
		}
		return SSEEvent{
			Type: e.Name(),
			Data: map[string]any{
				"trackId": e.TrackID,
				"title":   e.Title,
				"source":  e.Source,
			},
		}, true
	case music.ProgressEvent:
		return SSEEvent{
			Type: e.Name(),
			Data: map[string]any{
				"trackId":     e.TrackID,
				"positionSec": e.PositionSec,
				"durationSec": e.DurationSec,
				"serverTs":    e.ServerTime.Unix(),
			},
		}, true
	default:
		return SSEEvent{}, false
	}
}
