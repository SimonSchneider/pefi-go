package core

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/goslu/srvu"
	"github.com/SimonSchneider/goslu/static/shttp"
	"github.com/SimonSchneider/goslu/templ"
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
			ids[i] = string(acc.ID)
		}
		// Snapshots are ordered by date
		snapshots, err := ListAccountsSnapshots(ctx, db, ids)
		if err != nil {
			return fmt.Errorf("listing account snapshots: %w", err)
		}
		type DateIDKey struct {
			Date date.Date
			ID   AccountID
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
		s, err := UpsertAccountSnapshot(ctx, db, accID, inp)
		if err != nil {
			return fmt.Errorf("upserting snapshot: %w", err)
		}
		if r.Header.Get("HX-Request") == "true" {
			return view.SnapshotTableCell(w, r, s)
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
	mux.Handle("GET /charts/", TemplateHandler(tmpl, "chart.gohtml", nil))

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
