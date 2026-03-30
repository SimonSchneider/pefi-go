package model

import (
	"context"
	"sync"
	"time"
)

type ForecastRunner struct {
	debounce time.Duration
	runFn    func(ctx context.Context)

	mu       sync.Mutex
	timer    *time.Timer
	cancelFn context.CancelFunc
	stopped  bool

	runningMu sync.RWMutex
	running   bool
}

func NewForecastRunner(debounce time.Duration, runFn func(ctx context.Context)) *ForecastRunner {
	return &ForecastRunner{
		debounce: debounce,
		runFn:    runFn,
	}
}

func (r *ForecastRunner) Invalidate() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.stopped {
		return
	}

	// Cancel any in-progress run
	if r.cancelFn != nil {
		r.cancelFn()
	}

	// Reset or start debounce timer
	if r.timer != nil {
		r.timer.Stop()
	}
	r.timer = time.AfterFunc(r.debounce, r.startRun)
}

func (r *ForecastRunner) startRun() {
	r.mu.Lock()
	if r.stopped {
		r.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.cancelFn = cancel
	r.mu.Unlock()

	r.runningMu.Lock()
	r.running = true
	r.runningMu.Unlock()

	r.runFn(ctx)

	r.runningMu.Lock()
	r.running = false
	r.runningMu.Unlock()
}

func (r *ForecastRunner) IsRunning() bool {
	r.runningMu.RLock()
	defer r.runningMu.RUnlock()
	return r.running
}

func (r *ForecastRunner) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.stopped = true
	if r.timer != nil {
		r.timer.Stop()
	}
	if r.cancelFn != nil {
		r.cancelFn()
	}
}
