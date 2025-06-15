package core

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/goslu/sid"
	"github.com/SimonSchneider/goslu/static/shttp"
	"github.com/SimonSchneider/pefigo/internal/pdb"
	"github.com/SimonSchneider/pefigo/internal/uncertain"
	"net/http"
	"time"
	"unicode"
)

type UncertainValue struct {
	Distribution uncertain.DistributionType `json:"distribution"`
	Parameters   map[string]float64         `json:"parameters"`
	Samples      []float64                  `json:"samples"`
}

type Account struct {
	ID                    string
	Name                  string
	BalanceUpperLimit     *float64
	CashFlowFrequency     string
	CashFlowDestinationID string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type AccountSnapshot struct {
	AccountID string
	Date      date.Date
	Balance   uncertain.Value
}

type AccountInput struct {
	ID                    string
	Name                  string
	BalanceUpperLimit     *float64
	CashFlowFrequency     string
	CashFlowDestinationID string
}

func parseNullableFloat(val string) (*float64, error) {
	if val == "" {
		return nil, nil
	}
	f, err := shttp.ParseFloat(val)
	if err != nil {
		return nil, fmt.Errorf("parsing float: %w", err)
	}
	return &f, nil
}

func (a *AccountInput) FromForm(r *http.Request) error {
	a.ID = r.FormValue("id")
	a.Name = r.FormValue("name")
	if err := shttp.Parse(&a.BalanceUpperLimit, parseNullableFloat, r.FormValue("balance_upper_limit"), nil); err != nil {
		return fmt.Errorf("parsing balance limit: %w", err)
	}
	a.CashFlowFrequency = r.FormValue("cash_flow_frequency")
	a.CashFlowDestinationID = r.FormValue("cash_flow_destination_id")
	return nil
}

func orDefault[T any](val *T) T {
	if val == nil {
		var zero T
		return zero
	}
	return *val
}

func accountFromDB(a pdb.Account) Account {
	return Account{
		ID:                    a.ID,
		Name:                  a.Name,
		BalanceUpperLimit:     a.BalanceUpperLimit,
		CashFlowFrequency:     orDefault(a.CashFlowFrequency),
		CashFlowDestinationID: orDefault(a.CashFlowDestinationID),
		CreatedAt:             time.UnixMilli(a.CreatedAt),
		UpdatedAt:             time.UnixMilli(a.UpdatedAt),
	}
}

func GetAccount(ctx context.Context, db *sql.DB, id string) (Account, error) {
	acc, err := pdb.New(db).GetAccount(ctx, id)
	if err != nil {
		return Account{}, fmt.Errorf("failed to get account: %w", err)
	}
	return accountFromDB(acc), nil
}

type AccountSnapshotInput struct {
	Date         date.Date
	Balance      uncertain.Value
	EmptyBalance bool
}

func parseUncertainValue(val string) (uncertain.Value, error) {
	if unicode.IsDigit(rune(val[0])) || (len(val) > 1 && val[0] == '-' && unicode.IsDigit(rune(val[1]))) {
		// If the value is a simple float, return a fixed uncertain value
		f, err := shttp.ParseFloat(val)
		if err != nil {
			return uncertain.Value{}, fmt.Errorf("parsing float: %w", err)
		}
		return uncertain.NewFixed(f), nil
	}
	// Otherwise, parse it as an uncertain value
	var value uncertain.Value
	if err := value.Decode(val); err != nil {
		return uncertain.Value{}, fmt.Errorf("decoding uncertain value: %w", err)
	}
	return value, nil
}

func (a *AccountSnapshotInput) FromForm(r *http.Request) error {
	dateStr := r.PathValue("date")
	if dateStr == "" {
		dateStr = r.FormValue("date")
	}
	if err := shttp.Parse(&a.Date, date.ParseDate, dateStr, 0); err != nil {
		return fmt.Errorf("parsing date: %w", err)
	}
	balanceStr := r.FormValue("balance")
	if balanceStr == "" {
		a.EmptyBalance = true
	} else if err := shttp.Parse(&a.Balance, parseUncertainValue, balanceStr, uncertain.NewFixed(0)); err != nil {
		return fmt.Errorf("parsing balance: %w", err)
	}
	return nil
}

func UpsertAccountSnapshot(ctx context.Context, db *sql.DB, accountID string, inp AccountSnapshotInput) (AccountSnapshot, error) {
	balance, err := inp.Balance.Encode()
	if err != nil {
		return AccountSnapshot{}, fmt.Errorf("encoding balance: %w", err)
	}
	s, err := pdb.New(db).UpsertSnapshot(ctx, pdb.UpsertSnapshotParams{
		AccountID: accountID,
		Date:      int64(inp.Date),
		Balance:   balance,
	})
	if err != nil {
		return AccountSnapshot{}, fmt.Errorf("failed to upsert account: %w", err)
	}
	return accountSnapshotFromDB(s), nil
}

func accountSnapshotFromDB(s pdb.AccountSnapshot) AccountSnapshot {
	var balance uncertain.Value
	if err := balance.Decode(s.Balance); err != nil {
		panic(fmt.Errorf("decoding balance: %w", err))
	}
	return AccountSnapshot{
		AccountID: s.AccountID,
		Date:      date.Date(s.Date),
		Balance:   balance,
	}
}

func DeleteAccountSnapshot(ctx context.Context, db *sql.DB, id string, date date.Date) error {
	if err := pdb.New(db).DeleteSnapshot(ctx, pdb.DeleteSnapshotParams{AccountID: id, Date: int64(date)}); err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}
	return nil
}

