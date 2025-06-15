package finance

import (
	"context"
	"fmt"
	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/pefigo/internal/uncertain"
	"sort"
)

type BalanceSnapshot struct {
	Date    date.Date
	Balance uncertain.Value
}

type CashFlowModel struct {
	Frequency     date.Cron
	DestinationID string
}

type BalanceLimit struct {
	Upper uncertain.Value // Optional upper limit, if not set, no limit is applied
}

type Entity struct {
	ID   string
	Name string

	BalanceLimit BalanceLimit // Optional balance limit, if not set, no limit is applied

	// Balance snapshots for this entity, sorted by date
	Snapshots []BalanceSnapshot

	GrowthModel GrowthModel
	CashFlow    *CashFlowModel // Optional cash flow model, if not set, no cash flow is applied
}

func (fe *Entity) GetLatestSnapshot(day date.Date) BalanceSnapshot {
	foundSnapshot := sort.Search(len(fe.Snapshots), func(i int) bool {
		return fe.Snapshots[i].Date.After(day)
	})
	if foundSnapshot == 0 {
		// No snapshot before the given date, return zero balance
		return BalanceSnapshot{Balance: uncertain.NewFixed(0.0), Date: day}
	}
	return fe.Snapshots[foundSnapshot-1]
}

type ModeledEntity struct {
	Entity

	lastSnapshotDate    date.Date // Last date when the balance was updated
	balance             uncertain.Value
	accruedAppreciation uncertain.Value
}

func (fe *ModeledEntity) Init(day date.Date) {
	latestSnapshot := fe.GetLatestSnapshot(day)
	fe.lastSnapshotDate = latestSnapshot.Date
	fe.balance = latestSnapshot.Balance
}

func (fe *ModeledEntity) ApplyGrowth(ucfg *uncertain.Config, date date.Date) {
	if fe.GrowthModel == nil || !fe.GrowthModel.IsActiveOn(date) {
		return // No growth model or not active on this date
	}
	if fe.accruedAppreciation.Distribution == "" {
		fe.accruedAppreciation = uncertain.NewFixed(0.0) // Initialize if not set
	}
	totalBalance := fe.balance.Add(ucfg, fe.accruedAppreciation)
	dailyGrowth := fe.GrowthModel.Apply(ucfg, totalBalance)
	fe.accruedAppreciation = fe.accruedAppreciation.Add(ucfg, dailyGrowth)
}

func (fe *ModeledEntity) ApplyAppreciation(ucfg *uncertain.Config, entities map[string]*ModeledEntity, day date.Date) {
	if fe.accruedAppreciation.Zero() {
		return
	}
	if fe.CashFlow == nil || (fe.CashFlow.Frequency.Matches(day) && fe.CashFlow.DestinationID == "") {
		fe.balance = fe.balance.Add(ucfg, fe.accruedAppreciation)
		fe.accruedAppreciation = uncertain.NewFixed(0.0)
	} else if fe.CashFlow.Frequency.Matches(day) {
		// If a destination account is specified, add interest to that account
		if destAccount, ok := entities[fe.CashFlow.DestinationID]; ok {
			if destAccount.lastSnapshotDate.After(day) {
				panic("Destination account " + fe.CashFlow.DestinationID + " has a snapshot after the cash flow date " + day.String())
			}
			destAccount.balance = destAccount.balance.Add(ucfg, fe.accruedAppreciation)
		} else {
			panic("Could not find account with ID " + fe.CashFlow.DestinationID)
		}
		fe.accruedAppreciation = uncertain.NewFixed(0.0)
	}
}

func RunPrediction(ctx context.Context, ucfg *uncertain.Config, from, to date.Date, snapshotCron date.Cron, financialEntities []Entity, transfers []TransferTemplate, recorder Recorder) error {
	dailyTransfers := make([]TransferTemplate, 0)
	fes := make(map[string]*ModeledEntity)
	earliestDate := from
	for _, fe := range financialEntities {
		mfe := &ModeledEntity{
			Entity: fe,
		}
		mfe.Init(from) // Initialize each account with its balance on the most recent snapshot date
		if mfe.lastSnapshotDate.Before(earliestDate) {
			earliestDate = mfe.lastSnapshotDate // Find the earliest date across all financialEntities
		}
		fes[fe.ID] = mfe
	}
	for day := range date.Iter(earliestDate, to, date.Day) {
		if from <= day {
			for _, transfer := range transfers {
				if transfer.EffectiveFrom.After(day) || (transfer.EffectiveTo != nil && transfer.EffectiveTo.Before(day)) || !transfer.Enabled || !transfer.Recurrence.Matches(day) {
					continue // Skip transfers not effective on this day
				}
				dailyTransfers = append(dailyTransfers, transfer)
			}
			if len(dailyTransfers) > 0 {
				if err := applyDailyTransfers(ucfg, fes, dailyTransfers, day, recorder); err != nil {
					return fmt.Errorf("failed to apply daily transfers: %w", err)
				}
			}
		}

		for _, fe := range fes {
			if fe.lastSnapshotDate.Before(day) {
				fe.ApplyGrowth(ucfg, day)
			}
		}
		for _, fe := range fes {
			if fe.lastSnapshotDate.Before(day) {
				fe.ApplyAppreciation(ucfg, fes, day)
			}
		}
		if snapshotCron.Matches(day) {
			for _, fe := range fes {
				if fe.lastSnapshotDate.Before(day) {
					if err := recorder.OnSnapshot(fe.ID, day, fe.balance); err != nil {
						return fmt.Errorf("failed to record snapshot for %s: %w", fe.ID, err)
					}
					fe.lastSnapshotDate = day
				}
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		dailyTransfers = dailyTransfers[:0]
	}
	return nil
}
