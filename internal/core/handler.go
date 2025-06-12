package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/goslu/srvu"
	"github.com/SimonSchneider/goslu/static/shttp"
	"github.com/SimonSchneider/goslu/templ"
	"github.com/SimonSchneider/pefigo/internal/finance"
	"github.com/SimonSchneider/pefigo/internal/uncertain"
	"io/fs"
	"net/http"
	"time"
)

func HandlerIndexPage(db *sql.DB, view *View) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		accs, err := ListAccounts(ctx, db)
		if err != nil {
			return err
		}
		users, err := ListUsers(ctx, db)
		if err != nil {
			return err
		}
		return view.IndexPage(w, r, IndexView{
			Accounts: AccountsListView{Accounts: accs},
			Users:    UserListView{Users: users},
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
		return view.AccountPage(w, r, AccountView{
			Account:   acc,
			Snapshots: snapshots,
		})
	})
}

func HandlerAccountEditPage(db *sql.DB, view *View) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		account, err := GetAccount(ctx, db, r.PathValue("id"))
		if err != nil {
			return fmt.Errorf("getting account for edit: %w", err)
		}
		return view.AccountEditPage(w, r, AccountEditView{
			Account: account,
		})
	})
}

func HandlerAccountNewPage(view *View) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return view.AccountCreatePage(w, r, AccountEditView{})
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

func HandleDeleteUser(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return DeleteUser(ctx, db, r.PathValue("id"))
	})
}

func HandleListUsers(db *sql.DB, tmpl templ.TemplateProvider) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return Respond(ListUsers(ctx, db))(tmpl, w, "userList.gohtml")
	})
}

func Must[T any](v T, err error) T {
	if err != nil {
		panic(fmt.Errorf("must: %w", err))
	}
	return v
}

var entities = []finance.Entity{
	{
		ID:   "checking",
		Name: "Checking",
		Snapshots: []finance.BalanceSnapshot{
			{Balance: uncertain.NewFixed(500), Date: date.Today().Add(-30)},
			{Balance: uncertain.NewFixed(1000), Date: date.Today()},
		},
	},
	{
		ID:   "savings",
		Name: "Savings",
		Snapshots: []finance.BalanceSnapshot{
			{Balance: uncertain.NewFixed(800_000), Date: date.Today().Add(-30)},
			{Balance: uncertain.NewFixed(810_000), Date: date.Today()},
		},
		//GrowthModel: &finance.LogNormalGrowth{
		//	AnnualRate:       uncertain.NewUniform(0.0, 0.02),
		//	AnnualVolatility: uncertain.NewFixed(0.01),
		//},
		GrowthModel: &finance.LogNormalGrowth{
			AnnualRate:       uncertain.NewUniform(0.04, 0.08),
			AnnualVolatility: uncertain.NewUniform(0.06, 0.12),
		},
	},
	{
		ID:   "realEstate",
		Name: "Real Estate",
		Snapshots: []finance.BalanceSnapshot{
			{Balance: uncertain.NewUniform(3_800_000, 4_100_000), Date: date.Today()}}, //.Add(-1 * date.Year)
		GrowthModel: &finance.LogNormalGrowth{
			AnnualRate:       uncertain.NewUniform(0.03, 0.06),
			AnnualVolatility: uncertain.NewUniform(0.1, 0.2),
		},
	},
	//{
	//	ID:   "plantStocks",
	//	Name: "Plant stocks",
	//	Snapshots: []finance.BalanceSnapshot{
	//		{Balance: uncertain.NewFixed(2_437_000), Date: date.Today().Add(-1 * date.Year)},
	//	},
	//	GrowthModel: &finance.LogNormalGrowth{
	//		AnnualRate:       uncertain.NewUniform(0.2, 0.5),
	//		AnnualVolatility: uncertain.NewUniform(0.3, 0.5),
	//	},
	//},
	{
		ID:   "mortgage",
		Name: "Mortgage",
		BalanceLimit: finance.BalanceLimit{
			Upper: uncertain.NewFixed(0),
		},
		Snapshots: []finance.BalanceSnapshot{
			{Balance: uncertain.NewFixed(-1_300_000), Date: date.Today()},
		},
		GrowthModel: &finance.FixedGrowth{
			AnnualRate: uncertain.NewUniform(0.012, 0.045), // Negative growth for debt
		},
		CashFlow: &finance.CashFlowModel{
			Frequency:     "*-*-25",
			DestinationID: "checking", // Assume mortgage payments go to checking account
		},
	},
}

var transfers = []finance.TransferTemplate{
	{
		ID:            "savings",
		Name:          "Savings Transfer",
		FromAccountID: "checking",
		ToAccountID:   "savings",
		AmountType:    finance.AmountPercent,
		AmountPercent: finance.TransferPercent{
			Percent: 0.2,
		},
		Priority:   0,
		Recurrence: "*-*-25",
		Enabled:    true,
	},
	{
		ID:            "extraMortgagePayment",
		Name:          "extraMortgage Transfer",
		FromAccountID: "checking",
		ToAccountID:   "mortgage",
		AmountType:    finance.AmountPercent,
		AmountPercent: finance.TransferPercent{
			Percent: 0.8,
		},
		Priority:   0,
		Recurrence: "*-*-25",
		Enabled:    true,
	},
	{
		ID:            "finalSavings",
		Name:          "Final Savings",
		FromAccountID: "checking",
		ToAccountID:   "savings",
		AmountType:    finance.AmountPercent,
		AmountPercent: finance.TransferPercent{
			Percent: 1,
		},
		Priority:   1,
		Recurrence: "*-*-25",
		Enabled:    true,
	},
	{
		ID:            "salary",
		Name:          "Salary",
		FromAccountID: "",
		ToAccountID:   "checking",
		AmountType:    finance.AmountFixed,
		AmountFixed: finance.TransferFixed{
			Amount: uncertain.NewFixed(60_000),
		},
		Priority:   2,
		Recurrence: "*-*-25",
		Enabled:    true,
	},
	{
		ID:            "fixedCosts",
		Name:          "Fixed Costs Transfer",
		FromAccountID: "checking",
		ToAccountID:   "",
		AmountType:    finance.AmountFixed,
		AmountFixed: finance.TransferFixed{
			Amount: uncertain.NewFixed(30_000),
		},
		Priority:   3,
		Recurrence: "*-*-25",
		Enabled:    true,
	},
	{
		ID:            "mortgagePayment",
		Name:          "Mortgage Payment",
		FromAccountID: "checking",
		ToAccountID:   "mortgage",
		AmountType:    finance.AmountFixed,
		AmountFixed: finance.TransferFixed{
			Amount: uncertain.NewFixed(10_000),
		},
		Priority:   4,
		Recurrence: "*-*-25",
		Enabled:    true,
	},
}

