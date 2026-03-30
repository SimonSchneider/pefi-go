# Cached Forecast Dashboard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Precompute Monte Carlo forecasts in the background on data changes, cache results in SQLite, stream live updates via SSE to a new dashboard forecast chart.

**Architecture:** A `ForecastRunner` in the model layer manages debounced background forecast computation. It writes results to a `forecast_cache` table and broadcasts to SSE subscribers. The dashboard opens an SSE connection on load to display cached + live forecast data as a stacked area chart grouped by account type. The issue this solves is #107.

**Tech Stack:** Go (goroutines, sync, context cancellation), SQLite, SQLC, Templ, HTMX, ECharts, SSE

---

### Task 1: Database Migration and SQLC Queries for Forecast Cache

**Files:**
- Create: `static/migrations/0028_forecast_cache.sql`
- Create: `sqlc/queries/forecast_cache.sql`
- Modify: `internal/pdb/` (auto-generated after `make generate-watch`)

- [ ] **Step 1: Write the migration file**

```sql
-- static/migrations/0028_forecast_cache.sql
-- migrate:up
CREATE TABLE forecast_cache (
    date INTEGER NOT NULL,
    account_type_id TEXT NOT NULL,
    median REAL NOT NULL,
    lower_bound REAL NOT NULL,
    upper_bound REAL NOT NULL,
    PRIMARY KEY (date, account_type_id)
);
```

- [ ] **Step 2: Write the SQLC query file**

```sql
-- sqlc/queries/forecast_cache.sql

-- name: ListForecastCache :many
SELECT date, account_type_id, median, lower_bound, upper_bound
FROM forecast_cache
ORDER BY date, account_type_id;

-- name: DeleteAllForecastCache :exec
DELETE FROM forecast_cache;

-- name: InsertForecastCache :exec
INSERT INTO forecast_cache (date, account_type_id, median, lower_bound, upper_bound)
VALUES (?, ?, ?, ?, ?);
```

- [ ] **Step 3: Run code generation**

Run: `make generate-watch`
Expected: New files in `internal/pdb/` for the forecast cache queries. No errors.

- [ ] **Step 4: Verify compilation**

Run: `go build ./...`
Expected: Builds successfully.

- [ ] **Step 5: Commit**

```bash
git add static/migrations/0028_forecast_cache.sql sqlc/queries/forecast_cache.sql internal/pdb/
git commit -m "feat(#107): add forecast_cache migration and SQLC queries"
```

---

### Task 2: Settings for Forecast Confidence and Samples

**Files:**
- Modify: `internal/model/settings.go`
- Test: `internal/model/service_test.go`

- [ ] **Step 1: Write the failing test for forecast settings**

Add to `internal/model/service_test.go`:

```go
func TestForecastSettings(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()

	// Default values
	confidence, err := svc.GetForecastConfidence(ctx)
	if err != nil {
		t.Fatalf("get default confidence: %v", err)
	}
	if confidence != 0.80 {
		t.Fatalf("expected default confidence 0.80, got %f", confidence)
	}

	samples, err := svc.GetForecastSamples(ctx)
	if err != nil {
		t.Fatalf("get default samples: %v", err)
	}
	if samples != 10000 {
		t.Fatalf("expected default samples 10000, got %d", samples)
	}

	// Set custom values
	if err := svc.SetForecastConfidence(ctx, 0.90); err != nil {
		t.Fatalf("set confidence: %v", err)
	}
	confidence, err = svc.GetForecastConfidence(ctx)
	if err != nil {
		t.Fatalf("get confidence after set: %v", err)
	}
	if confidence != 0.90 {
		t.Fatalf("expected confidence 0.90, got %f", confidence)
	}

	if err := svc.SetForecastSamples(ctx, 5000); err != nil {
		t.Fatalf("set samples: %v", err)
	}
	samples, err = svc.GetForecastSamples(ctx)
	if err != nil {
		t.Fatalf("get samples after set: %v", err)
	}
	if samples != 5000 {
		t.Fatalf("expected samples 5000, got %d", samples)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/model/ -run TestForecastSettings -v`
Expected: FAIL — `GetForecastConfidence` not defined.

- [ ] **Step 3: Write the implementation**

Add to `internal/model/settings.go`:

```go
const (
	settingForecastConfidence = "forecast_confidence"
	settingForecastSamples    = "forecast_samples"
)

func (s *Service) GetForecastConfidence(ctx context.Context) (float64, error) {
	val, err := s.q.GetSetting(ctx, settingForecastConfidence)
	if errors.Is(err, sql.ErrNoRows) {
		return 0.80, nil
	}
	if err != nil {
		return 0, fmt.Errorf("getting forecast confidence: %w", err)
	}
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing forecast confidence: %w", err)
	}
	return f, nil
}

func (s *Service) SetForecastConfidence(ctx context.Context, confidence float64) error {
	return s.q.UpsertSetting(ctx, pdb.UpsertSettingParams{
		Key:   settingForecastConfidence,
		Value: strconv.FormatFloat(confidence, 'f', -1, 64),
	})
}

func (s *Service) GetForecastSamples(ctx context.Context) (int64, error) {
	val, err := s.q.GetSetting(ctx, settingForecastSamples)
	if errors.Is(err, sql.ErrNoRows) {
		return 10000, nil
	}
	if err != nil {
		return 0, fmt.Errorf("getting forecast samples: %w", err)
	}
	n, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing forecast samples: %w", err)
	}
	return n, nil
}

func (s *Service) SetForecastSamples(ctx context.Context, samples int64) error {
	return s.q.UpsertSetting(ctx, pdb.UpsertSettingParams{
		Key:   settingForecastSamples,
		Value: strconv.FormatInt(samples, 10),
	})
}
```

