package scheduler

import (
	"Jabba_The_Bot/internal/music/domain"
	"Jabba_The_Bot/internal/music/queue"
	"context"
)

type Scheduler struct {
	user     *queue.Queue
	autoplay *queue.Queue
}

func NewScheduler(user, autoplay *queue.Queue) *Scheduler {
	return &Scheduler{
		user:     user,
		autoplay: autoplay,
	}
}

func (s *Scheduler) Next(ctx context.Context) (*domain.Track, error) {
	if track, ok := s.user.TryDequeue(); ok {
		return track, nil
	}

	return s.waitForNext(ctx)
}

func (s *Scheduler) waitForNext(ctx context.Context) (*domain.Track, error) {
	for {
		// Check queues first so we do not depend only on ready-channel edges.
		if track, ok := s.user.TryDequeue(); ok {
			return track, nil
		}
		if track, ok := s.autoplay.TryDequeue(); ok {
			return track, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-s.user.Ready():
			if track, ok := s.user.TryDequeue(); ok {
				return track, nil
			}
		case <-s.autoplay.Ready():
			// Try if userqueue didnt "sneak in" ? In which case can that happen xD
			if track, ok := s.user.TryDequeue(); ok {
				return track, nil
			}
			if track, ok := s.autoplay.TryDequeue(); ok {
				return track, nil
			}
		}
	}
}
