package core

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/goslu/static/shttp"
	"github.com/SimonSchneider/pefigo/internal/finance"
	"github.com/SimonSchneider/pefigo/internal/pdb"
	"github.com/SimonSchneider/pefigo/internal/ui"
	"github.com/SimonSchneider/pefigo/internal/uncertain"
)

type BalanceChange int

const (
	BalanceUnchanged BalanceChange = iota
	BalanceIncreased
	BalanceDecreased
)

type AccountSnapshot struct {
	AccountID string
	Date      date.Date
	Balance   uncertain.Value
}

func (a AccountSnapshot) ToFinance() finance.BalanceSnapshot {
	return finance.BalanceSnapshot{
		Date:    a.Date,
		Balance: a.Balance,
	}
}

type AccountSnapshotInput struct {
	Date         date.Date
	Balance      uncertain.Value
	EmptyBalance bool
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
	} else if err := shttp.Parse(&a.Balance, ui.ParseUncertainValue, balanceStr, uncertain.NewFixed(0)); err != nil {
		return fmt.Errorf("parsing balance: %w", err)
	}
	return nil
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
