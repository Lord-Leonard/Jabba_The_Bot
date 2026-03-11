package queue

import "Jabba_The_Bot/internal/music/domain"

type Subscription struct {
	OnEnqueue func(track domain.Track)
	OnDequeue func(track domain.Track)
	OnRemove  func(track domain.Track)
	OnClear   func()
	OnSet     func([]domain.Track)
}

func (q *Queue) Subscribe(subscription Subscription) {
	q.mu.Lock()
	q.subscriptions = append(q.subscriptions, subscription)
	q.mu.Unlock()
}

func (q *Queue) emitEnqueue(t domain.Track) {
	q.mu.Lock()
	subs := make([]Subscription, len(q.subscriptions))
	copy(subs, q.subscriptions)
	q.mu.Unlock()

	for _, sub := range subs {
		if sub.OnEnqueue == nil {
			continue
		}
		go sub.OnEnqueue(t)
	}
}

func (q *Queue) emitDequeue(t domain.Track) {
	q.mu.Lock()
	subs := make([]Subscription, len(q.subscriptions))
	copy(subs, q.subscriptions)
	q.mu.Unlock()

	for _, sub := range subs {
		if sub.OnDequeue == nil {
			continue
		}
		go sub.OnDequeue(t)
	}
}

func (q *Queue) emitRemove(t domain.Track) {
	q.mu.Lock()
	subs := make([]Subscription, len(q.subscriptions))
	copy(subs, q.subscriptions)
	q.mu.Unlock()

	for _, sub := range subs {
		if sub.OnRemove == nil {
			continue
		}
		go sub.OnRemove(t)
	}
}

func (q *Queue) emitClear() {
	q.mu.Lock()
	subs := make([]Subscription, len(q.subscriptions))
	copy(subs, q.subscriptions)
	q.mu.Unlock()

	for _, sub := range subs {
		if sub.OnClear == nil {
			continue
		}
		go sub.OnClear()
	}
}

func (q *Queue) emitSet() {
	q.mu.Lock()
	subs := make([]Subscription, len(q.subscriptions))
	copy(subs, q.subscriptions)
	queueSnapshot := make([]domain.Track, len(q.queue))
	copy(queueSnapshot, q.queue)
	q.mu.Unlock()

	for _, sub := range subs {
		if sub.OnSet == nil {
			continue
		}
		go sub.OnSet(queueSnapshot)
	}
}
