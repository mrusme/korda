package events

import (
	"context"
	"sync"
)

type Listener func(ctx context.Context, event any)

type Bus struct {
	mu        sync.RWMutex
	listeners []Listener
}

func New() (*Bus, error) {
	return new(Bus), nil
}

func (bus *Bus) On(fn Listener) {
	bus.mu.Lock()
	bus.listeners = append(bus.listeners, fn)
	bus.mu.Unlock()
}

func (bus *Bus) Emit(ctx context.Context, event any) {
	bus.mu.RLock()
	listeners := bus.listeners
	bus.mu.RUnlock()

	for _, fn := range listeners {
		fn(context.WithoutCancel(ctx), event)
	}
}
