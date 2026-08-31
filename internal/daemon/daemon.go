package daemon

import (
	"context"
	"errors"
	"io"
	"log"
	"sync"
	"time"

	"logy/internal/events"
)

// Collector produces normalized events for the daemon to persist.
type Collector interface {
	Name() string
	Collect(ctx context.Context) ([]events.Event, error)
}

// Options configure a daemon instance.
type Options struct {
	LockPath        string
	FlushSize       int
	FlushInterval   time.Duration
	CollectInterval time.Duration
	Sink            EventSink
	Collectors      []Collector
	Logger          *log.Logger
}

// Status is a snapshot of daemon lifecycle state.
type Status struct {
	Running    bool
	StartedAt  time.Time
	Collectors []string
}

// Daemon owns collectors, batching, and the singleton lock.
type Daemon struct {
	opts      Options
	batcher   *Batcher
	logger    *log.Logger
	mu        sync.Mutex
	lock      *Lock
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	running   bool
	startedAt time.Time
}

// New constructs a daemon. Start acquires the lock.
func New(opts Options) (*Daemon, error) {
	if opts.LockPath == "" {
		return nil, errors.New("lock path required")
	}
	if opts.CollectInterval <= 0 {
		opts.CollectInterval = time.Minute
	}
	if opts.FlushInterval <= 0 {
		opts.FlushInterval = 5 * time.Second
	}
	logger := opts.Logger
	if logger == nil {
		logger = log.New(io.Discard, "logy: ", 0)
	}
	return &Daemon{
		opts:    opts,
		batcher: NewBatcher(opts.Sink, opts.FlushSize, opts.FlushInterval),
		logger:  logger,
	}, nil
}

// Start acquires the singleton lock and begins collector and flush loops.
func (d *Daemon) Start(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.running {
		return ErrAlreadyRunning
	}
	lock, err := AcquireLock(d.opts.LockPath)
	if err != nil {
		return err
	}
	runCtx, cancel := context.WithCancel(ctx)
	d.lock = lock
	d.cancel = cancel
	d.running = true
	d.startedAt = time.Now()
	d.wg.Add(2)
	go d.flushLoop(runCtx)
	go d.collectLoop(runCtx)
	return nil
}

// Stop cancels loops, flushes remaining events, and releases the lock.
func (d *Daemon) Stop(ctx context.Context) error {
	d.mu.Lock()
	if !d.running {
		d.mu.Unlock()
		return nil
	}
	cancel := d.cancel
	d.running = false
	d.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	done := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		d.wg.Wait()
	}

	flushErr := d.batcher.Flush(context.Background())
	d.mu.Lock()
	lock := d.lock
	d.lock = nil
	d.cancel = nil
	d.mu.Unlock()
	var releaseErr error
	if lock != nil {
		releaseErr = lock.Release()
	}
	if flushErr != nil {
		return flushErr
	}
	return releaseErr
}

// Status reports whether the daemon is running.
func (d *Daemon) Status(context.Context) Status {
	d.mu.Lock()
	defer d.mu.Unlock()
	names := make([]string, 0, len(d.opts.Collectors))
	for _, collector := range d.opts.Collectors {
		names = append(names, collector.Name())
	}
	return Status{
		Running:    d.running,
		StartedAt:  d.startedAt,
		Collectors: names,
	}
}

func (d *Daemon) collectLoop(ctx context.Context) {
	defer d.wg.Done()
	d.runCollectors(ctx)
	ticker := time.NewTicker(d.opts.CollectInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.runCollectors(ctx)
		}
	}
}

func (d *Daemon) flushLoop(ctx context.Context) {
	defer d.wg.Done()
	ticker := time.NewTicker(d.opts.FlushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := d.batcher.Flush(ctx); err != nil {
				d.logger.Printf("flush: %v", err)
			}
		}
	}
}

func (d *Daemon) runCollectors(ctx context.Context) {
	for _, collector := range d.opts.Collectors {
		if ctx.Err() != nil {
			return
		}
		evts, err := collector.Collect(ctx)
		if err != nil {
			d.logger.Printf("collector %s: %v", collector.Name(), err)
			continue
		}
		if err := d.batcher.Add(ctx, evts); err != nil {
			d.logger.Printf("batcher: %v", err)
		}
	}
}
