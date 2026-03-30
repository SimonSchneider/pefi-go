package model

import (
	"context"
	"fmt"
	"sort"

	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/pefigo/internal/pdb"
)

func (s *Service) ListForecastCache(ctx context.Context) ([]ForecastCacheRow, error) {
	rows, err := s.q.ListForecastCache(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing forecast cache: %w", err)
	}
	result := make([]ForecastCacheRow, len(rows))
	for i, r := range rows {
		result[i] = ForecastCacheRow{
			Date:          r.Date,
			AccountTypeID: r.AccountTypeID,
			Median:        r.Median,
			LowerBound:    r.LowerBound,
			UpperBound:    r.UpperBound,
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

	// Find last special date as end date
	sort.Slice(specialDates, func(i, j int) bool {
		return specialDates[i].Date.Before(specialDates[j].Date)
	})
	endDate := specialDates[len(specialDates)-1].Date

	today := date.Today()
	if !endDate.After(today) {
		return nil
	}

	duration := endDate.Sub(today)

	handler := &forecastCacheEventHandler{}

	params := PredictionParams{
		Duration:         duration,
		Samples:          samples,
		Quantile:         confidence,
		SnapshotInterval: "*-01-01",
		GroupBy:          GroupByType,
	}

	if err := s.RunPrediction(ctx, handler, params); err != nil {
		return fmt.Errorf("running prediction for forecast cache: %w", err)
	}

	// Clear old cache and insert new rows
	if err := s.q.DeleteAllForecastCache(ctx); err != nil {
		return fmt.Errorf("deleting old forecast cache: %w", err)
	}
	for _, row := range handler.rows {
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

	return nil
}

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

// forecastCacheEventHandler collects prediction snapshots into ForecastCacheRow slices.
type forecastCacheEventHandler struct {
	rows []ForecastCacheRow
}

func (h *forecastCacheEventHandler) Setup(_ PredictionSetupEvent) error {
	return nil
}

func (h *forecastCacheEventHandler) Snapshot(snap PredictionBalanceSnapshot) error {
	h.rows = append(h.rows, ForecastCacheRow{
		Date:          snap.Day,
		AccountTypeID: snap.ID,
		Median:        snap.Balance,
		LowerBound:    snap.LowerBound,
		UpperBound:    snap.UpperBound,
	})
	return nil
}

func (h *forecastCacheEventHandler) Close() error {
	return nil
}
