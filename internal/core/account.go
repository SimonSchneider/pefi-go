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

type GrowthModel struct {
	ID               string
	AccountID        string
	Type             string
	AnnualRate       uncertain.Value
	AnnualVolatility uncertain.Value
	StartDate        date.Date
	EndDate          *date.Date
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

type AccountGrowthModelInput struct {
	ID               string
	AccountID        string
	Type             string
	AnnualRate       uncertain.Value
	AnnualVolatility uncertain.Value

	StartDate date.Date
	EndDate   *date.Date
}

func (a *AccountGrowthModelInput) FromForm(r *http.Request) error {
	a.ID = r.FormValue("id")
	a.AccountID = r.FormValue("account_id")
	a.Type = r.FormValue("type")
	if a.Type != "fixed" && a.Type != "lognormal" {
		return fmt.Errorf("invalid growth model type: %s", a.Type)
	}
	if err := shttp.Parse(&a.AnnualRate, parseUncertainValue, r.FormValue("annual_rate"), uncertain.NewFixed(0)); err != nil {
		return fmt.Errorf("parsing annual rate: %w", err)
	}
	if err := shttp.Parse(&a.AnnualVolatility, parseUncertainValue, r.FormValue("annual_volatility"), uncertain.NewFixed(0)); err != nil {
		return fmt.Errorf("parsing annual volatility: %w", err)
	}
	if err := shttp.Parse(&a.StartDate, date.ParseDate, r.FormValue("start_date"), date.Date(0)); err != nil {
		return fmt.Errorf("parsing start date: %w", err)
	}
	if endDateStr := r.FormValue("end_date"); endDateStr != "" {
		var endDate date.Date
		if err := shttp.Parse(&endDate, date.ParseDate, endDateStr, date.Date(0)); err != nil {
			return fmt.Errorf("parsing end date: %w", err)
		}
		if endDate.IsZero() {
			a.EndDate = nil
		} else {
			a.EndDate = &endDate
		}
	}
	return nil
}

func UpsertAccountGrowthModel(ctx context.Context, db *sql.DB, inp AccountGrowthModelInput) (GrowthModel, error) {
	var endDate *int64
	if inp.EndDate != nil {
		*endDate = int64(*inp.EndDate)
	}
	annualRate, err := inp.AnnualRate.Encode()
	if err != nil {
		return GrowthModel{}, fmt.Errorf("encoding annual rate: %w", err)
	}
	annualVolatility, err := inp.AnnualVolatility.Encode()
	if err != nil {
		return GrowthModel{}, fmt.Errorf("encoding annual volatility: %w", err)
	}
	gm, err := pdb.New(db).UpsertGrowthModel(ctx, pdb.UpsertGrowthModelParams{
		ID:               inp.ID,
		AccountID:        inp.AccountID,
		ModelType:        inp.Type,
		AnnualGrowthRate: annualRate,
		AnnualVolatility: annualVolatility,
		StartDate:        int64(inp.StartDate),
		EndDate:          endDate,
		CreatedAt:        time.Now().UnixMilli(),
		UpdatedAt:        time.Now().UnixMilli(),
	})
	if err != nil {
		return GrowthModel{}, fmt.Errorf("failed to upsert growth model: %w", err)
	}
	return growthModelFromDB(gm)
}

func growthModelFromDB(g pdb.GrowthModel) (GrowthModel, error) {
	var annualRate, annualVolatility uncertain.Value
	if err := annualRate.Decode(g.AnnualGrowthRate); err != nil {
		return GrowthModel{}, fmt.Errorf("decoding annual growth rate: %w", err)
	}
	if err := annualVolatility.Decode(g.AnnualVolatility); err != nil {
		return GrowthModel{}, fmt.Errorf("decoding annual volatility: %w", err)
	}
	var endDate *date.Date
	if g.EndDate != nil {
		d := date.Date(*g.EndDate)
		endDate = &d
	}
	return GrowthModel{
		ID:               g.ID,
		AccountID:        g.AccountID,
		Type:             g.ModelType,
		AnnualRate:       annualRate,
		AnnualVolatility: annualVolatility,
		StartDate:        date.Date(g.StartDate),
		EndDate:          endDate,
	}, nil
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

func ListAccountGrowthModels(ctx context.Context, db *sql.DB, accountID string) ([]GrowthModel, error) {
	gms, err := pdb.New(db).GetGrowthModelsByAccount(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to list account growth models: %w", err)
	}
	gmList := make([]GrowthModel, len(gms))
	for i, g := range gms {
		gm, err := growthModelFromDB(g)
		if err != nil {
			return nil, fmt.Errorf("failed to convert growth model from db: %w", err)
		}
		gmList[i] = gm
	}
	return gmList, nil
}

func GetGrowthModel(ctx context.Context, db *sql.DB, id string) (GrowthModel, error) {
	g, err := pdb.New(db).GetGrowthModel(ctx, id)
	if err != nil {
		return GrowthModel{}, fmt.Errorf("failed to get account growth model: %w", err)
	}
	gm, err := growthModelFromDB(g)
	if err != nil {
		return GrowthModel{}, fmt.Errorf("failed to convert growth model from db: %w", err)
	}
	return gm, nil
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
