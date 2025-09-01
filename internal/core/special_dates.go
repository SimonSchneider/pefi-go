package core

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/SimonSchneider/goslu/sid"
	"github.com/SimonSchneider/goslu/srvu"
	"github.com/SimonSchneider/goslu/static/shttp"
	"github.com/SimonSchneider/pefigo/internal/pdb"
)

func SpecialDatesPage(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		specialDates, err := ListSpecialDates(ctx, db)
		if err != nil {
			return fmt.Errorf("listing special dates: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Special Dates", PageSpecialDates(SpecialDatesView(specialDates))))
	})
}

func SpecialDateNewPage(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return NewView(ctx, w, r).Render(Page("Special Dates", PageEditSpecialDate(SpecialDateEditView(SpecialDate{}))))
	})
}

func SpecialDateEditPage(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		sd, err := GetSpecialDate(ctx, db, r.PathValue("id"))
		if err != nil {
			return fmt.Errorf("getting special date: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Special Dates", PageEditSpecialDate(SpecialDateEditView(sd))))
	})
}

func HandlerSpecialDateUpsert(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var inp SpecialDateInput
		if err := srvu.Decode(r, &inp, false); err != nil {
			return fmt.Errorf("decoding input: %w", err)
		}
		_, err := UpsertSpecialDate(ctx, db, inp)
		if err != nil {
			return fmt.Errorf("upserting special date: %w", err)
		}
		shttp.RedirectToNext(w, r, "/special-dates")
		return nil
	})
}

func HandlerSpecialDateDelete(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if err := DeleteSpecialDate(ctx, db, r.PathValue("id")); err != nil {
			return fmt.Errorf("deleting special date: %w", err)
		}
		shttp.RedirectToNext(w, r, "/special-dates")
		return nil
	})
}

type SpecialDate struct {
	ID   string
	Name string
	Date string
}

type SpecialDateInput struct {
	ID   string
	Name string
	Date string
}

func (s *SpecialDateInput) FromForm(r *http.Request) error {
	s.ID = r.FormValue("id")
	s.Name = r.FormValue("name")
	s.Date = r.FormValue("date")
	return nil
}

func specialDateFromDB(sd pdb.SpecialDate) SpecialDate {
	return SpecialDate{
		ID:   sd.ID,
		Name: sd.Name,
		Date: sd.Date,
	}
}

func GetSpecialDate(ctx context.Context, db *sql.DB, id string) (SpecialDate, error) {
	sd, err := pdb.New(db).GetSpecialDate(ctx, id)
	if err != nil {
		return SpecialDate{}, fmt.Errorf("failed to get special date: %w", err)
	}
	return specialDateFromDB(sd), nil
}

func UpsertSpecialDate(ctx context.Context, db *sql.DB, inp SpecialDateInput) (SpecialDate, error) {
	var (
		q   = pdb.New(db)
		sd  pdb.SpecialDate
		err error
	)
	if inp.ID != "" {
		sd, err = q.UpsertSpecialDate(ctx, pdb.UpsertSpecialDateParams{
			ID:   inp.ID,
			Name: inp.Name,
			Date: inp.Date,
		})
	} else {
		sd, err = q.UpsertSpecialDate(ctx, pdb.UpsertSpecialDateParams{
			ID:   sid.MustNewString(15),
			Name: inp.Name,
			Date: inp.Date,
		})
	}
	if err != nil {
		return SpecialDate{}, fmt.Errorf("failed to upsert special date: %w", err)
	}
	return specialDateFromDB(sd), nil
}

func DeleteSpecialDate(ctx context.Context, db *sql.DB, id string) error {
	err := pdb.New(db).DeleteSpecialDate(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete special date: %w", err)
	}
	return nil
}

func specialDatesListFromDB(dbSDs []pdb.SpecialDate) []SpecialDate {
	sds := make([]SpecialDate, len(dbSDs))
	for i := range dbSDs {
		sds[i] = specialDateFromDB(dbSDs[i])
	}
	return sds
}

func ListSpecialDates(ctx context.Context, db *sql.DB) ([]SpecialDate, error) {
	sds, err := pdb.New(db).GetSpecialDates(ctx)
	if err != nil {
		return nil, err
	}
	return specialDatesListFromDB(sds), nil
}
