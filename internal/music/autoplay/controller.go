package autoplay

import (
	"Jabba_The_Bot/internal/music/domain"
	"Jabba_The_Bot/internal/music/queue"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

type LogLevel int

const (
	LogOff LogLevel = iota
	LogInfo
	LogDebug
)

const (
	defaultTargetQueueSize  = 20
	defaultMaxSeeds         = 30
	seedRecencyWindow       = 10
	backgroundRefillTTL     = 10 * time.Second
	defaultSeedWeight       = 1.0
	autoplayCycleContextKey = "autoplay_cycle_id"
)

type ControllerOption func(*Controller)

func WithLogLevel(level LogLevel) ControllerOption {
	return func(c *Controller) {
		c.logLevel = level
	}
}

func WithLogger(logger *slog.Logger) ControllerOption {
	return func(c *Controller) {
		if logger != nil {
			c.logger = logger
		}
	}
}

func WithMaxSeeds(maxSeeds int) ControllerOption {
	return func(c *Controller) {
		if maxSeeds > 0 {
			c.maxSeeds = maxSeeds
		}
	}
}

func ParseLogLevel(value string) (LogLevel, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "off", "none", "disabled":
		return LogOff, nil
	case "info":
		return LogInfo, nil
	case "debug", "verbose":
		return LogDebug, nil
	default:
		return LogOff, fmt.Errorf("unknown autoplay log level %q (use off|info|debug)", value)
	}
}

type seedEntry struct {
	Track  domain.Track
	Weight float64
}

type seedWeightDetail struct {
	Track         domain.Track
	BaseWeight    float64
	RecencyFactor float64
	Weighted      float64
	Normalized    float64
}

type Controller struct {
	queue       *queue.Queue
	recommender RecommendationProvider

	seeds    []seedEntry // ordered oldest -> newest
	rejected map[string]struct{}

	seedsMu     sync.Mutex
	replenishMu sync.Mutex

	targetSize int
	maxSeeds   int

	logger   *slog.Logger
	logLevel LogLevel
}

func NewController(autoplayQueue *queue.Queue, recommender RecommendationProvider, opts ...ControllerOption) *Controller {
	controller := &Controller{
		queue:       autoplayQueue,
		recommender: recommender,
		seeds:       make([]seedEntry, 0),
		rejected:    make(map[string]struct{}),
		seedsMu:     sync.Mutex{},
		replenishMu: sync.Mutex{},
		targetSize:  defaultTargetQueueSize,
		maxSeeds:    defaultMaxSeeds,
		logger:      slog.Default(),
		logLevel:    LogOff,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(controller)
		}
	}

	// Keep autoplay queue length steady by topping up after autoplay tracks are consumed.
	autoplayQueue.Subscribe(queue.Subscription{
		OnDequeue: func(track domain.Track) {
			controller.addTrack(track, "autoplay_dequeue", false)
			controller.triggerBackgroundRefill()
		},
	})

	return controller
}

func (a *Controller) Reject(t domain.Track) {
	videoID := strings.TrimSpace(t.VideoID)
	if videoID == "" {
		return
	}

	a.seedsMu.Lock()
	a.rejected[videoID] = struct{}{}
	a.seedsMu.Unlock()

	a.queue.Remove(t)
}

func (a *Controller) AddTrack(track domain.Track) {
	a.addTrack(track, "user_enqueue", true)
}

func (a *Controller) addTrack(track domain.Track, source string, infoLevel bool) {
	videoID := strings.TrimSpace(track.VideoID)
	if videoID == "" {
		return
	}

	a.seedsMu.Lock()
	defer a.seedsMu.Unlock()

	// If seed already exists, refresh metadata and move it to the most recent position.
	for i := range a.seeds {
		if a.seeds[i].Track.VideoID != videoID {
			continue
		}

		entry := a.seeds[i]
		entry.Track = track
		a.seeds = append(a.seeds[:i], a.seeds[i+1:]...)
		a.seeds = append(a.seeds, entry)
		a.logByLevel(infoLevel, "autoplay seed refreshed", "source", source, "track", describeTrack(track), "weight", entry.Weight, "seeds_total", len(a.seeds))
		return
	}

	a.seeds = append(a.seeds, seedEntry{
		Track:  track,
		Weight: defaultSeedWeight,
	})
	if a.maxSeeds > 0 && len(a.seeds) > a.maxSeeds {
		excess := len(a.seeds) - a.maxSeeds
		dropped := append([]seedEntry(nil), a.seeds[:excess]...)
		a.seeds = append([]seedEntry(nil), a.seeds[excess:]...)
		a.logDebug("autoplay seeds trimmed", "source", source, "dropped_count", len(dropped), "dropped_tracks", summarizeSeedEntries(dropped, 8))
	}

	a.logByLevel(infoLevel, "autoplay seed added", "source", source, "track", describeTrack(track), "weight", defaultSeedWeight, "seeds_total", len(a.seeds))
}

