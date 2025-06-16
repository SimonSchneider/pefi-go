package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/goslu/sid"
	"github.com/SimonSchneider/goslu/srvu"
	"github.com/SimonSchneider/goslu/static/shttp"
	"github.com/SimonSchneider/goslu/templ"
	"github.com/SimonSchneider/pefigo/internal/finance"
	"github.com/SimonSchneider/pefigo/internal/pdb"
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
		_, err := UpsertAccountGrowthModel(ctx, db, inp)
		if err != nil {
			return fmt.Errorf("upserting account growth model: %w", err)
		}
		shttp.RedirectToNext(w, r, fmt.Sprintf("/accounts/%s", inp.AccountID))
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
			fmt.Printf("creating account: %s (%s)\n", a.ID, a.Name)
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
		return tmpl.ExecuteTemplate(w, "chart.gohtml", nil)
	})
}

func HandlerChartsDataStream(db *sql.DB) http.Handler {
	q := pdb.New(db)
	type SSEBalanceSnapshot struct {
		ID         string  `json:"id"`
		Day        int64   `json:"day"`
		Balance    float64 `json:"balance"`
		LowerBound float64 `json:"lowerBound"`
		UpperBound float64 `json:"upperBound"`
	}
	type SSEFinancialEntity struct {
		ID        string               `json:"id"`
		Name      string               `json:"name"`
		Snapshots []SSEBalanceSnapshot `json:"snapshots"`
	}
	type SetupEvent struct {
		Max      int64                `json:"max"`
		Entities []SSEFinancialEntity `json:"entities"`
	}
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		startDate := date.Today()
		endDate := startDate.Add(365 * 10)
		samples := 2_000
		quantile := 0.8
		snapshotInterval := date.Cron("*-*-25")

		q1, q2 := (1-quantile)/2, (1+quantile)/2

		entities := make([]finance.Entity, len(staticEntities))
		copy(entities, staticEntities)
		accs, err := q.ListAccounts(ctx)
		if err != nil {
			return fmt.Errorf("listing accounts for SSE: %w", err)
		}
		for _, acc := range accs {
			snaps, err := ListAccountSnapshots(ctx, db, acc.ID)
			if err != nil {
				return fmt.Errorf("getting snapshots for account %s: %w", acc.ID, err)
			}
			gms, err := ListAccountGrowthModels(ctx, db, acc.ID)
			if err != nil {
				return fmt.Errorf("getting growth models for account %s: %w", acc.ID, err)
			}
			var balanceLimit finance.BalanceLimit
			if acc.BalanceUpperLimit != nil {
				balanceLimit = finance.BalanceLimit{
					Upper: uncertain.NewFixed(*acc.BalanceUpperLimit),
				}
			}
			entity := finance.Entity{
				ID:           acc.ID,
				Name:         acc.Name,
				BalanceLimit: balanceLimit,
				Snapshots:    make([]finance.BalanceSnapshot, 0, len(snaps)),
			}
			if acc.CashFlowFrequency != nil || acc.CashFlowDestinationID != nil {
				entity.CashFlow = &finance.CashFlowModel{
					Frequency:     date.Cron(orDefault(acc.CashFlowFrequency)),
					DestinationID: orDefault(acc.CashFlowDestinationID),
				}
			}
			for _, snap := range snaps {
				entity.Snapshots = append(entity.Snapshots, finance.BalanceSnapshot{
					Date:    snap.Date,
					Balance: snap.Balance,
				})
			}
			fgms := make([]finance.GrowthModel, 0, len(gms))
			for _, gm := range gms {
				if gm.Type == "fixed" {
					fgms = append(fgms, &finance.FixedGrowth{
						TimeFrameGrowth: finance.TimeFrameGrowth{
							StartDate: gm.StartDate,
							EndDate:   gm.EndDate,
						},
						AnnualRate: gm.AnnualRate,
					})
				} else if gm.Type == "lognormal" {
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
			if len(fgms) == 1 {
				entity.GrowthModel = finance.NewGrowthCombined(fgms...)
			}
			if len(entity.Snapshots) > 0 {
				entities = append(entities, entity)
			}
		}

		sssEntities := make([]SSEFinancialEntity, len(entities))
		for i, e := range entities {
			sssEntities[i] = SSEFinancialEntity{
				ID:   e.ID,
				Name: e.Name,
			}
			for _, s := range e.Snapshots {
				q := s.Balance.Quantiles()
				sssEntities[i].Snapshots = append(sssEntities[i].Snapshots, SSEBalanceSnapshot{
					ID:         e.ID,
					Day:        s.Date.ToStdTime().UnixMilli(),
					Balance:    s.Balance.Mean(),
					LowerBound: q(q1),
					UpperBound: q(q2),
				})
			}
		}

		ucfg := uncertain.NewConfig(time.Now().UnixMilli(), samples)
		sse := srvu.SSEResponse(w)
		if err := sse.SendNamedJson("setup", SetupEvent{
			Max:      endDate.ToStdTime().UnixMilli(),
			Entities: sssEntities,
		}); err != nil {
			return fmt.Errorf("sending SSE response: %w", err)
		}
		snapshotRecorder := finance.SnapshotRecorderFunc(func(accountID string, day date.Date, balance uncertain.Value) error {
			q := balance.Quantiles()
			event := SSEBalanceSnapshot{
				ID:         accountID,
				Day:        day.ToStdTime().UnixMilli(),
				Balance:    balance.Mean(),
				LowerBound: q(q1),
				UpperBound: q(q2),
			}
			return sse.SendNamedJson("balanceSnapshot", event)
		})
		if err := finance.RunPrediction(ctx, ucfg, startDate, endDate, snapshotInterval, entities, transfers, finance.CompositeRecorder{SnapshotRecorder: snapshotRecorder}); err != nil {
			return fmt.Errorf("running prediction for SSE: %w", err)
		}
		return sse.SendEventWithoutData("close")
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

	mux.Handle("GET /accounts/{id}/growth-models/new", HandlerAccountGrowthModelNewPage(db, view))
	mux.Handle("GET /growth-models/{id}/edit", HandlerAccountGrowthModelEditPage(db, view))
	mux.Handle("POST /growth-models/", HandlerAccountGrowthModelUpsert(db))

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
