# Cached Forecasting on Dashboard

**Issue:** #107
**Date:** 2026-03-30
**Status:** Draft

## Problem

The dashboard shows historical data only. Forecasts are computed on-demand from the chart page via SSE streaming, which is slow and requires manual navigation. Users have no at-a-glance view of their projected financial future.

## Solution

A background `ForecastRunner` service that precomputes Monte Carlo forecasts on data changes, caches results in SQLite, and streams live updates to the dashboard via SSE. The dashboard displays a stacked area chart grouped by account type with confidence bands, covering yearly snapshots and special dates from today until the last special date.

## Database Changes

### New Table: `forecast_cache`

```sql
CREATE TABLE forecast_cache (
    date INTEGER NOT NULL,            -- unix days
    account_type_id TEXT NOT NULL,     -- FK to account_type
    median REAL NOT NULL,
    lower_bound REAL NOT NULL,
    upper_bound REAL NOT NULL,
    PRIMARY KEY (date, account_type_id)
);
```

Rows are replaced wholesale after each successful forecast run. No partial updates.

### Settings Additions

Two new entries in the existing `settings` table:

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `forecast_confidence` | float | `0.80` | Confidence interval for lower/upper bounds (e.g., 0.80 = 80%) |
| `forecast_samples` | int | `10000` | Number of Monte Carlo samples per forecast run |

## ForecastRunner Service

### Responsibilities

- Manages the forecast computation lifecycle
- Owns debouncing, cancellation, and subscriber management
- Lives in the model/service layer

### State

- `cancelFunc context.CancelFunc` — cancels the current run
- `timer *time.Timer` — 5-second trailing-edge debounce timer
- `mu sync.RWMutex` — protects subscriber list and running state
- `subscribers []chan ForecastEvent` — active SSE subscribers
- `running bool` — whether a forecast is currently in progress

### Invalidate()

Called by service methods after any forecast-affecting mutation. Behavior:

1. If a debounce timer is pending, reset it to 5 seconds.
2. If a forecast goroutine is running, cancel its context.
3. Start (or restart) a 5-second trailing-edge debounce timer.
4. When the timer fires, launch a new forecast goroutine.

This ensures rapid successive changes (e.g., editing multiple accounts) result in a single forecast run 5 seconds after the last change.

### Subscribe() / Unsubscribe()

- `Subscribe() chan ForecastEvent` — adds a channel to the subscriber list, returns it.
- `Unsubscribe(ch chan ForecastEvent)` — removes the channel and closes it.

Used by the dashboard SSE handler.

### Forecast Goroutine

1. Read `forecast_confidence` and `forecast_samples` from settings.
2. Load all accounts, snapshots, growth models, transfers, special dates.
3. Determine end date: last special date. If no special dates exist, abort (no forecast to run).
4. Compute snapshot dates: every January 1st from now until end date, plus all special date dates.
5. Run `finance.RunPrediction()` with high sample count, grouped by account type.
6. For each snapshot result:
   - Write to `forecast_cache` table.
   - Broadcast to all subscribers as a `ForecastEvent`.
7. On completion: broadcast a "done" event to all subscribers.
8. On context cancellation: stop early, leave partial data in DB (next run will replace it).

## Invalidation Triggers

The following service methods call `ForecastRunner.Invalidate()` after a successful DB write:

- **Accounts:** `UpsertAccount()`, `DeleteAccount()`
- **Snapshots:** `UpsertAccountSnapshot()`, `DeleteAccountSnapshot()`
- **Growth models:** `UpsertAccountGrowthModel()`, `DeleteAccountGrowthModel()`
- **Transfers:** `UpsertTransferTemplate()`, `DeleteTransferTemplate()`
- **Special dates:** `UpsertSpecialDate()`, `DeleteSpecialDate()`
- **Account types:** `UpsertAccountType()`, `DeleteAccountType()`

## SSE Endpoint

### `GET /dashboard/forecast/stream`

1. Call `ForecastRunner.Subscribe()` to get a subscriber channel.
2. Read all current rows from `forecast_cache` and send as SSE events.
3. If `ForecastRunner.IsRunning()`, send a `status: running` event.
4. Enter SSE loop: forward `ForecastEvent` messages from the channel as SSE events.
5. On "done" event, send `status: idle` to the client.
6. On client disconnect, call `Unsubscribe()`.

### SSE Event Types

| Event | Data | Description |
|-------|------|-------------|
| `setup` | `{ dates: [...], accountTypes: [...] }` | Initial metadata for chart setup |
| `snapshot` | `{ date, accountTypeId, median, lowerBound, upperBound }` | One data point |
| `status` | `{ status: "running" \| "idle" }` | Forecast computation status |
| `done` | `{}` | Forecast run completed |

## Dashboard View

### Forecast Chart Section

- **Position:** Below the existing balance history chart.
- **Visibility:** Only rendered if at least one special date exists. If no special dates, the section is hidden entirely.
- **Chart type:** Stacked area chart (ECharts) grouped by account type.
  - Each account type gets a stacked area series using its configured color.
  - Confidence bounds rendered as shaded bands (upper/lower) around the median.
- **X-axis:** Snapshot dates (yearly Jan 1st + special dates). Not evenly spaced — dates are positioned proportionally.
- **Y-axis:** Projected balance.
- **Special dates:** Rendered as vertical marklines with labels and colors (reusing existing markline pattern from chart page).

### Loading State

- When the forecast is running, show a loading/spinner indicator overlaying or adjacent to the chart.
- The chart progressively fills in as snapshot events arrive via SSE.
- When the "done" event arrives, remove the loading indicator.

### SSE Connection

- The dashboard page opens an `EventSource` to `/dashboard/forecast/stream` on page load.
- JavaScript handles events to build/update the ECharts instance.
- On disconnect/error, the chart remains showing the last received data (from the initial DB read).

## Forecast Parameters

| Parameter | Value | Source |
|-----------|-------|--------|
| Start date | Today | Computed |
| End date | Last special date | Computed from `special_date` table |
| Snapshot dates | Every Jan 1st + all special dates | Computed |
| Sample count | Configurable | `settings.forecast_samples` (default: 10000) |
| Confidence interval | Configurable | `settings.forecast_confidence` (default: 0.80) |
| Grouping | By account type | Fixed |

## Startup Behavior

On application startup, the `ForecastRunner` triggers an initial `Invalidate()` call so the cache is populated. If the `forecast_cache` table already has data, the dashboard displays it immediately while the background run refreshes it.

## Testing Strategy

- **ForecastRunner unit tests:** Test debounce behavior, cancel-and-restart, subscriber management.
- **Integration tests:** Verify that mutations trigger invalidation, forecast results are written to DB, and SSE events are broadcast.
- **Dashboard view tests:** Verify chart section visibility (hidden when no special dates, shown otherwise).
