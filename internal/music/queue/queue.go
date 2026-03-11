package queue

import (
	"Jabba_The_Bot/internal/music/domain"
	"sync"
)

type Queue struct {
	queue []domain.Track

	subscriptions []Subscription

	mu    sync.Mutex
	ready chan struct{}
}

func New() *Queue {
	return &Queue{
		queue:         make([]domain.Track, 0),
		ready:         make(chan struct{}),
		subscriptions: make([]Subscription, 0),
	}
}

func (q *Queue) Enqueue(track domain.Track) error {
	q.mu.Lock()
	q.queue = append(q.queue, track)
	q.mu.Unlock()

	select {
	case q.ready <- struct{}{}:
	default:
	}

	q.emitEnqueue(track)

	return nil
}

func (q *Queue) TryDequeue() (*domain.Track, bool) {
	q.mu.Lock()
	if len(q.queue) == 0 {
		q.mu.Unlock()
		return nil, false
	}
	item := q.queue[0]
	q.queue = q.queue[1:]
	q.mu.Unlock()

	q.emitDequeue(item)
	return &item, true
}

func (q *Queue) Queue() []domain.Track {
	q.mu.Lock()
	defer q.mu.Unlock()
	result := make([]domain.Track, len(q.queue))
	copy(result, q.queue)
	return result
}

func (q *Queue) Ready() chan struct{} {
	return q.ready
}

func (q *Queue) Remove(t domain.Track) {
	q.mu.Lock()
	var removed *domain.Track
	for i, track := range q.queue {
		if track.VideoID == t.VideoID {
			q.queue = append(q.queue[:i], q.queue[i+1:]...)
			removed = &track
			break
		}
	}

	q.mu.Unlock()

	if removed != nil {
		q.emitRemove(*removed)
	}
}

func (q *Queue) Clear() {
	q.mu.Lock()
	q.queue = q.queue[:0]
	q.mu.Unlock()

	q.emitClear()
}

func (q *Queue) Set(newQueue []domain.Track) {
	q.mu.Lock()
	q.queue = make([]domain.Track, len(newQueue))
	copy(q.queue, newQueue)
	q.mu.Unlock()

	if len(newQueue) > 0 {
		select {
		case q.ready <- struct{}{}:
		default:
		}
	}

	q.emitSet()
}
