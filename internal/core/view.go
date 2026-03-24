package core

import (
	"context"
	"net/http"

	"github.com/SimonSchneider/pefigo/internal/service"
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
		v.w.Header().Set("Cache-Control", "public, max-age=3600")
	} else {
		v.w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		v.w.Header().Set("Pragma", "no-cache")
		v.w.Header().Set("Expires", "0")
	}
}

// Type aliases for service types used by templ templates.
// Templates stay in the core package and reference these aliases,
// while the real type definitions live in the service package.

type (
	Account                          = service.Account
	AccountDetailed                  = service.AccountDetailed
	AccountInput                     = service.AccountInput
	AccountType                      = service.AccountType
	AccountTypeInput                 = service.AccountTypeInput
	AccountTypeWithFilter            = service.AccountTypeWithFilter
	AccountTypesWithFilter           = service.AccountTypesWithFilter
	TransferTemplate                 = service.TransferTemplate
	TransferTemplateWithAmount       = service.TransferTemplateWithAmount
	TransferTemplateCategory         = service.TransferTemplateCategory
	TransferTemplateCategoryInput    = service.TransferTemplateCategoryInput
	AccountSnapshot                  = service.AccountSnapshot
	AccountSnapshotInput             = service.AccountSnapshotInput
	AccountSnapshotCell              = service.AccountSnapshotCell
	BalanceChange                    = service.BalanceChange
	GrowthModel                      = service.GrowthModel
	GrowthModels                     = service.GrowthModels
	AccountGrowthModelInput          = service.AccountGrowthModelInput
	InvestmentRound                  = service.InvestmentRound
	InvestmentRoundInput             = service.InvestmentRoundInput
	ShareChange                      = service.ShareChange
	ShareChangeInput                 = service.ShareChangeInput
	StartupShareOption               = service.StartupShareOption
	StartupShareOptionInput          = service.StartupShareOptionInput
	StartupShareAccount              = service.StartupShareAccount
	StartupShareAccountInput         = service.StartupShareAccountInput
	DerivedStartupShareSummary       = service.DerivedStartupShareSummary
	SpecialDate                      = service.SpecialDate
	SpecialDateInput                 = service.SpecialDateInput
	DashboardView                    = service.DashboardView
	BudgetView                       = service.BudgetView
	BudgetCategoryGroup              = service.BudgetCategoryGroup
	BudgetChartEntry                 = service.BudgetChartEntry
	BudgetItem                       = service.BudgetItem
	AccountsView                     = service.AccountsView
	AccountEditView2                 = service.AccountEditView2
	TransferTemplatesView2           = service.TransferTemplatesView2
	TransferTemplateEditView         = service.TransferTemplateEditView
	TransfersView                    = service.TransfersView
	Transfer                         = service.Transfer
	SnapshotsTableView               = service.SnapshotsTableView
	SnapshotsRow                     = service.SnapshotsRow
	TransferTemplateCategoriesView   = service.TransferTemplateCategoriesView
	TransferTemplateCategoryEditView = service.TransferTemplateCategoryEditView
	CategoriesPageView               = service.CategoriesPageView
	PredictionParams                 = service.PredictionParams
	AccountTypeGroup                 = service.AccountTypeGroup
	AccountTypeChartEntry            = service.AccountTypeChartEntry
	SnapshotHistoryChartData         = service.SnapshotHistoryChartData
	TransferChartGroupBy             = service.TransferChartGroupBy
	GroupBy                          = service.GroupBy
	TransferChartLink                = service.TransferChartLink
	TransferChartDataNode            = service.TransferChartDataNode
	TransferChartDataEnvelope        = service.TransferChartDataEnvelope
	PredictionSetupEvent             = service.PredictionSetupEvent
	PredictionBalanceSnapshot        = service.PredictionBalanceSnapshot
	PredictionFinancialEntity        = service.PredictionFinancialEntity
	Markline                         = service.Markline
	PredictionEventHandler           = service.PredictionEventHandler
	BudgetChartItem                  = service.BudgetChartItem
	SnapshotHistorySeries            = service.SnapshotHistorySeries
	AccountChartEntry                = service.AccountChartEntry
	Salary                           = service.Salary
	SalaryAmount                     = service.SalaryAmount
	SalaryAdjustment                 = service.SalaryAdjustment
	SalaryEditView                   = service.SalaryEditView
	Inkomstbasbelopp                 = service.Inkomstbasbelopp
	TransferTemplateSource           = service.TransferTemplateSource
)

const (
	BalanceUnchanged BalanceChange = service.BalanceUnchanged
	BalanceIncreased BalanceChange = service.BalanceIncreased
	BalanceDecreased BalanceChange = service.BalanceDecreased
)

const (
	GroupByNone  GroupBy = service.GroupByNone
	GroupByType  GroupBy = service.GroupByType
	GroupByTotal GroupBy = service.GroupByTotal
)

const (
	GroupByAccount     TransferChartGroupBy = service.GroupByAccount
	GroupByAccountType TransferChartGroupBy = service.GroupByAccountType
)
