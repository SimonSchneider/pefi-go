package core

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"net/http"
	"time"

	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/goslu/srvu"
	"github.com/SimonSchneider/goslu/static/shttp"
)

func HandlerAccountUpsert(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var inp AccountInput
		if err := srvu.Decode(r, &inp, false); err != nil {
			return fmt.Errorf("decoding input: %w", err)
		}
		acc, err := UpsertAccount(ctx, db, inp)
		if err != nil {
			return fmt.Errorf("upserting account: %w", err)
		}
		shttp.RedirectToNext(w, r, fmt.Sprintf("/accounts/%s", acc.ID))
		return nil
	})
}

func HandlerAccountDelete(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if err := DeleteAccount(ctx, db, r.PathValue("id")); err != nil {
			return fmt.Errorf("deleting account: %w", err)
		}
		shttp.RedirectToNext(w, r, "/accounts/")
		return nil
	})
}

func HandlerAccountGrowthModelUpsert(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var inp AccountGrowthModelInput
		if err := srvu.Decode(r, &inp, false); err != nil {
			return fmt.Errorf("decoding input: %w", err)
		}
		_, err := UpsertAccountGrowthModel(ctx, db, inp)
		if err != nil {
			return fmt.Errorf("upserting account growth model: %w", err)
		}
		shttp.RedirectToNext(w, r, fmt.Sprintf("/accounts/%s", inp.AccountID))
		return nil
	})
}

func HandlerAccountGrowthModelDelete(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if err := DeleteAccountGrowthModel(ctx, db, r.PathValue("id")); err != nil {
			return fmt.Errorf("deleting account growth model: %w", err)
		}
		shttp.RedirectToNext(w, r, fmt.Sprintf("/accounts/%s", r.PathValue("id")))
		return nil
	})
}

func HandlerTransferTemplateUpsert(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var inp TransferTemplate
		if err := srvu.Decode(r, &inp, false); err != nil {
			return fmt.Errorf("decoding input: %w", err)
		}
		t, err := UpsertTransferTemplate(ctx, db, inp)
		if err != nil {
			return fmt.Errorf("upserting transfer template: %w", err)
		}
		shttp.RedirectToNext(w, r, fmt.Sprintf("/transfers/%s", t.ID))
		return nil
	})
}

func HandlerTransferTemplateDuplicate(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		tr, err := DuplicateTransferTemplate(ctx, db, r.PathValue("id"))
		if err != nil {
			return fmt.Errorf("duplicating transfer template: %w", err)
		}
		shttp.RedirectToNext(w, r, fmt.Sprintf("/transfer-templates/%s/edit", tr.ID))
		return nil
	})
}

func HandlerTransferTemplateDelete(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if err := DeleteTransferTemplate(ctx, db, r.PathValue("id")); err != nil {
			return fmt.Errorf("deleting transfer template: %w", err)
		}
		shttp.RedirectToNext(w, r, "/")
		return nil
	})
}

func RootPage() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return NewView(ctx, w, r).Render(Page("App", App()))
	})
}

func AccountsPage(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		accs, err := ListAccountsDetailed(ctx, db, date.Today())
		if err != nil {
			return fmt.Errorf("listing accounts: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Accounts", PageAccounts(NewAccountsView(accs))))
	})
}

func TransferTemplatesPage(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		transferTemplates, err := ListTransferTemplates(ctx, db)
		if err != nil {
			return fmt.Errorf("listing transfer templates: %w", err)
		}
		accounts, err := ListAccounts(ctx, db)
		if err != nil {
			return fmt.Errorf("listing accounts: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Transfer Templates", PageTransferTemplates(NewTransferTemplatesView2(transferTemplates, accounts))))
	})
}

func AccountNewPage(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		accs, err := ListAccounts(ctx, db)
		if err != nil {
			return fmt.Errorf("listing accounts: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Accounts", PageEditAccount(NewAccountEditView2(Account{}, accs, nil))))
	})
}

