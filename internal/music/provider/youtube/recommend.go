package youtube

import (
	"Jabba_The_Bot/internal/music/domain"
	youtubepkg "Jabba_The_Bot/pkg/youtube"
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
)

type RecommendationLogLevel int

const (
	RecommendationLogOff RecommendationLogLevel = iota
	RecommendationLogInfo
	RecommendationLogDebug
)

const (
	rankSmoothingK = 60.0
	overlapBonus   = 0.05

	autoplayCycleContextKey = "autoplay_cycle_id"
)

type scoredCandidate struct {
	track   domain.Track
	score   float64
	seedHit int
}

func ParseRecommendationLogLevel(value string) (RecommendationLogLevel, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "off", "none", "disabled":
		return RecommendationLogOff, nil
	case "info":
		return RecommendationLogInfo, nil
	case "debug", "verbose":
		return RecommendationLogDebug, nil
	default:
		return RecommendationLogOff, fmt.Errorf("unknown recommendation log level %q (use off|info|debug)", value)
	}
}

func (p *Provider) Recommend(ctx context.Context, seeds []domain.WeightedSeed) ([]domain.Track, error) {
	activeSeeds := normalizeSeeds(seeds)
	if len(activeSeeds) == 0 {
		return nil, fmt.Errorf("at least one seed with video ID is required")
	}
	p.logRecommendationDebug(ctx, "recommendation seeds", "seeds", summarizeWeightedSeeds(activeSeeds))

	seedIDs := make(map[string]struct{}, len(activeSeeds))
	for _, s := range activeSeeds {
		seedIDs[s.Track.VideoID] = struct{}{}
	}

	candidates := make(map[string]*scoredCandidate)

	var firstErr error
	for _, seed := range activeSeeds {
		results, err := p.client.GetNextSongs(ctx, seed.Track.VideoID)
		if err != nil {
			p.logRecommendationWarn(
				ctx,
				"failed fetching recommendation seed",
				"seed_track", describeTrack(seed.Track),
				"seed_weight", seed.Weight,
				"error", err,
			)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		p.logRecommendationDebug(
			ctx,
			"seed recommendations fetched",
			"seed_track", describeTrack(seed.Track),
			"seed_weight", seed.Weight,
			"results_count", len(results),
			"sample", summarizeSearchResults(results, 6),
		)

		weight := seed.Weight
		for rank, res := range results {
			videoID := strings.TrimSpace(res.VideoID)
			if videoID == "" {
				continue
			}
			if _, isSeed := seedIDs[videoID]; isSeed {
				continue
			}

			rel := 1.0 / (rankSmoothingK + float64(rank) + 1.0)
			if existing, ok := candidates[videoID]; ok {
				existing.score += weight * rel
				existing.seedHit++
				continue
			}

			candidates[videoID] = &scoredCandidate{
				track: domain.Track{
					Title:    res.Title,
					Artist:   res.Artist,
					Album:    res.Album,
					VideoID:  res.VideoID,
					CoverUrl: res.CoverArtUrl,
				},
				score:   weight * rel,
				seedHit: 1,
			}
		}
	}

	if len(candidates) == 0 {
		if firstErr != nil {
			return nil, firstErr
		}
		return []domain.Track{}, nil
	}

	scored := make([]scoredCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.seedHit > 1 {
			candidate.score += overlapBonus
		}
		scored = append(scored, *candidate)
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return scored[i].track.VideoID < scored[j].track.VideoID
		}
		return scored[i].score > scored[j].score
	})

	out := make([]domain.Track, 0, len(scored))
	for _, item := range scored {
		out = append(out, item.track)
	}
	p.logRecommendationInfo(
		ctx,
		"recommendations scored",
		"seeds_count", len(activeSeeds),
		"candidate_count", len(candidates),
		"result_count", len(out),
		"top", summarizeTopCandidates(scored, 5),
	)
	p.logRecommendationDebug(ctx, "recommendation top results", "top", summarizeTopCandidates(scored, 10))

	return out, nil
}

