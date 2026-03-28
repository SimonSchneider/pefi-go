package service

import (
	"testing"

	"github.com/SimonSchneider/pefigo/internal/uncertain"
)

func TestNewTransferTemplatesView2_OneTimeTransfersExcludedFromMonthlyTotals(t *testing.T) {
	templates := []TransferTemplate{
		{
			ID:          "recurring-income",
			Name:        "Salary",
			ToAccountID: "acc1",
			AmountType:  "fixed",
			AmountFixed: uncertain.NewFixed(5000),
			Recurrence:  "*-*-25",
			Enabled:     true,
		},
		{
			ID:          "onetime-income",
			Name:        "Bonus",
			ToAccountID: "acc1",
			AmountType:  "fixed",
			AmountFixed: uncertain.NewFixed(10000),
			Recurrence:  "2024-06-01",
			Enabled:     true,
		},
		{
			ID:            "recurring-expense",
			Name:          "Rent",
			FromAccountID: "acc1",
			AmountType:    "fixed",
			AmountFixed:   uncertain.NewFixed(1200),
			Recurrence:    "*-*-1",
			Enabled:       true,
		},
		{
			ID:            "onetime-expense",
			Name:          "Moving cost",
			FromAccountID: "acc1",
			AmountType:    "fixed",
			AmountFixed:   uncertain.NewFixed(3000),
			Recurrence:    "2024-06-01",
			Enabled:       true,
		},
	}

	view := newTransferTemplatesView2(templates, templates, nil, nil)

	if view.MonthlyIncome != 5000 {
		t.Errorf("expected MonthlyIncome 5000 (excluding one-time bonus), got %f", view.MonthlyIncome)
	}
	if view.MonthlyExpenses != -1200 {
		t.Errorf("expected MonthlyExpenses -1200 (excluding one-time moving cost), got %f", view.MonthlyExpenses)
	}
}

func TestAccountFormMode(t *testing.T) {
	t.Run("standard when no startup share account", func(t *testing.T) {
		v := &AccountEditView2{}
		if got := v.AccountFormMode(); got != "standard" {
			t.Errorf("expected 'standard', got %q", got)
		}
	})

	t.Run("startup when startup share account exists", func(t *testing.T) {
		v := &AccountEditView2{
			StartupShareAccount: &StartupShareAccount{AccountID: "a1"},
		}
		if got := v.AccountFormMode(); got != "startup" {
			t.Errorf("expected 'startup', got %q", got)
		}
	})
}

func TestDerivedStartupShareFormatting(t *testing.T) {
	v := &AccountEditView2{
		DerivedStartupShareSummary: &DerivedStartupShareSummary{
			SharesOwned:              12345.67,
			TotalShares:              1000000.00,
			AvgPurchasePricePerShare: 0.0012345678,
		},
	}

	t.Run("shares owned uses thousand separator", func(t *testing.T) {
		got := v.GetStartupShareSharesOwned()
		if got != "12,346" {
			t.Errorf("expected '12,346', got %q", got)
		}
	})

	t.Run("total shares uses thousand separator", func(t *testing.T) {
		got := v.GetStartupShareTotalShares()
		if got != "1,000,000" {
			t.Errorf("expected '1,000,000', got %q", got)
		}
	})

	t.Run("avg purchase price keeps full precision", func(t *testing.T) {
		got := v.GetStartupSharePurchasePrice()
		if got != "0.0012345678" {
			t.Errorf("expected '0.0012345678', got %q", got)
		}
	})
}
