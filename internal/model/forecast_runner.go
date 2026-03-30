package model

import (
	"context"
	"sync"
	"time"
)

type ForecastEventType string

const (
	ForecastEventSnapshot ForecastEventType = "snapshot"
	ForecastEventDone     ForecastEventType = "done"
	ForecastEventStatus   ForecastEventType = "status"
)

type ForecastCacheRow struct {
	Date          int64
	AccountTypeID string
	Median        float64
	LowerBound    float64
	UpperBound    float64
}

type ForecastEvent struct {
	Type     ForecastEventType
	Snapshot *ForecastCacheRow
}

type ForecastRunner struct {
	debounce time.Duration
	runFn    func(ctx context.Context)

	mu       sync.Mutex
	timer    *time.Timer
	cancelFn context.CancelFunc
	stopped  bool

	runningMu sync.RWMutex
	running   bool

	subscribersMu sync.RWMutex
	subscribers   []chan ForecastEvent
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

func (r *ForecastRunner) Subscribe() chan ForecastEvent {
	ch := make(chan ForecastEvent, 64)
	r.subscribersMu.Lock()
	r.subscribers = append(r.subscribers, ch)
	r.subscribersMu.Unlock()
	return ch
}

func (r *ForecastRunner) Unsubscribe(ch chan ForecastEvent) {
	r.subscribersMu.Lock()
	defer r.subscribersMu.Unlock()
	for i, sub := range r.subscribers {
		if sub == ch {
			r.subscribers = append(r.subscribers[:i], r.subscribers[i+1:]...)
			close(ch)
			return
		}
	}
}

func (r *ForecastRunner) Broadcast(evt ForecastEvent) {
	r.subscribersMu.RLock()
	defer r.subscribersMu.RUnlock()
	for _, ch := range r.subscribers {
		select {
		case ch <- evt:
		default:
			// Drop if subscriber is slow
		}
	}
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
