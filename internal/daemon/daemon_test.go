package daemon

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"logy/internal/events"
)

type recordingSink struct {
	mu     sync.Mutex
	events []events.Event
}

func (s *recordingSink) AppendEvents(_ context.Context, evts []events.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, evts...)
	return nil
}

func (s *recordingSink) all() []events.Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]events.Event, len(s.events))
	copy(out, s.events)
	return out
}

type stubCollector struct {
	name string
	fn   func(context.Context) ([]events.Event, error)
}

func (c stubCollector) Name() string { return c.name }

func (c stubCollector) Collect(ctx context.Context) ([]events.Event, error) {
	return c.fn(ctx)
}

func TestAcquireLockRejectsDuplicate(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "logy.lock")

	first, err := AcquireLock(path)
	if err != nil {
		t.Fatalf("AcquireLock() error = %v", err)
	}
	t.Cleanup(func() { _ = first.Release() })

	second, err := AcquireLock(path)
	if err == nil {
		_ = second.Release()
		t.Fatal("second AcquireLock() succeeded, want error")
	}
	if !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("second AcquireLock() error = %v, want ErrAlreadyRunning", err)
	}

	if err := first.Release(); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	third, err := AcquireLock(path)
	if err != nil {
		t.Fatalf("AcquireLock() after release error = %v", err)
	}
	if err := third.Release(); err != nil {
		t.Fatalf("third Release() error = %v", err)
	}
}

func TestBatcherFlushesAtSizeLimit(t *testing.T) {
	t.Parallel()
	sink := &recordingSink{}
	b := NewBatcher(sink, 2, time.Hour)

	started := time.Date(2026, time.August, 31, 10, 0, 0, 0, time.UTC)
	if err := b.Add(context.Background(), []events.Event{
		{ID: "1", StartedAt: started, Type: "git.commit"},
		{ID: "2", StartedAt: started, Type: "git.commit"},
	}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	got := sink.all()
	if len(got) != 2 {
		t.Fatalf("flushed %d events, want 2", len(got))
	}
}

func TestBatcherFlushesOnStop(t *testing.T) {
	t.Parallel()
	sink := &recordingSink{}
	b := NewBatcher(sink, 10, time.Hour)
	if err := b.Add(context.Background(), []events.Event{{ID: "pending", Type: "note"}}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if err := b.Flush(context.Background()); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}
	got := sink.all()
	if len(got) != 1 || got[0].ID != "pending" {
		t.Fatalf("Flush() = %#v, want pending event", got)
	}
}

func TestDaemonStartStopAndStatus(t *testing.T) {
	t.Parallel()
	sink := &recordingSink{}
	d, err := New(Options{
		LockPath:        filepath.Join(t.TempDir(), "logy.lock"),
		FlushSize:       8,
		FlushInterval:   time.Hour,
		CollectInterval: 20 * time.Millisecond,
		Sink:            sink,
		Collectors: []Collector{
			stubCollector{name: "git", fn: func(context.Context) ([]events.Event, error) {
				return []events.Event{{ID: "git-1", Type: "git.commit", Summary: "ok"}}, nil
			}},
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := d.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	status := d.Status(ctx)
	if !status.Running {
		t.Fatal("Status().Running = false, want true")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(sink.all()) > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if len(sink.all()) == 0 {
		t.Fatal("expected collected events before stop")
	}

	if err := d.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	status = d.Status(context.Background())
	if status.Running {
		t.Fatal("Status().Running = true after Stop")
	}
}

func TestDaemonIsolatesCollectorFailure(t *testing.T) {
	t.Parallel()
	sink := &recordingSink{}
	var goodRuns atomic.Int32
	d, err := New(Options{
		LockPath:        filepath.Join(t.TempDir(), "logy.lock"),
		FlushSize:       1,
		FlushInterval:   time.Hour,
		CollectInterval: 15 * time.Millisecond,
		Sink:            sink,
		Collectors: []Collector{
			stubCollector{name: "boom", fn: func(context.Context) ([]events.Event, error) {
				return nil, errors.New("collector exploded")
			}},
			stubCollector{name: "git", fn: func(context.Context) ([]events.Event, error) {
				goodRuns.Add(1)
				return []events.Event{{ID: "ok", Type: "git.commit"}}, nil
			}},
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := d.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() { _ = d.Stop(context.Background()) })

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if goodRuns.Load() > 0 && len(sink.all()) > 0 {
			break
		}
		time.Sleep(15 * time.Millisecond)
	}
	if goodRuns.Load() == 0 {
		t.Fatal("healthy collector did not run after sibling failure")
	}
	if len(sink.all()) == 0 {
		t.Fatal("expected events from healthy collector")
	}
	status := d.Status(context.Background())
	if !status.Running {
		t.Fatal("daemon stopped after collector failure")
	}
}

func TestDaemonDuplicateStartRejected(t *testing.T) {
	t.Parallel()
	lockPath := filepath.Join(t.TempDir(), "logy.lock")
	sink := &recordingSink{}
	first, err := New(Options{
		LockPath:        lockPath,
		FlushSize:       8,
		CollectInterval: time.Hour,
		Sink:            sink,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := first.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() { _ = first.Stop(context.Background()) })

	second, err := New(Options{
		LockPath:        lockPath,
		FlushSize:       8,
		CollectInterval: time.Hour,
		Sink:            sink,
	})
	if err != nil {
		t.Fatalf("second New() error = %v", err)
	}
	if err := second.Start(context.Background()); !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("second Start() error = %v, want ErrAlreadyRunning", err)
	}
}
