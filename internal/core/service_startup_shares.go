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
	"github.com/SimonSchneider/pefigo/internal/uncertain"
)

type StartupShareAccount struct {
	AccountID               string
	TaxRate                 float64
	ValuationDiscountFactor float64
}

type InvestmentRound struct {
	ID              string
	AccountID       string
	Date            date.Date
	Valuation       float64 // pre-money valuation
	PreMoneyShares  float64
	Investment      float64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type ShareChange struct {
	ID          string
	AccountID   string
	Date        date.Date
	DeltaShares float64
	TotalPrice  float64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type StartupShareOption struct {
	ID                  string
	AccountID           string
	SourceAccountID     string
	Shares              float64
	StrikePricePerShare float64
	GrantDate           date.Date
	EndDate             date.Date
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type StartupShareAccountInput struct {
	AccountID               string
	TaxRate                 float64
	ValuationDiscountFactor float64
}

func (s *StartupShareAccountInput) FromForm(r *http.Request) error {
	s.AccountID = r.FormValue("account_id")
	if err := shttp.Parse(&s.TaxRate, shttp.ParseFloat, r.FormValue("tax_rate"), 0.0); err != nil {
		return fmt.Errorf("parsing tax rate: %w", err)
	}
	if err := shttp.Parse(&s.ValuationDiscountFactor, shttp.ParseFloat, r.FormValue("valuation_discount_factor"), 0.5); err != nil {
		return fmt.Errorf("parsing valuation discount factor: %w", err)
	}
	return nil
}

type InvestmentRoundInput struct {
	ID             string
	AccountID      string
	Date           date.Date
	Valuation      float64
	PreMoneyShares float64
	Investment     float64
}

func (i *InvestmentRoundInput) FromForm(r *http.Request) error {
	i.ID = r.FormValue("id")
	i.AccountID = r.FormValue("account_id")
	if err := shttp.Parse(&i.Date, date.ParseDate, r.FormValue("date"), date.Date(0)); err != nil {
		return fmt.Errorf("parsing date: %w", err)
	}
	if err := shttp.Parse(&i.Valuation, shttp.ParseFloat, r.FormValue("valuation"), 0.0); err != nil {
		return fmt.Errorf("parsing valuation: %w", err)
	}
	if err := shttp.Parse(&i.PreMoneyShares, shttp.ParseFloat, r.FormValue("pre_money_shares"), 0.0); err != nil {
		return fmt.Errorf("parsing pre_money_shares: %w", err)
	}
	if err := shttp.Parse(&i.Investment, shttp.ParseFloat, r.FormValue("investment"), 0.0); err != nil {
		return fmt.Errorf("parsing investment: %w", err)
	}
	return nil
}

type ShareChangeInput struct {
	ID          string
	AccountID   string
	Date        date.Date
	DeltaShares float64
	TotalPrice  float64
}

func (s *ShareChangeInput) FromForm(r *http.Request) error {
	s.ID = r.FormValue("id")
	s.AccountID = r.FormValue("account_id")
	if err := shttp.Parse(&s.Date, date.ParseDate, r.FormValue("date"), date.Date(0)); err != nil {
		return fmt.Errorf("parsing date: %w", err)
	}
	if err := shttp.Parse(&s.DeltaShares, shttp.ParseFloat, r.FormValue("delta_shares"), 0.0); err != nil {
		return fmt.Errorf("parsing delta_shares: %w", err)
	}
	if err := shttp.Parse(&s.TotalPrice, shttp.ParseFloat, r.FormValue("total_price"), 0.0); err != nil {
		return fmt.Errorf("parsing total_price: %w", err)
	}
	return nil
}

type StartupShareOptionInput struct {
	ID                  string
	AccountID           string
	SourceAccountID     string
	Shares              float64
	StrikePricePerShare float64
	GrantDate           date.Date
	EndDate             date.Date
}

func (o *StartupShareOptionInput) FromForm(r *http.Request) error {
	o.ID = r.FormValue("id")
	o.AccountID = r.FormValue("account_id")
	o.SourceAccountID = r.FormValue("source_account_id")
	if err := shttp.Parse(&o.Shares, shttp.ParseFloat, r.FormValue("shares"), 0.0); err != nil {
		return fmt.Errorf("parsing shares: %w", err)
	}
	if err := shttp.Parse(&o.StrikePricePerShare, shttp.ParseFloat, r.FormValue("strike_price_per_share"), 0.0); err != nil {
		return fmt.Errorf("parsing strike price per share: %w", err)
	}
	if err := shttp.Parse(&o.GrantDate, date.ParseDate, r.FormValue("grant_date"), date.Date(0)); err != nil {
		return fmt.Errorf("parsing grant date: %w", err)
	}
	if err := shttp.Parse(&o.EndDate, date.ParseDate, r.FormValue("end_date"), date.Date(0)); err != nil {
		return fmt.Errorf("parsing end date: %w", err)
	}
	return nil
}

func startupShareAccountFromDB(s pdb.StartupShareAccount) StartupShareAccount {
	return StartupShareAccount{
		AccountID:               s.AccountID,
		TaxRate:                 s.TaxRate,
		ValuationDiscountFactor: s.ValuationDiscountFactor,
	}
}

func investmentRoundFromDB(i pdb.InvestmentRound) InvestmentRound {
	return InvestmentRound{
		ID:             i.ID,
		AccountID:      i.AccountID,
		Date:           date.Date(i.Date),
		Valuation:      i.Valuation,
		PreMoneyShares: i.PreMoneyShares,
		Investment:     i.Investment,
		CreatedAt:      time.UnixMilli(i.CreatedAt),
		UpdatedAt:      time.UnixMilli(i.UpdatedAt),
	}
}

func shareChangeFromDB(s pdb.ShareChange) ShareChange {
	return ShareChange{
		ID:          s.ID,
		AccountID:   s.AccountID,
		Date:        date.Date(s.Date),
		DeltaShares: s.DeltaShares,
		TotalPrice:  s.TotalPrice,
		CreatedAt:   time.UnixMilli(s.CreatedAt),
		UpdatedAt:   time.UnixMilli(s.UpdatedAt),
	}
}

func startupShareOptionFromDB(o pdb.StartupShareOption) StartupShareOption {
	return StartupShareOption{
		ID:                  o.ID,
		AccountID:           o.AccountID,
		SourceAccountID:     o.SourceAccountID,
		Shares:              o.Shares,
		StrikePricePerShare: o.StrikePricePerShare,
		GrantDate:           date.Date(o.GrantDate),
		EndDate:             date.Date(o.EndDate),
		CreatedAt:           time.UnixMilli(o.CreatedAt),
		UpdatedAt:           time.UnixMilli(o.UpdatedAt),
	}
}

func UpsertStartupShareAccount(ctx context.Context, db *sql.DB, inp StartupShareAccountInput) (StartupShareAccount, error) {
	ssa, err := pdb.New(db).UpsertStartupShareAccount(ctx, pdb.UpsertStartupShareAccountParams{
		AccountID:               inp.AccountID,
		TaxRate:                 inp.TaxRate,
		ValuationDiscountFactor: inp.ValuationDiscountFactor,
	})
	if err != nil {
		return StartupShareAccount{}, fmt.Errorf("failed to upsert startup share account: %w", err)
	}
	return startupShareAccountFromDB(ssa), nil
}

func GetStartupShareAccount(ctx context.Context, db *sql.DB, accountID string) (StartupShareAccount, error) {
	ssa, err := pdb.New(db).GetStartupShareAccount(ctx, accountID)
	if err != nil {
		return StartupShareAccount{}, err // Return unwrapped error so caller can check for sql.ErrNoRows
	}
	return startupShareAccountFromDB(ssa), nil
}

func DeleteStartupShareAccount(ctx context.Context, db *sql.DB, accountID string) error {
	if err := pdb.New(db).DeleteStartupShareAccount(ctx, accountID); err != nil {
		return fmt.Errorf("failed to delete startup share account: %w", err)
	}
	return nil
}

func UpsertInvestmentRound(ctx context.Context, db *sql.DB, inp InvestmentRoundInput) (InvestmentRound, error) {
	if inp.ID == "" {
		inp.ID = sid.MustNewString(32)
	}
	ir, err := pdb.New(db).UpsertInvestmentRound(ctx, pdb.UpsertInvestmentRoundParams{
		ID:             inp.ID,
		AccountID:      inp.AccountID,
		Date:           int64(inp.Date),
		Valuation:      inp.Valuation,
		PreMoneyShares: inp.PreMoneyShares,
		Investment:     inp.Investment,
		CreatedAt:      time.Now().UnixMilli(),
		UpdatedAt:      time.Now().UnixMilli(),
	})
	if err != nil {
		return InvestmentRound{}, fmt.Errorf("failed to upsert investment round: %w", err)
	}
	return investmentRoundFromDB(ir), nil
}

func GetInvestmentRound(ctx context.Context, db *sql.DB, id string) (InvestmentRound, error) {
	ir, err := pdb.New(db).GetInvestmentRound(ctx, id)
	if err != nil {
		return InvestmentRound{}, fmt.Errorf("failed to get investment round: %w", err)
	}
	return investmentRoundFromDB(ir), nil
}

func ListInvestmentRounds(ctx context.Context, db *sql.DB, accountID string) ([]InvestmentRound, error) {
	rounds, err := pdb.New(db).ListInvestmentRounds(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to list investment rounds: %w", err)
	}
	result := make([]InvestmentRound, len(rounds))
	for i, r := range rounds {
		result[i] = investmentRoundFromDB(r)
	}
	return result, nil
}

func GetLatestInvestmentRound(ctx context.Context, db *sql.DB, accountID string, d date.Date) (InvestmentRound, error) {
	ir, err := pdb.New(db).GetLatestInvestmentRound(ctx, pdb.GetLatestInvestmentRoundParams{
		AccountID: accountID,
		Date:      int64(d),
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return InvestmentRound{}, sql.ErrNoRows
		}
		return InvestmentRound{}, fmt.Errorf("failed to get latest investment round: %w", err)
	}
	return investmentRoundFromDB(ir), nil
}

func DeleteInvestmentRound(ctx context.Context, db *sql.DB, id string) error {
	if err := pdb.New(db).DeleteInvestmentRound(ctx, id); err != nil {
		return fmt.Errorf("failed to delete investment round: %w", err)
	}
	return nil
}

func UpsertShareChange(ctx context.Context, db *sql.DB, inp ShareChangeInput) (ShareChange, error) {
	if inp.ID == "" {
		inp.ID = sid.MustNewString(32)
	}
	sc, err := pdb.New(db).UpsertShareChange(ctx, pdb.UpsertShareChangeParams{
		ID:          inp.ID,
		AccountID:   inp.AccountID,
		Date:        int64(inp.Date),
		DeltaShares: inp.DeltaShares,
		TotalPrice:  inp.TotalPrice,
		CreatedAt:   time.Now().UnixMilli(),
		UpdatedAt:   time.Now().UnixMilli(),
	})
	if err != nil {
		return ShareChange{}, fmt.Errorf("failed to upsert share change: %w", err)
	}
	return shareChangeFromDB(sc), nil
}

func GetShareChange(ctx context.Context, db *sql.DB, id string) (ShareChange, error) {
	sc, err := pdb.New(db).GetShareChange(ctx, id)
	if err != nil {
		return ShareChange{}, fmt.Errorf("failed to get share change: %w", err)
	}
	return shareChangeFromDB(sc), nil
}

func ListShareChanges(ctx context.Context, db *sql.DB, accountID string) ([]ShareChange, error) {
	changes, err := pdb.New(db).ListShareChanges(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to list share changes: %w", err)
	}
	result := make([]ShareChange, len(changes))
	for i, c := range changes {
		result[i] = shareChangeFromDB(c)
	}
	return result, nil
}

func DeleteShareChange(ctx context.Context, db *sql.DB, id string) error {
	if err := pdb.New(db).DeleteShareChange(ctx, id); err != nil {
		return fmt.Errorf("failed to delete share change: %w", err)
	}
	return nil
}

// DeriveShareState returns shares owned and weighted-average purchase price as of the given date.
func DeriveShareState(changes []ShareChange, asOf date.Date) (sharesOwned float64, avgPurchasePricePerShare float64) {
	var totalCostBasis float64
	var costBasisShares float64
	for _, c := range changes {
		if c.Date > asOf {
			continue
		}
		sharesOwned += c.DeltaShares
		if c.DeltaShares > 0 {
			totalCostBasis += c.TotalPrice
			costBasisShares += c.DeltaShares
		}
	}
	if costBasisShares <= 0 {
		return sharesOwned, 0
	}
	avgPurchasePricePerShare = totalCostBasis / costBasisShares
	return sharesOwned, avgPurchasePricePerShare
}

// PostMoneyValuationAndShares returns post-money valuation and total shares after the round.
func PostMoneyValuationAndShares(preMoneyValuation, preMoneyShares, investment float64) (postMoneyValuation, postMoneyShares float64) {
	postMoneyValuation = preMoneyValuation + investment
	if preMoneyShares <= 0 {
		return postMoneyValuation, preMoneyShares
	}
	pricePerShare := preMoneyValuation / preMoneyShares
	newShares := investment / pricePerShare
	postMoneyShares = preMoneyShares + newShares
	return postMoneyValuation, postMoneyShares
}

func UpsertStartupShareOption(ctx context.Context, db *sql.DB, inp StartupShareOptionInput) (StartupShareOption, error) {
	if inp.ID == "" {
		inp.ID = sid.MustNewString(32)
	}
	opt, err := pdb.New(db).UpsertStartupShareOption(ctx, pdb.UpsertStartupShareOptionParams{
		ID:                  inp.ID,
		AccountID:           inp.AccountID,
		SourceAccountID:     inp.SourceAccountID,
		Shares:              inp.Shares,
		StrikePricePerShare: inp.StrikePricePerShare,
		GrantDate:           int64(inp.GrantDate),
		EndDate:             int64(inp.EndDate),
		CreatedAt:           time.Now().UnixMilli(),
		UpdatedAt:           time.Now().UnixMilli(),
	})
	if err != nil {
		return StartupShareOption{}, fmt.Errorf("failed to upsert startup share option: %w", err)
	}
	return startupShareOptionFromDB(opt), nil
}

func GetStartupShareOption(ctx context.Context, db *sql.DB, id string) (StartupShareOption, error) {
	opt, err := pdb.New(db).GetStartupShareOption(ctx, id)
	if err != nil {
		return StartupShareOption{}, fmt.Errorf("failed to get startup share option: %w", err)
	}
	return startupShareOptionFromDB(opt), nil
}

func ListStartupShareOptions(ctx context.Context, db *sql.DB, accountID string) ([]StartupShareOption, error) {
	opts, err := pdb.New(db).ListStartupShareOptions(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to list startup share options: %w", err)
	}
	result := make([]StartupShareOption, len(opts))
	for i, o := range opts {
		result[i] = startupShareOptionFromDB(o)
	}
	return result, nil
}

func DeleteStartupShareOption(ctx context.Context, db *sql.DB, id string) error {
	if err := pdb.New(db).DeleteStartupShareOption(ctx, id); err != nil {
		return fmt.Errorf("failed to delete startup share option: %w", err)
	}
	return nil
}

// CalculateStartupShareBalance calculates the net balance (after tax) for a startup share account
func CalculateStartupShareBalance(cfg *uncertain.Config, valuation uncertain.Value, sharesOwned float64, purchasePricePerShare float64, taxRate float64, totalShares float64, discountFactor float64) uncertain.Value {
	grossValue := valuation.Mul(cfg, uncertain.NewFixed(sharesOwned/totalShares*discountFactor))
	purchasePrice := purchasePricePerShare * sharesOwned

	if purchasePrice > grossValue.Mean() {
		return grossValue
	}
	capitalGains := grossValue.Sub(cfg, uncertain.NewFixed(purchasePrice))
	tax := capitalGains.Mul(cfg, uncertain.NewFixed(taxRate))
	return grossValue.Sub(cfg, tax)
}
