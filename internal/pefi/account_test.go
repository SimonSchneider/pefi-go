package pefi_test

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/SimonSchneider/goslu/srvu"
	"github.com/SimonSchneider/pefigo"
	"github.com/SimonSchneider/pefigo/internal/pefi"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func TestCreateAccount(t *testing.T) {
	ctx, pt, cancel := Setup()
	defer cancel()
	data, err := pt.ServeAndGetSingleTemplate(FormReq(ctx, "POST", "/accounts/", map[string]string{
		"name": "test",
	}), http.StatusOK, "account.gohtml")
	if err != nil {
		t.Fatalf("failed to execute request: %s", err)
	}
	if acc, ok := data.(pefi.Account); !ok {
		t.Fatalf("expected data to be of type Account, got %T", data)
	} else {
		if acc.Name != "test" {
			t.Fatalf("expected account name 'test', got %s", acc.Name)
		}
		if acc.ID == "" {
			t.Fatalf("expected account id to be set")
		}
		if acc.CreatedAt.IsZero() {
			t.Fatalf("expected created at to be set")
		}
	}
}

func TestUpdateAccount(t *testing.T) {
	ctx, pt, cancel := Setup()
	defer cancel()
	initial := Must(pt.UpsertAccount(ctx, map[string]string{
		"name": "test",
	}))
	data, err := pt.ServeAndGetSingleTemplate(FormReq(ctx, "POST", "/accounts/", map[string]string{
		"id":   initial.ID,
		"name": "test2",
	}), http.StatusOK, "account.gohtml")
	if err != nil {
		t.Fatalf("updating account: %s", err)
	}
	if acc, ok := data.(pefi.Account); !ok {
		t.Fatalf("template excution for account: %T", data)
	} else {
		if acc.ID != initial.ID {
			t.Fatalf("unexpected id (%s) when updating (%s)", acc.ID, initial.ID)
		}
		if acc.Name != "test2" {
			t.Fatalf("unexpected name (%s) wanted (test2)", acc.Name)
		}
	}
}

func TestListAccounts(t *testing.T) {
	ctx, pt, cancel := Setup()
	defer cancel()
	a1 := Must(pt.UpsertAccount(ctx, map[string]string{"name": "a1"}))
	a2 := Must(pt.UpsertAccount(ctx, map[string]string{"name": "a2"}))
	data, err := pt.ServeAndGetSingleTemplate(httptest.NewRequest("GET", "/accounts/", nil), http.StatusOK, "accountList.gohtml")
	if err != nil {
		t.Fatalf("getting accs: %s", err)
	}
	if accs, ok := data.(*pefi.AccountsList); !ok {
		t.Fatalf("template exec for accounts list: %T", data)
	} else {
		if len(accs.Accounts) != 2 {
			t.Fatalf("unexpected lenght, want 2 got %d", len(accs.Accounts))
		}
		if accs.Accounts[0].ID != a1.ID {
			t.Fatalf("first account should be a1, got %+v", accs.Accounts[0])
		}
		if accs.Accounts[1].ID != a2.ID {
			t.Fatalf("second account should be a2, got %+v", accs.Accounts[1])
		}
	}
}

func TestCreateUserAccount(t *testing.T) {
	ctx, pt, cancel := Setup()
	defer cancel()
	bob := Must(pt.UpsertUser(ctx, map[string]string{
		"name": "bob",
	}))
	acc, err := pt.UpsertAccount(ctx, map[string]string{
		"name":     "checking",
		"type":     "personal",
		"owner_id": bob.ID,
	})
	if err != nil {
		t.Fatalf("cant create account for bob: %s", err)
	}
	if acc.OwnerID != bob.ID {
		t.Fatalf("owner is not bob (%s) but %s", bob.ID, acc.OwnerID)
	}
}

type RecordedTemplate struct {
	name string
	data interface{}
}

type TemplateRecorder struct {
	executed []RecordedTemplate
}

func (t *TemplateRecorder) Lookup(name string) *template.Template {
	panic("no supported")
}

func (t *TemplateRecorder) ExecuteTemplate(w io.Writer, name string, data interface{}) error {
	t.executed = append(t.executed, RecordedTemplate{name, data})
	return nil
}

func (t *TemplateRecorder) Executed() []RecordedTemplate {
	return t.executed
}

func (t *TemplateRecorder) Reset() {
	t.executed = nil
}

func (t *TemplateRecorder) GetSingle(name string) (interface{}, error) {
	if len(t.executed) != 1 {
		return nil, fmt.Errorf("unexpected number of executed templates (%d) expected (%d)", len(t.executed), 1)
	}
	exec := t.executed[0]
	if exec.name != name {
		return nil, fmt.Errorf("expected executed template (%s) to be (%s)", exec.name, name)
	}
	return exec.data, nil
}

func NewTemplateRecorder() *TemplateRecorder {
	return &TemplateRecorder{}
}

type PefiTest struct {
	Handler http.Handler
	Tmpl    *TemplateRecorder
	DB      *sql.DB
}

func (pt *PefiTest) ServeHttp(r *http.Request, statusCode int) error {
	w := httptest.NewRecorder()
	pt.Handler.ServeHTTP(w, r)
	res := w.Result()
	if res.StatusCode != statusCode {
		return fmt.Errorf("expected status code 200, got %d", res.StatusCode)
	}
	return nil
}

func (pt *PefiTest) ServeAndGetSingleTemplate(r *http.Request, statusCode int, templateName string) (interface{}, error) {
	if err := pt.ServeHttp(r, statusCode); err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	return pt.Tmpl.GetSingle(templateName)
}

func (pt *PefiTest) UpsertAccount(ctx context.Context, inp map[string]string) (pefi.Account, error) {
	data, err := pt.ServeAndGetSingleTemplate(FormReq(ctx, "POST", "/accounts/", inp), http.StatusOK, "account.gohtml")
	if err != nil {
		return pefi.Account{}, err
	}
	acc := data.(pefi.Account)
	pt.Tmpl.Reset()
	return acc, nil
}

func (pt *PefiTest) UpsertUser(ctx context.Context, inp map[string]string) (pefi.User, error) {
	data, err := pt.ServeAndGetSingleTemplate(FormReq(ctx, "POST", "/users/", inp), http.StatusOK, "user.gohtml")
	if err != nil {
		return pefi.User{}, err
	}
	user := data.(pefi.User)
	pt.Tmpl.Reset()
	return user, nil
}

func Setup() (context.Context, *PefiTest, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	db, err := pefi.GetMigratedDB(ctx, pefigo.StaticEmbeddedFS, "static/migrations", ":memory:")
	if err != nil {
		panic(fmt.Sprintf("failed to create test db: %s", err))
	}
	ctx = srvu.ContextWithLogger(ctx, srvu.LogToOutput(log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)))
	tmpl := NewTemplateRecorder()
	h := pefi.NewHandler(db, pefigo.StaticEmbeddedFS, tmpl)
	return ctx, &PefiTest{DB: db, Handler: h, Tmpl: tmpl}, cancel
}

func FormReq(ctx context.Context, method, target string, m map[string]string) *http.Request {
	body := make(url.Values)
	for k, v := range m {
		body.Set(k, v)
	}
	r := httptest.NewRequestWithContext(ctx, method, target, strings.NewReader(body.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r
}

func Must[T any](r T, err error) T {
	if err != nil {
		panic(err)
	}
	return r
}
