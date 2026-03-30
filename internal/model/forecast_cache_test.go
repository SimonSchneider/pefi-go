package model_test

import (
	"testing"

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

	// Should have rows for the account type
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
