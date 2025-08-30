package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"time"

	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/goslu/sid"
	"github.com/SimonSchneider/goslu/srvu"
	"github.com/SimonSchneider/goslu/static/shttp"
	"github.com/SimonSchneider/goslu/templ"
	"github.com/SimonSchneider/pefigo/internal/pdb"
	"github.com/SimonSchneider/pefigo/internal/uncertain"
)

func HandlerIndexPage(db *sql.DB, view *View) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		accs, err := ListAccounts(ctx, db)
		if err != nil {
			return err
		}
		trans, err := ListTransferTemplates(ctx, db)
		if err != nil {
			return fmt.Errorf("listing transfer templates: %w", err)
		}
		users, err := ListUsers(ctx, db)
		if err != nil {
			return err
		}
		return view.IndexPage(w, r, IndexView{
			Accounts:  AccountsListView{Accounts: accs},
			Transfers: TransferTemplatesView{Transfers: trans},
			Users:     UserListView{Users: users},
		})
	})
}

func HandlerTable(db *sql.DB, view *View) http.Handler {
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
		return view.TablePage(w, r, TableView{
			Accounts: accounts,
			Rows:     rows,
		})
	})
}

func HandlerTransferTable(db *sql.DB, view *View) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		accounts, err := ListAccounts(ctx, db)
		if err != nil {
			return fmt.Errorf("listing accounts: %w", err)
		}
		transfers, err := ListTransferTemplates(ctx, db)
		type DateIDKey struct {
			Date date.Date
			ID   string
		}
		reqD := &RequestDetails{req: r}
		rows := make([]TransferTableRow, 0, len(transfers))
		for _, t := range transfers {
			rows = append(rows, TransferTableRow{
				RequestDetails: reqD,
				Transfer:       t,
				Accounts:       accounts,
			})
		}
		return view.TransferTablePage(w, r, TransferTableView{
			Rows: rows,
		})
	})
}

func HandlerAccountUpsert(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var inp AccountInput
		if err := srvu.Decode(r, &inp, false); err != nil {
			return fmt.Errorf("decoding input: %w", err)
		}
		if inp.ID == "" {
			inp.ID = sid.MustNewString(32)
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

func HandlerAccountPage(db *sql.DB, view *View) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		acc, err := GetAccount(ctx, db, r.PathValue("id"))
		if err != nil {
			return fmt.Errorf("getting account: %w", err)
		}
		snapshots, err := ListAccountSnapshots(ctx, db, string(acc.ID))
		if err != nil {
			return fmt.Errorf("listing account snapshots: %w", err)
		}
		growthModels, err := ListAccountGrowthModels(ctx, db, string(acc.ID))
		if err != nil {
			return fmt.Errorf("listing account growth models: %w", err)
		}
		return view.AccountPage(w, r, AccountView{
			Account:      acc,
			Snapshots:    snapshots,
			GrowthModels: growthModels,
		})
	})
}

func HandlerAccountEditPage(db *sql.DB, view *View) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		account, err := GetAccount(ctx, db, r.PathValue("id"))
		if err != nil {
			return fmt.Errorf("getting account for edit: %w", err)
		}
		accs, err := ListAccounts(ctx, db)
		if err != nil {
			return fmt.Errorf("listing accounts for edit page: %w", err)
		}
		return view.AccountEditPage(w, r, AccountEditView{
			Account:  account,
			Accounts: accs,
		})
	})
}

func HandlerAccountNewPage(db *sql.DB, view *View) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		accs, err := ListAccounts(ctx, db)
		if err != nil {
			return fmt.Errorf("listing accounts for new account page: %w", err)
		}
		return view.AccountCreatePage(w, r, AccountEditView{
			Accounts: accs,
		})
	})
}

