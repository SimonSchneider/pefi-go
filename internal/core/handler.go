package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"time"

	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/goslu/srvu"
	"github.com/SimonSchneider/goslu/static/shttp"
	"github.com/SimonSchneider/pefigo/internal/service"
	"github.com/SimonSchneider/pefigo/internal/ui"
)

type Handler struct {
	svc *service.Service
}

func NewHandler(svc *service.Service, public fs.FS) http.Handler {
	h := &Handler{svc: svc}
	mux := http.NewServeMux()
	mux.Handle("GET /static/public/", srvu.With(http.StripPrefix("/static/public/", http.FileServerFS(public)), srvu.WithCacheCtrlHeader(365*24*time.Hour)))

	mux.Handle("GET /{$}", h.dashboardPage())
	mux.Handle("GET /accounts", h.accountsPage())
	mux.Handle("GET /accounts/new", h.accountNewPage())
	mux.Handle("GET /accounts/{id}/edit", h.accountEditPage())
	mux.Handle("GET /transfer-templates", h.transferTemplatesPage())
	mux.Handle("GET /transfer-templates/new", h.transferTemplatesNewPage())
	mux.Handle("GET /transfer-templates/{id}/edit", h.transferTemplatesEditPage())

	mux.Handle("GET /categories", h.categoriesPage())

	mux.Handle("GET /account-types/new", h.accountTypeNewPage())
	mux.Handle("GET /account-types/{id}/edit", h.accountTypeEditPage())
	mux.Handle("POST /account-types/{$}", h.accountTypeUpsert())
	mux.Handle("POST /account-types/{id}/delete", h.accountTypeDelete())

	mux.Handle("GET /special-dates", h.specialDatesPage())
	mux.Handle("GET /special-dates/new", h.specialDateNewPage())
	mux.Handle("GET /special-dates/{id}/edit", h.specialDateEditPage())
	mux.Handle("POST /special-dates/{$}", h.specialDateUpsert())
	mux.Handle("POST /special-dates/{id}/delete", h.specialDateDelete())

	mux.Handle("GET /snapshots-table", h.snapshotsTablePage())
	mux.Handle("POST /snapshots-table/modify-date", h.snapshotsTableModifyDate())
	mux.Handle("GET /snapshots-table/empty-row", h.snapshotsTableEmptyRow())
	mux.Handle("POST /accounts/{id}/snapshots/{date}/", h.accountSnapshotUpsert())

	mux.Handle("GET /transfers", h.transfersPage())
	mux.Handle("GET /transfers/chart/{$}", h.transferChartPage())
	mux.Handle("GET /transfers/chart/data", h.transferChartData())

	mux.Handle("GET /budget", h.budgetPage())

	mux.Handle("GET /chart", h.chartPage())
	mux.Handle("GET /chart/stream", h.chartsDataStream())

	mux.Handle("POST /accounts/{$}", h.accountUpsert())
	mux.Handle("POST /accounts/{id}/delete", h.accountDelete())

	mux.Handle("POST /growth-models/", h.accountGrowthModelUpsert())
	mux.Handle("POST /growth-models/{id}/delete", h.accountGrowthModelDelete())

	mux.Handle("POST /startup-share-accounts/", h.startupShareAccountUpsert())
	mux.Handle("POST /investment-rounds/", h.investmentRoundUpsert())
	mux.Handle("POST /investment-rounds/{id}/delete", h.investmentRoundDelete())
	mux.Handle("POST /share-changes/", h.shareChangeUpsert())
	mux.Handle("POST /share-changes/{id}/delete", h.shareChangeDelete())
	mux.Handle("POST /startup-share-options/", h.startupShareOptionUpsert())
	mux.Handle("POST /startup-share-options/{id}/delete", h.startupShareOptionDelete())

	mux.Handle("GET /salaries", h.salariesPage())
	mux.Handle("GET /salaries/new", h.salaryNewPage())
	mux.Handle("GET /salaries/{id}/edit", h.salaryEditPage())
	mux.Handle("POST /salaries/{$}", h.salaryUpsert())
	mux.Handle("POST /salaries/{id}/delete", h.salaryDelete())
	mux.Handle("POST /salary-amounts/{$}", h.salaryAmountUpsert())
	mux.Handle("POST /salary-amounts/{id}/delete", h.salaryAmountDelete())

	mux.Handle("POST /transfers/{$}", h.transferTemplateUpsert())
	mux.Handle("POST /transfers/{id}/duplicate", h.transferTemplateDuplicate())
	mux.Handle("POST /transfers/{id}/delete", h.transferTemplateDelete())

	mux.Handle("GET /transfer-template-categories/new", h.transferTemplateCategoryNewPage())
	mux.Handle("GET /transfer-template-categories/{id}/edit", h.transferTemplateCategoryEditPage())
	mux.Handle("POST /transfer-template-categories/{$}", h.transferTemplateCategoryUpsert())
	mux.Handle("POST /transfer-template-categories/{id}/delete", h.transferTemplateCategoryDelete())

	mux.Handle("POST /sleep/{$}", srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		time.Sleep(1 * time.Second)
		w.WriteHeader(200)
		return nil
	}))

	return mux
}

