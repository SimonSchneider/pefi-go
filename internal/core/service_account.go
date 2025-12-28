package core

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/goslu/sid"
	"github.com/SimonSchneider/goslu/static/shttp"
	"github.com/SimonSchneider/pefigo/internal/pdb"
	"github.com/SimonSchneider/pefigo/internal/ui"
)

type Account struct {
	ID                    string
	Name                  string
	BalanceUpperLimit     *float64
	CashFlowFrequency     string
	CashFlowDestinationID string
	TypeID                string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type AccountDetailed struct {
	Account
	LastSnapshot        *AccountSnapshot
	GrowthModel         *GrowthModel
	StartupShareAccount *StartupShareAccount
}

type AccountInput struct {
	ID                    string
	Name                  string
	BalanceUpperLimit     *float64
	CashFlowFrequency     string
	CashFlowDestinationID string
	TypeID                string
}

func (a *AccountInput) FromForm(r *http.Request) error {
	a.ID = r.FormValue("id")
	a.Name = r.FormValue("name")
	if err := shttp.Parse(&a.BalanceUpperLimit, ui.ParseNullableFloat, r.FormValue("balance_upper_limit"), nil); err != nil {
		return fmt.Errorf("parsing balance limit: %w", err)
	}
	a.CashFlowFrequency = r.FormValue("cash_flow_frequency")
	a.CashFlowDestinationID = r.FormValue("cash_flow_destination_id")
	a.TypeID = r.FormValue("type_id")
	return nil
}

func accountFromDB(a pdb.Account) Account {
	return Account{
		ID:                    a.ID,
		Name:                  a.Name,
		BalanceUpperLimit:     a.BalanceUpperLimit,
		CashFlowFrequency:     ui.OrDefault(a.CashFlowFrequency),
		CashFlowDestinationID: ui.OrDefault(a.CashFlowDestinationID),
		TypeID:                ui.OrDefault(a.TypeID),
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
			CashFlowFrequency:     ui.WithDefaultNull(inp.CashFlowFrequency),
			CashFlowDestinationID: ui.WithDefaultNull(inp.CashFlowDestinationID),
			TypeID:                ui.WithDefaultNull(inp.TypeID),
			UpdatedAt:             time.Now().UnixMilli(),
		})
	} else {
		acc, err = q.CreateAccount(ctx, pdb.CreateAccountParams{
			ID:                    sid.MustNewString(15),
			Name:                  inp.Name,
			BalanceUpperLimit:     inp.BalanceUpperLimit,
			CashFlowFrequency:     ui.WithDefaultNull(inp.CashFlowFrequency),
			CashFlowDestinationID: ui.WithDefaultNull(inp.CashFlowDestinationID),
			TypeID:                ui.WithDefaultNull(inp.TypeID),
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

func accountsListFromDBDetailed(dbAccs []pdb.Account, snapshots map[string]pdb.AccountSnapshot, growthModels map[string]pdb.GrowthModel, startupShareAccounts map[string]pdb.StartupShareAccount) []AccountDetailed {
	accs := make([]AccountDetailed, len(dbAccs))
	for i := range dbAccs {
		accs[i].Account = accountFromDB(dbAccs[i])
		if snapshot, ok := snapshots[accs[i].ID]; ok {
			s := accountSnapshotFromDB(snapshot)
			accs[i].LastSnapshot = &s
		}
		if gm, ok := growthModels[accs[i].ID]; ok {
			gmd, err := growthModelFromDB(gm)
			if err != nil {
				panic(err)
			}
			accs[i].GrowthModel = &gmd
		}
		if ssa, ok := startupShareAccounts[accs[i].ID]; ok {
			ssaCore := startupShareAccountFromDB(ssa)
			accs[i].StartupShareAccount = &ssaCore
		}
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

func ptr[T any](v T) *T {
	return &v
}

func ListAccountsDetailed(ctx context.Context, db *sql.DB, today date.Date) ([]AccountDetailed, error) {
	accs, err := pdb.New(db).ListAccounts(ctx)
	if err != nil {
		return nil, err
	}
	snapshots, err := pdb.New(db).ListLatestSnapshotPerAccount(ctx)
	if err != nil {
		return nil, err
	}
	snapshotsMap := make(map[string]pdb.AccountSnapshot)
	for _, snapshot := range snapshots {
		snapshotsMap[snapshot.AccountID] = snapshot
	}
	growthModels, err := pdb.New(db).ListActiveGrowthModels(ctx, ptr(int64(today)))
	if err != nil {
		return nil, err
	}
	growthModelsMap := make(map[string]pdb.GrowthModel)
	for _, growthModel := range growthModels {
		growthModelsMap[growthModel.AccountID] = growthModel
	}
	// Load startup share accounts for all accounts
	startupShareAccountsMap := make(map[string]pdb.StartupShareAccount)
	for _, acc := range accs {
		ssa, err := pdb.New(db).GetStartupShareAccount(ctx, acc.ID)
		if err != nil {
			if err != sql.ErrNoRows {
				return nil, fmt.Errorf("failed to get startup share account: %w", err)
			}
			// No startup share account for this account, skip
			continue
		}
		startupShareAccountsMap[acc.ID] = ssa
	}
	return accountsListFromDBDetailed(accs, snapshotsMap, growthModelsMap, startupShareAccountsMap), nil
}
