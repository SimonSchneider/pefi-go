package finance_test

import (
	"context"
	"fmt"
	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/goslu/sid"
	"github.com/SimonSchneider/pefigo/internal/finance"
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

func newAccount(name string, withFeatures ...func(*finance.Entity)) *finance.Entity {
	acc := &finance.Entity{
		ID:   fmt.Sprintf("'%s'(%s)", name, sid.MustNewString(2)),
		Name: name,
	}
	for _, f := range withFeatures {
		f(acc)
	}
	return acc
}

func withInterest(annualRate uncertain.Value, paymentDate date.Cron, payoutAccountID string) func(*finance.Entity) {
	return func(acc *finance.Entity) {
		acc.GrowthModel = &finance.FixedGrowth{
			AnnualRate: annualRate,
		}
		acc.CashFlow = &finance.CashFlowModel{
			Frequency:     paymentDate,
			DestinationID: payoutAccountID,
		}
	}
}

func withLogNormGrowth(annualRate, annualVolatility uncertain.Value) func(*finance.Entity) {
	return func(acc *finance.Entity) {
		acc.GrowthModel = &finance.LogNormalGrowth{
			AnnualRate:       annualRate,
			AnnualVolatility: annualVolatility,
		}
	}
}

func withGrowthModel(annualRate uncertain.Value) func(*finance.Entity) {
	return func(acc *finance.Entity) {
		acc.GrowthModel = &finance.LogNormalGrowth{
			AnnualRate: annualRate,
		}
	}
}

func withBalance(d date.Date, v uncertain.Value) func(*finance.Entity) {
	return func(acc *finance.Entity) {
		acc.Snapshots = append(acc.Snapshots, finance.BalanceSnapshot{
			Date:    d,
			Balance: v,
		})
	}
}

func newTransfer(from, to string, prio int64, cron date.Cron, transferFeatures ...func(template *finance.TransferTemplate)) finance.TransferTemplate {
	tt := finance.TransferTemplate{
		ID:            sid.MustNewString(32),
		FromAccountID: from,
		ToAccountID:   to,
		Priority:      prio,
		Recurrence:    cron,
		Enabled:       true,
	}
	for _, f := range transferFeatures {
		f(&tt)
	}
	return tt
}

func withFixed(amount uncertain.Value) func(*finance.TransferTemplate) {
	return func(tt *finance.TransferTemplate) {
		tt.AmountType = finance.AmountFixed
		tt.AmountFixed = finance.TransferFixed{
			Amount: amount,
		}
	}
}

func mks[T any](accounts ...T) []T {
	return accounts
}

func runPredict(accounts []finance.Entity, transfers []finance.TransferTemplate) (map[string]uncertain.Value, error) {
	m := make(map[string]finance.BalanceSnapshot)
	err := finance.RunPrediction(
		context.Background(),
		uncertain.NewConfig(time.Now().UnixMilli(), 2_000),
		startDate,
		startDate.Add(1*date.Year).Add(2*date.Day),
		"*-*-01",
		accounts,
		transfers,
		func(accountID string, day date.Date, balance uncertain.Value) error {
			if s, ok := m[accountID]; !ok || s.Date < day {
				m[accountID] = finance.BalanceSnapshot{
					Date:    day,
					Balance: balance,
				}
			}
			return nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to run prediction: %w", err)
	}
	results := make(map[string]uncertain.Value)
	for _, acc := range accounts {
		if s, ok := m[acc.ID]; ok {
			results[acc.ID] = s.Balance
		}
	}
	return results, nil
}

func isAround(value uncertain.Value, target float64) bool {
	if value.Distribution != uncertain.DistEmpirical {
		return value.Mean()*0.98 <= target && target <= value.Mean()*1.02
	}
	q := value.Quantiles()
	q1, q9 := q(0.025), q(0.975)
	return q1 <= target && target <= q9
}

var (
	firstDate = startDate.Add(-1 * date.Day)
	startDate = Must(date.ParseDate("2000-01-01"))
)

func TestMortgageInterestPayments(t *testing.T) {
	checkAcc := newAccount("Checking Account", withBalance(firstDate, uncertain.NewFixed(1000)))
	mortgAcc := newAccount("Mortgage Account",
		withInterest(uncertain.NewFixed(0.03), "*-*-01", checkAcc.ID),
		withBalance(firstDate, uncertain.NewFixed(-10000)),
	)
	bals, err := runPredict(mks(*checkAcc, *mortgAcc), nil)
	if err != nil {
		t.Fatalf("failed to run simulation: %s", err)
	}
	if cBal := bals[checkAcc.ID]; !isAround(cBal, 700) {
		t.Errorf("checking account balance after interest payment is %s, expected around 700", cBal)
	}
	if mBal := bals[mortgAcc.ID].Mean(); mBal != -10000 {
		t.Errorf("mortgage account balance after interest payment is %f, expected -10000", mBal)
	}
}

func TestSavingsAccountInterestPayments(t *testing.T) {
	savingsAcc := newAccount("Savings Account",
		withLogNormGrowth(uncertain.NewFixed(0.04), uncertain.NewFixed(0.04)),
		withBalance(firstDate, uncertain.NewFixed(1000)),
	)
	bals, err := runPredict(mks(*savingsAcc), nil)
	if err != nil {
		t.Fatalf("failed to run simulation: %s", err)
	}
	if bal := bals[savingsAcc.ID]; !isAround(bal, 1040) {
		t.Errorf("savings account balance after interest payment is %s, expected around 1040", bal)
	}
}

func TestTransfersToAndFromExternal(t *testing.T) {
	salaryAcc := newAccount("Salary Account")
	salaryTrn := newTransfer("", salaryAcc.ID, 1, "*-*-25", withFixed(uncertain.NewFixed(1000)))
	bals, err := runPredict(mks(*salaryAcc), mks(salaryTrn))
	if err != nil {
		t.Fatalf("failed to run simulation: %s", err)
	}
	if bal := bals[salaryAcc.ID].Mean(); bal != 12000 {
		t.Errorf("salary account balance after transfer is %f, expected to be 12000", bal)
	}
}

func TestTransfersBetweenAccounts(t *testing.T) {
	checkingAcc := newAccount("Checking Account", withBalance(firstDate, uncertain.NewFixed(13000)))
	savingsAcc := newAccount("Savings Account")
	savingsTrn := newTransfer(checkingAcc.ID, savingsAcc.ID, 1, "*-*-25", withFixed(uncertain.NewFixed(1000)))
	bals, err := runPredict(mks(*checkingAcc, *savingsAcc), mks(savingsTrn))
	if err != nil {
		t.Fatalf("failed to run simulation: %s", err)
	}
	if bal := bals[checkingAcc.ID].Mean(); bal != 1000 {
		t.Errorf("checking account balance after transfers is %f, expected to be 1000", bal)
	}
	if bal := bals[savingsAcc.ID].Mean(); bal != 12000 {
		t.Errorf("savings account balance after transfers is %f, expected to be 12000", bal)
	}
}

func TestRealEstateAppreciation(t *testing.T) {
	realEstateAcc := newAccount("Real Estate",
		withBalance(firstDate, uncertain.NewUniform(99_000, 101_000)),
		withGrowthModel(uncertain.NewUniform(0.00, 0.06)),
	)
	bals, err := runPredict(mks(*realEstateAcc), nil)
	if err != nil {
		t.Fatalf("failed to run simulation: %s", err)
	}
	if bal := bals[realEstateAcc.ID]; !isAround(bal, 103_000) {
		t.Errorf("real estate account balance after appreciation is %s, expected around 103000", bal)
	}
}

func BenchmarkRealEstateAppreciation(b *testing.B) {
	realEstateAcc := newAccount("Real Estate",
		withBalance(firstDate, uncertain.NewUniform(99_000, 101_000)),
		withGrowthModel(uncertain.NewUniform(0.00, 0.06)),
	)
	for b.Loop() {
		if _, err := runPredict(mks(*realEstateAcc), nil); err != nil {
			b.Fatalf("failed to run simulation: %s", err)
		}
	}
}

func TestInterestForwardingUntilFromDate(t *testing.T) {
	checkingAcc := newAccount("Checking Account", withBalance(firstDate, uncertain.NewFixed(1000)))
	savingsAcc := newAccount("Savings Account",
		withBalance(firstDate.Add(-date.Year), uncertain.NewFixed(1000)),
		withLogNormGrowth(uncertain.NewFixed(0.50), uncertain.NewFixed(0.05)),
	)
	salary := newTransfer("", checkingAcc.ID, 1, "*-*-25", withFixed(uncertain.NewFixed(10_000)))
	transfer := newTransfer(checkingAcc.ID, savingsAcc.ID, 1, "*-01-24", withFixed(uncertain.NewFixed(1000)))
	bals, err := runPredict(mks(*checkingAcc, *savingsAcc), mks(salary, transfer))
	if err != nil {
		t.Fatalf("failed to run simulation: %s", err)
	}
	if bal := bals[checkingAcc.ID]; !isAround(bal, 1000+10_000*12-1000) {
		t.Errorf("checking account balance after transfers is %s, expected to be around 1000", bal)
	}
	savingsTarget := 4327.0
	if bal := bals[savingsAcc.ID]; !isAround(bal, savingsTarget) {
		t.Errorf("savings account balance after interest payment is %s, expected around %f", bal, savingsTarget)
	}
}