func normalizeSeeds(seeds []domain.WeightedSeed) []domain.WeightedSeed {
	if len(seeds) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(seeds))
	out := make([]domain.WeightedSeed, 0, len(seeds))
	total := 0.0

	for _, seed := range seeds {
		videoID := strings.TrimSpace(seed.Track.VideoID)
		if videoID == "" {
			continue
		}
		if _, ok := seen[videoID]; ok {
			continue
		}

		weight := seed.Weight
		if weight <= 0 {
			continue
		}

		seen[videoID] = struct{}{}
		out = append(out, domain.WeightedSeed{
			Track:  seed.Track,
			Weight: weight,
		})
		total += weight
	}

	if len(out) == 0 {
		return nil
	}
	if total <= 0 {
		uniform := 1.0 / float64(len(out))
		for i := range out {
			out[i].Weight = uniform
		}
		return out
	}
	for i := range out {
		out[i].Weight = out[i].Weight / total
	}

	return out
}

func (p *Provider) logRecommendationInfo(ctx context.Context, msg string, args ...any) {
	if p.recommendationLogLevel < RecommendationLogInfo {
		return
	}
	p.logRecommendation(ctx, slog.LevelInfo, msg, args...)
}

func (p *Provider) logRecommendationWarn(ctx context.Context, msg string, args ...any) {
	if p.recommendationLogLevel < RecommendationLogInfo {
		return
	}
	p.logRecommendation(ctx, slog.LevelWarn, msg, args...)
}

func (p *Provider) logRecommendationDebug(ctx context.Context, msg string, args ...any) {
	if p.recommendationLogLevel < RecommendationLogDebug {
		return
	}
	p.logRecommendation(ctx, slog.LevelDebug, msg, args...)
}

func (p *Provider) logRecommendation(ctx context.Context, level slog.Level, msg string, args ...any) {
	if p == nil || p.recommendationLogLogger == nil {
		return
	}
	p.recommendationLogLogger.Log(context.Background(), level, msg, withAutoplayCycleID(ctx, args...)...)
}

func withAutoplayCycleID(ctx context.Context, args ...any) []any {
	if ctx == nil {
		return args
	}
	cycleID, ok := ctx.Value(autoplayCycleContextKey).(string)
	if !ok || strings.TrimSpace(cycleID) == "" {
		return args
	}

	withCycleID := make([]any, 0, len(args)+2)
	withCycleID = append(withCycleID, "cycle_id", cycleID)
	withCycleID = append(withCycleID, args...)
	return withCycleID
}

func summarizeWeightedSeeds(seeds []domain.WeightedSeed) string {
	parts := make([]string, 0, len(seeds))
	for _, seed := range seeds {
		parts = append(parts, fmt.Sprintf("%s weight=%.4f", describeTrack(seed.Track), seed.Weight))
	}
	return strings.Join(parts, ", ")
}

func summarizeTopCandidates(candidates []scoredCandidate, limit int) string {
	if limit <= 0 {
		limit = 1
	}
	if len(candidates) < limit {
		limit = len(candidates)
	}
	if limit == 0 {
		return ""
	}

	parts := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		c := candidates[i]
		parts = append(parts, fmt.Sprintf("%s score=%.5f seed_hits=%d", describeTrack(c.track), c.score, c.seedHit))
	}
	return strings.Join(parts, ", ")
}

func summarizeSearchResults(results []youtubepkg.SearchResult, limit int) string {
	if limit <= 0 {
		limit = 1
	}
	if len(results) < limit {
		limit = len(results)
	}
	if limit == 0 {
		return ""
	}

	parts := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		parts = append(parts, describeSearchResult(results[i]))
	}
	return strings.Join(parts, ", ")
}

func describeSearchResult(result youtubepkg.SearchResult) string {
	title := strings.TrimSpace(result.Title)
	artist := strings.TrimSpace(result.Artist)
	videoID := strings.TrimSpace(result.VideoID)

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
		return "<unknown recommendation>"
	}
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