func HandlerAccountSnapshotUpsert(db *sql.DB, view *View) http.Handler {
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
			return view.SnapshotTableCell(w, r, snap)
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

func HandlerAccountSnapshotNewPage(db *sql.DB, view *View) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		acc, err := GetAccount(ctx, db, r.PathValue("id"))
		if err != nil {
			return fmt.Errorf("getting account for snapshot creation: %w", err)
		}
		return view.AccountSnapshotEditPage(w, r, AccountSnapshotEditView{
			Account:  acc,
			Snapshot: AccountSnapshot{Date: date.Today()},
		})
	})
}

func HandlerAccountGrowthModelNewPage(db *sql.DB, view *View) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		acc, err := GetAccount(ctx, db, r.PathValue("id"))
		if err != nil {
			return fmt.Errorf("getting account for creation: %w", err)
		}
		return view.AccountGrowthModelEditPage(w, r, AccountGrowthModelView{
			Account: acc,
			GrowthModel: GrowthModel{
				ID:        sid.MustNewString(32),
				AccountID: acc.ID,
			},
		})
	})
}

func HandlerAccountGrowthModelEditPage(db *sql.DB, view *View) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		gm, err := GetGrowthModel(ctx, db, r.PathValue("id"))
		if err != nil {
			return fmt.Errorf("getting account for creation: %w", err)
		}
		acc, err := GetAccount(ctx, db, gm.AccountID)
		if err != nil {
			return fmt.Errorf("getting account for growth model edit: %w", err)
		}
		return view.AccountGrowthModelEditPage(w, r, AccountGrowthModelView{
			Account:     acc,
			GrowthModel: gm,
		})
	})
}

func HandlerAccountGrowthModelUpsert(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var inp AccountGrowthModelInput
		if err := srvu.Decode(r, &inp, false); err != nil {
			return fmt.Errorf("decoding input: %w", err)
		}
		if inp.ID == "" {
			inp.ID = sid.MustNewString(32)
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

func HandlerAccountSnapshotEditPage(db *sql.DB, view *View) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		acc, err := GetAccount(ctx, db, r.PathValue("id"))
		if err != nil {
			return fmt.Errorf("getting account for snapshot creation: %w", err)
		}
		var d date.Date
		if err := shttp.Parse(&d, date.ParseDate, r.PathValue("date"), date.Today()); err != nil {
			return fmt.Errorf("parsing date: %w", err)
		}
		snapshot, err := GetAccountSnapshot(ctx, db, acc.ID, d)
		if err != nil {
			return fmt.Errorf("getting account snapshot: %w", err)
		}
		return view.AccountSnapshotEditPage(w, r, AccountSnapshotEditView{
			Account:  acc,
			Snapshot: snapshot,
		})
	})
}

func HandlerGetUser(db *sql.DB, tmpl templ.TemplateProvider) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return Respond(GetUser(ctx, db, r.PathValue("id")))(tmpl, w, "user.gohtml")
	})
}

func HandlerUpsertUser(db *sql.DB, tmpl templ.TemplateProvider) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var inp UserInput
		if err := srvu.Decode(r, &inp, false); err != nil {
			return fmt.Errorf("decoding input: %w", err)
		}
		return Respond(UpsertUser(ctx, db, inp))(tmpl, w, "user.gohtml")
	})
}

func HandlerEditUser(db *sql.DB, tmpl templ.TemplateProvider) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return Respond(GetUser(ctx, db, r.PathValue("id")))(tmpl, w, "userModal.gohtml")
	})
}

func HandlerDeleteUser(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return DeleteUser(ctx, db, r.PathValue("id"))
	})
}

func HandlerListUsers(db *sql.DB, tmpl templ.TemplateProvider) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return Respond(ListUsers(ctx, db))(tmpl, w, "userList.gohtml")
	})
}

