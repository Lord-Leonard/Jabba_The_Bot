package music

import (
	"Jabba_The_Bot/internal/music/provider"
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"
)

type Manager struct {
	mu        sync.Mutex
	cond      *sync.Cond
	queue     []queueItem
	eventSink EventSink

	provider *provider.YouTubeProvider
	out      playbackController

	workerRunning bool
	current       *playbackControl

	autoplayEnabled    bool
	autoplayMinQueue   int
	autoplayBatchSize  int
	autoplayPrefetch   int
	prefetchTimeout    time.Duration
	prefetchDelay      time.Duration
	prefetchLead       time.Duration
	prefetchMinElapsed time.Duration
	prefetchFallback   time.Duration
	prefetchPoll       time.Duration
	autoplayKnownIDs   map[string]struct{}
	autoplayInFlight   map[string]struct{}

	playbackState string
	nowPlaying    *provider.Track
	progressTick  time.Duration
}

type playbackController interface {
	PlayOpus(source provider.AudioStream) error
}

// Todo: remove Query from queue Item. At all times a queue Item is determinisic! No search at play time.
// TODO: also remove requestedAt and IsAuto, who the fuck cares.
type queueItem struct {
	Query     string
	Track     *provider.Track
	IsAuto    bool
	Requested time.Time
}

type playbackControl struct {
	stop   chan struct{}
	done   chan struct{}
	once   sync.Once
	stream provider.AudioStream
}

type ManagerOption func(*Manager)

func WithEventSink(sink EventSink) ManagerOption {
	return func(m *Manager) {
		m.eventSink = sink
	}
}

func NewManager(ytp *provider.YouTubeProvider, out playbackController, options ...ManagerOption) *Manager {
	m := &Manager{
		provider: ytp,
		queue:    make([]queueItem, 0),

		out: out,

		autoplayEnabled:    true,
		autoplayMinQueue:   2,
		autoplayBatchSize:  5,
		autoplayPrefetch:   2,
		prefetchTimeout:    90 * time.Second,
		prefetchDelay:      12 * time.Second,
		prefetchLead:       25 * time.Second,
		prefetchMinElapsed: 20 * time.Second,
		prefetchFallback:   60 * time.Second,
		prefetchPoll:       1 * time.Second,
		autoplayKnownIDs:   make(map[string]struct{}),
		autoplayInFlight:   make(map[string]struct{}),
		playbackState:      "stopped",
		progressTick:       1 * time.Second,
	}
	for _, option := range options {
		option(m)
	}
	m.cond = sync.NewCond(&m.mu)
	return m
}

type QueueSnapshotItem struct {
	Query      string
	TrackTitle string
	VideoID    string
	URL        string
	IsAuto     bool
	Requested  time.Time
}

type NowPlayingSnapshot struct {
	Title    string
	VideoID  string
	URL      string
	Position time.Duration
	Duration time.Duration
}

func (m *Manager) SetOnFirstBytes(fn func(videoID, title string, duration time.Duration, cacheHit bool)) {
	m.provider.SetOnFirstBytes(fn)
}

func (m *Manager) Enqueue(query string) (int, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return 0, fmt.Errorf("query cannot be empty")
	}

	m.mu.Lock()
	m.queue = append(m.queue, queueItem{
		Query:     query,
		Track:     nil,
		IsAuto:    false,
		Requested: time.Now(),
	})
	position := len(m.queue)
	m.cond.Signal()

	if !m.workerRunning {
		m.workerRunning = true
		go m.playbackLoop()
	}
	m.mu.Unlock()

	m.emitEvent(QueueUpdatedEvent{QueueSize: len(m.QueueSnapshot())})
	slog.Info("music queued", "query", query, "position", position)
	return position, nil
}

func (m *Manager) QueueLength() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.queue)
}

func (m *Manager) ClearQueue() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	qlen := len(m.queue)
	m.queue = m.queue[:0]

	return qlen
}

func (m *Manager) Skip() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current == nil {
		return false
	}
	m.current.once.Do(func() {
		close(m.current.stop)
	})
	return true
}