Add `"strconv"` to the imports in `settings.go`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/model/ -run TestForecastSettings -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/model/settings.go internal/model/service_test.go
git commit -m "feat(#107): add forecast confidence and samples settings"
```

---

### Task 3: ForecastRunner Core — Debounce, Cancel, and Run

This is the central piece: a `ForecastRunner` struct that manages debounced background forecast computation with cancel-and-restart semantics.

**Files:**
- Create: `internal/model/forecast_runner.go`
- Create: `internal/model/forecast_runner_test.go`

- [ ] **Step 1: Write the failing test for debounce and cancel behavior**

Create `internal/model/forecast_runner_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/model/ -run TestForecastRunner -v`
Expected: FAIL — `model.NewForecastRunner` not defined.

- [ ] **Step 3: Write the ForecastRunner implementation**

Create `internal/model/forecast_runner.go`:

```go
package model

import (
	"context"
	"sync"
	"time"
)

type ForecastRunner struct {
	debounce time.Duration
	runFn    func(ctx context.Context)

	mu        sync.Mutex
	timer     *time.Timer
	cancelFn  context.CancelFunc
	stopped   bool

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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/model/ -run TestForecastRunner -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/model/forecast_runner.go internal/model/forecast_runner_test.go
git commit -m "feat(#107): add ForecastRunner with debounce and cancel-restart"
```

---

### Task 4: ForecastRunner Pub/Sub for SSE Subscribers

**Files:**
- Modify: `internal/model/forecast_runner.go`
- Modify: `internal/model/forecast_runner_test.go`

- [ ] **Step 1: Write the failing test for subscribe/broadcast**

Add to `internal/model/forecast_runner_test.go`:

```go
func TestForecastRunnerSubscribe(t *testing.T) {
	runner := model.NewForecastRunner(50*time.Millisecond, func(ctx context.Context) {
		// no-op
	})
	defer runner.Stop()

	ch := runner.Subscribe()
	defer runner.Unsubscribe(ch)

	// Broadcast an event
	runner.Broadcast(model.ForecastEvent{
		Type: model.ForecastEventSnapshot,
		Snapshot: &model.ForecastCacheRow{
			Date:          20000,
			AccountTypeID: "savings",
			Median:        1000,
			LowerBound:    800,
			UpperBound:    1200,
		},
	})

	select {
	case evt := <-ch:
		if evt.Type != model.ForecastEventSnapshot {
			t.Fatalf("expected snapshot event, got %v", evt.Type)
		}
		if evt.Snapshot.Median != 1000 {
			t.Fatalf("expected median 1000, got %f", evt.Snapshot.Median)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for broadcast event")
	}
}

func TestForecastRunnerUnsubscribe(t *testing.T) {
	runner := model.NewForecastRunner(50*time.Millisecond, func(ctx context.Context) {
		// no-op
	})
	defer runner.Stop()

	ch := runner.Subscribe()
	runner.Unsubscribe(ch)

	// After unsubscribe, channel should be closed
	_, ok := <-ch
	if ok {
		t.Fatal("expected channel to be closed after unsubscribe")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/model/ -run TestForecastRunnerSubscribe -v`
Expected: FAIL — `ForecastEvent`, `Broadcast`, `Subscribe` not defined.

- [ ] **Step 3: Add pub/sub types and methods to ForecastRunner**

Add to `internal/model/forecast_runner.go`:

```go
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
```

Add these fields to the `ForecastRunner` struct:

```go
	subscribersMu sync.RWMutex
	subscribers   []chan ForecastEvent
```

Add these methods:

```go
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/model/ -run TestForecastRunner -v`
Expected: All tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/model/forecast_runner.go internal/model/forecast_runner_test.go
git commit -m "feat(#107): add pub/sub to ForecastRunner for SSE broadcasting"
```

---

### Task 5: Forecast Computation Logic

Wire the `ForecastRunner` into the `Service` so it runs the actual Monte Carlo forecast, writes to `forecast_cache`, and broadcasts to subscribers.

**Files:**
- Create: `internal/model/forecast_cache.go`
- Create: `internal/model/forecast_cache_test.go`

- [ ] **Step 1: Write the failing test for forecast cache computation**

Create `internal/model/forecast_cache_test.go`:

```go
package model_test

import (
	"testing"
	"time"

	"github.com/SimonSchneider/pefigo/internal/model"
)

func TestRunForecastCache(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()

	// Create account type
	at, err := svc.UpsertAccountType(ctx, model.AccountTypeInput{Name: "Savings", Color: "#00ff00"})
	if err != nil {
		t.Fatalf("create account type: %v", err)
	}

	// Create account with type
	acc, err := svc.UpsertAccount(ctx, model.AccountInput{
		Name:   "My Savings",
		TypeID: at.ID,
	})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}

	// Add a snapshot
	_, err = svc.UpsertAccountSnapshot(ctx, acc.ID, model.AccountSnapshotInput{
		Date:    mustParseDate("2026-01-01"),
		Balance: newFixedValue(10000),
	})
	if err != nil {
		t.Fatalf("create snapshot: %v", err)
	}

	// Add a growth model
	_, err = svc.UpsertAccountGrowthModel(ctx, model.AccountGrowthModelInput{
		AccountID:        acc.ID,
		Type:             "fixed",
		AnnualRate:       newFixedValue(0.05),
		AnnualVolatility: newFixedValue(0),
		StartDate:        mustParseDate("2026-01-01"),
	})
	if err != nil {
		t.Fatalf("create growth model: %v", err)
	}

	// Add a special date (required for forecast to run)
	_, err = svc.UpsertSpecialDate(ctx, model.SpecialDateInput{
		Name: "Retirement",
		Date: mustParseDate("2028-01-01"),
	})
	if err != nil {
		t.Fatalf("create special date: %v", err)
	}

	// Run the forecast cache
	err = svc.RunForecastCache(ctx)
	if err != nil {
		t.Fatalf("run forecast cache: %v", err)
	}

	// Verify cache has data
	rows, err := svc.ListForecastCache(ctx)
	if err != nil {
		t.Fatalf("list forecast cache: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("expected forecast cache rows, got none")
	}

	// Should have snapshots for Jan 1 2027 and Jan 1 2028 (yearly) + special date (2028-01-01 overlaps)
	// At minimum we expect rows for the account type
	foundType := false
	for _, row := range rows {
		if row.AccountTypeID == at.ID {
			foundType = true
			if row.Median <= 0 {
				t.Fatalf("expected positive median, got %f", row.Median)
			}
		}
	}
	if !foundType {
		t.Fatalf("expected rows for account type %s, got none", at.ID)
	}
}

func TestRunForecastCacheNoSpecialDates(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()

	// No special dates — forecast should not run (no error, just no data)
	err := svc.RunForecastCache(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rows, err := svc.ListForecastCache(ctx)
	if err != nil {
		t.Fatalf("list forecast cache: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected no rows when no special dates, got %d", len(rows))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/model/ -run TestRunForecastCache -v`
Expected: FAIL — `RunForecastCache`, `ListForecastCache` not defined.

- [ ] **Step 3: Write the forecast cache computation and read logic**

Create `internal/model/forecast_cache.go`:

```go
package model

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/pefigo/internal/pdb"
	finance2 "github.com/SimonSchneider/pefigo/pkg/finance"
	"github.com/SimonSchneider/pefigo/pkg/uncertain"
)

func (s *Service) ListForecastCache(ctx context.Context) ([]ForecastCacheRow, error) {
	rows, err := s.q.ListForecastCache(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing forecast cache: %w", err)
	}
	result := make([]ForecastCacheRow, len(rows))
	for i, row := range rows {
		result[i] = ForecastCacheRow{
			Date:          row.Date,
			AccountTypeID: row.AccountTypeID,
			Median:        row.Median,
			LowerBound:    row.LowerBound,
			UpperBound:    row.UpperBound,
		}
	}
	return result, nil
}

func (s *Service) RunForecastCache(ctx context.Context) error {
	specialDates, err := s.ListSpecialDates(ctx)
	if err != nil {
		return fmt.Errorf("listing special dates: %w", err)
	}
	if len(specialDates) == 0 {
		return nil
	}

	confidence, err := s.GetForecastConfidence(ctx)
	if err != nil {
		return fmt.Errorf("getting forecast confidence: %w", err)
	}
	samples, err := s.GetForecastSamples(ctx)
	if err != nil {
		return fmt.Errorf("getting forecast samples: %w", err)
	}

	// Find last special date for end date
	sort.Slice(specialDates, func(i, j int) bool {
		return specialDates[i].Date.Before(specialDates[j].Date)
	})
	endDate := specialDates[len(specialDates)-1].Date

	today := date.Today()
	if !endDate.After(today) {
		return nil
	}

	// Build snapshot dates: every Jan 1st + special dates
	snapshotDates := make(map[date.Date]bool)
	for year := today.ToStdTime().Year() + 1; year <= endDate.ToStdTime().Year(); year++ {
		jan1 := date.FromStdTime(time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC))
		if jan1.After(today) && !jan1.After(endDate) {
			snapshotDates[jan1] = true
		}
	}
	for _, sd := range specialDates {
		if sd.Date.After(today) && !sd.Date.After(endDate) {
			snapshotDates[sd.Date] = true
		}
	}

	// Build a date cron that matches our snapshot dates
	sortedDates := make([]date.Date, 0, len(snapshotDates))
	for d := range snapshotDates {
		sortedDates = append(sortedDates, d)
	}
	sort.Slice(sortedDates, func(i, j int) bool {
		return sortedDates[i].Before(sortedDates[j])
	})

	// Use a custom recorder that collects on snapshot dates
	q1, q2 := (1-confidence)/2, (1+confidence)/2

	// Reuse the existing RunPrediction with GroupByType
	collector := &forecastCacheCollector{
		rows: make([]ForecastCacheRow, 0),
	}
	handler := &forecastCacheEventHandler{
		q1:        q1,
		q2:        q2,
		collector: collector,
	}

	params := PredictionParams{
		Duration:         date.DurationBetween(today, endDate),
		Samples:          samples,
		Quantile:         confidence,
		SnapshotInterval: buildSnapshotCronFromDates(sortedDates),
		GroupBy:          GroupByType,
	}

	if err := s.RunPrediction(ctx, handler, params); err != nil {
		return fmt.Errorf("running forecast prediction: %w", err)
	}

	// Write results to DB
	if err := s.q.DeleteAllForecastCache(ctx); err != nil {
		return fmt.Errorf("clearing forecast cache: %w", err)
	}
	for _, row := range collector.rows {
		if err := s.q.InsertForecastCache(ctx, pdb.InsertForecastCacheParams{
			Date:          row.Date,
			AccountTypeID: row.AccountTypeID,
			Median:        row.Median,
			LowerBound:    row.LowerBound,
			UpperBound:    row.UpperBound,
		}); err != nil {
			return fmt.Errorf("inserting forecast cache row: %w", err)
		}
	}

	return nil
}

type forecastCacheCollector struct {
	rows []ForecastCacheRow
}

type forecastCacheEventHandler struct {
	q1, q2    float64
	collector *forecastCacheCollector
}

func (h *forecastCacheEventHandler) Setup(e PredictionSetupEvent) error {
	return nil
}

func (h *forecastCacheEventHandler) Snapshot(e PredictionBalanceSnapshot) error {
	h.collector.rows = append(h.collector.rows, ForecastCacheRow{
		Date:          e.Day,
		AccountTypeID: e.ID,
		Median:        e.Balance,
		LowerBound:    e.LowerBound,
		UpperBound:    e.UpperBound,
	})
	return nil
}

func (h *forecastCacheEventHandler) Close() error {
	return nil
}

// buildSnapshotCronFromDates creates a cron that approximates the desired dates.
// Since the finance engine uses date.Cron for snapshot scheduling, we use a
// daily cron and filter in the recorder. Alternatively, we can use "*-01-01"
// for yearly snapshots. For the cached forecast, we use a daily cron and
// let the groupingEventHandler do its work — the special dates are already
// included as marklines.
//
// For simplicity, use monthly snapshots ("*-*-01") which captures Jan 1st
// naturally and gives enough resolution. The special dates that fall on
// non-first-of-month days will be close enough to a monthly snapshot.
func buildSnapshotCronFromDates(dates []date.Date) date.Cron {
	return date.Cron("*-*-01")
}
```

Note: This initial implementation uses monthly snapshots via the existing `RunPrediction` + `groupingEventHandler` pipeline. The snapshot dates include Jan 1st naturally. Special dates that don't fall on the 1st will be approximated — this is a pragmatic starting point. If exact special date snapshots are needed, we can refine the cron or post-filter later.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/model/ -run TestRunForecastCache -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/model/forecast_cache.go internal/model/forecast_cache_test.go
git commit -m "feat(#107): add forecast cache computation and storage"
```

---

### Task 6: Wire ForecastRunner into Service

Connect the `ForecastRunner` to the `Service` so it can be started/stopped with the application lifecycle and called from mutation methods.

**Files:**
- Modify: `internal/model/service.go`
- Modify: `internal/model/forecast_runner.go`
- Modify: `internal/controller/run.go`

- [ ] **Step 1: Write the failing test for service with forecast runner**

Add to `internal/model/forecast_cache_test.go`:

```go
func TestServiceForecastRunnerInvalidation(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()

	runner := model.NewForecastRunner(50*time.Millisecond, func(ctx context.Context) {
		svc.RunForecastCache(ctx)
	})
	svc.SetForecastRunner(runner)
	defer runner.Stop()

	// Create account type
	at, err := svc.UpsertAccountType(ctx, model.AccountTypeInput{Name: "Savings", Color: "#00ff00"})
	if err != nil {
		t.Fatalf("create account type: %v", err)
	}

	// Create account with snapshot and growth model
	acc, err := svc.UpsertAccount(ctx, model.AccountInput{Name: "Test", TypeID: at.ID})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	_, err = svc.UpsertAccountSnapshot(ctx, acc.ID, model.AccountSnapshotInput{
		Date:    mustParseDate("2026-01-01"),
		Balance: newFixedValue(10000),
	})
	if err != nil {
		t.Fatalf("create snapshot: %v", err)
	}
	_, err = svc.UpsertAccountGrowthModel(ctx, model.AccountGrowthModelInput{
		AccountID:        acc.ID,
		Type:             "fixed",
		AnnualRate:       newFixedValue(0.05),
		AnnualVolatility: newFixedValue(0),
		StartDate:        mustParseDate("2026-01-01"),
	})
	if err != nil {
		t.Fatalf("create growth model: %v", err)
	}

	// Add special date
	_, err = svc.UpsertSpecialDate(ctx, model.SpecialDateInput{
		Name: "Target",
		Date: mustParseDate("2028-01-01"),
	})
	if err != nil {
		t.Fatalf("create special date: %v", err)
	}

	// Wait for debounce + forecast run
	time.Sleep(500 * time.Millisecond)

	// Verify cache was populated by the runner
	rows, err := svc.ListForecastCache(ctx)
	if err != nil {
		t.Fatalf("list forecast cache: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("expected forecast cache to be populated after invalidation")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/model/ -run TestServiceForecastRunnerInvalidation -v`
Expected: FAIL — `SetForecastRunner` not defined.

- [ ] **Step 3: Add ForecastRunner to Service and invalidation calls**

Add to `internal/model/service.go`, the `forecastRunner` field to the `Service` struct:

```go
type Service struct {
	db             *sql.DB
	q              *pdb.Queries
	sweClient      *swe.Client
	currencyClient *currency.Client
	forecastRunner *ForecastRunner
}
```

Add the setter method:

```go
func (s *Service) SetForecastRunner(runner *ForecastRunner) {
	s.forecastRunner = runner
}

func (s *Service) ForecastRunner() *ForecastRunner {
	return s.forecastRunner
}

func (s *Service) invalidateForecast() {
	if s.forecastRunner != nil {
		s.forecastRunner.Invalidate()
	}
}
```

Now add `s.invalidateForecast()` calls after the successful DB write in each of these methods:

**`internal/model/account.go`** — `UpsertAccount()`: add `s.invalidateForecast()` after the `if err != nil { return ... }` block, before the return.
Same for `UpsertAccountWithStartupShares()` and `DeleteAccount()`.

**`internal/model/snapshot.go`** — `UpsertAccountSnapshot()` and `DeleteAccountSnapshot()`: add `s.invalidateForecast()` before the return.

**`internal/model/growth_model.go`** — `UpsertAccountGrowthModel()` and `DeleteAccountGrowthModel()`: add `s.invalidateForecast()` before the return.

**`internal/model/transfer_template.go`** — `UpsertTransferTemplate()` and `DeleteTransferTemplate()`: add `s.invalidateForecast()` before the return.

**`internal/model/special_date.go`** — `UpsertSpecialDate()` and `DeleteSpecialDate()`: add `s.invalidateForecast()` before the return.

**`internal/model/account_type.go`** — `UpsertAccountType()` and `DeleteAccountType()`: add `s.invalidateForecast()` before the return.

**`internal/model/startup_shares.go`** — `UpsertInvestmentRound()`, `DeleteInvestmentRound()`, `UpsertShareChange()`, `DeleteShareChange()`, `UpsertStartupShareOption()`, `DeleteStartupShareOption()`: add `s.invalidateForecast()` before the return.

Each call is a single line added right before the success return, for example in `UpsertAccount()`:

```go
	// ... existing code ...
	if err != nil {
		return Account{}, fmt.Errorf("failed to upsert account: %w", err)
	}
	s.invalidateForecast()
	return accountFromDB(acc), nil
```

- [ ] **Step 4: Wire runner in controller/run.go**

Modify `internal/controller/run.go` to create and start the runner:

```go
	svc := model.New(db)

	runner := model.NewForecastRunner(5*time.Second, func(ctx context.Context) {
		if err := svc.RunForecastCache(ctx); err != nil {
			logger.Printf("forecast cache error: %v", err)
		}
	})
	svc.SetForecastRunner(runner)
	defer runner.Stop()

	// Trigger initial forecast on startup
	runner.Invalidate()
```

Add `"time"` to the imports.

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/model/ -run TestServiceForecastRunnerInvalidation -v`
Expected: PASS

- [ ] **Step 6: Run all tests**

Run: `go test ./...`
Expected: All tests PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/model/service.go internal/model/account.go internal/model/snapshot.go internal/model/growth_model.go internal/model/transfer_template.go internal/model/special_date.go internal/model/account_type.go internal/model/startup_shares.go internal/model/forecast_runner.go internal/model/forecast_cache.go internal/model/forecast_cache_test.go internal/controller/run.go
git commit -m "feat(#107): wire ForecastRunner into Service with invalidation triggers"
```

---

### Task 7: SSE Endpoint for Dashboard Forecast Stream

**Files:**
- Modify: `internal/controller/handler.go`
- Create: `internal/model/forecast_cache_test.go` (add SSE-related service test)

- [ ] **Step 1: Write the failing test for ListForecastCacheGrouped**

We need a method that returns forecast data organized for the SSE setup event. Add to `internal/model/forecast_cache_test.go`:

```go
func TestListForecastCacheGrouped(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()

	at, err := svc.UpsertAccountType(ctx, model.AccountTypeInput{Name: "Savings", Color: "#00ff00"})
	if err != nil {
		t.Fatalf("create account type: %v", err)
	}

	acc, err := svc.UpsertAccount(ctx, model.AccountInput{Name: "Test", TypeID: at.ID})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	_, err = svc.UpsertAccountSnapshot(ctx, acc.ID, model.AccountSnapshotInput{
		Date:    mustParseDate("2026-01-01"),
		Balance: newFixedValue(10000),
	})
	if err != nil {
		t.Fatalf("create snapshot: %v", err)
	}
	_, err = svc.UpsertAccountGrowthModel(ctx, model.AccountGrowthModelInput{
		AccountID:        acc.ID,
		Type:             "fixed",
		AnnualRate:       newFixedValue(0.05),
		AnnualVolatility: newFixedValue(0),
		StartDate:        mustParseDate("2026-01-01"),
	})
	if err != nil {
		t.Fatalf("create growth model: %v", err)
	}
	_, err = svc.UpsertSpecialDate(ctx, model.SpecialDateInput{
		Name:  "Retirement",
		Date:  mustParseDate("2028-01-01"),
		Color: "#ff0000",
	})
	if err != nil {
		t.Fatalf("create special date: %v", err)
	}

	if err := svc.RunForecastCache(ctx); err != nil {
		t.Fatalf("run forecast cache: %v", err)
	}

	data, err := svc.GetForecastCacheForDashboard(ctx)
	if err != nil {
		t.Fatalf("get forecast data: %v", err)
	}
	if data == nil {
		t.Fatal("expected forecast data, got nil")
	}
	if len(data.Entities) == 0 {
		t.Fatal("expected entities in forecast data")
	}
	if len(data.Marklines) == 0 {
		t.Fatal("expected marklines in forecast data")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/model/ -run TestListForecastCacheGrouped -v`
Expected: FAIL — `GetForecastCacheForDashboard` not defined.

- [ ] **Step 3: Implement GetForecastCacheForDashboard**

Add to `internal/model/forecast_cache.go`:

```go
type ForecastDashboardData struct {
	Entities  []PredictionFinancialEntity `json:"entities"`
	Marklines []Markline                  `json:"marklines"`
}

func (s *Service) GetForecastCacheForDashboard(ctx context.Context) (*ForecastDashboardData, error) {
	rows, err := s.ListForecastCache(ctx)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}

	accountTypes, err := s.ListAccountTypes(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing account types: %w", err)
	}
	typesByID := make(map[string]AccountType)
	for _, at := range accountTypes {
		typesByID[at.ID] = at
	}

	// Group rows by account type
	entitiesByID := make(map[string]*PredictionFinancialEntity)
	for _, row := range rows {
		entity, ok := entitiesByID[row.AccountTypeID]
		if !ok {
			at := typesByID[row.AccountTypeID]
			entity = &PredictionFinancialEntity{
				ID:    row.AccountTypeID,
				Name:  at.Name,
				Color: at.Color,
			}
			entitiesByID[row.AccountTypeID] = entity
		}
		entity.Snapshots = append(entity.Snapshots, PredictionBalanceSnapshot{
			ID:         row.AccountTypeID,
			Day:        row.Date,
			Balance:    row.Median,
			LowerBound: row.LowerBound,
			UpperBound: row.UpperBound,
		})
	}

	entities := make([]PredictionFinancialEntity, 0, len(entitiesByID))
	for _, e := range entitiesByID {
		entities = append(entities, *e)
	}

	// Build marklines from special dates
	specialDates, err := s.ListSpecialDates(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing special dates: %w", err)
	}
	marklines := make([]Markline, 0, len(specialDates))
	for _, sd := range specialDates {
		marklines = append(marklines, Markline{
			Date:  sd.Date.ToStdTime().UnixMilli(),
			Color: sd.Color,
			Name:  sd.Name,
		})
	}

	return &ForecastDashboardData{
		Entities:  entities,
		Marklines: marklines,
	}, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/model/ -run TestListForecastCacheGrouped -v`
Expected: PASS

- [ ] **Step 5: Add the SSE endpoint to handler**

Add to `internal/controller/handler.go` in the route registration (in `NewHandler`):

```go
	mux.Handle("GET /dashboard/forecast/stream", h.dashboardForecastStream())
```

Add the handler method:

```go
func (h *Handler) dashboardForecastStream() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		sse := srvu.SSEResponse(w)
		runner := h.svc.ForecastRunner()

		// Subscribe to live updates
		var ch chan model.ForecastEvent
		if runner != nil {
			ch = runner.Subscribe()
			defer runner.Unsubscribe(ch)
		}

		// Send cached data
		data, err := h.svc.GetForecastCacheForDashboard(ctx)
		if err != nil {
			return fmt.Errorf("getting forecast cache: %w", err)
		}
		if data != nil {
			if err := sse.SendNamedJson("setup", data); err != nil {
				return err
			}
		}

		// Send current status
		if runner != nil && runner.IsRunning() {
			if err := sse.SendNamedJson("status", map[string]string{"status": "running"}); err != nil {
				return err
			}
		}

		// Stream live updates
		if ch == nil {
			return sse.SendEventWithoutData("close")
		}
		for {
			select {
			case <-ctx.Done():
				return nil
			case evt, ok := <-ch:
				if !ok {
					return nil
				}
				switch evt.Type {
				case model.ForecastEventSnapshot:
					if err := sse.SendNamedJson("snapshot", evt.Snapshot); err != nil {
						return err
					}
				case model.ForecastEventDone:
					if err := sse.SendNamedJson("status", map[string]string{"status": "idle"}); err != nil {
						return err
					}
				}
			}
		}
	})
}
```

- [ ] **Step 6: Add broadcasting to RunForecastCache**

Modify `internal/model/forecast_cache.go` — update `RunForecastCache` to broadcast events. After inserting each row into the DB, broadcast to subscribers. After all rows are written, broadcast a done event.

Replace the "Write results to DB" section:

```go
	// Write results to DB and broadcast to subscribers
	if err := s.q.DeleteAllForecastCache(ctx); err != nil {
		return fmt.Errorf("clearing forecast cache: %w", err)
	}
	for _, row := range collector.rows {
		if err := s.q.InsertForecastCache(ctx, pdb.InsertForecastCacheParams{
			Date:          row.Date,
			AccountTypeID: row.AccountTypeID,
			Median:        row.Median,
			LowerBound:    row.LowerBound,
			UpperBound:    row.UpperBound,
		}); err != nil {
			return fmt.Errorf("inserting forecast cache row: %w", err)
		}
		if s.forecastRunner != nil {
			s.forecastRunner.Broadcast(ForecastEvent{
				Type:     ForecastEventSnapshot,
				Snapshot: &row,
			})
		}
	}
	if s.forecastRunner != nil {
		s.forecastRunner.Broadcast(ForecastEvent{Type: ForecastEventDone})
	}
```

- [ ] **Step 7: Run all tests**

Run: `go test ./...`
Expected: All tests PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/model/forecast_cache.go internal/model/forecast_cache_test.go internal/controller/handler.go
git commit -m "feat(#107): add SSE endpoint and dashboard forecast data retrieval"
```

---

### Task 8: Dashboard Forecast Chart View

**Files:**
- Modify: `internal/view/dashboard_view.templ`
- Modify: `internal/model/dashboard.go`
- Create: `static/public/dashboard-forecast.js`

- [ ] **Step 1: Add HasSpecialDates to DashboardView**

Modify `internal/model/dashboard.go` — add field to `DashboardView`:

```go
type DashboardView struct {
	TotalBalance         float64
	TotalAssets          float64
	TotalLiabilities     float64
	Budget               *BudgetView
	AccountTypeGroups    []AccountTypeGroup
	AccountChartData     []AccountTypeChartEntry
	SnapshotHistoryChart SnapshotHistoryChartData
	HasSpecialDates      bool
}
```

In `GetDashboardData()`, add after the `snapshotHistory` section:

```go
	specialDates, err := s.ListSpecialDates(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing special dates: %w", err)
	}
```

And set the field in the return:

```go
	return &DashboardView{
		// ... existing fields ...
		HasSpecialDates: len(specialDates) > 0,
	}, nil
```

- [ ] **Step 2: Add the forecast chart section to the dashboard template**

Modify `internal/view/dashboard_view.templ`. After the balance history / accounts section (before `@ItemsTooltipScript()`), add:

```templ
			if view.HasSpecialDates {
				<div class="grid grid-cols-1 gap-6 mt-6">
					<div class="card bg-base-100 shadow-sm border border-base-300">
						<div class="card-body">
							<h2 class="card-title">
								Forecast
								<span id="forecast-status" class="loading loading-spinner loading-sm hidden"></span>
							</h2>
							<div id="dashboard-forecast-chart" style="width: 100%; height: 400px;"></div>
						</div>
					</div>
				</div>
				<script src="/static/public/dashboard-forecast.js"></script>
			}
```

- [ ] **Step 3: Create the forecast chart JavaScript**

Create `static/public/dashboard-forecast.js`:

```javascript
(function() {
    function getThemeColor(cssVar, fallback) {
        try {
            var v = getComputedStyle(document.documentElement).getPropertyValue(cssVar).trim();
            return v || fallback;
        } catch (_) {
            return fallback;
        }
    }

    function getThemeOpts() {
        var baseContent = getThemeColor('--color-base-content', '#333');
        var baseBg = getThemeColor('--color-base-100', '#fff');
        var tooltipBorder = getThemeColor('--color-base-300', '#ccc');
        return {
            baseContent: baseContent,
            baseBg: baseBg,
            tooltipBorder: tooltipBorder,
            textStyle: { color: baseContent, textBorderWidth: 0 },
            axisLabel: { color: baseContent, textBorderWidth: 0 }
        };
    }

    function formatThousands(val) {
        var n = Math.round(val);
        var neg = n < 0;
        var s = Math.abs(n).toString();
        var pre = s.length % 3 || 3;
        var out = s.slice(0, pre);
        for (var i = pre; i < s.length; i += 3) {
            out += ',' + s.slice(i, i + 3);
        }
        return neg ? '-' + out : out;
    }

    function lightenColor(hex) {
        var r = parseInt(hex.slice(1, 3), 16);
        var g = parseInt(hex.slice(3, 5), 16);
        var b = parseInt(hex.slice(5, 7), 16);
        return 'rgba(' + r + ', ' + g + ', ' + b + ', 0.2)';
    }

    var dom = document.getElementById('dashboard-forecast-chart');
    if (!dom) return;

    var chart = echarts.init(dom);
    var series = {};
    var legendData = [];
    var marklineSeries = [];
    var statusEl = document.getElementById('forecast-status');

    function showLoading() {
        if (statusEl) statusEl.classList.remove('hidden');
    }

    function hideLoading() {
        if (statusEl) statusEl.classList.add('hidden');
    }

    function addEntity(e) {
        var color = e.color || '#666';
        series[e.id] = {
            id: e.id,
            name: e.name,
            type: 'line',
            data: [],
            showSymbol: false,
            stack: 'forecast',
            areaStyle: { color: color, opacity: 0.6 },
            lineStyle: { color: color },
            itemStyle: { color: color },
            emphasis: { focus: 'series' }
        };
        series[e.id + '_min'] = {
            id: e.id + '_min',
            name: e.name + ' min',
            type: 'line',
            data: [],
            lineStyle: { opacity: 0 },
            stack: e.id + '-band',
            symbol: 'none',
            showSymbol: false,
            tooltip: { show: false },
            label: { show: false }
        };
        series[e.id + '_max'] = {
            id: e.id + '_max',
            name: e.name + ' max',
            type: 'line',
            data: [],
            lineStyle: { opacity: 0 },
            stack: e.id + '-band',
            showSymbol: false,
            areaStyle: { color: lightenColor(color) },
            tooltip: { show: false },
            label: { show: false }
        };
        legendData.push({ name: e.name });
    }

    function addSnapshot(s) {
        if (!series[s.accountTypeID]) return;
        series[s.accountTypeID].data.push([s.date, s.median]);
        series[s.accountTypeID + '_min'].data.push([s.date, s.lowerBound]);
        series[s.accountTypeID + '_max'].data.push([s.date, s.upperBound - s.lowerBound]);
    }

    function updateChart() {
        var theme = getThemeOpts();
        chart.setOption({
            backgroundColor: theme.baseBg,
            tooltip: {
                trigger: 'axis',
                backgroundColor: theme.baseBg,
                borderColor: theme.tooltipBorder,
                extraCssText: 'box-shadow: 0 2px 4px rgba(0,0,0,0.15);',
                formatter: function(params) {
                    var items = params.filter(function(p) {
                        return p.seriesName.indexOf(' min') === -1 &&
                               p.seriesName.indexOf(' max') === -1 &&
                               p.value && p.value[1] !== 0;
                    });
                    if (items.length === 0) return '';
                    var date = new Date(items[0].value[0]);
                    var header = date.getFullYear() + '-' + String(date.getMonth()+1).padStart(2,'0') + '-' + String(date.getDate()).padStart(2,'0');
                    var total = items.reduce(function(sum, p) { return sum + p.value[1]; }, 0);
                    var lines = items.map(function(p) {
                        return '<div style="display:flex;justify-content:space-between;gap:16px">' +
                            '<span>' + p.marker + ' ' + p.seriesName + '</span>' +
                            '<span style="font-weight:500">' + formatThousands(p.value[1]) + '</span></div>';
                    });
                    var totalLine = '<div style="display:flex;justify-content:space-between;gap:16px;border-top:1px solid ' + theme.tooltipBorder + ';margin-top:4px;padding-top:4px">' +
                        '<span>Total</span><span style="font-weight:700">' + formatThousands(total) + '</span></div>';
                    return header + '<br/>' + lines.join('') + totalLine;
                }
            },
            legend: {
                data: legendData,
                type: 'scroll',
                bottom: 0,
                textStyle: theme.textStyle
            },
            grid: {
                left: '3%',
                right: '4%',
                bottom: '15%',
                containLabel: true
            },
            xAxis: {
                type: 'time',
                axisLabel: theme.axisLabel
            },
            yAxis: {
                type: 'value',
                axisLabel: theme.axisLabel
            },
            series: Object.values(series).concat(marklineSeries)
        });
    }

    var evtSource = new EventSource('/dashboard/forecast/stream');

    evtSource.addEventListener('setup', function(event) {
        var data = JSON.parse(event.data);
        (data.entities || []).forEach(function(e) {
            addEntity(e);
            (e.snapshots || []).forEach(function(s) {
                addSnapshot({
                    accountTypeID: e.id,
                    date: s.day,
                    median: s.balance,
                    lowerBound: s.lowerBound,
                    upperBound: s.upperBound
                });
            });
        });

        var themeText = getThemeColor('--color-base-content', '#333');
        marklineSeries = (data.marklines || []).map(function(m, idx) {
            return {
                name: m.name,
                type: 'line',
                markLine: {
                    symbol: ['none', 'none'],
                    data: [{
                        xAxis: new Date(m.date),
                        lineStyle: { color: m.color || themeText, type: 'dashed' },
                        label: {
                            offset: [0, idx % 2 !== 0 ? 0 : -15],
                            formatter: m.name,
                            color: m.color || themeText,
                            textBorderWidth: 0
                        }
                    }]
                }
            };
        });

        updateChart();
    });

    evtSource.addEventListener('snapshot', function(event) {
        var s = JSON.parse(event.data);
        addSnapshot(s);
        updateChart();
    });

    evtSource.addEventListener('status', function(event) {
        var data = JSON.parse(event.data);
        if (data.status === 'running') {
            showLoading();
        } else {
            hideLoading();
        }
    });

    evtSource.addEventListener('close', function() {
        evtSource.close();
        hideLoading();
    });

    window.addEventListener('resize', function() { chart.resize(); });
    window.addEventListener('themechange', function() { updateChart(); chart.resize(); });
})();
```

- [ ] **Step 4: Run code generation for templ changes**

Run: `make generate-watch`
(Note: if `make watch-templ` is running, templ changes are picked up automatically — just run `make generate-watch` to regenerate tailwind.)

- [ ] **Step 5: Verify compilation**

Run: `go build ./...`
Expected: Builds successfully.

- [ ] **Step 6: Commit**

```bash
git add internal/model/dashboard.go internal/view/dashboard_view.templ static/public/dashboard-forecast.js
git commit -m "feat(#107): add forecast chart to dashboard with SSE streaming"
```

---

### Task 9: Settings UI for Forecast Configuration

**Files:**
- Modify: `internal/controller/handler.go` (settings handlers)
- Modify: settings view templ file

- [ ] **Step 1: Find the settings page view and handler**

Check the existing settings page implementation to understand the pattern. Look at:
- The `settingsPage()` handler in `internal/controller/handler.go`
- The settings view template

- [ ] **Step 2: Add forecast settings to the settings page handler**

In the settings page handler, load the forecast confidence and samples values and pass them to the view. In the settings POST handler, handle saving the new values.

Add forecast settings fields to whatever view data struct the settings page uses. Add form fields for:
- Forecast Confidence (dropdown or input: 0.80, 0.90, 0.95)
- Forecast Samples (input: number)

Add a POST handler for saving these settings (or extend the existing settings POST handler).

- [ ] **Step 3: Add forecast settings form to the settings view template**

Add a section to the settings page with form fields for the two new settings. Follow the existing form patterns in the codebase.

- [ ] **Step 4: Run code generation and verify compilation**

Run: `make generate-watch && go build ./...`
Expected: Builds successfully.

- [ ] **Step 5: Commit**

```bash
git add internal/controller/handler.go internal/view/
git commit -m "feat(#107): add forecast settings UI for confidence and samples"
```

---

### Task 10: Final Integration and Manual Testing

- [ ] **Step 1: Run the full test suite**

Run: `go test ./...`
Expected: All tests PASS.

- [ ] **Step 2: Run CI checks**

Run: `make ci-pr`
Expected: All checks pass.

- [ ] **Step 3: Manual browser test**

Use `rodney` to verify the dashboard:
1. Navigate to the dashboard (http://localhost:3002/)
2. Verify the forecast chart appears (if special dates exist)
3. Verify the loading spinner shows during forecast computation
4. Verify the chart updates after making a data change

- [ ] **Step 4: Commit any fixes**

If any issues found during manual testing, fix and commit.

---

### Task 11: Create Pull Request

- [ ] **Step 1: Create PR for issue #107**

```bash
gh pr create --title "feat: cached forecasting on dashboard (#107)" --body "$(cat <<'EOF'
## Summary
- Background `ForecastRunner` precomputes Monte Carlo forecasts with 5s trailing-edge debounce and cancel-restart semantics
- Results cached in `forecast_cache` SQLite table, streamed live via SSE to dashboard
- Dashboard shows stacked area chart by account type with confidence bands and special date marklines
- Forecast settings (confidence interval, sample count) configurable in settings page
- Only shows forecast chart when special dates exist

Closes #107

## Test plan
- [ ] `go test ./...` passes
- [ ] `make ci-pr` passes
- [ ] Dashboard shows forecast chart with cached data
- [ ] Modifying account/snapshot/growth model triggers debounced re-forecast
- [ ] Loading spinner shows during forecast computation
- [ ] SSE stream delivers live updates to open dashboard
- [ ] No forecast chart when no special dates defined
- [ ] Settings page allows configuring confidence and samples

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```