func ListAccountSnapshots(ctx context.Context, db *sql.DB, id string) ([]AccountSnapshot, error) {
	return ListAccountsSnapshots(ctx, db, []string{id})
}

func ListAccountsSnapshots(ctx context.Context, db *sql.DB, ids []string) ([]AccountSnapshot, error) {
	snapshots, err := pdb.New(db).GetSnapshotsByAccounts(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to list account snapshots: %w", err)
	}
	snapshotList := make([]AccountSnapshot, len(snapshots))
	for i, s := range snapshots {
		snapshotList[i] = accountSnapshotFromDB(s)
	}
	return snapshotList, nil
}

func GetAccountSnapshot(ctx context.Context, db *sql.DB, id string, date date.Date) (AccountSnapshot, error) {
	s, err := pdb.New(db).GetSnapshot(ctx, pdb.GetSnapshotParams{
		AccountID: string(id),
		Date:      int64(date),
	})
	if err != nil {
		return AccountSnapshot{}, fmt.Errorf("failed to get account snapshot: %w", err)
	}
	return accountSnapshotFromDB(s), nil
}

func withDefaultNull[T comparable](val T) *T {
	var zero T
	if val == zero {
		return nil
	}
	return &val
}

func UpsertAccount(ctx context.Context, db *sql.DB, inp AccountInput) (Account, error) {
	var (
		q   = pdb.New(db)
		acc pdb.Account
		err error
	)
	if inp.ID != "" {
		acc, err = q.UpdateAccount(ctx, pdb.UpdateAccountParams{
			ID:                    inp.ID,
			Name:                  inp.Name,
			BalanceUpperLimit:     inp.BalanceUpperLimit,
			CashFlowFrequency:     withDefaultNull(inp.CashFlowFrequency),
			CashFlowDestinationID: withDefaultNull(inp.CashFlowDestinationID),
			UpdatedAt:             time.Now().UnixMilli(),
		})
	} else {
		acc, err = q.CreateAccount(ctx, pdb.CreateAccountParams{
			ID:                    sid.MustNewString(15),
			Name:                  inp.Name,
			BalanceUpperLimit:     inp.BalanceUpperLimit,
			CashFlowFrequency:     withDefaultNull(inp.CashFlowFrequency),
			CashFlowDestinationID: withDefaultNull(inp.CashFlowDestinationID),
			CreatedAt:             time.Now().UnixMilli(),
			UpdatedAt:             time.Now().UnixMilli(),
		})
	}
	if err != nil {
		return Account{}, fmt.Errorf("failed to upsert account: %w", err)
	}
	return accountFromDB(acc), nil
}

func DeleteAccount(ctx context.Context, db *sql.DB, id string) error {
	_, err := pdb.New(db).DeleteAccount(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}
	return nil
}

func accountsListFromDB(dbAccs []pdb.Account) []Account {
	accs := make([]Account, len(dbAccs))
	for i := range dbAccs {
		accs[i] = accountFromDB(dbAccs[i])
	}
	return accs
}

func ListAccounts(ctx context.Context, db *sql.DB) ([]Account, error) {
	accs, err := pdb.New(db).ListAccounts(ctx)
	if err != nil {
		return nil, err
	}
	return accountsListFromDB(accs), nil
}
