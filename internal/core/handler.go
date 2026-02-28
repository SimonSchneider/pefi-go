package core

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"time"

	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/goslu/srvu"
	"github.com/SimonSchneider/goslu/static/shttp"
)

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
		// Handle startup share account configuration
		enableStartupShares := r.FormValue("enable_startup_shares") == "on"
		if enableStartupShares {
			// User wants to enable/update startup shares
			var ssaInp StartupShareAccountInput
			if err := srvu.Decode(r, &ssaInp, false); err == nil {
				ssaInp.AccountID = acc.ID
				// Convert tax rate and discount factor from percentage to decimal
				ssaInp.TaxRate = ssaInp.TaxRate / 100.0
				ssaInp.ValuationDiscountFactor = ssaInp.ValuationDiscountFactor / 100.0
				_, err := UpsertStartupShareAccount(ctx, db, ssaInp)
				if err != nil {
					return fmt.Errorf("upserting startup share account: %w", err)
				}
			}
		} else {
			// User unchecked the box, delete startup share account if it exists
			_, err := GetStartupShareAccount(ctx, db, acc.ID)
			if err == nil {
				// Startup share account exists, delete it
				if err := DeleteStartupShareAccount(ctx, db, acc.ID); err != nil {
					return fmt.Errorf("deleting startup share account: %w", err)
				}
			}
			// If it doesn't exist (sql.ErrNoRows), that's fine, nothing to do
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

func HandlerAccountGrowthModelUpsert(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var inp AccountGrowthModelInput
		if err := srvu.Decode(r, &inp, false); err != nil {
			return fmt.Errorf("decoding input: %w", err)
		}
		_, err := UpsertAccountGrowthModel(ctx, db, inp)
		if err != nil {
			return fmt.Errorf("upserting account growth model: %w", err)
		}
		shttp.RedirectToNext(w, r, fmt.Sprintf("/accounts/%s", inp.AccountID))
		return nil
	})
}

func HandlerAccountGrowthModelDelete(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if err := DeleteAccountGrowthModel(ctx, db, r.PathValue("id")); err != nil {
			return fmt.Errorf("deleting account growth model: %w", err)
		}
		shttp.RedirectToNext(w, r, fmt.Sprintf("/accounts/%s", r.PathValue("id")))
		return nil
	})
}

func HandlerStartupShareAccountUpsert(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var inp StartupShareAccountInput
		if err := srvu.Decode(r, &inp, false); err != nil {
			return fmt.Errorf("decoding input: %w", err)
		}
		// Convert tax rate and discount factor from percentage to decimal
		inp.TaxRate = inp.TaxRate / 100.0
		inp.ValuationDiscountFactor = inp.ValuationDiscountFactor / 100.0
		_, err := UpsertStartupShareAccount(ctx, db, inp)
		if err != nil {
			return fmt.Errorf("upserting startup share account: %w", err)
		}
		shttp.RedirectToNext(w, r, fmt.Sprintf("/accounts/%s/edit", inp.AccountID))
		return nil
	})
}

func HandlerInvestmentRoundUpsert(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var inp InvestmentRoundInput
		if err := srvu.Decode(r, &inp, false); err != nil {
			return fmt.Errorf("decoding input: %w", err)
		}
		_, err := UpsertInvestmentRound(ctx, db, inp)
		if err != nil {
			return fmt.Errorf("upserting investment round: %w", err)
		}
		shttp.RedirectToNext(w, r, fmt.Sprintf("/accounts/%s/edit", inp.AccountID))
		return nil
	})
}

func HandlerInvestmentRoundDelete(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		roundID := r.PathValue("id")
		round, err := GetInvestmentRound(ctx, db, roundID)
		if err != nil {
			return fmt.Errorf("getting investment round: %w", err)
		}
		if err := DeleteInvestmentRound(ctx, db, roundID); err != nil {
			return fmt.Errorf("deleting investment round: %w", err)
		}
		shttp.RedirectToNext(w, r, fmt.Sprintf("/accounts/%s/edit", round.AccountID))
		return nil
	})
}

func HandlerStartupShareOptionUpsert(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var inp StartupShareOptionInput
		if err := srvu.Decode(r, &inp, false); err != nil {
			return fmt.Errorf("decoding input: %w", err)
		}
		_, err := UpsertStartupShareOption(ctx, db, inp)
		if err != nil {
			return fmt.Errorf("upserting startup share option: %w", err)
		}
		shttp.RedirectToNext(w, r, fmt.Sprintf("/accounts/%s/edit", inp.AccountID))
		return nil
	})
}

