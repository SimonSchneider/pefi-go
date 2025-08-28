package core

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/goslu/static/shttp"
	"github.com/SimonSchneider/pefigo/internal/finance"
	"github.com/SimonSchneider/pefigo/internal/pdb"
	"github.com/SimonSchneider/pefigo/internal/uncertain"
	"net/http"
	"time"
)

type PredictionParams struct {
	Duration         date.Duration
	Samples          int64
	Quantile         float64
	SnapshotInterval date.Cron
}

func (p *PredictionParams) FromForm(r *http.Request) error {
	if err := shttp.Parse(&p.Duration, date.ParseDuration, r.FormValue("duration"), 365); err != nil {
		return fmt.Errorf("parsing duration: %w", err)
	}
	if err := shttp.Parse(&p.Samples, parseHumanNumber(parseInt64), r.FormValue("samples"), 2000); err != nil {
		return fmt.Errorf("parsing samples: %w", err)
	}
	if err := shttp.Parse(&p.Quantile, shttp.ParseFloat, r.FormValue("quantile"), 0.8); err != nil {
		return fmt.Errorf("parsing quantile: %w", err)
	}
	if err := shttp.Parse(&p.SnapshotInterval, parseDateCron, r.FormValue("snapshot_interval"), "*-*-25"); err != nil {
		return fmt.Errorf("parsing snapshot interval: %w", err)
	}
	return nil
}

type PredictionBalanceSnapshot struct {
	ID         string  `json:"id"`
	Day        int64   `json:"day"`
	Balance    float64 `json:"balance"`
	LowerBound float64 `json:"lowerBound"`
	UpperBound float64 `json:"upperBound"`
}
type PredictionFinancialEntity struct {
	ID        string                      `json:"id"`
	Name      string                      `json:"name"`
	Snapshots []PredictionBalanceSnapshot `json:"snapshots"`
}
type PredictionSetupEvent struct {
	Max      int64                       `json:"max"`
	Entities []PredictionFinancialEntity `json:"entities"`
}

type PredictionEventHandler interface {
	Setup(PredictionSetupEvent) error
	Snapshot(PredictionBalanceSnapshot) error
	Close() error
}

