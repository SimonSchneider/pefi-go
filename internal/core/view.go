package core

import (
	"context"
	"fmt"
	"math"
	"net/http"

	"github.com/SimonSchneider/goslu/date"
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
	Categories       []TransferTemplateCategory
	AllTemplates     []TransferTemplate // For selecting parent template
}

func (c TransferTemplateEditView) IsEdit() bool {
	return c.TransferTemplate.ID != ""
}

type TransferTemplateWithAmount struct {
	TransferTemplate
	Amount float64

	SimDate date.Date
}

func (c *TransferTemplateWithAmount) HasDifferentAmount() bool {
	return math.Abs(c.Amount-c.AmountFixed.Mean()) > 1
}

func (t *TransferTemplateWithAmount) ActiveState() string {
	if !t.Enabled {
		return "Disabled"
	}
	if t.EndDate != nil && t.EndDate.Before(t.SimDate) || t.StartDate.After(t.SimDate) {
		return "Inactive"
	}
	return "Active"
}

func makeTransferTemplatesWithAmount(transfers []TransferTemplate, day date.Date) []TransferTemplateWithAmount {
	type AccBalance struct {
		Starting float64
		Current  float64
	}
	accs := make(map[string]AccBalance)
	ttwas := make([]TransferTemplateWithAmount, len(transfers))
	// handle the percentage transfers with the same priority using the same initial account balance
	currIter := ""
	for i, t := range transfers {
		ttwa := TransferTemplateWithAmount{TransferTemplate: t, SimDate: day}
		if !t.Enabled || (t.EndDate != nil && t.EndDate.Before(day) || t.StartDate.After(day)) {
			ttwas[i] = ttwa
			continue
		}
		nextIter := fmt.Sprintf("%s%d", t.Recurrence, t.Priority)
		if nextIter != currIter {
			currIter = nextIter
			for k, acc := range accs {
				acc.Starting = acc.Current
				accs[k] = acc
			}
		}
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
	Categories        map[string]TransferTemplateCategory
	MonthlyIncome     float64
	MonthlyExpenses   float64
}

func NewTransferTemplatesView2(transferTemplates []TransferTemplate, accounts []Account, categories []TransferTemplateCategory) *TransferTemplatesView2 {
	v := &TransferTemplatesView2{
		TransferTemplates: makeTransferTemplatesWithAmount(transferTemplates, date.Today()),
		Accounts:          KeyBy(accounts, func(a Account) string { return a.ID }),
		Categories:        KeyBy(categories, func(c TransferTemplateCategory) string { return c.ID }),
	}
	for _, t := range v.TransferTemplates {
		if t.FromAccountID == "" {
			v.MonthlyIncome += t.Amount
		} else if t.ToAccountID == "" {
			v.MonthlyExpenses += -t.Amount
		}
	}
	return v
}

func (v *TransferTemplatesView2) GetBudgetCategory(id *string) *TransferTemplateCategory {
	if id == nil {
		return nil
	}
	cat, ok := v.Categories[*id]
	if !ok {
		return nil
	}
	return &cat
}

func (v *TransferTemplatesView2) GetAccount(id string) *Account {
	a := v.Accounts[id]
	return &a
}

type AccountTypeWithFilter struct {
	AccountType
	Exclude bool
}

type AccountsView struct {
	Accounts         []AccountDetailed
	AccountTypes     AccountTypesWithFilter
	Categories       map[string]TransferTemplateCategory
	TotalBalance     float64
	TotalAssets      float64
	TotalLiabilities float64
}

func NewAccountsView(accounts []AccountDetailed, accountTypes []AccountTypeWithFilter, categories []TransferTemplateCategory) *AccountsView {
	v := &AccountsView{
		Accounts:     accounts,
		AccountTypes: accountTypes,
		Categories:   KeyBy(categories, func(c TransferTemplateCategory) string { return c.ID }),
	}
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

func (v *AccountsView) GetBudgetCategory(id *string) *TransferTemplateCategory {
	if id == nil {
		return nil
	}
	cat, ok := v.Categories[*id]
	if !ok {
		return nil
	}
	return &cat
}

func (v *AccountsView) GetAccountType(typeID string) AccountTypeWithFilter {
	return v.AccountTypes.GetAccountType(typeID)
}

func KeyBy[T any](items []T, key func(T) string) map[string]T {
	m := make(map[string]T)
	for _, item := range items {
		m[key(item)] = item
	}
	return m
}

type AccountEditView2 struct {
	Account             Account
	Accounts            []Account
	GrowthModels        []GrowthModel
	AccountTypes        AccountTypesWithFilter
	Categories          []TransferTemplateCategory
	StartupShareAccount *StartupShareAccount
	InvestmentRounds    []InvestmentRound
	Options             []StartupShareOption
}

func NewAccountEditView2(account Account, accounts []Account, growthModels []GrowthModel, accountTypes []AccountTypeWithFilter, categories []TransferTemplateCategory, startupShareAccount *StartupShareAccount, investmentRounds []InvestmentRound, options []StartupShareOption) *AccountEditView2 {
	return &AccountEditView2{
		Account:             account,
		Accounts:            accounts,
		GrowthModels:        growthModels,
		AccountTypes:        accountTypes,
		Categories:          categories,
		StartupShareAccount: startupShareAccount,
		InvestmentRounds:    investmentRounds,
		Options:             options,
	}
}

func (v *AccountEditView2) GetStartupShareSharesOwned() string {
	if v.StartupShareAccount == nil {
		return "0"
	}
	return fmt.Sprintf("%.2f", v.StartupShareAccount.SharesOwned)
}

func (v *AccountEditView2) GetStartupShareTotalShares() string {
	if v.StartupShareAccount == nil {
		return "0"
	}
	return fmt.Sprintf("%.2f", v.StartupShareAccount.TotalShares)
}

func (v *AccountEditView2) GetStartupSharePurchasePrice() string {
	if v.StartupShareAccount == nil {
		return "0"
	}
	return fmt.Sprintf("%.10f", v.StartupShareAccount.PurchasePricePerShare)
}

func (v *AccountEditView2) GetStartupShareTaxRate() string {
	if v.StartupShareAccount == nil {
		return "15"
	}
	return fmt.Sprintf("%.2f", v.StartupShareAccount.TaxRate*100)
}

func (v *AccountEditView2) GetStartupShareDiscountFactor() string {
	if v.StartupShareAccount == nil {
		return "50"
	}
	return fmt.Sprintf("%.2f", v.StartupShareAccount.ValuationDiscountFactor*100)
}

func (v *AccountEditView2) HasStartupShareAccount() bool {
	return v.StartupShareAccount != nil
}

func (v *AccountEditView2) GetStartupShareFieldsStyle() string {
	if v.HasStartupShareAccount() {
		return "display: block;"
	}
	return "display: none;"
}

func (ir InvestmentRound) GetDateString() string {
	if ir.ID == "" {
		return ""
	}
	return ir.Date.String()
}

func (ir InvestmentRound) GetValuationString() string {
	if ir.ID == "" {
		return ""
	}
	return fmt.Sprintf("%.2f", ir.Valuation)
}

func (opt StartupShareOption) GetSharesString() string {
	if opt.ID == "" {
		return ""
	}
	return fmt.Sprintf("%.2f", opt.Shares)
}

func (opt StartupShareOption) GetStrikePriceString() string {
	if opt.ID == "" {
		return ""
	}
	return fmt.Sprintf("%.2f", opt.StrikePricePerShare)
}

func (opt StartupShareOption) GetGrantDateString() string {
	if opt.ID == "" {
		return ""
	}
	return opt.GrantDate.String()
}

func (opt StartupShareOption) GetEndDateString() string {
	if opt.ID == "" {
		return ""
	}
	return opt.EndDate.String()
}

func (gm GrowthModel) GetEndDateString() string {
	if gm.ID == "" || gm.EndDate == nil {
		return ""
	}
	return gm.EndDate.String()
}

func (v *AccountEditView2) IsEdit() bool {
	return v.Account.ID != ""
}

func (v *AccountEditView2) GetAccountTypeName(typeID string) string {
	return v.AccountTypes.GetAccountType(typeID).Name
}

// Transfer Template Category Views
type TransferTemplateCategoriesView struct {
	Categories []TransferTemplateCategory
}

func NewTransferTemplateCategoriesView(categories []TransferTemplateCategory) *TransferTemplateCategoriesView {
	return &TransferTemplateCategoriesView{Categories: categories}
}

type TransferTemplateCategoryEditView struct {
	Category TransferTemplateCategory
}

func (v TransferTemplateCategoryEditView) IsEdit() bool {
	return v.Category.ID != ""
}
