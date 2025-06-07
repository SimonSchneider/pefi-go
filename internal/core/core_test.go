package core

import (
	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/pefigo/internal/uncertain"
	"testing"
	"time"
)

func Must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func TestIdea(t *testing.T) {
	startDate := Must(date.ParseDate("2000-01-01"))
	accounts := []*ModeledAccount{
		{
			Account: Account{
				ID:   "1",
				Name: "Test Account 1",
			},
			YieldModels: []YieldModel{
				{
					ID:        "model1",
					AccountID: "1",
					//AnnualRate: uncertain.NewFixed(0.04),
					AnnualRate:  uncertain.NewUniform(0.02, 0.06),
					Compounding: "daily",
				},
			},
		},
	}
	transfers := []TransferTemplate{
		{
			ID:            "transfer1",
			FromAccountID: "",
			ToAccountID:   "1",
			AmountType:    AmountFixed,
			AmountFixed: TransferFixed{
				Amount: uncertain.NewFixed(1000),
			},
			Priority:   1,
			Recurrence: date.Cron("*-01-01"),
			Enabled:    true,
		},
	}
	if err := Run(
		uncertain.NewConfig(time.Now().UnixMilli(), 100),
		startDate,
		startDate.Add(1*date.Year),
		"*-*-01",
		accounts,
		transfers,
	); err != nil {
		t.Fatalf("failed to run simulation: %s", err)
	}
	for _, acc := range accounts {
		t.Logf("account: %s", acc.ID)
		for _, snapshot := range acc.Snapshots {
			t.Logf("    snapshot: %s, balance: %s", snapshot.Date.String(), snapshot.Balance.String())
		}
	}
}
