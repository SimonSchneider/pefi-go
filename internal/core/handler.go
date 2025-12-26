package core

import (
	"context"
	"database/sql"
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

		// Handle category assignments
		// Get existing categories
		existingCategories, _ := GetCategoriesForTemplate(ctx, db, t.ID)
		existingCategoryMap := make(map[string]bool)
		for _, c := range existingCategories {
			existingCategoryMap[c.ID] = true
		}

		// Parse category IDs from form (multiple values with same name)
		categoryIDs := r.Form["category_ids"]
		if len(categoryIDs) == 0 {
			// Try single value
			if catID := r.FormValue("category_ids"); catID != "" {
				categoryIDs = []string{catID}
			}
		}

		newCategoryMap := make(map[string]bool)
		for _, catID := range categoryIDs {
			if catID != "" {
				newCategoryMap[catID] = true
				if !existingCategoryMap[catID] {
					// Assign new category
					if err := AssignCategoryToTemplate(ctx, db, t.ID, catID); err != nil {
						return fmt.Errorf("assigning category: %w", err)
					}
				}
			}
		}

		// Remove categories that are no longer assigned
		for _, existingCat := range existingCategories {
			if !newCategoryMap[existingCat.ID] {
				if err := RemoveCategoryFromTemplate(ctx, db, t.ID, existingCat.ID); err != nil {
					return fmt.Errorf("removing category: %w", err)
				}
			}
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

func RootPage() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return NewView(ctx, w, r).Render(Page("App", App()))
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
		return NewView(ctx, w, r).Render(Page("Accounts", PageAccounts(NewAccountsView(accs, accountTypesWithFilter))))
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
		return NewView(ctx, w, r).Render(Page("Transfer Templates", PageTransferTemplates(NewTransferTemplatesView2(transferTemplates, accounts))))
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
		return NewView(ctx, w, r).Render(Page("Accounts", PageEditAccount(NewAccountEditView2(Account{}, accs, nil, accountTypesWithFilter))))
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
		return NewView(ctx, w, r).Render(Page("Accounts", PageEditAccount(NewAccountEditView2(acc, accs, growthModels, accountTypesWithFilter))))
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

	mux.Handle("GET /{$}", RootPage())
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

	// Chart
	mux.Handle("GET /chart", ChartPage())
	mux.Handle("GET /chart/stream", HandlerChartsDataStream(db))

	mux.Handle("POST /accounts/{$}", HandlerAccountUpsert(db))
	mux.Handle("POST /accounts/{id}/delete", HandlerAccountDelete(db))

	mux.Handle("POST /growth-models/", HandlerAccountGrowthModelUpsert(db))
	mux.Handle("POST /growth-models/{id}/delete", HandlerAccountGrowthModelDelete(db))

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