func HandlerImport(db *sql.DB) http.Handler {
	dbi := pdb.New(db)
	type ImportSnapshot struct {
		Date    *date.Date `json:"date"`
		Balance float64    `json:"balance"`
	}
	type ImportAccount struct {
		ID        string           `json:"id"`
		Name      string           `json:"name"`
		Snapshots []ImportSnapshot `json:"snapshots"`
	}
	type ImportData struct {
		Accounts []ImportAccount `json:"accounts"`
	}
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return nil
		}
		var imp ImportData
		if err := json.NewDecoder(r.Body).Decode(&imp); err != nil {
			return fmt.Errorf("decoding import data: %w", err)
		}
		accs := make([]Account, 0, len(imp.Accounts))
		snapshots := make([]AccountSnapshot, 0)
		for _, a := range imp.Accounts {
			var currDate date.Date
			for _, s := range a.Snapshots {
				if s.Date == nil && currDate == 0 {
					return fmt.Errorf("missing date for snapshot")
				}
				if s.Date != nil {
					currDate = *s.Date
				} else {
					t := currDate.ToStdTime()
					y, m, d := t.Date()
					if m == 12 {
						y++
						m = 1
					} else {
						m++
					}
					parsed, err := date.ParseDate(fmt.Sprintf("%04d-%02d-%02d", y, m, d))
					if err != nil {
						return fmt.Errorf("parsing date from snapshot: %w", err)
					}
					currDate = parsed
				}
				balance := uncertain.NewFixed(s.Balance)
				snapshots = append(snapshots, AccountSnapshot{
					AccountID: a.ID,
					Date:      currDate,
					Balance:   balance,
				})
			}
			accs = append(accs, Account{
				ID:   a.ID,
				Name: a.Name,
			})
		}
		for _, a := range accs {
			if _, err := dbi.CreateAccount(ctx, pdb.CreateAccountParams{
				ID:        a.ID,
				Name:      a.Name,
				CreatedAt: time.Now().UnixMilli(),
				UpdatedAt: time.Now().UnixMilli(),
			}); err != nil {
				return fmt.Errorf("creating account: %w", err)
			}
		}
		for _, s := range snapshots {
			balance, err := s.Balance.Encode()
			if err != nil {
				return fmt.Errorf("encoding balance for snapshot (%s, %s): %w", s.AccountID, s.Date, err)
			}
			if _, err := dbi.UpsertSnapshot(ctx, pdb.UpsertSnapshotParams{
				AccountID: s.AccountID,
				Date:      int64(s.Date),
				Balance:   balance,
			}); err != nil {
				return fmt.Errorf("creating account snapshot (%s): %w", s.AccountID, err)
			}
		}
		return nil
	})
}

func HandlerCharts(tmpl templ.TemplateProvider) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return tmpl.ExecuteTemplate(w, "chart.gohtml", struct{ Query string }{fmt.Sprintf("?%s", r.URL.Query().Encode())})
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

func HandlerTransferTemplateUpsertPage(db *sql.DB, view *View) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		accs, err := ListAccounts(ctx, db)
		if err != nil {
			return fmt.Errorf("listing accounts for transfer template edit: %w", err)
		}
		viewData := TransferTemplateEditView{
			Accounts: accs,
		}
		if id != "" {
			t, err := GetTransferTemplate(ctx, db, id)
			if err != nil {
				return fmt.Errorf("getting transfer template for edit: %w", err)
			}
			viewData.TransferTemplate = t
		}
		return view.TransferTemplateEditPage(w, r, viewData)
	})
}

func HandlerTransferTemplatePage(db *sql.DB, view *View) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		t, err := GetTransferTemplate(ctx, db, r.PathValue("id"))
		if err != nil {
			return fmt.Errorf("getting transfer template for show: %w", err)
		}
		return view.TransferTemplatePage(w, r, TransferTemplateView{
			TransferTemplate: t,
		})
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