func (m *Manager) Stop() (bool, int) {
	m.mu.Lock()
	cleared := len(m.queue)
	if cleared > 0 {
		m.queue = m.queue[:0]
	}

	stopped := false
	if m.current != nil {
		m.current.once.Do(func() {
			close(m.current.stop)
			stopped = true
		})
	}
	stateChanged := m.playbackState != "stopped"
	nowPlayingCleared := m.nowPlaying != nil
	m.playbackState = "stopped"
	m.nowPlaying = nil
	m.mu.Unlock()

	if stateChanged {
		m.emitEvent(PlaybackStateChangedEvent{PlaybackState: "stopped"})
	}
	if nowPlayingCleared {
		m.emitEvent(NowPlayingEvent{Cleared: true})
	}
	if cleared > 0 {
		m.emitEvent(QueueUpdatedEvent{QueueSize: 0})
	}

	return stopped, cleared
}

func (m *Manager) Seek(target time.Duration) error {
	m.mu.Lock()
	ctrl := m.current
	var stream provider.AudioStream
	if ctrl != nil {
		stream = ctrl.stream
	}
	m.mu.Unlock()

	if ctrl == nil || stream == nil {
		return fmt.Errorf("nothing is currently playing")
	}
	if target < 0 {
		target = 0
	}

	seekable, ok := stream.(provider.SeekableAudioStream)
	if !ok {
		return fmt.Errorf("current stream is not seekable")
	}
	if err := seekable.Seek(target); err != nil {
		return err
	}
	m.emitProgress()
	return nil
}

func (m *Manager) Start(query string) error {
	_, err := m.Enqueue(query)
	return err
}

func (m *Manager) playbackLoop() {
	for {
		item, ok := m.dequeue()
		if !ok {
			return
		}
		m.playItem(item)
	}
}

func (m *Manager) dequeue() (queueItem, bool) {
	m.mu.Lock()
	for len(m.queue) == 0 {
		m.cond.Wait()
	}

	item := m.queue[0]
	m.queue = m.queue[1:]
	queueSize := len(m.queue)
	m.mu.Unlock()

	m.emitEvent(QueueUpdatedEvent{QueueSize: queueSize})
	return item, true
}

func (m *Manager) playItem(item queueItem) {
	started := time.Now()
	m.setPlaybackState("loading")
	slog.Info("music start requested", "query", item.Query, "autoplay", item.IsAuto)

	ctrl := &playbackControl{stop: make(chan struct{}), done: make(chan struct{})}
	m.setCurrent(ctrl)
	defer func() {
		close(ctrl.done)
		m.clearCurrent(ctrl)
	}()

	var track provider.Track
	if item.Track != nil {
		track = *item.Track
	} else {
		searchStarted := time.Now()
		tracks, err := m.provider.Search(context.Background(), item.Query)
		if err != nil {
			slog.Error("music search failed", "query", item.Query, "error", err)
			return
		}
		slog.Info("music search completed", "query", item.Query, "results", len(tracks), "duration", time.Since(searchStarted))

		if len(tracks) == 0 {
			m.setPlaybackState("stopped")
			slog.Error("no tracks found", "query", item.Query)
			return
		}
		if ctrl.stopped() {
			m.setPlaybackState("stopped")
			slog.Info("music skipped before stream start", "query", item.Query)
			return
		}

		track = tracks[0]
	}
	m.maybeTriggerAutoplay(track)

	cacheHit := m.provider.HasCachedTrack(track.VideoID)
	if item.IsAuto {
		slog.Info("autoplay track selected", "videoID", track.VideoID, "title", track.Title, "cacheHit", cacheHit)
	}

	streamInitStarted := time.Now()
	err := track.StartStream()
	if err != nil {
		m.setPlaybackState("stopped")
		slog.Error("could not start stream", "query", item.Query, "videoID", track.VideoID, "error", err)
		return
	}
	slog.Info("music stream initialized", "videoID", track.VideoID, "title", track.Title, "duration", time.Since(streamInitStarted))
	if ctrl.stopped() {
		m.setPlaybackState("stopped")
		slog.Info("music skipped before playback", "videoID", track.VideoID)
		return
	}
	m.setNowPlaying(track)
	m.setPlaybackState("playing")
	go m.emitProgressLoop(ctrl)

	stream := &interruptibleStream{
		source: track.Stream,
		stop:   ctrl.stop,
	}
	m.setCurrentStream(ctrl, stream)
	go m.prefetchNearEnd(ctrl, track, stream)
	err = m.out.PlayOpus(stream)
	if err != nil {
		slog.Error("Error playing opus", "error", err, "videoID", track.VideoID)
	}
	m.setPlaybackState("stopped")
	m.clearNowPlaying(track.VideoID)
	slog.Info("music playback finished", "videoID", track.VideoID, "duration", time.Since(started))
}