// ---- Dashboard ----

func (h *Handler) dashboardPage() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		view, err := h.svc.GetDashboardData(ctx)
		if err != nil {
			return fmt.Errorf("computing dashboard view: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Dashboard", PageDashboard(view)))
	})
}

// ---- Budget ----

func (h *Handler) budgetPage() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		view, err := h.svc.GetBudgetData(ctx)
		if err != nil {
			return fmt.Errorf("computing budget view: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Budget", PageBudget(view)))
	})
}

// ---- Accounts ----

func (h *Handler) accountsPage() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		view, err := h.svc.GetAccountsPageData(ctx, extractExcludedTypeIDs(r))
		if err != nil {
			return fmt.Errorf("getting accounts page data: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Accounts", PageAccounts(view)))
	})
}

func (h *Handler) accountNewPage() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		view, err := h.svc.GetAccountNewPageData(ctx, extractExcludedTypeIDs(r))
		if err != nil {
			return fmt.Errorf("getting account new page data: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Accounts", PageEditAccount(view)))
	})
}

func (h *Handler) accountEditPage() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		view, err := h.svc.GetAccountEditPageData(ctx, r.PathValue("id"), extractExcludedTypeIDs(r))
		if err != nil {
			return fmt.Errorf("getting account edit page data: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Accounts", PageEditAccount(view)))
	})
}

func (h *Handler) accountUpsert() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var inp accountInputForm
		if err := srvu.Decode(r, &inp, false); err != nil {
			return fmt.Errorf("decoding input: %w", err)
		}
		var startupShares *service.StartupShareAccountInput
		if r.FormValue("enable_startup_shares") == "on" {
			var ssaInp startupShareAccountInputForm
			if err := srvu.Decode(r, &ssaInp, false); err == nil {
				startupShares = &ssaInp.StartupShareAccountInput
			}
		}
		acc, err := h.svc.UpsertAccountWithStartupShares(ctx, inp.AccountInput, startupShares)
		if err != nil {
			return fmt.Errorf("upserting account: %w", err)
		}
		shttp.RedirectToNext(w, r, fmt.Sprintf("/accounts/%s", acc.ID))
		return nil
	})
}

func (h *Handler) accountDelete() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if err := h.svc.DeleteAccount(ctx, r.PathValue("id")); err != nil {
			return fmt.Errorf("deleting account: %w", err)
		}
		shttp.RedirectToNext(w, r, "/accounts/")
		return nil
	})
}

// ---- Account Growth Models ----

func (h *Handler) accountGrowthModelUpsert() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var inp accountGrowthModelInputForm
		if err := srvu.Decode(r, &inp, false); err != nil {
			return fmt.Errorf("decoding input: %w", err)
		}
		_, err := h.svc.UpsertAccountGrowthModel(ctx, inp.AccountGrowthModelInput)
		if err != nil {
			return fmt.Errorf("upserting account growth model: %w", err)
		}
		shttp.RedirectToNext(w, r, fmt.Sprintf("/accounts/%s", inp.AccountID))
		return nil
	})
}

func (h *Handler) accountGrowthModelDelete() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if err := h.svc.DeleteAccountGrowthModel(ctx, r.PathValue("id")); err != nil {
			return fmt.Errorf("deleting account growth model: %w", err)
		}
		shttp.RedirectToNext(w, r, fmt.Sprintf("/accounts/%s", r.PathValue("id")))
		return nil
	})
}

// ---- Startup Shares ----

