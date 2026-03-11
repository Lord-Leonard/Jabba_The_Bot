package autoplay

import (
	"Jabba_The_Bot/internal/music/domain"
	"Jabba_The_Bot/internal/music/queue"
	"context"
	"fmt"
	"math"
	"testing"
	"time"
)

type stubRecommender struct{}

func (stubRecommender) Recommend(context.Context, []domain.WeightedSeed) ([]domain.Track, error) {
	return []domain.Track{}, nil
}

func TestControllerOnDequeueAddsSeed(t *testing.T) {
	autoplayQueue := queue.New()
	controller := NewController(autoplayQueue, stubRecommender{}, WithMaxSeeds(10))

	track := domain.Track{
		Title:   "Track 1",
		Artist:  "Artist 1",
		VideoID: "id-1",
	}
	if err := autoplayQueue.Enqueue(track); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	if _, ok := autoplayQueue.TryDequeue(); !ok {
		t.Fatal("expected track to dequeue")
	}

	waitFor(t, 1500*time.Millisecond, func() bool {
		controller.seedsMu.Lock()
		defer controller.seedsMu.Unlock()
		return len(controller.seeds) == 1 && controller.seeds[0].Track.VideoID == track.VideoID
	})
}

func TestControllerAddTrackTrimsSeedsToMax(t *testing.T) {
	autoplayQueue := queue.New()
	controller := NewController(autoplayQueue, stubRecommender{}, WithMaxSeeds(3))

	controller.AddTrack(domain.Track{Title: "Track 1", VideoID: "id-1"})
	controller.AddTrack(domain.Track{Title: "Track 2", VideoID: "id-2"})
	controller.AddTrack(domain.Track{Title: "Track 3", VideoID: "id-3"})
	controller.AddTrack(domain.Track{Title: "Track 4", VideoID: "id-4"})

	controller.seedsMu.Lock()
	defer controller.seedsMu.Unlock()

	if got := len(controller.seeds); got != 3 {
		t.Fatalf("expected 3 seeds, got %d", got)
	}

	seedIDs := make(map[string]struct{}, len(controller.seeds))
	for _, seed := range controller.seeds {
		seedIDs[seed.Track.VideoID] = struct{}{}
	}

	if _, ok := seedIDs["id-1"]; ok {
		t.Fatal("expected oldest seed id-1 to be trimmed")
	}
	if _, ok := seedIDs["id-2"]; !ok {
		t.Fatal("expected id-2 to remain in seeds")
	}
	if _, ok := seedIDs["id-3"]; !ok {
		t.Fatal("expected id-3 to remain in seeds")
	}
	if _, ok := seedIDs["id-4"]; !ok {
		t.Fatal("expected id-4 to remain in seeds")
	}
}

func TestCurrentSeedsUsesLastTenWithLinearRecency(t *testing.T) {
	autoplayQueue := queue.New()
	controller := NewController(autoplayQueue, stubRecommender{}, WithMaxSeeds(50))

	for i := 1; i <= 12; i++ {
		controller.AddTrack(domain.Track{
			Title:   "Track",
			VideoID: fmt.Sprintf("id-%d", i),
		})
	}

	seeds := controller.currentSeeds()
	if got := len(seeds); got != 10 {
		t.Fatalf("expected 10 weighted seeds, got %d", got)
	}

	// Oldest 2 are dropped from weighting window.
	if got := seeds[0].Track.VideoID; got != "id-3" {
		t.Fatalf("expected first weighted seed id-3, got %s", got)
	}
	if got := seeds[len(seeds)-1].Track.VideoID; got != "id-12" {
		t.Fatalf("expected last weighted seed id-12, got %s", got)
	}

	const sumLinearFactors = 55.0 // 1+2+...+10
	for i, seed := range seeds {
		expected := float64(i+1) / sumLinearFactors
		if math.Abs(seed.Weight-expected) > 1e-9 {
			t.Fatalf("unexpected weight at index %d: got %.10f expected %.10f", i, seed.Weight, expected)
		}
	}
}

func waitFor(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("condition not met before timeout")
}