func (m *Manager) maybeTriggerAutoplay(track provider.Track) {
	if !m.autoplayEnabled || strings.TrimSpace(track.VideoID) == "" {
		return
	}

	m.mu.Lock()
	if len(m.queue) >= m.autoplayMinQueue {
		m.mu.Unlock()
		return
	}
	if _, busy := m.autoplayInFlight[track.VideoID]; busy {
		m.mu.Unlock()
		return
	}
	m.autoplayInFlight[track.VideoID] = struct{}{}
	m.mu.Unlock()

	go func(seed provider.Track) {
		defer func() {
			m.mu.Lock()
			delete(m.autoplayInFlight, seed.VideoID)
			m.mu.Unlock()
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		recs, err := m.provider.Recommend(ctx, seed.VideoID, m.autoplayBatchSize*2)
		if err != nil {
			slog.Warn("autoplay recommendations failed", "seedVideoID", seed.VideoID, "error", err)
			return
		}
		if len(recs) == 0 {
			return
		}

		added := 0
		prefetchCandidates := make([]provider.Track, 0, m.autoplayPrefetch)
		for i := range recs {
			if m.enqueueAutoTrack(recs[i]) {
				added++
				if len(prefetchCandidates) < m.autoplayPrefetch {
					prefetchCandidates = append(prefetchCandidates, recs[i])
				}
				if added >= m.autoplayBatchSize {
					break
				}
			}
		}
		if added > 0 {
			slog.Info("autoplay queued recommendations", "seedVideoID", seed.VideoID, "added", added)
		}
		if len(prefetchCandidates) > 0 {
			slog.Info("autoplay prefetch deferred", "count", len(prefetchCandidates))
		}
	}(track)
}

func (m *Manager) enqueueAutoTrack(track provider.Track) bool {
	if strings.TrimSpace(track.VideoID) == "" {
		return false
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, seen := m.autoplayKnownIDs[track.VideoID]; seen {
		return false
	}
	for _, q := range m.queue {
		if q.Track != nil && q.Track.VideoID == track.VideoID {
			return false
		}
	}

	m.autoplayKnownIDs[track.VideoID] = struct{}{}
	trackCopy := track
	m.queue = append(m.queue, queueItem{
		Query:     track.Title,
		Track:     &trackCopy,
		IsAuto:    true,
		Requested: time.Now(),
	})
	m.cond.Signal()
	return true
}

func (m *Manager) prefetchTracks(tracks []provider.Track) {
	go func(items []provider.Track) {
		if m.prefetchDelay > 0 {
			time.Sleep(m.prefetchDelay)
		}
		for i := range items {
			track := items[i]
			ctx, cancel := context.WithTimeout(context.Background(), m.prefetchTimeout)
			slog.Info("autoplay prefetch attempt", "videoID", track.VideoID, "title", track.Title)
			if err := m.provider.Prefetch(ctx, track); err != nil {
				slog.Warn("autoplay prefetch failed", "videoID", track.VideoID, "error", err)
			}
			cancel()
		}
	}(tracks)
}

func (m *Manager) prefetchNearEnd(ctrl *playbackControl, track provider.Track, stream provider.AudioStream) {
	if m.autoplayPrefetch <= 0 {
		return
	}

	target := m.prefetchFallback
	if timed, ok := stream.(provider.TimedAudioStream); ok {
		duration := timed.Duration()
		if duration <= 0 {
			duration = 0
		}
		lead := m.prefetchLead
		if lead < 0 {
			lead = 0
		}
		if duration > 0 {
			target = duration - lead
			if target < m.prefetchMinElapsed {
				target = m.prefetchMinElapsed
			}
		} else {
			target = m.prefetchFallback
		}
	}
	if target < 0 {
		target = 0
	}

	if _, ok := stream.(provider.TimedAudioStream); !ok {
		timer := time.NewTimer(target)
		defer timer.Stop()
		select {
		case <-ctrl.stop:
			return
		case <-timer.C:
			m.prefetchNextTracks()
			return
		}
	}

	ticker := time.NewTicker(m.prefetchPoll)
	defer ticker.Stop()

	for {
		select {
		case <-ctrl.stop:
			return
		case <-ticker.C:
			elapsed := stream.(provider.TimedAudioStream).Position()
			if elapsed >= target {
				slog.Info("prefetch window reached", "videoID", track.VideoID, "elapsed", elapsed, "target", target)
				m.prefetchNextTracks()
				return
			}
		}
	}
}

func (m *Manager) prefetchNextTracks() {
	targets := m.collectPrefetchTargets(m.autoplayPrefetch)
	if len(targets) == 0 {
		return
	}

	for _, track := range targets {
		ctx, cancel := context.WithTimeout(context.Background(), m.prefetchTimeout)
		slog.Info("prefetch attempt", "videoID", track.VideoID, "title", track.Title)
		if err := m.provider.Prefetch(ctx, track); err != nil {
			slog.Warn("prefetch failed", "videoID", track.VideoID, "error", err)
		}
		cancel()
	}
}

func (m *Manager) collectPrefetchTargets(max int) []provider.Track {
	if max <= 0 {
		return nil
	}

	m.mu.Lock()
	queueLen := len(m.queue)
	m.mu.Unlock()
	if queueLen == 0 {
		return nil
	}

	targets := make([]provider.Track, 0, max)
	for i := 0; i < queueLen && len(targets) < max; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		track, ok := m.resolveQueueTrack(ctx, i)
		cancel()
		if !ok {
			continue
		}
		if m.provider.HasCachedTrack(track.VideoID) {
			continue
		}
		targets = append(targets, track)
	}
	return targets
}

func (m *Manager) resolveQueueTrack(ctx context.Context, idx int) (provider.Track, bool) {
	m.mu.Lock()
	if idx < 0 || idx >= len(m.queue) {
		m.mu.Unlock()
		return provider.Track{}, false
	}
	item := m.queue[idx]
	if item.Track != nil {
		track := *item.Track
		m.mu.Unlock()
		return track, true
	}
	query := item.Query
	m.mu.Unlock()
	if strings.TrimSpace(query) == "" {
		return provider.Track{}, false
	}

	results, err := m.provider.Search(ctx, query)
	if err != nil || len(results) == 0 {
		if err != nil {
			slog.Warn("prefetch search failed", "query", query, "error", err)
		}
		return provider.Track{}, false
	}

	track := results[0]

	m.mu.Lock()
	if idx < len(m.queue) && m.queue[idx].Track == nil && m.queue[idx].Query == query {
		trackCopy := track
		m.queue[idx].Track = &trackCopy
	}
	m.mu.Unlock()

	return track, true
}

func (m *Manager) setCurrent(ctrl *playbackControl) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.current = ctrl
}

