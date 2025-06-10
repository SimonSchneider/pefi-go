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

func newAccount(id, name string, withFeatures ...func(*AccountModel)) *AccountModel {
	acc := &AccountModel{
		Account: Account{
			ID:   AccountID(id),
			Name: name,
		},
	}
	for _, f := range withFeatures {
		f(acc)
	}
	return acc
}

func withInterest(id string, annualRate uncertain.Value, compounding string, paymentDate date.Cron, payoutAccountID string) func(*AccountModel) {
	return func(acc *AccountModel) {
		acc.InterestModels = append(acc.InterestModels, InterestModel{
			ID:                   InterestModelID(id),
			AccountID:            acc.ID,
			AnnualRate:           annualRate,
			Compounding:          compounding,
			PaymentDate:          paymentDate,
			DestinationAccountID: AccountID(payoutAccountID),
		})
	}
}

func withBalance(d date.Date, v uncertain.Value) func(*AccountModel) {
	return func(acc *AccountModel) {
		acc.Snapshots = append(acc.Snapshots, AccountSnapshot{
			AccountID: acc.ID,
			Date:      d,
			Balance:   v,
		})
	}
}

func newTransfer(id TransferTemplateID, from, to AccountID, prio int64, cron date.Cron, transferFeatures ...func(template *TransferTemplate)) TransferTemplate {
	tt := TransferTemplate{
		ID:            id,
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

func withFixed(amount uncertain.Value) func(*TransferTemplate) {
	return func(tt *TransferTemplate) {
		tt.AmountType = AmountFixed
		tt.AmountFixed = TransferFixed{
			Amount: amount,
		}
	}
}

func runPredict(accounts []*AccountModel, transfers []TransferTemplate) error {
	return RunPrediction(
		uncertain.NewConfig(time.Now().UnixMilli(), 100),
		startDate,
		startDate.Add(1*date.Year).Add(2*date.Day),
		"*-*-01",
		accounts,
		transfers,
	)
}

var (
	firstDate = startDate.Add(-1 * date.Day)
	startDate = Must(date.ParseDate("2000-01-01"))
	endDate   = startDate.Add(1 * date.Year).Add(2 * date.Day)
)

func TestMortgageInterestPayments(t *testing.T) {
	checkAcc := newAccount("2", "Checking Account", withBalance(firstDate, uncertain.NewFixed(1000)))
	mortgAcc := newAccount("3", "Mortgage Account",
		withInterest("2", uncertain.NewFixed(0.03), "daily", "*-*-01", "2"),
		withBalance(firstDate, uncertain.NewFixed(-10000)),
	)
	if err := runPredict([]*AccountModel{checkAcc, mortgAcc}, nil); err != nil {
		t.Fatalf("failed to run simulation: %s", err)
	}
	if cBal := checkAcc.GetBalanceOn(endDate).Mean(); cBal > 710 || cBal < 690 {
		t.Errorf("checking account balance after interest payment is %f, expected around 700", cBal)
	}
	if mBal := mortgAcc.GetBalanceOn(endDate).Mean(); mBal != -10000 {
		t.Errorf("mortgage account balance after interest payment is %f, expected -10000", mBal)
	}
}

func TestSavingsAccountInterestPayments(t *testing.T) {
	savingsAcc := newAccount("1", "Savings Account",
		withInterest("1", uncertain.NewFixed(0.04), "daily", "*-*-01", ""),
		withBalance(firstDate, uncertain.NewFixed(1000)),
	)
	if err := runPredict([]*AccountModel{savingsAcc}, nil); err != nil {
		t.Fatalf("failed to run simulation: %s", err)
	}
	if bal := savingsAcc.GetBalanceOn(endDate).Mean(); bal < 1035 || bal > 1045 {
		t.Errorf("savings account balance after interest payment is %f, expected around 1040", bal)
	}
}

func TestTransfersToAndFromExternal(t *testing.T) {
	salaryAcc := newAccount("1", "Salary Account")
	salaryTrn := newTransfer("t1", "", "1", 1, "*-*-25", withFixed(uncertain.NewFixed(1000)))
	if err := runPredict([]*AccountModel{salaryAcc}, []TransferTemplate{salaryTrn}); err != nil {
		t.Fatalf("failed to run simulation: %s", err)
	}
	if bal := salaryAcc.GetBalanceOn(endDate).Mean(); bal != 12000 {
		t.Errorf("salary account balance after transfer is %f, expected to be 12000", bal)
	}
}

func TestTransfersBetweenAccounts(t *testing.T) {
	checkingAcc := newAccount("2", "Checking Account", withBalance(firstDate, uncertain.NewFixed(13000)))
	savingsAcc := newAccount("1", "Savings Account")
	savingsTrn := newTransfer("t1", "2", "1", 1, "*-*-25", withFixed(uncertain.NewFixed(1000)))
	if err := runPredict([]*AccountModel{checkingAcc, savingsAcc}, []TransferTemplate{savingsTrn}); err != nil {
		t.Fatalf("failed to run simulation: %s", err)
	}
	if bal := checkingAcc.GetBalanceOn(endDate).Mean(); bal != 1000 {
		t.Errorf("checking account balance after transfers is %f, expected to be 1000", bal)
	}
	if bal := savingsAcc.GetBalanceOn(endDate).Mean(); bal != 12000 {
		t.Errorf("savings account balance after transfers is %f, expected to be 12000", bal)
	}
}
