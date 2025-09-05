package core

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/goslu/sid"
	"github.com/SimonSchneider/goslu/static/shttp"
	"github.com/SimonSchneider/pefigo/internal/finance"
	"github.com/SimonSchneider/pefigo/internal/pdb"
	"github.com/SimonSchneider/pefigo/internal/ui"
	"github.com/SimonSchneider/pefigo/internal/uncertain"
)

type TransferTemplate struct {
	ID            string
	Name          string
	FromAccountID string
	ToAccountID   string
	AmountType    string // "fixed" or "percent"
	AmountFixed   uncertain.Value
	AmountPercent float64
	Priority      int64     // lower number = happens earlier
	Recurrence    date.Cron // e.g. "*-*-25"

	StartDate date.Date
	EndDate   *date.Date
	Enabled   bool
}

func (t *TransferTemplate) FromForm(r *http.Request) error {
	t.ID = r.FormValue("id")
	t.Name = r.FormValue("name")
	t.FromAccountID = r.FormValue("from_account_id")
	t.ToAccountID = r.FormValue("to_account_id")
	t.AmountType = r.FormValue("amount_type")
	if t.AmountType != "fixed" && t.AmountType != "percent" {
		return fmt.Errorf("invalid amount type: %s", t.AmountType)
	}
	if err := shttp.Parse(&t.AmountFixed, ui.ParseUncertainValue, r.FormValue("amount_fixed"), uncertain.NewFixed(0)); err != nil {
		return fmt.Errorf("parsing amount fixed: %w", err)
	}
	if err := shttp.Parse(&t.AmountPercent, shttp.ParseFloat, r.FormValue("amount_percent"), 0); err != nil {
		return fmt.Errorf("parsing amount percent: %w", err)
	}
	if err := shttp.Parse(&t.Priority, ui.ParseInt64, r.FormValue("priority"), int64(0)); err != nil {
		return fmt.Errorf("parsing priority: %w", err)
	}
	if err := shttp.Parse(&t.Recurrence, ui.ParseDateCron, r.FormValue("recurrence"), date.Cron("")); err != nil {
		return fmt.Errorf("parsing recurrence: %w", err)
	}
	if err := shttp.Parse(&t.StartDate, date.ParseDate, r.FormValue("start_date"), date.Date(0)); err != nil {
		return fmt.Errorf("parsing effective from: %w", err)
	}
	if endDateStr := r.FormValue("end_date"); endDateStr != "" {
		var endDate date.Date
		if err := shttp.Parse(&endDate, date.ParseDate, endDateStr, date.Date(0)); err != nil {
			return fmt.Errorf("parsing effective to: %w", err)
		}
		t.EndDate = &endDate
	} else {
		t.EndDate = nil
	}
	t.Enabled = r.FormValue("enabled") == "on"
	return nil
}

func (t *TransferTemplate) ToFinanceTransferTemplate() finance.TransferTemplate {
	return finance.TransferTemplate{
		ID:            t.ID,
		Name:          t.Name,
		FromAccountID: t.FromAccountID,
		ToAccountID:   t.ToAccountID,
		AmountType:    finance.TransferAmountType(t.AmountType),
		AmountFixed: finance.TransferFixed{
			Amount: t.AmountFixed,
		},
		AmountPercent: finance.TransferPercent{
			Percent: t.AmountPercent,
		},
		Priority:      t.Priority,
		EffectiveFrom: t.StartDate,
		EffectiveTo:   t.EndDate,
		Recurrence:    t.Recurrence,
		Enabled:       t.Enabled,
	}
}

func transferTemplateFromDB(t pdb.TransferTemplate) (TransferTemplate, error) {
	var endDate *date.Date
	if t.EndDate != nil {
		d := date.Date(*t.EndDate)
		endDate = &d
	}
	var amountFixed uncertain.Value
	if err := amountFixed.Decode(t.AmountFixed); err != nil {
		return TransferTemplate{}, fmt.Errorf("decoding amount fixed: %w", err)
	}
	return TransferTemplate{
		ID:            t.ID,
		Name:          t.Name,
		FromAccountID: ui.OrDefault(t.FromAccountID),
		ToAccountID:   ui.OrDefault(t.ToAccountID),
		AmountType:    t.AmountType,
		AmountFixed:   amountFixed,
		AmountPercent: t.AmountPercent,
		Priority:      t.Priority,
		Recurrence:    date.Cron(t.Recurrence),
		StartDate:     date.Date(t.StartDate),
		EndDate:       endDate,
		Enabled:       t.Enabled,
	}, nil
}

func UpsertTransferTemplate(ctx context.Context, db *sql.DB, inp TransferTemplate) (TransferTemplate, error) {
	var endDate *int64
	if inp.EndDate != nil {
		d := int64(*inp.EndDate)
		endDate = &d
	}
	amountFixed, err := inp.AmountFixed.Encode()
	if err != nil {
		return TransferTemplate{}, fmt.Errorf("encoding amount fixed: %w", err)
	}
	if inp.ID == "" {
		inp.ID = sid.MustNewString(32)
	}
	t, err := pdb.New(db).UpsertTransferTemplate(ctx, pdb.UpsertTransferTemplateParams{
		ID:            inp.ID,
		Name:          inp.Name,
		FromAccountID: ui.WithDefaultNull(inp.FromAccountID),
		ToAccountID:   ui.WithDefaultNull(inp.ToAccountID),
		AmountType:    inp.AmountType,
		AmountFixed:   amountFixed,
		AmountPercent: inp.AmountPercent,
		Priority:      inp.Priority,
		Recurrence:    string(inp.Recurrence),
		StartDate:     int64(inp.StartDate),
		EndDate:       endDate,
		Enabled:       inp.Enabled,
	})
	if err != nil {
		return TransferTemplate{}, fmt.Errorf("failed to upsert  template: %w", err)
	}
	return transferTemplateFromDB(t)
}

func DuplicateTransferTemplate(ctx context.Context, db *sql.DB, id string) (TransferTemplate, error) {
	t, err := GetTransferTemplate(ctx, db, id)
	if err != nil {
		return TransferTemplate{}, fmt.Errorf("failed to get transfer template: %w", err)
	}
	t.ID = ""
	return UpsertTransferTemplate(ctx, db, t)
}

func ListTransferTemplates(ctx context.Context, db *sql.DB) ([]TransferTemplate, error) {
	templates, err := pdb.New(db).GetTransferTemplates(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list  templates: %w", err)
	}
	var result []TransferTemplate
	for _, t := range templates {
		template, err := transferTemplateFromDB(t)
		if err != nil {
			return nil, fmt.Errorf("converting transfer template from DB: %w", err)
		}
		result = append(result, template)
	}
	return result, nil
}

func GetTransferTemplate(ctx context.Context, db *sql.DB, id string) (TransferTemplate, error) {
	t, err := pdb.New(db).GetTransferTemplate(ctx, id)
	if err != nil {
		return TransferTemplate{}, fmt.Errorf("failed to get transfer template: %w", err)
	}
	return transferTemplateFromDB(t)
}

func DeleteTransferTemplate(ctx context.Context, db *sql.DB, id string) error {
	if err := pdb.New(db).DeleteTransferTemplate(ctx, id); err != nil {
		return fmt.Errorf("failed to delete  template: %w", err)
	}
	return nil
}
