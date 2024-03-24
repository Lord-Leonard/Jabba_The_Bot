package eventbus

import (
	"sync"

	"Jabba_The_Bot/internal/music"
)

type Publisher interface {
	Publish(event music.Event)
}

type Subscriber interface {
	Subscribe(buffer int) (<-chan music.Event, func())
}

type Bus interface {
	Publisher
	Subscriber
}

type InProcBus struct {
	mu          sync.RWMutex
	subscribers map[chan music.Event]struct{}
}

func NewInProcBus() Bus {
	return &InProcBus{
		subscribers: make(map[chan music.Event]struct{}),
	}
}

func (b *InProcBus) Publish(event music.Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.subscribers {
		select {
		case ch <- event:
		default:
		}
	}
}

func (b *InProcBus) Subscribe(buffer int) (<-chan music.Event, func()) {
	if buffer <= 0 {
		buffer = 16
	}
	ch := make(chan music.Event, buffer)

	b.mu.Lock()
	b.subscribers[ch] = struct{}{}
	b.mu.Unlock()

	cancel := func() {
		b.mu.Lock()
		if _, ok := b.subscribers[ch]; ok {
			delete(b.subscribers, ch)
			close(ch)
		}
		b.mu.Unlock()
	}

	return ch, cancel
}