func (a *Controller) RemoveTrack(track domain.Track) {
	videoID := strings.TrimSpace(track.VideoID)
	if videoID == "" {
		return
	}

	a.seedsMu.Lock()
	defer a.seedsMu.Unlock()

	for i := range a.seeds {
		if a.seeds[i].Track.VideoID != videoID {
			continue
		}
		a.seeds = append(a.seeds[:i], a.seeds[i+1:]...)
		break
	}
}

func (a *Controller) SetSeedWeight(videoID string, weight float64) error {
	videoID = strings.TrimSpace(videoID)
	if videoID == "" {
		return fmt.Errorf("video ID cannot be empty")
	}
	if weight <= 0 {
		return fmt.Errorf("weight must be > 0")
	}

	a.seedsMu.Lock()
	defer a.seedsMu.Unlock()

	for i := range a.seeds {
		if a.seeds[i].Track.VideoID != videoID {
			continue
		}
		track := a.seeds[i].Track
		a.seeds[i].Weight = weight
		a.logInfo("autoplay seed weight updated", "track", describeTrack(track), "weight", weight)
		return nil
	}

	return fmt.Errorf("seed %q not found", videoID)
}

func (a *Controller) currentSeeds() []domain.WeightedSeed {
	a.seedsMu.Lock()
	defer a.seedsMu.Unlock()

	if len(a.seeds) == 0 {
		return nil
	}

	activeSeeds := make([]seedEntry, 0, len(a.seeds))
	for _, seed := range a.seeds {
		videoID := strings.TrimSpace(seed.Track.VideoID)
		if videoID == "" {
			continue
		}
		activeSeeds = append(activeSeeds, seed)
	}
	if len(activeSeeds) == 0 {
		return nil
	}
	if len(activeSeeds) > seedRecencyWindow {
		activeSeeds = append([]seedEntry(nil), activeSeeds[len(activeSeeds)-seedRecencyWindow:]...)
	}

	result := make([]domain.WeightedSeed, 0, len(activeSeeds))
	details := make([]seedWeightDetail, 0, len(activeSeeds))
	total := 0.0

	totalSeeds := len(activeSeeds)
	for i, seed := range activeSeeds {
		baseWeight := seed.Weight
		if baseWeight <= 0 {
			baseWeight = defaultSeedWeight
		}
		recencyFactor := linearRecencyFactor(i, totalSeeds)
		weighted := baseWeight * recencyFactor

		result = append(result, domain.WeightedSeed{
			Track:  seed.Track,
			Weight: weighted,
		})
		details = append(details, seedWeightDetail{
			Track:         seed.Track,
			BaseWeight:    baseWeight,
			RecencyFactor: recencyFactor,
			Weighted:      weighted,
		})
		total += weighted
	}

	if len(result) == 0 {
		return nil
	}

	if total <= 0 {
		uniform := 1.0 / float64(len(result))
		for i := range result {
			result[i].Weight = uniform
			if i < len(details) {
				details[i].Normalized = uniform
			}
		}
		if a.logLevel >= LogDebug {
			a.logDebug("autoplay seeds normalized", "seeds", summarizeSeedWeights(details))
		}
		return result
	}

	for i := range result {
		result[i].Weight = result[i].Weight / total
		if i < len(details) {
			details[i].Normalized = result[i].Weight
		}
	}

	if a.logLevel >= LogDebug {
		a.logDebug("autoplay seeds normalized", "seeds", summarizeSeedWeights(details))
	}

	return result
}

func linearRecencyFactor(index, total int) float64 {
	if total <= 1 {
		return 1.0
	}
	if index < 0 || index >= total {
		return 1.0
	}
	// Oldest in window gets 1/total, newest gets 1.0 (before normalization).
	return float64(index+1) / float64(total)
}

func (a *Controller) rejectedIDs() map[string]struct{} {
	a.seedsMu.Lock()
	defer a.seedsMu.Unlock()

	out := make(map[string]struct{}, len(a.rejected))
	for id := range a.rejected {
		out[id] = struct{}{}
	}
	return out
}

func (a *Controller) triggerBackgroundRefill() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), backgroundRefillTTL)
		defer cancel()

		if err := a.ensureTarget(ctx, false); err != nil {
			a.logWarn("autoplay refill failed", "mode", modeName(false), "error", err)
		}
	}()
}