func RunPrediction(ctx context.Context, db *sql.DB, eventHandler PredictionEventHandler, params PredictionParams) error {
	q := pdb.New(db)
	q1, q2 := (1-params.Quantile)/2, (1+params.Quantile)/2

	transfers := make([]finance.TransferTemplate, 0)
	entities := make([]finance.Entity, 0)
	accs, err := q.ListAccounts(ctx)
	if err != nil {
		return fmt.Errorf("listing accounts for Prediction: %w", err)
	}
	trans, err := ListTransferTemplates(ctx, db)
	if err != nil {
		return fmt.Errorf("listing transfers for Prediction: %w", err)
	}
	startDate := date.Date(0)
	for _, acc := range accs {
		snaps, err := ListAccountSnapshots(ctx, db, acc.ID)
		if err != nil {
			return fmt.Errorf("getting snapshots for account %s: %w", acc.ID, err)
		}
		gms, err := ListAccountGrowthModels(ctx, db, acc.ID)
		if err != nil {
			return fmt.Errorf("getting growth models for account %s: %w", acc.ID, err)
		}
		var balanceLimit finance.BalanceLimit
		if acc.BalanceUpperLimit != nil {
			balanceLimit = finance.BalanceLimit{
				Upper: uncertain.NewFixed(*acc.BalanceUpperLimit),
			}
		}
		entity := finance.Entity{
			ID:           acc.ID,
			Name:         acc.Name,
			BalanceLimit: balanceLimit,
			Snapshots:    make([]finance.BalanceSnapshot, 0, len(snaps)),
		}
		if acc.CashFlowFrequency != nil || acc.CashFlowDestinationID != nil {
			entity.CashFlow = &finance.CashFlowModel{
				Frequency:     date.Cron(orDefault(acc.CashFlowFrequency)),
				DestinationID: orDefault(acc.CashFlowDestinationID),
			}
		}
		for _, snap := range snaps {
			entity.Snapshots = append(entity.Snapshots, finance.BalanceSnapshot{
				Date:    snap.Date,
				Balance: snap.Balance,
			})
			if snap.Date.After(startDate) {
				startDate = snap.Date
			}
		}
		fgms := make([]finance.GrowthModel, 0, len(gms))
		for _, gm := range gms {
			if gm.Type == "fixed" {
				fgms = append(fgms, &finance.FixedGrowth{
					TimeFrameGrowth: finance.TimeFrameGrowth{
						StartDate: gm.StartDate,
						EndDate:   gm.EndDate,
					},
					AnnualRate: gm.AnnualRate,
				})
			} else if gm.Type == "lognormal" {
				fgms = append(fgms, &finance.LogNormalGrowth{
					TimeFrameGrowth: finance.TimeFrameGrowth{
						StartDate: gm.StartDate,
						EndDate:   gm.EndDate,
					},
					AnnualRate:       gm.AnnualRate,
					AnnualVolatility: gm.AnnualVolatility,
				})
			}
		}
		if len(fgms) == 1 {
			entity.GrowthModel = finance.NewGrowthCombined(fgms...)
		}
		if len(entity.Snapshots) > 0 {
			entities = append(entities, entity)
		}
	}
	for _, t := range trans {
		transfers = append(transfers, finance.TransferTemplate{
			ID:            t.ID,
			Name:          t.Name,
			FromAccountID: t.FromAccountID,
			ToAccountID:   t.ToAccountID,
			AmountType:    finance.TransferAmountType(t.AmountType),
			AmountFixed: finance.TransferFixed{
				Amount: t.AmountFixed,
			},
			AmountPercent: finance.TransferPercent{
				Percent: t.AmountPercent,
			},
			Priority:      t.Priority,
			EffectiveFrom: t.StartDate,
			EffectiveTo:   t.EndDate,
			Recurrence:    t.Recurrence,
			Enabled:       t.Enabled,
		})
	}

	sssEntities := make([]PredictionFinancialEntity, len(entities))
	for i, e := range entities {
		sssEntities[i] = PredictionFinancialEntity{
			ID:   e.ID,
			Name: e.Name,
		}
		for _, s := range e.Snapshots {
			q := s.Balance.Quantiles()
			sssEntities[i].Snapshots = append(sssEntities[i].Snapshots, PredictionBalanceSnapshot{
				ID:         e.ID,
				Day:        s.Date.ToStdTime().UnixMilli(),
				Balance:    s.Balance.Mean(),
				LowerBound: q(q1),
				UpperBound: q(q2),
			})
		}
	}

	startDate += 1
	endDate := startDate.Add(params.Duration)

	ucfg := uncertain.NewConfig(time.Now().UnixMilli(), params.Samples)
	if err := eventHandler.Setup(PredictionSetupEvent{
		Max:      endDate.ToStdTime().UnixMilli(),
		Entities: sssEntities,
	}); err != nil {
		return fmt.Errorf("sending SSE response: %w", err)
	}
	snapshotRecorder := finance.SnapshotRecorderFunc(func(accountID string, day date.Date, balance uncertain.Value) error {
		q := balance.Quantiles()
		event := PredictionBalanceSnapshot{
			ID:         accountID,
			Day:        day.ToStdTime().UnixMilli(),
			Balance:    balance.Mean(),
			LowerBound: q(q1),
			UpperBound: q(q2),
		}
		return eventHandler.Snapshot(event)
	})
	if err := finance.RunPrediction(ctx, ucfg, startDate, endDate, params.SnapshotInterval, entities, transfers, finance.CompositeRecorder{SnapshotRecorder: snapshotRecorder}); err != nil {
		return fmt.Errorf("running prediction for SSE: %w", err)
	}
	return eventHandler.Close()
}
