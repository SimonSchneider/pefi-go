package core

import (
	"context"
	"fmt"
	"net/http"

	"github.com/a-h/templ"
)

type View struct {
	ctx context.Context
	w   http.ResponseWriter
	r   *http.Request
}

func NewView(ctx context.Context, w http.ResponseWriter, r *http.Request) *View {
	return &View{ctx: ctx, w: w, r: r}
}

func (v *View) Render(c templ.Component) error {
	v.setupHeaders(false)
	return c.Render(v.ctx, v.w)
}

func (v *View) setupHeaders(cache bool) {
	v.w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if cache {
		v.w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour
	} else {
		// No caching headers
		v.w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		v.w.Header().Set("Pragma", "no-cache")
		v.w.Header().Set("Expires", "0")
	}
}

type RequestDetails struct {
	req *http.Request
}

func (r *RequestDetails) CurrPath() string {
	return r.req.URL.RequestURI()
}

func (r *RequestDetails) PrevPath() string {
	return r.req.FormValue("prev")
}

type TransferTemplateEditView struct {
	*RequestDetails
	TransferTemplate TransferTemplate
	Accounts         []Account
}

func (c TransferTemplateEditView) IsEdit() bool {
	return c.TransferTemplate.ID != ""
}

type TransferTemplateWithAmount struct {
	TransferTemplate
	Amount float64
}

func makeTransferTemplatesWithAmount(transfers []TransferTemplate) []TransferTemplateWithAmount {
	type AccBalance struct {
		Starting float64
		Current  float64
	}
	accs := make(map[string]AccBalance)
	ttwas := make([]TransferTemplateWithAmount, len(transfers))
	// handle the percentage transfers with the same priority using the same initial account balance
	currIter := ""
	for i, t := range transfers {
		nextIter := fmt.Sprintf("%s%d", t.Recurrence, t.Priority)
		if nextIter != currIter {
			currIter = nextIter
			for k, acc := range accs {
				acc.Starting = acc.Current
				accs[k] = acc
			}
		}
		ttwa := TransferTemplateWithAmount{TransferTemplate: t}
		accs = initMap(accs, t.FromAccountID)
		accs = initMap(accs, t.ToAccountID)
		switch t.AmountType {
		case "fixed":
			ttwa.Amount = t.AmountFixed.Mean()
		case "percent":
			ttwa.Amount = t.AmountPercent * accs[t.FromAccountID].Starting
		}
		fromAcc := accs[t.FromAccountID]
		fromAcc.Current -= ttwa.Amount
		accs[t.FromAccountID] = fromAcc
		toAcc := accs[t.ToAccountID]
		toAcc.Current += ttwa.Amount
		accs[t.ToAccountID] = toAcc
		ttwas[i] = ttwa
	}
	return ttwas
}

func initMap[K comparable, V any](m map[K]V, ks ...K) map[K]V {
	var v V
	for _, k := range ks {
		if _, ok := m[k]; !ok {
			m[k] = v
		}
	}
	return m
}

type TransferTemplatesView2 struct {
	TransferTemplates []TransferTemplateWithAmount
	Accounts          map[string]Account
	MonthlyIncome     float64
	MonthlyExpenses   float64
}

func NewTransferTemplatesView2(transferTemplates []TransferTemplate, accounts []Account) *TransferTemplatesView2 {
	v := &TransferTemplatesView2{TransferTemplates: makeTransferTemplatesWithAmount(transferTemplates), Accounts: KeyBy(accounts, func(a Account) string { return a.ID })}
	for _, t := range v.TransferTemplates {
		if t.FromAccountID == "" {
			v.MonthlyIncome += t.Amount
		} else if t.ToAccountID == "" {
			v.MonthlyExpenses += -t.Amount
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
