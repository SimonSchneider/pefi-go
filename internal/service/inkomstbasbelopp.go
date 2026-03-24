package service

import (
	"context"
	"fmt"
	"time"

	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/goslu/sid"
	"github.com/SimonSchneider/pefigo/internal/pdb"
)

type Inkomstbasbelopp struct {
	ID            string
	Amount        float64
	Prisbasbelopp float64
	ValidFrom     date.Date
}

func inkomstbasbeloppFromDB(row pdb.Inkomstbasbelopp) Inkomstbasbelopp {
	return Inkomstbasbelopp{
		ID:            row.ID,
		Amount:        row.Amount,
		Prisbasbelopp: row.Prisbasbelopp,
		ValidFrom:     date.Date(row.ValidFrom),
	}
}

func (s *Service) ListInkomstbasbelopp(ctx context.Context) ([]Inkomstbasbelopp, error) {
	rows, err := pdb.New(s.db).ListInkomstbasbelopp(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing inkomstbasbelopp: %w", err)
	}
	result := make([]Inkomstbasbelopp, len(rows))
	for i, r := range rows {
		result[i] = inkomstbasbeloppFromDB(r)
	}
	return result, nil
}

func (s *Service) GetInkomstbasbelopp(ctx context.Context, id string) (Inkomstbasbelopp, error) {
	row, err := pdb.New(s.db).GetInkomstbasbelopp(ctx, id)
	if err != nil {
		return Inkomstbasbelopp{}, fmt.Errorf("getting inkomstbasbelopp: %w", err)
	}
	return inkomstbasbeloppFromDB(row), nil
}

func (s *Service) UpsertInkomstbasbelopp(ctx context.Context, inp Inkomstbasbelopp) (Inkomstbasbelopp, error) {
	if inp.ID == "" {
		inp.ID = sid.MustNewString(32)
	}
	now := time.Now().Unix()
	row, err := pdb.New(s.db).UpsertInkomstbasbelopp(ctx, pdb.UpsertInkomstbasbeloppParams{
		ID:            inp.ID,
		Amount:        inp.Amount,
		Prisbasbelopp: inp.Prisbasbelopp,
		ValidFrom:     int64(inp.ValidFrom),
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	if err != nil {
		return Inkomstbasbelopp{}, fmt.Errorf("upserting inkomstbasbelopp: %w", err)
	}
	return inkomstbasbeloppFromDB(row), nil
}

func (s *Service) DeleteInkomstbasbelopp(ctx context.Context, id string) error {
	if err := pdb.New(s.db).DeleteInkomstbasbelopp(ctx, id); err != nil {
		return fmt.Errorf("deleting inkomstbasbelopp: %w", err)
	}
	return nil
}

// activeIBBAt returns the IBB value active at a given date (the latest one with ValidFrom <= d).
func activeIBBAt(ibbs []Inkomstbasbelopp, d date.Date) float64 {
	var active float64
	for _, ibb := range ibbs {
		if ibb.ValidFrom <= d {
			active = ibb.Amount
		}
	}
	return active
}

// activePBBAt returns the PBB (prisbasbelopp) value active at a given date.
func activePBBAt(ibbs []Inkomstbasbelopp, d date.Date) float64 {
	var active float64
	for _, ibb := range ibbs {
		if ibb.ValidFrom <= d {
			active = ibb.Prisbasbelopp
		}
	}
	return active
}
