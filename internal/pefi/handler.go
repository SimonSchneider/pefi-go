package pefi

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/SimonSchneider/goslu/srvu"
	"github.com/SimonSchneider/goslu/templ"
	"io/fs"
	"net/http"
	"time"
)

func HandlerUpsertAccount(db *sql.DB, tmpl templ.TemplateProvider) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var inp AccountInput
		if err := srvu.Decode(r, &inp, false); err != nil {
			return fmt.Errorf("decoding input: %w", err)
		}
		return Respond(UpsertAccount(ctx, db, inp))(tmpl, w, "account.gohtml")
	})
}

func HandlerGetAccount(db *sql.DB, tmpl templ.TemplateProvider) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return Respond(GetAccount(ctx, db, r.PathValue("id")))(tmpl, w, "account.gohtml")
	})
}

func HandlerListAccounts(db *sql.DB, tmpl templ.TemplateProvider) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return Respond(ListAccounts(ctx, db))(tmpl, w, "accountList.gohtml")
	})
}

func HandlerIndex(db *sql.DB, tmpl templ.TemplateProvider) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return Respond(IndexPage(ctx, db))(tmpl, w, "index.gohtml")
	})
}

func HandlerEditAccount(db *sql.DB, tmpl templ.TemplateProvider) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return Respond(GetAccountModal(ctx, db, r.PathValue("id")))(tmpl, w, "accountModal.gohtml")
	})
}

func HandlerDeleteAccount(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return DeleteAccount(ctx, db, r.PathValue("id"))
	})
}

func HandlerNewAccount(db *sql.DB, tmpl templ.TemplateProvider) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return Respond(NewAccount(ctx, db))(tmpl, w, "accountModal.gohtml")
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

func NewHandler(db *sql.DB, public fs.FS, tmpl templ.TemplateProvider) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /static/public/", srvu.With(http.StripPrefix("/static/public/", http.FileServerFS(public)), srvu.WithCacheCtrlHeader(365*24*time.Hour)))

	mux.Handle("POST /accounts/{$}", HandlerUpsertAccount(db, tmpl))
	mux.Handle("GET /accounts/{$}", HandlerListAccounts(db, tmpl))
	mux.Handle("GET /accounts/{id}/edit", HandlerEditAccount(db, tmpl))
	mux.Handle("DELETE /accounts/{id}", HandlerDeleteAccount(db))
	mux.Handle("GET /accounts/{id}", HandlerGetAccount(db, tmpl))
	mux.Handle("GET /accounts/new", HandlerNewAccount(db, tmpl))

	mux.Handle("POST /users/{$}", HandlerUpsertUser(db, tmpl))
	mux.Handle("GET /users/{$}", HandleListUsers(db, tmpl))
	mux.Handle("GET /users/{id}/edit", HandlerEditUser(db, tmpl))
	mux.Handle("DELETE /users/{id}", HandleDeleteUser(db))
	mux.Handle("GET /users/{id}", HandlerGetUser(db, tmpl))
	mux.Handle("GET /users/new", TemplateHandler(tmpl, "userModal.gohtml", EmptyUser()))

	mux.Handle("GET /{$}", HandlerIndex(db, tmpl))
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