func (h *Handler) startupShareAccountUpsert() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var inp startupShareAccountInputForm
		if err := srvu.Decode(r, &inp, false); err != nil {
			return fmt.Errorf("decoding input: %w", err)
		}
		_, err := h.svc.UpsertStartupShareAccount(ctx, inp.StartupShareAccountInput)
		if err != nil {
			return fmt.Errorf("upserting startup share account: %w", err)
		}
		shttp.RedirectToNext(w, r, fmt.Sprintf("/accounts/%s/edit", inp.AccountID))
		return nil
	})
}

func (h *Handler) investmentRoundUpsert() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var inp investmentRoundInputForm
		if err := srvu.Decode(r, &inp, false); err != nil {
			return fmt.Errorf("decoding input: %w", err)
		}
		_, err := h.svc.UpsertInvestmentRound(ctx, inp.InvestmentRoundInput)
		if err != nil {
			return fmt.Errorf("upserting investment round: %w", err)
		}
		shttp.RedirectToNext(w, r, fmt.Sprintf("/accounts/%s/edit", inp.AccountID))
		return nil
	})
}

func (h *Handler) investmentRoundDelete() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		roundID := r.PathValue("id")
		round, err := h.svc.GetInvestmentRound(ctx, roundID)
		if err != nil {
			return fmt.Errorf("getting investment round: %w", err)
		}
		if err := h.svc.DeleteInvestmentRound(ctx, roundID); err != nil {
			return fmt.Errorf("deleting investment round: %w", err)
		}
		shttp.RedirectToNext(w, r, fmt.Sprintf("/accounts/%s/edit", round.AccountID))
		return nil
	})
}

func (h *Handler) shareChangeUpsert() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var inp shareChangeInputForm
		if err := srvu.Decode(r, &inp, false); err != nil {
			return fmt.Errorf("decoding input: %w", err)
		}
		_, err := h.svc.UpsertShareChange(ctx, inp.ShareChangeInput)
		if err != nil {
			return fmt.Errorf("upserting share change: %w", err)
		}
		shttp.RedirectToNext(w, r, fmt.Sprintf("/accounts/%s/edit", inp.AccountID))
		return nil
	})
}

func (h *Handler) shareChangeDelete() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		changeID := r.PathValue("id")
		sc, err := h.svc.GetShareChange(ctx, changeID)
		if err != nil {
			return fmt.Errorf("getting share change: %w", err)
		}
		if err := h.svc.DeleteShareChange(ctx, changeID); err != nil {
			return fmt.Errorf("deleting share change: %w", err)
		}
		shttp.RedirectToNext(w, r, fmt.Sprintf("/accounts/%s/edit", sc.AccountID))
		return nil
	})
}

func (h *Handler) startupShareOptionUpsert() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var inp startupShareOptionInputForm
		if err := srvu.Decode(r, &inp, false); err != nil {
			return fmt.Errorf("decoding input: %w", err)
		}
		_, err := h.svc.UpsertStartupShareOption(ctx, inp.StartupShareOptionInput)
		if err != nil {
			return fmt.Errorf("upserting startup share option: %w", err)
		}
		shttp.RedirectToNext(w, r, fmt.Sprintf("/accounts/%s/edit", inp.AccountID))
		return nil
	})
}

func (h *Handler) startupShareOptionDelete() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		optionID := r.PathValue("id")
		option, err := h.svc.GetStartupShareOption(ctx, optionID)
		if err != nil {
			return fmt.Errorf("getting startup share option: %w", err)
		}
		if err := h.svc.DeleteStartupShareOption(ctx, optionID); err != nil {
			return fmt.Errorf("deleting startup share option: %w", err)
		}
		shttp.RedirectToNext(w, r, fmt.Sprintf("/accounts/%s/edit", option.AccountID))
		return nil
	})
}

// ---- Salaries ----

func (h *Handler) salariesPage() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		salaries, err := h.svc.GetSalariesPageData(ctx)
		if err != nil {
			return fmt.Errorf("getting salaries page data: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Salaries", PageSalaries(SalariesListView(salaries))))
	})
}

func (h *Handler) salaryNewPage() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		view, err := h.svc.GetSalaryNewPageData(ctx)
		if err != nil {
			return fmt.Errorf("getting salary new page data: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Salaries", PageEditSalary(SalaryEditContent(view))))
	})
}

func (h *Handler) salaryEditPage() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		view, err := h.svc.GetSalaryEditPageData(ctx, r.PathValue("id"))
		if err != nil {
			return fmt.Errorf("getting salary edit page data: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Salaries", PageEditSalary(SalaryEditContent(view))))
	})
}

