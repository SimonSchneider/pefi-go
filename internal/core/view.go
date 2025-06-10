package core

import (
	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/goslu/templ"
	"io"
	"net/http"
)

type RequestDetails struct {
	req *http.Request
}

func (r *RequestDetails) CurrPath() string {
	return r.req.URL.RequestURI()
}

func (r *RequestDetails) PrevPath() string {
	return r.req.URL.Query().Get("prev")
}

type HtmlTemplateProvider struct {
	templ.TemplateProvider
}

func (p *HtmlTemplateProvider) ExecuteTemplate(w io.Writer, name string, data interface{}) error {
	if rw, ok := w.(http.ResponseWriter); ok {
		if rw.Header().Get("Content-Type") == "" {
			rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		}
		rw.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate, private")
		rw.Header().Set("Pragma", "no-cache")
		rw.Header().Set("Expires", "0")
	}
	return p.TemplateProvider.ExecuteTemplate(w, name, data)
}

type View struct {
	p *HtmlTemplateProvider
}

func NewView(p templ.TemplateProvider) *View {
	return &View{p: &HtmlTemplateProvider{TemplateProvider: p}}
}

type AccountsListView struct {
	*RequestDetails
	Accounts []Account
}

type UserListView struct {
	*RequestDetails
	Users []User
}

type IndexView struct {
	*RequestDetails
	Accounts AccountsListView
	Users    UserListView
}

func (v *View) IndexPage(w http.ResponseWriter, r *http.Request, d IndexView) error {
	rd := &RequestDetails{req: r}
	d.RequestDetails = rd
	d.Accounts.RequestDetails = rd
	d.Users.RequestDetails = rd
	return v.p.ExecuteTemplate(w, "index.page.gohtml", d)
}

type TableRow struct {
	Date      date.Date
	Snapshots []AccountSnapshot
}

type TableView struct {
	*RequestDetails
	Accounts []Account
	Rows     []TableRow
}

func (v *View) TablePage(w http.ResponseWriter, r *http.Request, d TableView) error {
	d.RequestDetails = &RequestDetails{req: r}
	return v.p.ExecuteTemplate(w, "table.page.gohtml", d)
}

func (v *View) SnapshotTableCell(w http.ResponseWriter, r *http.Request, d AccountSnapshot) error {
	return v.p.ExecuteTemplate(w, "table_cell.partial.gohtml", d)
}

type AccountEditView struct {
	*RequestDetails
	Account Account
}

func (c AccountEditView) IsEdit() bool {
	return c.Account.ID != ""
}

func (v *View) AccountEditPage(w http.ResponseWriter, r *http.Request, d AccountEditView) error {
	d.RequestDetails = &RequestDetails{req: r}
	return v.p.ExecuteTemplate(w, "account_edit.page.gohtml", d)
}

func (v *View) AccountCreatePage(w http.ResponseWriter, r *http.Request, d AccountEditView) error {
	d.RequestDetails = &RequestDetails{req: r}
	return v.p.ExecuteTemplate(w, "account_edit.page.gohtml", d)
}

type AccountSnapshotEditView struct {
	*RequestDetails
	Account  Account
	Snapshot AccountSnapshot
}

func (c AccountSnapshotEditView) IsEdit() bool {
	return c.Snapshot.AccountID != ""
}

func (v *View) AccountSnapshotEditPage(w http.ResponseWriter, r *http.Request, d AccountSnapshotEditView) error {
	d.RequestDetails = &RequestDetails{req: r}
	return v.p.ExecuteTemplate(w, "account_snapshot_edit.page.gohtml", d)
}

type AccountView struct {
	*RequestDetails
	Account   Account
	Snapshots []AccountSnapshot
}

func (v *View) AccountPage(w http.ResponseWriter, r *http.Request, d AccountView) error {
	d.RequestDetails = &RequestDetails{req: r}
	return v.p.ExecuteTemplate(w, "account.page.gohtml", d)
}
