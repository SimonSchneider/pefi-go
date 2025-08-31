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

func AccountTypesPage(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		accountTypes, err := ListAccountTypes(ctx, db)
		if err != nil {
			return fmt.Errorf("listing account types: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Account Types", PageAccountTypes(AccountTypesView(accountTypes))))
	})
}

func AccountTypeNewPage(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return NewView(ctx, w, r).Render(Page("Account Types", PageEditAccountType(AccountTypeEditView(AccountType{}))))
	})
}

func AccountTypeEditPage(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		at, err := GetAccountType(ctx, db, r.PathValue("id"))
		if err != nil {
			return fmt.Errorf("getting account type: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Account Types", PageEditAccountType(AccountTypeEditView(at))))
	})
}

func HandlerAccountTypeUpsert(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var inp AccountTypeInput
		if err := srvu.Decode(r, &inp, false); err != nil {
			return fmt.Errorf("decoding input: %w", err)
		}
		_, err := UpsertAccountType(ctx, db, inp)
		if err != nil {
			return fmt.Errorf("upserting account type: %w", err)
		}
		shttp.RedirectToNext(w, r, "/account-types")
		return nil
	})
}

func HandlerAccountTypeDelete(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if err := DeleteAccountType(ctx, db, r.PathValue("id")); err != nil {
			return fmt.Errorf("deleting account type: %w", err)
		}
		shttp.RedirectToNext(w, r, "/account-types")
		return nil
	})
}

type AccountType struct {
	ID   string
	Name string
}

type AccountTypeInput struct {
	ID   string
	Name string
}

func (a *AccountTypeInput) FromForm(r *http.Request) error {
	a.ID = r.FormValue("id")
	a.Name = r.FormValue("name")
	return nil
}

func accountTypeFromDB(at pdb.AccountType) AccountType {
	return AccountType{
		ID:   at.ID,
		Name: at.Name,
	}
}

func GetAccountType(ctx context.Context, db *sql.DB, id string) (AccountType, error) {
	at, err := pdb.New(db).GetAccountType(ctx, id)
	if err != nil {
		return AccountType{}, fmt.Errorf("failed to get account type: %w", err)
	}
	return accountTypeFromDB(at), nil
}

func UpsertAccountType(ctx context.Context, db *sql.DB, inp AccountTypeInput) (AccountType, error) {
	var (
		q   = pdb.New(db)
		at  pdb.AccountType
		err error
	)
	if inp.ID != "" {
		at, err = q.UpsertAccountType(ctx, pdb.UpsertAccountTypeParams{
			ID:   inp.ID,
			Name: inp.Name,
		})
	} else {
		at, err = q.UpsertAccountType(ctx, pdb.UpsertAccountTypeParams{
			ID:   sid.MustNewString(15),
			Name: inp.Name,
		})
	}
	if err != nil {
		return AccountType{}, fmt.Errorf("failed to upsert account type: %w", err)
	}
	return accountTypeFromDB(at), nil
}

func DeleteAccountType(ctx context.Context, db *sql.DB, id string) error {
	err := pdb.New(db).DeleteAccountType(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete account type: %w", err)
	}
	return nil
}

func accountTypesListFromDB(dbATs []pdb.AccountType) []AccountType {
	ats := make([]AccountType, len(dbATs))
	for i := range dbATs {
		ats[i] = accountTypeFromDB(dbATs[i])
	}
	return ats
}

func ListAccountTypes(ctx context.Context, db *sql.DB) ([]AccountType, error) {
	ats, err := pdb.New(db).ListAccountTypes(ctx)
	if err != nil {
		return nil, err
	}
	return accountTypesListFromDB(ats), nil
}
