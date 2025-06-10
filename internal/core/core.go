package core

import (
	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/pefigo/internal/uncertain"
	"sort"
	"time"
)

type AccountID string
type InterestModelID string
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

type InterestModel struct {
	ID                   InterestModelID
	AccountID            AccountID
	AnnualRate           uncertain.Value
	Compounding          string
	PaymentDate          date.Cron
	DestinationAccountID AccountID // optional, if not set, interest is added to the same account
	StartDate            date.Date
	EndDate              *date.Date // optional
}

func (i InterestModel) IsActiveOn(day date.Date) bool {
	// Check if the interest model applies to the given day
	if i.StartDate.After(day) {
		return false
	}
	if i.EndDate != nil && i.EndDate.Before(day) {
		return false
	}
	return true
}

func (i InterestModel) CompoundsOn(day date.Date) bool {
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

func (i InterestModel) PaysOutOn(d date.Date) bool {
	return i.PaymentDate.Matches(d)
}

func calculateInterest(ucfg *uncertain.Config, v uncertain.Value, r uncertain.Value, mult float64) uncertain.Value {
	return v.Mul(ucfg, r.Add(ucfg, uncertain.NewFixed(1)).Pow(ucfg, uncertain.NewFixed(mult)).Sub(ucfg, uncertain.NewFixed(1))) // Daily compounding
}

func (i InterestModel) Calculate(ucfg *uncertain.Config, v uncertain.Value) uncertain.Value {
	// Calculate the interest based on the annual rate and return it
	switch i.Compounding {
	case "daily":
		return calculateInterest(ucfg, v, i.AnnualRate, 1.0/365.0) // Daily compounding
	case "monthly":
		return calculateInterest(ucfg, v, i.AnnualRate, 1.0/12.0) // Monthly compounding
	case "yearly":
		return calculateInterest(ucfg, v, i.AnnualRate, 1.0) // Yearly compounding
	default:
		panic("Unknown compounding compounding")
	}
}

type TransferAmountType string

const (
	AmountFixed   TransferAmountType = "fixed"
	AmountPercent TransferAmountType = "percent"
)

type TransferTemplate struct {
	ID   TransferTemplateID
	Name string

	FromAccountID AccountID
	ToAccountID   AccountID

	AmountType    TransferAmountType
	AmountFixed   TransferFixed
	AmountPercent TransferPercent
	Priority      int64     // lower number = happens earlier
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

type AccountModel struct {
	Account
	InterestModels []InterestModel
	Snapshots      []AccountSnapshot
}

func (a *AccountModel) GetBalanceOn(date date.Date) uncertain.Value {
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
	*AccountModel
	Balance         uncertain.Value
	AccruedInterest map[InterestModelID]uncertain.Value // This is the interest accrued so far, not yet applied to the balance
}

func (m *ModeledAccountWithBalance) ApplyInterest(ucfg *uncertain.Config, accounts map[AccountID]*ModeledAccountWithBalance, date date.Date) {
	for _, interest := range m.InterestModels {
		if interest.IsActiveOn(date) {
			if interest.CompoundsOn(date) {
				accruedInterest := m.AccruedInterest[interest.ID]
				if accruedInterest.Distribution == "" {
					accruedInterest = uncertain.NewFixed(0.0) // Initialize if not set
				}
				totalBalance := m.Balance.Add(ucfg, accruedInterest)
				dailyInterest := interest.Calculate(ucfg, totalBalance)
				m.AccruedInterest[interest.ID] = accruedInterest.Add(ucfg, dailyInterest)
			}
		}
		accruedInterest := m.AccruedInterest[interest.ID]
		if !accruedInterest.Zero() && interest.PaysOutOn(date) {
			if interest.DestinationAccountID == "" {
				// If no destination account is specified, add interest to the same account
				m.Balance = m.Balance.Add(ucfg, accruedInterest)
			} else {
				// If a destination account is specified, add interest to that account
				if destAccount, ok := accounts[interest.DestinationAccountID]; ok {
					destAccount.Balance = destAccount.Balance.Add(ucfg, accruedInterest)
				} else {
					panic("Could not find account with ID " + interest.DestinationAccountID)
				}
			}
			// Reset accrued interest after payout
			delete(m.AccruedInterest, interest.ID)
		}
	}
}

func makeAccountWithBalance(accounts []*AccountModel, date date.Date) map[AccountID]*ModeledAccountWithBalance {
	accountsWithBalance := make(map[AccountID]*ModeledAccountWithBalance)
	for _, account := range accounts {
		lastSnapshot := AccountSnapshot{Balance: uncertain.NewFixed(0.0), AccountID: account.ID}
		for _, snapshot := range account.Snapshots {
			if snapshot.Date.Before(date) {
				lastSnapshot = snapshot
			}
		}
		accountsWithBalance[account.ID] = &ModeledAccountWithBalance{
			AccountModel:    account,
			Balance:         lastSnapshot.Balance,
			AccruedInterest: make(map[InterestModelID]uncertain.Value),
		}
	}
	return accountsWithBalance
}

func applyDailyTransfers(ucfg *uncertain.Config, accounts map[AccountID]*ModeledAccountWithBalance, transfers []TransferTemplate) {
	// TODO: transfers of equal priority should be applied at the same time
	// All transfers in the same priority will run simultaneously and % will be based on balance before the priority group
	// We could have a flag to allow transfers to draw money from the target account if the source account is negative
	// ie. A1=-100, A2=200,
	// T1: A1 -> A2, 50% -> A1=-50, A2=150
	// T2: A2 -> A1, 50% -> A1= 25, A2= 75

	// TODO: Do we need to support transfers which can understand if the destination account requires money?
	// For example we have a negative account and want to transfer money to it if it is negative but not if it is positive. (doesn't this solve the above question too?)
	// This is useful for loan payments and fill ups of salary accounts from saving accounts if they can't cover basic expenses.
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

func RunPrediction(ucfg *uncertain.Config, from, to date.Date, snapshotCron date.Cron, accounts []*AccountModel, transfers []TransferTemplate) error {
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

		// Apply interest for each account on this day
		for _, account := range accountsWithBalance {
			account.ApplyInterest(ucfg, accountsWithBalance, day)
		}
		if snapshotCron.Matches(day) {
			// Create a balance snapshot for each account
			for i, account := range accountsWithBalance {
				accountsWithBalance[i].AccountModel.Snapshots = append(account.Snapshots, AccountSnapshot{
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