func (m *Manager) clearCurrent(ctrl *playbackControl) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current == ctrl {
		m.current = nil
	}
}

func (m *Manager) setCurrentStream(ctrl *playbackControl, stream provider.AudioStream) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current == ctrl {
		m.current.stream = stream
	}
}

func (m *Manager) PlaybackState() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.playbackState
}

func (m *Manager) QueueSnapshot() []QueueSnapshotItem {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]QueueSnapshotItem, 0, len(m.queue))
	for _, item := range m.queue {
		snapshot := QueueSnapshotItem{
			Query:     item.Query,
			IsAuto:    item.IsAuto,
			Requested: item.Requested,
		}
		if item.Track != nil {
			snapshot.TrackTitle = item.Track.Title
			snapshot.VideoID = item.Track.VideoID
			snapshot.URL = item.Track.URL
		}
		out = append(out, snapshot)
	}
	return out
}

func (m *Manager) NowPlayingSnapshot() *NowPlayingSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.nowPlaying == nil {
		return nil
	}
	track := *m.nowPlaying
	var pos time.Duration
	var dur time.Duration
	if m.current != nil {
		if timed, ok := m.current.stream.(provider.TimedAudioStream); ok {
			pos = timed.Position()
			dur = timed.Duration()
		}
	}

	return &NowPlayingSnapshot{
		Title:    track.Title,
		VideoID:  track.VideoID,
		URL:      track.URL,
		Position: pos,
		Duration: dur,
	}
}

