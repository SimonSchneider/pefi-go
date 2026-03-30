package model_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/SimonSchneider/pefigo/internal/model"
)

func TestForecastRunnerDebounce(t *testing.T) {
	var runCount atomic.Int32
	runner := model.NewForecastRunner(50*time.Millisecond, func(ctx context.Context) {
		runCount.Add(1)
	})
	defer runner.Stop()

	// Multiple rapid invalidations should result in a single run
	runner.Invalidate()
	runner.Invalidate()
	runner.Invalidate()

	// Wait for debounce + execution
	time.Sleep(200 * time.Millisecond)

	if got := runCount.Load(); got != 1 {
		t.Fatalf("expected 1 run after debounce, got %d", got)
	}
}

func TestForecastRunnerCancelAndRestart(t *testing.T) {
	var started atomic.Int32
	var completed atomic.Int32
	runner := model.NewForecastRunner(50*time.Millisecond, func(ctx context.Context) {
		started.Add(1)
		select {
		case <-ctx.Done():
			return
		case <-time.After(500 * time.Millisecond):
			completed.Add(1)
		}
	})
	defer runner.Stop()

	// First invalidation triggers a run after debounce
	runner.Invalidate()
	time.Sleep(100 * time.Millisecond) // Let first run start

	// Second invalidation should cancel the first and start a new one
	runner.Invalidate()
	time.Sleep(600 * time.Millisecond) // Let second run complete

	if got := started.Load(); got != 2 {
		t.Fatalf("expected 2 starts, got %d", got)
	}
	if got := completed.Load(); got != 1 {
		t.Fatalf("expected 1 completion (first cancelled), got %d", got)
	}
}

func TestForecastRunnerStop(t *testing.T) {
	var runCount atomic.Int32
	runner := model.NewForecastRunner(50*time.Millisecond, func(ctx context.Context) {
		runCount.Add(1)
	})

	runner.Invalidate()
	runner.Stop()

	// After stop, the pending run should not execute
	time.Sleep(200 * time.Millisecond)
	if got := runCount.Load(); got != 0 {
		t.Fatalf("expected 0 runs after stop, got %d", got)
	}
}
