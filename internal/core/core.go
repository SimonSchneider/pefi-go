package core

import (
	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/pefigo/internal/uncertain"
	"sort"
	"time"
)

type AccountID string
type YieldModelID string
type TransferTemplateID string

type Account struct {
	ID        AccountID
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type AccountSnapshot struct {
	AccountID AccountID
	Date      date.Date
	Balance   uncertain.Value
}

type YieldModel struct {
	ID          YieldModelID
	AccountID   AccountID
	AnnualRate  uncertain.Value
	Compounding string
	StartDate   date.Date
	EndDate     *date.Date // optional
}

func (i YieldModel) AppliesTo(day date.Date) bool {
	// Check if the interest model applies to the given day
	if i.StartDate.After(day) {
		return false
	}
	if i.EndDate != nil && i.EndDate.Before(day) {
		return false
	}
	if i.Compounding == "daily" {
		return true // Daily compounding applies to every day
	} else if i.Compounding == "monthly" {
		if day.ToStdTime().Day() == 1 { // Monthly compounding applies to the first day of each month
			return true
		}
	} else if i.Compounding == "yearly" {
		if day.ToStdTime().Month() == time.January && day.ToStdTime().Day() == 1 { // Yearly compounding applies to the first day of the year
			return true
		}
	}
	return false // For any other compounding type, we assume it does not apply
}

func addYield(ucfg *uncertain.Config, v uncertain.Value, r uncertain.Value, mult float64) uncertain.Value {
	return v.Mul(ucfg, uncertain.NewFixed(1.0).Add(ucfg, r).Pow(ucfg, uncertain.NewFixed(mult))) // Daily compounding
}

func (i YieldModel) AddTo(ucfg *uncertain.Config, v uncertain.Value) uncertain.Value {
	// Calculate the yield based on the annual rate and return it
	switch i.Compounding {
	case "daily":
		return addYield(ucfg, v, i.AnnualRate, 1.0/365.0) // Daily compounding
	case "monthly":
		return addYield(ucfg, v, i.AnnualRate, 1.0/12.0) // Monthly compounding
	case "yearly":
		return addYield(ucfg, v, i.AnnualRate, 1.0) // Yearly compounding
	default:
		panic("Unknown compounding compounding")
	}
}

type TransferAmountType string

const (
	AmountFixed     TransferAmountType = "fixed"
	AmountPercent   TransferAmountType = "percent"
	AmountRemainder TransferAmountType = "remainder"
)

type TransferTemplate struct {
	ID   TransferTemplateID
	Name string

	FromAccountID AccountID
	ToAccountID   AccountID

	AmountType    TransferAmountType
	AmountFixed   TransferFixed
	AmountPercent TransferPercent
	Priority      int       // lower number = happens earlier
	Recurrence    date.Cron // e.g. "*-*-25"

	EffectiveFrom date.Date
	EffectiveTo   *date.Date
	Enabled       bool
}

type TransferFixed struct {
	Amount uncertain.Value
}
type TransferPercent struct {
	Percent float64 // e.g. 0.1 for 10%
}

type ModeledAccount struct {
	Account
	YieldModels []YieldModel
	Snapshots   []AccountSnapshot
}

func (a *ModeledAccount) GetBalanceOn(date date.Date) uncertain.Value {
	foundSnapshot := sort.Search(len(a.Snapshots), func(i int) bool {
		return a.Snapshots[i].Date.After(date)
	})
	if foundSnapshot == 0 {
		// No snapshot before the given date, return zero balance
		return uncertain.NewFixed(0.0)
	}
	return a.Snapshots[foundSnapshot-1].Balance
}

type ModeledAccountWithBalance struct {
	*ModeledAccount
	Balance uncertain.Value
}

func (m *ModeledAccountWithBalance) ApplyYield(ucfg *uncertain.Config, date date.Date) {
	for _, yield := range m.YieldModels {
		if yield.AppliesTo(date) {
			// Apply yield logic here, e.g., calculate yield based on the balance and annual rate
			// This is a placeholder for actual yield calculation logic
			m.Balance = yield.AddTo(ucfg, m.Balance)
		}
	}
}

func makeAccountWithBalance(accounts []*ModeledAccount, date date.Date) map[AccountID]*ModeledAccountWithBalance {
	accountsWithBalance := make(map[AccountID]*ModeledAccountWithBalance)
	for _, account := range accounts {
		lastSnapshot := AccountSnapshot{Balance: uncertain.NewFixed(0.0), AccountID: account.ID}
		for _, snapshot := range account.Snapshots {
			if snapshot.Date.Before(date) {
				lastSnapshot = snapshot
			}
		}
		accountsWithBalance[account.ID] = &ModeledAccountWithBalance{
			ModeledAccount: account,
			Balance:        lastSnapshot.Balance,
		}
	}
	return accountsWithBalance
}

func applyDailyTransfers(ucfg *uncertain.Config, accounts map[AccountID]*ModeledAccountWithBalance, transfers []TransferTemplate) {
	// Apply transfers for this day
	for _, transfer := range transfers {
		fromAccount, okFrom := accounts[transfer.FromAccountID]
		toAccount, okTo := accounts[transfer.ToAccountID]

		// These transfers will transfer as given, no uncertainty here, for empirical they are just changed by the transfer amount
		var amount uncertain.Value
		switch transfer.AmountType {
		case AmountFixed:
			amount = transfer.AmountFixed.Amount
		case AmountPercent:
			amount = uncertain.NewFixed(fromAccount.Balance.Sample(ucfg) * transfer.AmountPercent.Percent)
		case AmountRemainder:
			// In this case the from Account balance will become fixed to 0 so remember to set the balance correctly to fixed(0)
			amount = fromAccount.Balance // Remainder is the entire balance
		default:
			continue // Unknown amount type
		}

		if okFrom {
			// Perform the transfer
			fromAccount.Balance = fromAccount.Balance.Sub(ucfg, amount)
		}
		if okTo {
			toAccount.Balance = toAccount.Balance.Add(ucfg, amount)
		}
	}
}

func RunPrediction(ucfg *uncertain.Config, from, to date.Date, snapshotCron date.Cron, accounts []*ModeledAccount, transfers []TransferTemplate) error {
	dailyTransfers := make([]TransferTemplate, 0)
	accountsWithBalance := makeAccountWithBalance(accounts, from)
	for day := range date.Iter(from, to, date.Day) {
		for _, transfer := range transfers {
			if transfer.EffectiveFrom.After(day) || (transfer.EffectiveTo != nil && transfer.EffectiveTo.Before(day)) || !transfer.Enabled || !transfer.Recurrence.Matches(day) {
				continue // Skip transfers not effective on this day
			}
			dailyTransfers = append(dailyTransfers, transfer)
		}
		if len(dailyTransfers) > 0 {
			// Apply transfers for this day
			applyDailyTransfers(ucfg, accountsWithBalance, dailyTransfers)
		}

		// Apply yield for each account on this day
		for _, account := range accountsWithBalance {
			account.ApplyYield(ucfg, day)
		}

		if snapshotCron.Matches(day) {
			// Create a balance snapshot for each account
			for i, account := range accountsWithBalance {
				accountsWithBalance[i].ModeledAccount.Snapshots = append(account.Snapshots, AccountSnapshot{
					AccountID: account.ID,
					Date:      day,
					Balance:   account.Balance,
				})
			}
		}
		dailyTransfers = dailyTransfers[:0] // Reset for the next day
	}
	return nil
}
