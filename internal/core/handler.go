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

func HandlerAccountSnapshotUpsert(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var inp AccountSnapshotInput
		if err := srvu.Decode(r, &inp, false); err != nil {
			return fmt.Errorf("decoding input: %w", err)
		}
		accID := r.PathValue("id")
		var snap AccountSnapshot
		if inp.EmptyBalance {
			if err := DeleteAccountSnapshot(ctx, db, accID, inp.Date); err != nil {
				return fmt.Errorf("deleting existing snapshot: %w", err)
			}
			snap = AccountSnapshot{
				AccountID: accID,
				Date:      inp.Date,
			}
		} else {
			s, err := UpsertAccountSnapshot(ctx, db, accID, inp)
			if err != nil {
				return fmt.Errorf("upserting snapshot: %w", err)
			}
			snap = s
		}
		if r.Header.Get("HX-Request") == "true" {
			return NewTemplView(ctx, w, r).Render(SnapshotCell(accID, inp.Date, snap))
		} else {
			shttp.RedirectToNext(w, r, fmt.Sprintf("/accounts/%s", accID))
		}
		return nil
	})
}

func HandlerAccountSnapshotDelete(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var d date.Date
		if err := shttp.Parse(&d, date.ParseDate, r.FormValue("date"), date.Today()); err != nil {
			return fmt.Errorf("parsing date: %w", err)
		}
		if err := DeleteAccountSnapshot(ctx, db, r.PathValue("id"), d); err != nil {
			return fmt.Errorf("deleting account snapshot: %w", err)
		}
		shttp.RedirectToNext(w, r, fmt.Sprintf("/accounts/%s", r.PathValue("id")))
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

type SSEPredictionEventHandler struct {
	w *srvu.SSESender
}

func (s *SSEPredictionEventHandler) Setup(e PredictionSetupEvent) error {
	return s.w.SendNamedJson("setup", e)
}
func (s *SSEPredictionEventHandler) Snapshot(e PredictionBalanceSnapshot) error {
	return s.w.SendNamedJson("balanceSnapshot", e)
}
func (s *SSEPredictionEventHandler) Close() error {
	return s.w.SendEventWithoutData("close")
}

func HandlerChartsDataStream(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var params PredictionParams
		if err := srvu.Decode(r, &params, false); err != nil {
			return fmt.Errorf("decoding input: %w", err)
		}
		if err := RunPrediction(ctx, db, &SSEPredictionEventHandler{w: srvu.SSEResponse(w)}, params); err != nil {
			return fmt.Errorf("running prediction: %w", err)
		}
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
		shttp.RedirectToNext(w, r, fmt.Sprintf("/templ/transfer-templates/%s/edit", tr.ID))
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
		return NewTemplView(ctx, w, r).Render(Page("App", App()))
	})
}

func ChartPage() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		p := PredictionParams{}
		if err := srvu.Decode(r, &p, false); err != nil {
			return fmt.Errorf("decoding input: %w", err)
		}
		return NewTemplView(ctx, w, r).Render(Page("Chart", PageChart(p)))
	})
}

func AccountsPage(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		accs, err := ListAccountsDetailed(ctx, db, date.Today())
		if err != nil {
			return fmt.Errorf("listing accounts: %w", err)
		}
		return NewTemplView(ctx, w, r).Render(Page("Accounts", PageAccounts(NewAccountsView(accs))))
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
		return NewTemplView(ctx, w, r).Render(Page("Transfer Templates", PageTransferTemplates(NewTransferTemplatesView2(transferTemplates, accounts))))
	})
}

func AccountNewPage(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		accs, err := ListAccounts(ctx, db)
		if err != nil {
			return fmt.Errorf("listing accounts: %w", err)
		}
		return NewTemplView(ctx, w, r).Render(Page("Accounts", PageEditAccount(NewAccountEditView2(Account{}, accs, nil))))
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
		return NewTemplView(ctx, w, r).Render(Page("Accounts", PageEditAccount(NewAccountEditView2(acc, accs, growthModels))))
	})
}

