package daemon

import (
	"context"
	"sync"
	"time"

	"logy/internal/events"
)

// EventSink persists a batch of events.
type EventSink interface {
	AppendEvents(ctx context.Context, evts []events.Event) error
}

// Batcher groups events to reduce SQLite write frequency.
type Batcher struct {
	mu       sync.Mutex
	sink     EventSink
	maxSize  int
	interval time.Duration
	buf      []events.Event
}

// NewBatcher returns a batcher that flushes at maxSize or when Flush is called.
func NewBatcher(sink EventSink, maxSize int, interval time.Duration) *Batcher {
	if maxSize <= 0 {
		maxSize = 32
	}
	if interval <= 0 {
		interval = 5 * time.Second
	}
	return &Batcher{sink: sink, maxSize: maxSize, interval: interval}
}

// Add appends events and flushes when the size limit is reached.
func (b *Batcher) Add(ctx context.Context, evts []events.Event) error {
	if b == nil || b.sink == nil || len(evts) == 0 {
		return nil
	}
	b.mu.Lock()
	b.buf = append(b.buf, evts...)
	shouldFlush := len(b.buf) >= b.maxSize
	b.mu.Unlock()
	if shouldFlush {
		return b.Flush(ctx)
	}
	return nil
}

// Flush writes any buffered events to the sink.
func (b *Batcher) Flush(ctx context.Context) error {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	if len(b.buf) == 0 {
		b.mu.Unlock()
		return nil
	}
	batch := b.buf
	b.buf = nil
	b.mu.Unlock()
	if b.sink == nil {
		return nil
	}
	return b.sink.AppendEvents(ctx, batch)
}