func (h *Handler) salaryUpsert() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var inp salaryInputForm
		if err := srvu.Decode(r, &inp, false); err != nil {
			return fmt.Errorf("decoding input: %w", err)
		}
		sal, err := h.svc.UpsertSalary(ctx, inp.Salary)
		if err != nil {
			return fmt.Errorf("upserting salary: %w", err)
		}
		shttp.RedirectToNext(w, r, fmt.Sprintf("/salaries/%s/edit", sal.ID))
		return nil
	})
}

func (h *Handler) salaryDelete() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if err := h.svc.DeleteSalary(ctx, r.PathValue("id")); err != nil {
			return fmt.Errorf("deleting salary: %w", err)
		}
		shttp.RedirectToNext(w, r, "/salaries")
		return nil
	})
}

func (h *Handler) salaryAmountUpsert() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var inp salaryAmountInputForm
		if err := srvu.Decode(r, &inp, false); err != nil {
			return fmt.Errorf("decoding input: %w", err)
		}
		_, err := h.svc.UpsertSalaryAmount(ctx, inp.SalaryAmount)
		if err != nil {
			return fmt.Errorf("upserting salary amount: %w", err)
		}
		shttp.RedirectToNext(w, r, fmt.Sprintf("/salaries/%s/edit", inp.SalaryID))
		return nil
	})
}

func (h *Handler) salaryAmountDelete() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		amountID := r.PathValue("id")
		if err := h.svc.DeleteSalaryAmount(ctx, amountID); err != nil {
			return fmt.Errorf("deleting salary amount: %w", err)
		}
		shttp.RedirectToNext(w, r, "/salaries")
		return nil
	})
}

// ---- Transfer Templates ----

func (h *Handler) transferTemplatesPage() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		view, err := h.svc.GetTransferTemplatesPageData(ctx)
		if err != nil {
			return fmt.Errorf("getting transfer templates page data: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Transfer Templates", PageTransferTemplates(view)))
	})
}

func (h *Handler) transferTemplatesNewPage() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		view, err := h.svc.GetTransferTemplateNewPageData(ctx)
		if err != nil {
			return fmt.Errorf("getting transfer template new page data: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Transfer Templates", PageEditTransferTemplate(view)))
	})
}

func (h *Handler) transferTemplatesEditPage() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		view, err := h.svc.GetTransferTemplateEditPageData(ctx, r.PathValue("id"))
		if err != nil {
			return fmt.Errorf("getting transfer template edit page data: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Transfer Templates", PageEditTransferTemplate(view)))
	})
}

func (h *Handler) transferTemplateUpsert() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if err := r.ParseForm(); err != nil {
			return fmt.Errorf("parsing form: %w", err)
		}
		var inp transferTemplateForm
		if err := srvu.Decode(r, &inp, false); err != nil {
			return fmt.Errorf("decoding input: %w", err)
		}
		t, err := h.svc.UpsertTransferTemplate(ctx, inp.TransferTemplate)
		if err != nil {
			return fmt.Errorf("upserting transfer template: %w", err)
		}
		shttp.RedirectToNext(w, r, fmt.Sprintf("/transfer-templates/%s/edit", t.ID))
		return nil
	})
}

func (h *Handler) transferTemplateDuplicate() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		tr, err := h.svc.DuplicateTransferTemplate(ctx, r.PathValue("id"))
		if err != nil {
			return fmt.Errorf("duplicating transfer template: %w", err)
		}
		shttp.RedirectToNext(w, r, fmt.Sprintf("/transfer-templates/%s/edit", tr.ID))
		return nil
	})
}

func (h *Handler) transferTemplateDelete() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if err := h.svc.DeleteTransferTemplate(ctx, r.PathValue("id")); err != nil {
			return fmt.Errorf("deleting transfer template: %w", err)
		}
		shttp.RedirectToNext(w, r, "/")
		return nil
	})
}

// ---- Categories (combined) ----

func (h *Handler) categoriesPage() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		view, err := h.svc.GetCategoriesPageData(ctx)
		if err != nil {
			return fmt.Errorf("getting categories page data: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Categories", PageCategories(view)))
	})
}

// ---- Transfer Template Categories ----

func (h *Handler) transferTemplateCategoryNewPage() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return NewView(ctx, w, r).Render(Page("Transfer Template Categories", PageEditTransferTemplateCategory(&TransferTemplateCategoryEditView{})))
	})
}