func NewHandler(db *sql.DB, public fs.FS, tmpl templ.TemplateProvider, view *View) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /static/public/", srvu.With(http.StripPrefix("/static/public/", http.FileServerFS(public)), srvu.WithCacheCtrlHeader(365*24*time.Hour)))

	mux.Handle("GET /templ/app", RootPage())
	mux.Handle("GET /templ/chart", ChartPage())
	mux.Handle("GET /templ/accounts", AccountsPage(db))
	mux.Handle("GET /templ/accounts/new", AccountNewPage(db))
	mux.Handle("GET /templ/accounts/{id}/edit", AccountEditPage(db))
	mux.Handle("GET /templ/transfer-templates", TransferTemplatesPage(db))

	mux.Handle("GET /{$}", HandlerIndexPage(db, view))

	mux.Handle("GET /accounts/new", HandlerAccountNewPage(db, view))
	mux.Handle("GET /accounts/{id}/edit", HandlerAccountEditPage(db, view))
	mux.Handle("GET /accounts/{id}", HandlerAccountPage(db, view))
	mux.Handle("POST /accounts/{$}", HandlerAccountUpsert(db))
	mux.Handle("POST /accounts/{id}/delete", HandlerAccountDelete(db))

	mux.Handle("GET /accounts/{id}/snapshots/new", HandlerAccountSnapshotNewPage(db, view))
	mux.Handle("GET /accounts/{id}/snapshots/{date}/edit", HandlerAccountSnapshotEditPage(db, view))
	mux.Handle("POST /accounts/{id}/snapshots/", HandlerAccountSnapshotUpsert(db, view))
	mux.Handle("POST /accounts/{id}/snapshots/{date}/", HandlerAccountSnapshotUpsert(db, view))
	mux.Handle("POST /accounts/{id}/snapshots/delete", HandlerAccountSnapshotDelete(db))

	mux.Handle("GET /accounts/{id}/growth-models/new", HandlerAccountGrowthModelNewPage(db, view))
	mux.Handle("GET /growth-models/{id}/edit", HandlerAccountGrowthModelEditPage(db, view))
	mux.Handle("POST /growth-models/", HandlerAccountGrowthModelUpsert(db))
	mux.Handle("POST /growth-models/{id}/delete", HandlerAccountGrowthModelDelete(db))

	mux.Handle("GET /transfers/new", HandlerTransferTemplateUpsertPage(db, view))
	mux.Handle("GET /transfers/{id}/edit", HandlerTransferTemplateUpsertPage(db, view))
	mux.Handle("GET /transfers/{id}", HandlerTransferTemplatePage(db, view))
	mux.Handle("POST /transfers/{$}", HandlerTransferTemplateUpsert(db))
	mux.Handle("POST /transfers/{id}/delete", HandlerTransferTemplateDelete(db))
	mux.Handle("GET /transfers/table/{$}", HandlerTransferTable(db, view))

	mux.Handle("GET /tables/{$}", HandlerTable(db, view))

	mux.Handle("POST /sleep/{$}", srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Simulate a long-running operation
		time.Sleep(1 * time.Second)
		w.WriteHeader(200)
		return nil
	}))

	mux.Handle("GET /charts/{$}", HandlerCharts(tmpl))
	mux.Handle("GET /charts/stream", HandlerChartsDataStream(db))

	// OLD
	mux.Handle("POST /users/{$}", HandlerUpsertUser(db, tmpl))
	mux.Handle("GET /users/{$}", HandlerListUsers(db, tmpl))
	mux.Handle("GET /users/{id}/edit", HandlerEditUser(db, tmpl))
	mux.Handle("DELETE /users/{id}", HandlerDeleteUser(db))
	mux.Handle("GET /users/{id}", HandlerGetUser(db, tmpl))
	mux.Handle("GET /users/new", TemplateHandler(tmpl, "userModal.gohtml", EmptyUser()))

	mux.Handle("POST /import/{$}", HandlerImport(db))

	return mux
}

func TemplateHandler(tmpl templ.TemplateProvider, name string, data interface{}) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return tmpl.ExecuteTemplate(w, name, data)
	})
}

func Respond[T any](v T, err error) func(provider templ.TemplateProvider, w http.ResponseWriter, name string) error {
	return func(provider templ.TemplateProvider, w http.ResponseWriter, name string) error {
		if err != nil {
			return err
		}
		return provider.ExecuteTemplate(w, name, v)
	}
}