func TransferTemplatesNewPage(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		accs, err := ListAccounts(ctx, db)
		if err != nil {
			return fmt.Errorf("listing accounts: %w", err)
		}
		return NewTemplView(ctx, w, r).Render(Page("Transfer Templates", PageEditTransferTemplate(&TransferTemplateEditView{
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
		return NewTemplView(ctx, w, r).Render(Page("Transfer Templates", PageEditTransferTemplate(&TransferTemplateEditView{
			Accounts:         accs,
			TransferTemplate: t,
		})))
	})
}

func SnapshotsTablePage(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		accounts, err := ListAccounts(ctx, db)
		if err != nil {
			return fmt.Errorf("listing accounts: %w", err)
		}
		ids := make([]string, len(accounts))
		for i, acc := range accounts {
			ids[i] = acc.ID
		}
		// Snapshots are ordered by date
		snapshots, err := ListAccountsSnapshots(ctx, db, ids)
		if err != nil {
			return fmt.Errorf("listing account snapshots: %w", err)
		}
		type DateIDKey struct {
			Date date.Date
			ID   string
		}
		dates := make([]date.Date, 0)
		snaps := make(map[DateIDKey]AccountSnapshot)
		for _, s := range snapshots {
			snaps[DateIDKey{Date: s.Date, ID: s.AccountID}] = s
			if len(dates) == 0 || dates[len(dates)-1] != s.Date {
				dates = append(dates, s.Date)
			}
		}
		rows := make([]TableRow, 0)
		for _, d := range dates {
			rows = append(rows, TableRow{
				Date:      d,
				Snapshots: make([]AccountSnapshot, 0, len(accounts)),
			})
			for _, acc := range accounts {
				if snap, ok := snaps[DateIDKey{Date: d, ID: acc.ID}]; ok {
					rows[len(rows)-1].Snapshots = append(rows[len(rows)-1].Snapshots, snap)
				} else {
					rows[len(rows)-1].Snapshots = append(rows[len(rows)-1].Snapshots, AccountSnapshot{
						AccountID: acc.ID,
						Date:      d,
					})
				}
			}
		}

		return NewTemplView(ctx, w, r).Render(Page("Snapshots Table", PageSnapshotsTable(&TableView{
			Accounts: accounts,
			Rows:     rows,
		})))
	})
}

func NewHandler(db *sql.DB, public fs.FS) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /static/public/", srvu.With(http.StripPrefix("/static/public/", http.FileServerFS(public)), srvu.WithCacheCtrlHeader(365*24*time.Hour)))

	mux.Handle("GET /templ/app", RootPage())
	mux.Handle("GET /templ/chart", ChartPage())
	mux.Handle("GET /templ/accounts", AccountsPage(db))
	mux.Handle("GET /templ/accounts/new", AccountNewPage(db))
	mux.Handle("GET /templ/accounts/{id}/edit", AccountEditPage(db))
	mux.Handle("GET /templ/transfer-templates", TransferTemplatesPage(db))
	mux.Handle("GET /templ/transfer-templates/new", TransferTemplatesNewPage(db))
	mux.Handle("GET /templ/transfer-templates/{id}/edit", TransferTemplatesEditPage(db))
	mux.Handle("GET /templ/snapshots-table", SnapshotsTablePage(db))

	mux.Handle("POST /accounts/{$}", HandlerAccountUpsert(db))
	mux.Handle("POST /accounts/{id}/delete", HandlerAccountDelete(db))

	mux.Handle("POST /accounts/{id}/snapshots/", HandlerAccountSnapshotUpsert(db))
	mux.Handle("POST /accounts/{id}/snapshots/{date}/", HandlerAccountSnapshotUpsert(db))
	mux.Handle("POST /accounts/{id}/snapshots/delete", HandlerAccountSnapshotDelete(db))

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

	mux.Handle("GET /charts/stream", HandlerChartsDataStream(db))

	return mux
}
