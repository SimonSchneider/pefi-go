package core

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/SimonSchneider/goslu/sid"
	"github.com/SimonSchneider/pefigo/internal/pdb"
)

type TransferTemplateCategory struct {
	ID        string
	Name      string
	Color     *string
	CreatedAt int64
	UpdatedAt int64
}

func (c *TransferTemplateCategory) FromForm(r *http.Request) error {
	c.ID = r.FormValue("id")
	c.Name = r.FormValue("name")
	color := r.FormValue("color")
	if color != "" {
		c.Color = &color
	} else {
		c.Color = nil
	}
	return nil
}

func categoryFromDB(c pdb.TransferTemplateCategory) TransferTemplateCategory {
	return TransferTemplateCategory{
		ID:        c.ID,
		Name:      c.Name,
		Color:     c.Color,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func ListCategories(ctx context.Context, db *sql.DB) ([]TransferTemplateCategory, error) {
	categories, err := pdb.New(db).ListTransferTemplateCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list categories: %w", err)
	}
	var result []TransferTemplateCategory
	for _, c := range categories {
		result = append(result, categoryFromDB(c))
	}
	return result, nil
}

func GetCategory(ctx context.Context, db *sql.DB, id string) (TransferTemplateCategory, error) {
	c, err := pdb.New(db).GetTransferTemplateCategory(ctx, id)
	if err != nil {
		return TransferTemplateCategory{}, fmt.Errorf("failed to get category: %w", err)
	}
	return categoryFromDB(c), nil
}

func UpsertCategory(ctx context.Context, db *sql.DB, inp TransferTemplateCategory) (TransferTemplateCategory, error) {
	now := time.Now().Unix()
	if inp.ID == "" {
		inp.ID = sid.MustNewString(32)
		inp.CreatedAt = now
	}
	inp.UpdatedAt = now

	c, err := pdb.New(db).UpsertTransferTemplateCategory(ctx, pdb.UpsertTransferTemplateCategoryParams{
		ID:        inp.ID,
		Name:      inp.Name,
		Color:     inp.Color,
		CreatedAt: inp.CreatedAt,
		UpdatedAt: inp.UpdatedAt,
	})
	if err != nil {
		return TransferTemplateCategory{}, fmt.Errorf("failed to upsert category: %w", err)
	}
	return categoryFromDB(c), nil
}

func DeleteCategory(ctx context.Context, db *sql.DB, id string) error {
	if err := pdb.New(db).DeleteTransferTemplateCategory(ctx, id); err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}
	return nil
}