func (h *Handler) transferTemplateCategoryEditPage() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		c, err := h.svc.GetCategory(ctx, r.PathValue("id"))
		if err != nil {
			return fmt.Errorf("getting category: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Transfer Template Categories", PageEditTransferTemplateCategory(&TransferTemplateCategoryEditView{
			Category: c,
		})))
	})
}

func (h *Handler) transferTemplateCategoryUpsert() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var inp transferTemplateCategoryInputForm
		if err := srvu.Decode(r, &inp, false); err != nil {
			return fmt.Errorf("decoding input: %w", err)
		}
		if _, err := h.svc.UpsertCategory(ctx, inp.TransferTemplateCategoryInput); err != nil {
			return fmt.Errorf("upserting category: %w", err)
		}
		shttp.RedirectToNext(w, r, "/categories")
		return nil
	})
}

func (h *Handler) transferTemplateCategoryDelete() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if err := h.svc.DeleteCategory(ctx, r.PathValue("id")); err != nil {
			return fmt.Errorf("deleting category: %w", err)
		}
		shttp.RedirectToNext(w, r, "/categories")
		return nil
	})
}

// ---- Account Types ----

func (h *Handler) accountTypeNewPage() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return NewView(ctx, w, r).Render(Page("Account Types", PageEditAccountType(AccountTypeEditView(AccountType{}))))
	})
}

func (h *Handler) accountTypeEditPage() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		at, err := h.svc.GetAccountType(ctx, r.PathValue("id"))
		if err != nil {
			return fmt.Errorf("getting account type: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Account Types", PageEditAccountType(AccountTypeEditView(at))))
	})
}

func (h *Handler) accountTypeUpsert() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var inp accountTypeInputForm
		if err := srvu.Decode(r, &inp, false); err != nil {
			return fmt.Errorf("decoding input: %w", err)
		}
		_, err := h.svc.UpsertAccountType(ctx, inp.AccountTypeInput)
		if err != nil {
			return fmt.Errorf("upserting account type: %w", err)
		}
		shttp.RedirectToNext(w, r, "/categories")
		return nil
	})
}

func (h *Handler) accountTypeDelete() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if err := h.svc.DeleteAccountType(ctx, r.PathValue("id")); err != nil {
			return fmt.Errorf("deleting account type: %w", err)
		}
		shttp.RedirectToNext(w, r, "/categories")
		return nil
	})
}

// ---- Special Dates ----

func (h *Handler) specialDatesPage() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		specialDates, err := h.svc.ListSpecialDates(ctx)
		if err != nil {
			return fmt.Errorf("listing special dates: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Special Dates", PageSpecialDates(SpecialDatesView(specialDates))))
	})
}

func (h *Handler) specialDateNewPage() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return NewView(ctx, w, r).Render(Page("Special Dates", PageEditSpecialDate(SpecialDateEditView(SpecialDate{}))))
	})
}

func (h *Handler) specialDateEditPage() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		sd, err := h.svc.GetSpecialDate(ctx, r.PathValue("id"))
		if err != nil {
			return fmt.Errorf("getting special date: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Special Dates", PageEditSpecialDate(SpecialDateEditView(sd))))
	})
}

func (h *Handler) specialDateUpsert() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var inp specialDateInputForm
		if err := srvu.Decode(r, &inp, false); err != nil {
			return fmt.Errorf("decoding input: %w", err)
		}
		_, err := h.svc.UpsertSpecialDate(ctx, inp.SpecialDateInput)
		if err != nil {
			return fmt.Errorf("upserting special date: %w", err)
		}
		shttp.RedirectToNext(w, r, "/special-dates")
		return nil
	})
}

func (h *Handler) specialDateDelete() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if err := h.svc.DeleteSpecialDate(ctx, r.PathValue("id")); err != nil {
			return fmt.Errorf("deleting special date: %w", err)
		}
		shttp.RedirectToNext(w, r, "/special-dates")
		return nil
	})
}

// ---- Snapshots Table ----

func (h *Handler) snapshotsTablePage() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		view, err := h.svc.GetSnapshotsTablePageData(ctx, extractExcludedTypeIDs(r))
		if err != nil {
			return fmt.Errorf("getting snapshots table page data: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Snapshots Table", PageSnapshotsTable(view)))
	})
}