type ChartDataView struct {
	Entities []finance.Entity
}

func HandlerCharts(db *sql.DB, tmpl templ.TemplateProvider) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// This is a placeholder for the actual chart handler
		// In a real application, you would implement the logic to fetch and return chart data
		return tmpl.ExecuteTemplate(w, "chart.gohtml", ChartDataView{
			Entities: entities,
		})
	})
}

// Creates a SSE subscription handler for chart data.
func HandlerChartsDataSub(db *sql.DB) http.Handler {
	type SSEEvent struct {
		ID         string
		Day        int64
		Balance    float64
		LowerBound float64
		UpperBound float64
	}
	ucfg := uncertain.NewConfig(time.Now().UnixMilli(), 10_000)
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// This is a placeholder for the actual SSE subscription handler
		// In a real application, you would implement the logic to stream chart data
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		startDate := date.Today()
		endDate := startDate.Add(365 * 10)
		if _, err := fmt.Fprintf(w, "event: setup\ndata: {\"max\": %d}\n\n", endDate.ToStdTime().UnixMilli()); err != nil {
			return fmt.Errorf("writing setup event: %w", err)
		}
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		if err := finance.RunPrediction(ctx, ucfg, startDate, endDate, "*-*-25", entities, transfers, func(accountID string, day date.Date, balance uncertain.Value) error {
			// This is where you would send the data to the SSE client
			q := balance.Quantiles()
			event := SSEEvent{
				ID:         accountID,
				Day:        day.ToStdTime().UnixMilli(),
				Balance:    balance.Mean(),
				LowerBound: q(0.1),
				UpperBound: q(0.9),
			}
			data, err := json.Marshal(event)
			if err != nil {
				return fmt.Errorf("marshalling SSE event: %w", err)
			}
			if _, err := fmt.Fprintf(w, "event: balanceSnapshot\ndata: %s\n\n", data); err != nil {
				return fmt.Errorf("writing SSE event: %w", err)
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			return nil
		}); err != nil {
			return fmt.Errorf("running prediction for SSE: %w", err)
		}
		if _, err := fmt.Fprintf(w, "event: close\ndata:\n\n"); err != nil {
			return fmt.Errorf("writing close event: %w", err)
		}
		return nil
	})
}

func NewHandler(db *sql.DB, public fs.FS, tmpl templ.TemplateProvider, view *View) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /static/public/", srvu.With(http.StripPrefix("/static/public/", http.FileServerFS(public)), srvu.WithCacheCtrlHeader(365*24*time.Hour)))

	mux.Handle("GET /{$}", HandlerIndexPage(db, view))

	mux.Handle("GET /accounts/new", HandlerAccountNewPage(view))
	mux.Handle("GET /accounts/{id}/edit", HandlerAccountEditPage(db, view))
	mux.Handle("GET /accounts/{id}", HandlerAccountPage(db, view))
	mux.Handle("POST /accounts/{$}", HandlerAccountUpsert(db))
	mux.Handle("POST /accounts/{id}/delete", HandlerAccountDelete(db))

	mux.Handle("GET /accounts/{id}/snapshots/new", HandlerAccountSnapshotNewPage(db, view))
	mux.Handle("GET /accounts/{id}/snapshots/{date}/edit", HandlerAccountSnapshotEditPage(db, view))
	mux.Handle("POST /accounts/{id}/snapshots/", HandlerAccountSnapshotUpsert(db, view))
	mux.Handle("POST /accounts/{id}/snapshots/{date}/", HandlerAccountSnapshotUpsert(db, view))
	mux.Handle("POST /accounts/{id}/snapshots/delete", HandlerAccountSnapshotDelete(db))

	mux.Handle("GET /tables/{$}", HandlerTable(db, view))

	mux.Handle("POST /sleep/{$}", srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Simulate a long-running operation
		time.Sleep(1 * time.Second)
		w.WriteHeader(200)
		return nil
	}))

	// OLD
	mux.Handle("GET /charts/{$}", HandlerCharts(db, tmpl))
	mux.Handle("GET /charts/sub", HandlerChartsDataSub(db))

	mux.Handle("POST /users/{$}", HandlerUpsertUser(db, tmpl))
	mux.Handle("GET /users/{$}", HandleListUsers(db, tmpl))
	mux.Handle("GET /users/{id}/edit", HandlerEditUser(db, tmpl))
	mux.Handle("DELETE /users/{id}", HandleDeleteUser(db))
	mux.Handle("GET /users/{id}", HandlerGetUser(db, tmpl))
	mux.Handle("GET /users/new", TemplateHandler(tmpl, "userModal.gohtml", EmptyUser()))

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
