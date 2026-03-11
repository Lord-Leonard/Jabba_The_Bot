package http

import (
	"Jabba_The_Bot/internal/music"
)

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
