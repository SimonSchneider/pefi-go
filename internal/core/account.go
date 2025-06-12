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
)

type Account struct {
	ID        string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type AccountSnapshot struct {
	AccountID string
	Date      date.Date
	Balance   uncertain.Value
}

type AccountInput struct {
	ID   string
	Name string
}

func (a *AccountInput) FromForm(r *http.Request) error {
	a.ID = r.FormValue("id")
	a.Name = r.FormValue("name")
	return nil
}

func accountFromDB(a pdb.Account) Account {
	return Account{
		ID:        a.ID,
		Name:      a.Name,
		CreatedAt: time.UnixMilli(a.CreatedAt),
		UpdatedAt: time.UnixMilli(a.UpdatedAt),
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
	f, err := shttp.ParseFloat(val)
	if err != nil {
		return uncertain.Value{}, fmt.Errorf("parsing uncertain value: %w", err)
	}
	return uncertain.NewFixed(f), nil
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
	s, err := pdb.New(db).UpsertSnapshot(ctx, pdb.UpsertSnapshotParams{
		AccountID: accountID,
		Date:      int64(inp.Date),
		Balance:   inp.Balance.Mean(),
	})
	if err != nil {
		return AccountSnapshot{}, fmt.Errorf("failed to upsert account: %w", err)
	}
	return accountSnapshotFromDB(s), nil
}

func accountSnapshotFromDB(s pdb.AccountSnapshot) AccountSnapshot {
	return AccountSnapshot{
		AccountID: s.AccountID,
		Date:      date.Date(s.Date),
		Balance:   uncertain.NewFixed(s.Balance),
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

func UpsertAccount(ctx context.Context, db *sql.DB, inp AccountInput) (Account, error) {
	var (
		q   = pdb.New(db)
		acc pdb.Account
		err error
	)
	if inp.ID != "" {
		acc, err = q.UpdateAccount(ctx, pdb.UpdateAccountParams{
			ID:        inp.ID,
			Name:      inp.Name,
			UpdatedAt: time.Now().UnixMilli(),
		})
	} else {
		acc, err = q.CreateAccount(ctx, pdb.CreateAccountParams{
			ID:        sid.MustNewString(15),
			Name:      inp.Name,
			CreatedAt: time.Now().UnixMilli(),
			UpdatedAt: time.Now().UnixMilli(),
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