func HandlerStartupShareOptionDelete(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		optionID := r.PathValue("id")
		option, err := GetStartupShareOption(ctx, db, optionID)
		if err != nil {
			return fmt.Errorf("getting startup share option: %w", err)
		}
		if err := DeleteStartupShareOption(ctx, db, optionID); err != nil {
			return fmt.Errorf("deleting startup share option: %w", err)
		}
		shttp.RedirectToNext(w, r, fmt.Sprintf("/accounts/%s/edit", option.AccountID))
		return nil
	})
}

func HandlerTransferTemplateUpsert(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if err := r.ParseForm(); err != nil {
			return fmt.Errorf("parsing form: %w", err)
		}
		var inp TransferTemplate
		if err := srvu.Decode(r, &inp, false); err != nil {
			return fmt.Errorf("decoding input: %w", err)
		}
		t, err := UpsertTransferTemplate(ctx, db, inp)
		if err != nil {
			return fmt.Errorf("upserting transfer template: %w", err)
		}

		shttp.RedirectToNext(w, r, fmt.Sprintf("/transfers/%s", t.ID))
		return nil
	})
}

func HandlerTransferTemplateDuplicate(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		tr, err := DuplicateTransferTemplate(ctx, db, r.PathValue("id"))
		if err != nil {
			return fmt.Errorf("duplicating transfer template: %w", err)
		}
		shttp.RedirectToNext(w, r, fmt.Sprintf("/transfer-templates/%s/edit", tr.ID))
		return nil
	})
}

func HandlerTransferTemplateDelete(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if err := DeleteTransferTemplate(ctx, db, r.PathValue("id")); err != nil {
			return fmt.Errorf("deleting transfer template: %w", err)
		}
		shttp.RedirectToNext(w, r, "/")
		return nil
	})
}

// Transfer Template Category Handlers
func HandlerTransferTemplateCategoryUpsert(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var inp TransferTemplateCategory
		if err := srvu.Decode(r, &inp, false); err != nil {
			return fmt.Errorf("decoding input: %w", err)
		}
		c, err := UpsertCategory(ctx, db, inp)
		if err != nil {
			return fmt.Errorf("upserting category: %w", err)
		}
		shttp.RedirectToNext(w, r, fmt.Sprintf("/transfer-template-categories/%s", c.ID))
		return nil
	})
}

func HandlerTransferTemplateCategoryDelete(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if err := DeleteCategory(ctx, db, r.PathValue("id")); err != nil {
			return fmt.Errorf("deleting category: %w", err)
		}
		shttp.RedirectToNext(w, r, "/transfer-template-categories")
		return nil
	})
}

func TransferTemplateCategoriesPage(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		categories, err := ListCategories(ctx, db)
		if err != nil {
			return fmt.Errorf("listing categories: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Transfer Template Categories", PageTransferTemplateCategories(NewTransferTemplateCategoriesView(categories))))
	})
}

func TransferTemplateCategoryNewPage(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return NewView(ctx, w, r).Render(Page("Transfer Template Categories", PageEditTransferTemplateCategory(&TransferTemplateCategoryEditView{})))
	})
}

func TransferTemplateCategoryEditPage(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		c, err := GetCategory(ctx, db, r.PathValue("id"))
		if err != nil {
			return fmt.Errorf("getting category: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Transfer Template Categories", PageEditTransferTemplateCategory(&TransferTemplateCategoryEditView{
			Category: c,
		})))
	})
}

type AccountTypesWithFilter []AccountTypeWithFilter

func (a AccountTypesWithFilter) GetAccountType(typeID string) AccountTypeWithFilter {
	if typeID == "" {
		return AccountTypeWithFilter{}
	}
	for _, at := range a {
		if at.ID == typeID {
			return at
		}
	}
	return AccountTypeWithFilter{}
}

func getAccountTypesWithFilter(r *http.Request, accountTypes []AccountType) AccountTypesWithFilter {
	accountTypesWithFilter := make(AccountTypesWithFilter, 0, len(accountTypes))
	for _, accountType := range accountTypes {
		exclude := r.FormValue("exclude_at_"+accountType.ID) == "on"
		accountTypesWithFilter = append(accountTypesWithFilter, AccountTypeWithFilter{AccountType: accountType, Exclude: exclude})
	}
	return accountTypesWithFilter
}

func AccountsPage(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		accs, err := ListAccountsDetailed(ctx, db, date.Today())
		if err != nil {
			return fmt.Errorf("listing accounts: %w", err)
		}
		accountTypes, err := ListAccountTypes(ctx, db)
		if err != nil {
			return fmt.Errorf("listing account types: %w", err)
		}
		accountTypesWithFilter := getAccountTypesWithFilter(r, accountTypes)
		categories, err := ListCategories(ctx, db)
		if err != nil {
			return fmt.Errorf("listing categories: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Accounts", PageAccounts(NewAccountsView(accs, accountTypesWithFilter, categories))))
	})
}

