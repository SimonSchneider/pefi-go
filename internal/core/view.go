package core

import (
	"math"
	"net/http"
	"strconv"

	"github.com/SimonSchneider/goslu/date"
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

type TableRow struct {
	Date      date.Date
	Snapshots []AccountSnapshot
}

type TableView struct {
	*RequestDetails
	Accounts []Account
	Rows     []TableRow
}

type AccountEditView struct {
	*RequestDetails
	Account  Account
	Accounts []Account
}

func (c AccountEditView) IsEdit() bool {
	return c.Account.ID != ""
}

type AccountSnapshotEditView struct {
	*RequestDetails
	Account  Account
	Snapshot AccountSnapshot
}

func (c AccountSnapshotEditView) IsEdit() bool {
	return c.Snapshot.AccountID != ""
}

type AccountGrowthModelView struct {
	*RequestDetails
	Account     Account
	GrowthModel GrowthModel
}

func (c AccountGrowthModelView) IsEdit() bool {
	return c.GrowthModel.ID != ""
}

type AccountView struct {
	*RequestDetails
	Account      Account
	Snapshots    []AccountSnapshot
	GrowthModels []GrowthModel
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

type TransferTemplateView struct {
	*RequestDetails
	TransferTemplate TransferTemplate
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
