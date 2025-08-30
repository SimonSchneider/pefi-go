package core

import (
	"io"
	"math"
	"net/http"
	"strconv"

	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/goslu/templ"
)

type RequestDetails struct {
	req *http.Request
}

func (r *RequestDetails) CurrPath() string {
	return r.req.URL.RequestURI()
}

func (r *RequestDetails) PrevPath() string {
	return r.req.FormValue("prev")
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
}

type IndexView struct {
	*RequestDetails
	Accounts  AccountsListView
	Transfers TransferTemplatesView
	Users     UserListView
}

func (v *View) IndexPage(w http.ResponseWriter, r *http.Request, d IndexView) error {
	rd := &RequestDetails{req: r}
	d.RequestDetails = rd
	d.Accounts.RequestDetails = rd
	d.Transfers.RequestDetails = rd
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
	Account  Account
	Accounts []Account
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

type AccountGrowthModelView struct {
	*RequestDetails
	Account     Account
	GrowthModel GrowthModel
}

func (c AccountGrowthModelView) IsEdit() bool {
	return c.GrowthModel.ID != ""
}

func (v *View) AccountGrowthModelEditPage(w http.ResponseWriter, r *http.Request, d AccountGrowthModelView) error {
	d.RequestDetails = &RequestDetails{req: r}
	return v.p.ExecuteTemplate(w, "account_growth_model_edit.page.gohtml", d)
}

type AccountView struct {
	*RequestDetails
	Account      Account
	Snapshots    []AccountSnapshot
	GrowthModels []GrowthModel
}

func (v *View) AccountPage(w http.ResponseWriter, r *http.Request, d AccountView) error {
	d.RequestDetails = &RequestDetails{req: r}
	return v.p.ExecuteTemplate(w, "account.page.gohtml", d)
}

type TransferTemplatesView struct {
	*RequestDetails
	Transfers []TransferTemplate
}

type TransferTemplateEditView struct {
	*RequestDetails
	TransferTemplate TransferTemplate
	Accounts         []Account
}

func (c TransferTemplateEditView) IsEdit() bool {
	return c.TransferTemplate.ID != ""
}

func (v *View) TransferTemplateEditPage(w http.ResponseWriter, r *http.Request, d TransferTemplateEditView) error {
	d.RequestDetails = &RequestDetails{req: r}
	return v.p.ExecuteTemplate(w, "transfer_template_edit.page.gohtml", d)
}

type TransferTemplateView struct {
	*RequestDetails
	TransferTemplate TransferTemplate
}

func (v *View) TransferTemplatePage(w http.ResponseWriter, r *http.Request, d TransferTemplateView) error {
	d.RequestDetails = &RequestDetails{req: r}
	return v.p.ExecuteTemplate(w, "transfer_template.page.gohtml", d)
}

type TransferTableRow struct {
	*RequestDetails
	Transfer TransferTemplate
	Accounts []Account
}

type TransferTableView struct {
	*RequestDetails
	Rows []TransferTableRow
}

func (v *View) TransferTablePage(w http.ResponseWriter, r *http.Request, d TransferTableView) error {
	d.RequestDetails = &RequestDetails{req: r}
	return v.p.ExecuteTemplate(w, "transfer_template_table.page.gohtml", d)
}

type TransferTemplatesView2 struct {
	TransferTemplates []TransferTemplate
	Accounts          map[string]Account
	MonthlyIncome     float64
	MonthlyExpenses   float64
}

func NewTransferTemplatesView2(transferTemplates []TransferTemplate, accounts []Account) *TransferTemplatesView2 {
	v := &TransferTemplatesView2{TransferTemplates: transferTemplates, Accounts: KeyBy(accounts, func(a Account) string { return a.ID })}
	for _, t := range transferTemplates {
		if t.FromAccountID == "" {
			v.MonthlyIncome += t.AmountFixed.Mean()
		} else if t.ToAccountID == "" {
			v.MonthlyExpenses += -t.AmountFixed.Mean()
		}
	}
	return v
}

func (v *TransferTemplatesView2) GetAccount(id string) *Account {
	a := v.Accounts[id]
	return &a
}

type AccountsView struct {
	Accounts         []AccountDetailed
	TotalBalance     float64
	TotalAssets      float64
	TotalLiabilities float64
}

func NewAccountsView(accounts []AccountDetailed) *AccountsView {
	v := &AccountsView{Accounts: accounts}
	for _, account := range accounts {
		if account.LastSnapshot != nil {
			v.TotalBalance += account.LastSnapshot.Balance.Mean()
			if account.LastSnapshot.Balance.Mean() > 0 {
				v.TotalAssets += account.LastSnapshot.Balance.Mean()
			} else {
				v.TotalLiabilities += account.LastSnapshot.Balance.Mean()
			}
		}
	}
	return v
}

func FormatWithThousands(val float64) string {
	// Round the value to the nearest integer
	rounded := math.Round(val)
	s := strconv.FormatInt(int64(rounded), 10)
	n := len(s)
	neg := false
	if n > 0 && s[0] == '-' {
		neg = true
		s = s[1:]
		n--
	}
	if n <= 3 {
		if neg {
			return "-" + s
		}
		return s
	}
	var out []byte
	pre := n % 3
	if pre == 0 {
		pre = 3
	}
	out = append(out, s[:pre]...)
	for i := pre; i < n; i += 3 {
		out = append(out, ',')
		out = append(out, s[i:i+3]...)
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}

func KeyBy[T any](items []T, key func(T) string) map[string]T {
	m := make(map[string]T)
	for _, item := range items {
		m[key(item)] = item
	}
	return m
}

type AccountEditView2 struct {
	Account      Account
	Accounts     []Account
	GrowthModels []GrowthModel
}

func NewAccountEditView2(account Account, accounts []Account, growthModels []GrowthModel) *AccountEditView2 {
	return &AccountEditView2{Account: account, Accounts: accounts, GrowthModels: growthModels}
}

func (v *AccountEditView2) IsEdit() bool {
	return v.Account.ID != ""
}