func TransferTemplatesPage(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		transferTemplates, err := ListTransferTemplatesWithChildren(ctx, db)
		if err != nil {
			return fmt.Errorf("listing transfer templates: %w", err)
		}
		accounts, err := ListAccounts(ctx, db)
		if err != nil {
			return fmt.Errorf("listing accounts: %w", err)
		}
		categories, err := ListCategories(ctx, db)
		if err != nil {
			return fmt.Errorf("listing categories: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Transfer Templates", PageTransferTemplates(NewTransferTemplatesView2(transferTemplates, accounts, categories))))
	})
}

func AccountNewPage(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		accs, err := ListAccounts(ctx, db)
		if err != nil {
			return fmt.Errorf("listing accounts: %w", err)
		}
		accountTypes, err := ListAccountTypes(ctx, db)
		if err != nil {
			return fmt.Errorf("listing account types: %w", err)
		}
		accountTypesWithFilter := getAccountTypesWithFilter(r, accountTypes)
		categories, err := ListCategories(ctx, db)
		if err != nil {
			return fmt.Errorf("listing categories: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Accounts", PageEditAccount(NewAccountEditView2(Account{}, accs, nil, accountTypesWithFilter, categories, nil, nil, nil))))
	})
}

func AccountEditPage(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		accs, err := ListAccounts(ctx, db)
		if err != nil {
			return fmt.Errorf("listing accounts: %w", err)
		}
		acc, err := GetAccount(ctx, db, r.PathValue("id"))
		if err != nil {
			return fmt.Errorf("getting account: %w", err)
		}
		growthModels, err := ListAccountGrowthModels(ctx, db, string(acc.ID))
		if err != nil {
			return fmt.Errorf("listing account growth models: %w", err)
		}
		accountTypes, err := ListAccountTypes(ctx, db)
		if err != nil {
			return fmt.Errorf("listing account types: %w", err)
		}
		accountTypesWithFilter := getAccountTypesWithFilter(r, accountTypes)

		// Load startup share data if account has startup share configuration
		var startupShareAccount *StartupShareAccount
		var investmentRounds []InvestmentRound
		var options []StartupShareOption
		ssa, err := GetStartupShareAccount(ctx, db, acc.ID)
		if err == nil {
			startupShareAccount = &ssa
			investmentRounds, err = ListInvestmentRounds(ctx, db, acc.ID)
			if err != nil {
				return fmt.Errorf("listing investment rounds: %w", err)
			}
			options, err = ListStartupShareOptions(ctx, db, acc.ID)
			if err != nil {
				return fmt.Errorf("listing startup share options: %w", err)
			}
		} else if errors.Is(err, sql.ErrNoRows) {
			// Account doesn't have startup share configuration, which is fine
			// Leave startupShareAccount, investmentRounds, and options as nil/empty
		} else {
			return fmt.Errorf("getting startup share account: %w", err)
		}

		categories, err := ListCategories(ctx, db)
		if err != nil {
			return fmt.Errorf("listing categories: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Accounts", PageEditAccount(NewAccountEditView2(acc, accs, growthModels, accountTypesWithFilter, categories, startupShareAccount, investmentRounds, options))))
	})
}

func TransferTemplatesNewPage(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		accs, err := ListAccounts(ctx, db)
		if err != nil {
			return fmt.Errorf("listing accounts: %w", err)
		}
		categories, err := ListCategories(ctx, db)
		if err != nil {
			return fmt.Errorf("listing categories: %w", err)
		}
		allTemplates, err := ListTransferTemplatesWithChildren(ctx, db)
		if err != nil {
			return fmt.Errorf("listing templates: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Transfer Templates", PageEditTransferTemplate(&TransferTemplateEditView{
			Accounts:     accs,
			Categories:   categories,
			AllTemplates: allTemplates,
		})))
	})
}

func TransferTemplatesEditPage(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		t, err := GetTransferTemplate(ctx, db, id)
		if err != nil {
			return fmt.Errorf("getting transfer template: %w", err)
		}
		accs, err := ListAccounts(ctx, db)
		if err != nil {
			return fmt.Errorf("listing accounts: %w", err)
		}
		categories, err := ListCategories(ctx, db)
		if err != nil {
			return fmt.Errorf("listing categories: %w", err)
		}
		allTemplates, err := ListTransferTemplatesWithChildren(ctx, db)
		if err != nil {
			return fmt.Errorf("listing templates: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Transfer Templates", PageEditTransferTemplate(&TransferTemplateEditView{
			Accounts:         accs,
			TransferTemplate: t,
			Categories:       categories,
			AllTemplates:     allTemplates,
		})))
	})
}

