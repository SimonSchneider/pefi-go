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
	"github.com/SimonSchneider/pefigo/internal/finance"
	"github.com/SimonSchneider/pefigo/internal/pdb"
	"github.com/SimonSchneider/pefigo/internal/ui"
	"github.com/SimonSchneider/pefigo/internal/uncertain"
)

type GrowthModel struct {
	ID               string
	AccountID        string
	Type             string
	AnnualRate       uncertain.Value
	AnnualVolatility uncertain.Value
	StartDate        date.Date
	EndDate          *date.Date
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
	if err := shttp.Parse(&a.AnnualRate, ui.ParseUncertainValue, r.FormValue("annual_rate"), uncertain.NewFixed(0)); err != nil {
		return fmt.Errorf("parsing annual rate: %w", err)
	}
	if err := shttp.Parse(&a.AnnualVolatility, ui.ParseUncertainValue, r.FormValue("annual_volatility"), uncertain.NewFixed(0)); err != nil {
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

func UpsertAccountGrowthModel(ctx context.Context, db *sql.DB, inp AccountGrowthModelInput) (GrowthModel, error) {
	var endDate *int64
	if inp.EndDate != nil {
		endDate = ptr(int64(*inp.EndDate))
	}
	if inp.ID == "" {
		inp.ID = sid.MustNewString(32)
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

func DeleteAccountGrowthModel(ctx context.Context, db *sql.DB, id string) error {
	if err := pdb.New(db).DeleteGrowthModel(ctx, id); err != nil {
		return fmt.Errorf("failed to delete account growth model: %w", err)
	}
	return nil
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

type GrowthModels []GrowthModel

func (gms GrowthModels) ToFinance() finance.GrowthModel {
	fgms := make([]finance.GrowthModel, 0, len(gms))
	for _, gm := range gms {
		switch gm.Type {
		case "fixed":
			fgms = append(fgms, &finance.FixedGrowth{
				TimeFrameGrowth: finance.TimeFrameGrowth{
					StartDate: gm.StartDate,
					EndDate:   gm.EndDate,
				},
				AnnualRate: gm.AnnualRate,
			})
		case "lognormal":
			fgms = append(fgms, &finance.LogNormalGrowth{
				TimeFrameGrowth: finance.TimeFrameGrowth{
					StartDate: gm.StartDate,
					EndDate:   gm.EndDate,
				},
				AnnualRate:       gm.AnnualRate,
				AnnualVolatility: gm.AnnualVolatility,
			})
		}
	}
	return finance.NewGrowthCombined(fgms...)
}