func (h *Handler) snapshotsTableModifyDate() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var inp dateInputForm
		if err := srvu.Decode(r, &inp, false); err != nil {
			return fmt.Errorf("decoding input: %w", err)
		}
		row, err := h.svc.ModifySnapshotDateRow(ctx, inp.OldDate, inp.NewDate)
		if err != nil {
			return fmt.Errorf("modifying snapshot date: %w", err)
		}
		return NewView(ctx, w, r).Render(SnapshotsTableRow(row))
	})
}

func (h *Handler) snapshotsTableEmptyRow() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		row, err := h.svc.GetEmptySnapshotRow(ctx)
		if err != nil {
			return fmt.Errorf("getting empty snapshot row: %w", err)
		}
		return NewView(ctx, w, r).Render(SnapshotsTableRow(row))
	})
}

func (h *Handler) accountSnapshotUpsert() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var inp accountSnapshotInputForm
		if err := srvu.Decode(r, &inp, false); err != nil {
			return fmt.Errorf("decoding input: %w", err)
		}
		accID := r.PathValue("id")
		snap, err := h.svc.UpsertOrDeleteSnapshot(ctx, accID, inp.AccountSnapshotInput)
		if err != nil {
			return fmt.Errorf("upserting snapshot: %w", err)
		}
		if r.Header.Get("HX-Request") == "true" {
			return NewView(ctx, w, r).Render(SnapshotCell(accID, inp.Date, snap))
		}
		shttp.RedirectToNext(w, r, fmt.Sprintf("/accounts/%s", accID))
		return nil
	})
}

// ---- Transfers ----

func (h *Handler) transfersPage() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var day date.Date
		if err := shttp.Parse(&day, date.ParseDate, r.FormValue("day"), date.Today()); err != nil {
			return fmt.Errorf("parsing day: %w", err)
		}
		amounts := make(map[string]float64)
		if err := r.ParseForm(); err == nil {
			for key, values := range r.Form {
				if len(key) > 7 && key[:7] == "amount_" {
					templateID := key[7:]
					if amount, err := ui.ParseAmount(values[0]); err == nil {
						amounts[templateID] = amount
					}
				}
			}
		}
		view, err := h.svc.ComputeTransfersView(ctx, day, amounts)
		if err != nil {
			return fmt.Errorf("computing transfers view: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Transfers", PageTransfers(view)))
	})
}

func (h *Handler) transferChartPage() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var groupBy TransferChartGroupBy
		if err := shttp.Parse(&groupBy, service.ParseTransferChartGroupBy, r.FormValue("group_by"), service.GroupByAccount); err != nil {
			return fmt.Errorf("parsing group_by: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Transfers Chart", PageTransfersChart(groupBy)))
	})
}

func (h *Handler) transferChartData() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var groupBy TransferChartGroupBy
		if err := shttp.Parse(&groupBy, service.ParseTransferChartGroupBy, r.FormValue("group_by"), service.GroupByAccount); err != nil {
			return fmt.Errorf("parsing group_by: %w", err)
		}
		data, err := h.svc.GetTransferChartData(ctx, groupBy)
		if err != nil {
			return fmt.Errorf("getting transfer chart data: %w", err)
		}
		if err := json.NewEncoder(w).Encode(data); err != nil {
			return fmt.Errorf("encoding JSON: %w", err)
		}
		return nil
	})
}

// ---- Chart ----

func (h *Handler) chartPage() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var p predictionParamsForm
		if err := srvu.Decode(r, &p, false); err != nil {
			return fmt.Errorf("decoding input: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Chart", PageChart(p.PredictionParams)))
	})
}

type ssePredictionEventHandler struct {
	w *srvu.SSESender
}

func (s *ssePredictionEventHandler) Setup(e service.PredictionSetupEvent) error {
	return s.w.SendNamedJson("setup", e)
}
func (s *ssePredictionEventHandler) Snapshot(e service.PredictionBalanceSnapshot) error {
	return s.w.SendNamedJson("balanceSnapshot", e)
}
func (s *ssePredictionEventHandler) Close() error {
	return s.w.SendEventWithoutData("close")
}

func (h *Handler) chartsDataStream() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var params predictionParamsForm
		if err := srvu.Decode(r, &params, false); err != nil {
			return fmt.Errorf("decoding input: %w", err)
		}
		if err := h.svc.RunPrediction(ctx, &ssePredictionEventHandler{w: srvu.SSEResponse(w)}, params.PredictionParams); err != nil {
			return fmt.Errorf("running prediction: %w", err)
		}
		return nil
	})
}
