package http

import (
	"Jabba_The_Bot/internal/music"
	"Jabba_The_Bot/internal/music/domain"
	"Jabba_The_Bot/internal/music/playback"
	"net/http"
)

type PlaybackState string

const (
	PlaybackStateStopped PlaybackState = "stopped"
	PlaybackStateLoading PlaybackState = "loading"
	PlaybackStatePlaying PlaybackState = "playing"
	PlaybackStatePaused  PlaybackState = "paused"
)

type StateView struct {
	PlaybackState PlaybackState `json:"playbackState"`
	IsConnected   bool          `json:"isConnected"`
	Volume        int           `json:"volume,omitempty"`
}

func ToPlaybackState(e playback.EngineState) PlaybackState {
	switch e {
	case playback.EngineStateStopped:
		return PlaybackStateStopped
	case playback.EngineStatePlaying:
		return PlaybackStatePlaying
	case playback.EngineStatePaused:
		return PlaybackStatePaused
	default:
		return PlaybackStateStopped
	}
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	sv := StateView{
		PlaybackState: ToPlaybackState(s.svc.State()),
		IsConnected:   true,
		Volume:        100,
	}

	writeJSON(w, http.StatusOK, sv) // See inbound.go StateView
}

func (s *Server) handleQueue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	writeJSON(w, http.StatusOK, s.queueWithProxiedCover())
}

func (s *Server) handleNowPlaying(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	writeJSON(w, http.StatusOK, s.nowPlayingWithProxiedCover())
}

func (s *Server) queueWithProxiedCover() []domain.Track {
	currentQueue := s.svc.Queue()
	items := make([]domain.Track, len(currentQueue))
	copy(items, currentQueue)
	for i := range items {
		items[i].CoverUrl = coverProxyURL(items[i].CoverUrl, items[i].VideoID)
	}
	return items
}

func (s *Server) nowPlayingWithProxiedCover() *music.NowPlayingSnapshot {
	nowPlaying := s.svc.NowPlaying()
	if nowPlaying == nil {
		return nil
	}

	cp := *nowPlaying
	cp.CoverArtURL = coverProxyURL(cp.CoverArtURL, cp.VideoID)
	return &cp
}