func NewHandler(db *sql.DB, public fs.FS) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /static/public/", srvu.With(http.StripPrefix("/static/public/", http.FileServerFS(public)), srvu.WithCacheCtrlHeader(365*24*time.Hour)))

	mux.Handle("GET /{$}", DashboardPage(db))
	mux.Handle("GET /accounts", AccountsPage(db))
	mux.Handle("GET /accounts/new", AccountNewPage(db))
	mux.Handle("GET /accounts/{id}/edit", AccountEditPage(db))
	mux.Handle("GET /transfer-templates", TransferTemplatesPage(db))
	mux.Handle("GET /transfer-templates/new", TransferTemplatesNewPage(db))
	mux.Handle("GET /transfer-templates/{id}/edit", TransferTemplatesEditPage(db))

	// Account types
	mux.Handle("GET /account-types", AccountTypesPage(db))
	mux.Handle("GET /account-types/new", AccountTypeNewPage(db))
	mux.Handle("GET /account-types/{id}/edit", AccountTypeEditPage(db))
	mux.Handle("POST /account-types/{$}", HandlerAccountTypeUpsert(db))
	mux.Handle("POST /account-types/{id}/delete", HandlerAccountTypeDelete(db))

	// Special dates
	mux.Handle("GET /special-dates", SpecialDatesPage(db))
	mux.Handle("GET /special-dates/new", SpecialDateNewPage(db))
	mux.Handle("GET /special-dates/{id}/edit", SpecialDateEditPage(db))
	mux.Handle("POST /special-dates/{$}", HandlerSpecialDateUpsert(db))
	mux.Handle("POST /special-dates/{id}/delete", HandlerSpecialDateDelete(db))

	// Snapshots table
	mux.Handle("GET /snapshots-table", SnapshotsTablePage(db))
	mux.Handle("POST /snapshots-table/modify-date", SnapshotsTableModifyDate(db))
	mux.Handle("GET /snapshots-table/empty-row", SnapshotsTableEmptyRow(db))
	mux.Handle("POST /accounts/{id}/snapshots/{date}/", HandlerAccountSnapshotUpsert(db))

	// Transfers
	mux.Handle("GET /transfers", TransfersPage(db))
	mux.Handle("GET /transfers/chart/{$}", TransferChartPage(db))
	mux.Handle("GET /transfers/chart/data", TransferChartData(db))

	// Budget
	mux.Handle("GET /budget", BudgetPage(db))

	// Chart
	mux.Handle("GET /chart", ChartPage())
	mux.Handle("GET /chart/stream", HandlerChartsDataStream(db))

	mux.Handle("POST /accounts/{$}", HandlerAccountUpsert(db))
	mux.Handle("POST /accounts/{id}/delete", HandlerAccountDelete(db))

	mux.Handle("POST /growth-models/", HandlerAccountGrowthModelUpsert(db))
	mux.Handle("POST /growth-models/{id}/delete", HandlerAccountGrowthModelDelete(db))

	// Startup shares
	mux.Handle("POST /startup-share-accounts/", HandlerStartupShareAccountUpsert(db))
	mux.Handle("POST /investment-rounds/", HandlerInvestmentRoundUpsert(db))
	mux.Handle("POST /investment-rounds/{id}/delete", HandlerInvestmentRoundDelete(db))
	mux.Handle("POST /startup-share-options/", HandlerStartupShareOptionUpsert(db))
	mux.Handle("POST /startup-share-options/{id}/delete", HandlerStartupShareOptionDelete(db))

	mux.Handle("POST /transfers/{$}", HandlerTransferTemplateUpsert(db))
	mux.Handle("POST /transfers/{id}/duplicate", HandlerTransferTemplateDuplicate(db))
	mux.Handle("POST /transfers/{id}/delete", HandlerTransferTemplateDelete(db))

	// Transfer template categories
	mux.Handle("GET /transfer-template-categories", TransferTemplateCategoriesPage(db))
	mux.Handle("GET /transfer-template-categories/new", TransferTemplateCategoryNewPage(db))
	mux.Handle("GET /transfer-template-categories/{id}/edit", TransferTemplateCategoryEditPage(db))
	mux.Handle("POST /transfer-template-categories/{$}", HandlerTransferTemplateCategoryUpsert(db))
	mux.Handle("POST /transfer-template-categories/{id}/delete", HandlerTransferTemplateCategoryDelete(db))

	mux.Handle("POST /sleep/{$}", srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Simulate a long-running operation
		time.Sleep(1 * time.Second)
		w.WriteHeader(200)
		return nil
	}))

	return mux
}