func (a *Controller) Update(ctx context.Context) error {
	return a.ensureTarget(ctx, true)
}

func (a *Controller) ensureTarget(ctx context.Context, rebuild bool) error {
	a.replenishMu.Lock()
	defer a.replenishMu.Unlock()

	cycleID := fmt.Sprintf("%s-%d", modeName(rebuild), time.Now().UnixNano())
	ctx = context.WithValue(ctx, autoplayCycleContextKey, cycleID)

	seeds := a.currentSeeds()
	if len(seeds) == 0 {
		a.logDebug("autoplay skipped: no seeds", "cycle_id", cycleID, "mode", modeName(rebuild))
		return nil
	}

	recommendations, err := a.recommender.Recommend(ctx, seeds)
	if err != nil {
		a.logWarn("autoplay recommendation fetch failed", "cycle_id", cycleID, "mode", modeName(rebuild), "error", err)
		return err
	}
	if len(recommendations) == 0 {
		a.logWarn("autoplay recommendation fetch returned no tracks", "cycle_id", cycleID, "mode", modeName(rebuild), "seeds_count", len(seeds))
		return nil
	}
	a.logInfo(
		"autoplay recommendations fetched",
		"cycle_id", cycleID,
		"mode", modeName(rebuild),
		"seeds_count", len(seeds),
		"recommendations_count", len(recommendations),
		"seeds", summarizeWeightedSeeds(seeds),
	)
	a.logDebug("autoplay recommendation sample", "cycle_id", cycleID, "mode", modeName(rebuild), "tracks", summarizeTracks(recommendations, 8))

	blocked := a.rejectedIDs()
	seedIDs := make(map[string]struct{}, len(seeds))
	for _, seed := range seeds {
		id := strings.TrimSpace(seed.Track.VideoID)
		if id == "" {
			continue
		}
		seedIDs[id] = struct{}{}
	}

	existing := make(map[string]struct{})
	queueLenBefore := len(a.queue.Queue())
	if !rebuild {
		for _, track := range a.queue.Queue() {
			id := strings.TrimSpace(track.VideoID)
			if id == "" {
				continue
			}
			existing[id] = struct{}{}
		}
	}

	selected := make([]domain.Track, 0, a.targetSize)
	skippedMissingID := 0
	skippedSeed := 0
	skippedRejected := 0
	skippedAlreadyQueued := 0
	for _, candidate := range recommendations {
		id := strings.TrimSpace(candidate.VideoID)
		if id == "" {
			skippedMissingID++
			continue
		}
		if _, isSeed := seedIDs[id]; isSeed {
			skippedSeed++
			continue
		}
		if _, isRejected := blocked[id]; isRejected {
			skippedRejected++
			continue
		}
		if _, isAlreadyQueued := existing[id]; isAlreadyQueued {
			skippedAlreadyQueued++
			continue
		}

		existing[id] = struct{}{}
		selected = append(selected, candidate)
		if len(selected) >= a.targetSize {
			break
		}
	}
	a.logDebug(
		"autoplay candidate filtering",
		"cycle_id", cycleID,
		"mode", modeName(rebuild),
		"eligible_candidates", len(selected),
		"skipped_missing_id", skippedMissingID,
		"skipped_seed", skippedSeed,
		"skipped_rejected", skippedRejected,
		"skipped_already_queued", skippedAlreadyQueued,
		"queue_len_before", queueLenBefore,
		"selected_tracks", summarizeTracks(selected, 10),
	)

	if rebuild {
		a.queue.Clear()
		for _, track := range selected {
			if err := a.queue.Enqueue(track); err != nil {
				return err
			}
			a.logDebug("autoplay track enqueued", "cycle_id", cycleID, "mode", modeName(rebuild), "track", describeTrack(track))
		}
		if len(selected) < a.targetSize {
			a.logWarn(
				"autoplay queue under target after rebuild",
				"cycle_id", cycleID,
				"enqueued", len(selected),
				"target", a.targetSize,
			)
		}
		queueLenAfter := len(a.queue.Queue())
		a.logInfo(
			"autoplay queue rebuilt",
			"cycle_id", cycleID,
			"enqueued", len(selected),
			"target", a.targetSize,
			"queue_len_before", queueLenBefore,
			"queue_len_after", queueLenAfter,
			"tracks", summarizeTracks(selected, 10),
		)
		return nil
	}

	currentLen := len(a.queue.Queue())
	if currentLen >= a.targetSize {
		a.logDebug("autoplay top-up skipped: queue already at target", "cycle_id", cycleID, "queue_len_before", currentLen, "target", a.targetSize)
		return nil
	}

	needed := a.targetSize - currentLen
	added := 0
	addedTracks := make([]domain.Track, 0, needed)
	for _, track := range selected {
		if added >= needed {
			break
		}
		if err := a.queue.Enqueue(track); err != nil {
			return err
		}
		added++
		addedTracks = append(addedTracks, track)
		a.logDebug("autoplay track enqueued", "cycle_id", cycleID, "mode", modeName(rebuild), "track", describeTrack(track))
	}

	if added < needed {
		a.logWarn(
			"autoplay queue under target after top-up",
			"cycle_id", cycleID,
			"added", added,
			"needed", needed,
			"target", a.targetSize,
		)
	}
	queueLenAfter := len(a.queue.Queue())
	a.logInfo(
		"autoplay queue topped up",
		"cycle_id", cycleID,
		"added", added,
		"needed", needed,
		"target", a.targetSize,
		"queue_len_before", queueLenBefore,
		"queue_len_after", queueLenAfter,
		"tracks", summarizeTracks(addedTracks, 10),
	)

	return nil
}