func (m *Manager) setPlaybackState(state string) {
	m.mu.Lock()
	changed := m.playbackState != state
	m.playbackState = state
	m.mu.Unlock()

	if changed {
		m.emitEvent(PlaybackStateChangedEvent{PlaybackState: state})
	}
}

func (m *Manager) setNowPlaying(track provider.Track) {
	m.mu.Lock()
	trackCopy := track
	m.nowPlaying = &trackCopy
	m.mu.Unlock()

	m.emitNowPlaying(trackCopy)
}

func (m *Manager) clearNowPlaying(videoID string) {
	m.mu.Lock()
	cleared := false
	if m.nowPlaying != nil && m.nowPlaying.VideoID == videoID {
		m.nowPlaying = nil
		cleared = true
	}
	m.mu.Unlock()

	if cleared {
		m.emitEvent(NowPlayingEvent{Cleared: true})
	}
}

func (m *Manager) emitEvent(event Event) {
	if m.eventSink == nil {
		return
	}
	m.eventSink.HandleEvent(event)
}

func (m *Manager) emitNowPlaying(track provider.Track) {
	m.emitEvent(NowPlayingEvent{
		TrackID: track.VideoID,
		Title:   track.Title,
		Source:  track.URL,
		Cleared: false,
	})
}

func (m *Manager) emitProgressLoop(ctrl *playbackControl) {
	if m.progressTick <= 0 {
		return
	}
	ticker := time.NewTicker(m.progressTick)
	defer ticker.Stop()

	for {
		select {
		case <-ctrl.stop:
			return
		case <-ctrl.done:
			return
		case <-ticker.C:
			m.emitProgress()
		}
	}
}

func (m *Manager) emitProgress() {
	now := m.NowPlayingSnapshot()
	if now == nil {
		return
	}
	m.emitEvent(ProgressEvent{
		TrackID:     now.VideoID,
		PositionSec: int64(now.Position.Seconds()),
		DurationSec: int64(now.Duration.Seconds()),
		ServerTime:  time.Now().UTC(),
	})
}

func (c *playbackControl) stopped() bool {
	select {
	case <-c.stop:
		return true
	default:
		return false
	}
}

type interruptibleStream struct {
	source provider.AudioStream
	stop   <-chan struct{}
	once   sync.Once
}

func (s *interruptibleStream) ProvideFrame() ([]byte, error) {
	select {
	case <-s.stop:
		s.once.Do(func() {
			if c, ok := s.source.(interface{ Close() error }); ok {
				_ = c.Close()
			}
		})
		return nil, io.EOF
	default:
	}
	return s.source.ProvideFrame()
}

func (s *interruptibleStream) Duration() time.Duration {
	if timed, ok := s.source.(provider.TimedAudioStream); ok {
		return timed.Duration()
	}
	return 0
}

func (s *interruptibleStream) Position() time.Duration {
	if timed, ok := s.source.(provider.TimedAudioStream); ok {
		return timed.Position()
	}
	return 0
}

func (s *interruptibleStream) Seek(target time.Duration) error {
	seekable, ok := s.source.(provider.SeekableAudioStream)
	if !ok {
		return fmt.Errorf("underlying stream is not seekable")
	}
	return seekable.Seek(target)
}