func AccountEditPage(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		accs, err := ListAccounts(ctx, db)
		if err != nil {
			return fmt.Errorf("listing accounts: %w", err)
		}
		acc, err := GetAccount(ctx, db, r.PathValue("id"))
		if err != nil {
			return fmt.Errorf("getting account: %w", err)
		}
		growthModels, err := ListAccountGrowthModels(ctx, db, string(acc.ID))
		if err != nil {
			return fmt.Errorf("listing account growth models: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Accounts", PageEditAccount(NewAccountEditView2(acc, accs, growthModels))))
	})
}

func TransferTemplatesNewPage(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		accs, err := ListAccounts(ctx, db)
		if err != nil {
			return fmt.Errorf("listing accounts: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Transfer Templates", PageEditTransferTemplate(&TransferTemplateEditView{
			Accounts: accs,
		})))
	})
}

func TransferTemplatesEditPage(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		t, err := GetTransferTemplate(ctx, db, id)
		if err != nil {
			return fmt.Errorf("getting transfer template: %w", err)
		}
		accs, err := ListAccounts(ctx, db)
		if err != nil {
			return fmt.Errorf("listing accounts: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Transfer Templates", PageEditTransferTemplate(&TransferTemplateEditView{
			Accounts:         accs,
			TransferTemplate: t,
		})))
	})
}

func NewHandler(db *sql.DB, public fs.FS) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /static/public/", srvu.With(http.StripPrefix("/static/public/", http.FileServerFS(public)), srvu.WithCacheCtrlHeader(365*24*time.Hour)))

	mux.Handle("GET /{$}", RootPage())
	mux.Handle("GET /accounts", AccountsPage(db))
	mux.Handle("GET /accounts/new", AccountNewPage(db))
	mux.Handle("GET /accounts/{id}/edit", AccountEditPage(db))
	mux.Handle("GET /transfer-templates", TransferTemplatesPage(db))
	mux.Handle("GET /transfer-templates/new", TransferTemplatesNewPage(db))
	mux.Handle("GET /transfer-templates/{id}/edit", TransferTemplatesEditPage(db))

	// Account types
	mux.Handle("GET /account-types", AccountTypesPage(db))
	mux.Handle("GET /account-types/new", AccountTypeNewPage(db))
	mux.Handle("GET /account-types/{id}/edit", AccountTypeEditPage(db))
	mux.Handle("POST /account-types/{$}", HandlerAccountTypeUpsert(db))
	mux.Handle("POST /account-types/{id}/delete", HandlerAccountTypeDelete(db))

	// Snapshots table
	mux.Handle("GET /snapshots-table", SnapshotsTablePage(db))
	mux.Handle("POST /snapshots-table/modify-date", SnapshotsTableModifyDate(db))
	mux.Handle("GET /snapshots-table/empty-row", SnapshotsTableEmptyRow(db))
	mux.Handle("POST /accounts/{id}/snapshots/{date}/", HandlerAccountSnapshotUpsert(db))

	// Chart
	mux.Handle("GET /chart", ChartPage())
	mux.Handle("GET /chart/stream", HandlerChartsDataStream(db))

	mux.Handle("POST /accounts/{$}", HandlerAccountUpsert(db))
	mux.Handle("POST /accounts/{id}/delete", HandlerAccountDelete(db))

	mux.Handle("POST /growth-models/", HandlerAccountGrowthModelUpsert(db))
	mux.Handle("POST /growth-models/{id}/delete", HandlerAccountGrowthModelDelete(db))

	mux.Handle("POST /transfers/{$}", HandlerTransferTemplateUpsert(db))
	mux.Handle("POST /transfers/{id}/duplicate", HandlerTransferTemplateDuplicate(db))
	mux.Handle("POST /transfers/{id}/delete", HandlerTransferTemplateDelete(db))

	mux.Handle("POST /sleep/{$}", srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Simulate a long-running operation
		time.Sleep(1 * time.Second)
		w.WriteHeader(200)
		return nil
	}))

	return mux
}