func (a *Controller) SetTargetSize(size int) error {
	if size <= 0 {
		return fmt.Errorf("target size must be > 0")
	}

	a.replenishMu.Lock()
	defer a.replenishMu.Unlock()
	a.targetSize = size
	a.logInfo("autoplay target size updated", "target", size)
	return nil
}

func (a *Controller) logInfo(msg string, args ...any) {
	a.log(slog.LevelInfo, msg, args...)
}

func (a *Controller) logByLevel(info bool, msg string, args ...any) {
	if info {
		a.logInfo(msg, args...)
		return
	}
	a.logDebug(msg, args...)
}

func (a *Controller) logDebug(msg string, args ...any) {
	a.log(slog.LevelDebug, msg, args...)
}

func (a *Controller) logWarn(msg string, args ...any) {
	a.log(slog.LevelWarn, msg, args...)
}

func (a *Controller) log(level slog.Level, msg string, args ...any) {
	if a == nil || a.logger == nil {
		return
	}
	switch level {
	case slog.LevelDebug:
		if a.logLevel < LogDebug {
			return
		}
	default:
		if a.logLevel < LogInfo {
			return
		}
	}
	a.logger.Log(context.Background(), level, msg, args...)
}

func modeName(rebuild bool) string {
	if rebuild {
		return "rebuild"
	}
	return "topup"
}

func summarizeWeightedSeeds(seeds []domain.WeightedSeed) string {
	parts := make([]string, 0, len(seeds))
	for _, s := range seeds {
		parts = append(parts, fmt.Sprintf("%s weight=%.4f", describeTrack(s.Track), s.Weight))
	}
	return strings.Join(parts, ", ")
}

func summarizeSeedWeights(details []seedWeightDetail) string {
	parts := make([]string, 0, len(details))
	for _, detail := range details {
		parts = append(parts, fmt.Sprintf(
			"%s base=%.3f recency=%.3f weighted=%.3f normalized=%.4f",
			describeTrack(detail.Track),
			detail.BaseWeight,
			detail.RecencyFactor,
			detail.Weighted,
			detail.Normalized,
		))
	}
	return strings.Join(parts, " | ")
}

func summarizeSeedEntries(entries []seedEntry, limit int) string {
	if limit <= 0 {
		limit = 1
	}
	if len(entries) < limit {
		limit = len(entries)
	}
	if limit == 0 {
		return ""
	}

	parts := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		parts = append(parts, fmt.Sprintf("%s weight=%.3f", describeTrack(entries[i].Track), entries[i].Weight))
	}
	return strings.Join(parts, ", ")
}

func summarizeTracks(tracks []domain.Track, limit int) string {
	if limit <= 0 {
		limit = 1
	}
	if len(tracks) < limit {
		limit = len(tracks)
	}
	if limit == 0 {
		return ""
	}

	parts := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		parts = append(parts, describeTrack(tracks[i]))
	}
	return strings.Join(parts, ", ")
}

func describeTrack(track domain.Track) string {
	title := strings.TrimSpace(track.Title)
	artist := strings.TrimSpace(track.Artist)
	videoID := strings.TrimSpace(track.VideoID)

	switch {
	case title != "" && artist != "" && videoID != "":
		return fmt.Sprintf("%s - %s (%s)", title, artist, videoID)
	case title != "" && artist != "":
		return fmt.Sprintf("%s - %s", title, artist)
	case title != "" && videoID != "":
		return fmt.Sprintf("%s (%s)", title, videoID)
	case title != "":
		return title
	case videoID != "":
		return videoID
	default:
		return "<unknown track>"
	}
}
