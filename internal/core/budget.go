package core

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"

	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/goslu/srvu"
)

type BudgetItem struct {
	Name   string
	Amount float64
	Source string // "transfer" or "interest"
}

type BudgetCategoryGroup struct {
	Category TransferTemplateCategory
	Items    []BudgetItem
	Total    float64
}

type BudgetChartEntry struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
	Color string  `json:"color"`
}

type BudgetView struct {
	Categories []BudgetCategoryGroup
	GrandTotal float64
	ChartData  []BudgetChartEntry
}

func BudgetPage(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		view, err := computeBudgetView(ctx, db)
		if err != nil {
			return fmt.Errorf("computing budget view: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Budget", PageBudget(view)))
	})
}

func computeBudgetView(ctx context.Context, db *sql.DB) (*BudgetView, error) {
	today := date.Today()

	// 1. Load all transfer templates and resolve amounts using the existing estimation logic
	allTemplates, err := ListTransferTemplates(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("listing transfer templates: %w", err)
	}
	templatesWithAmount := makeTransferTemplatesWithAmount(allTemplates, today)

	// 2. Get budget accounts with growth models for interest estimation
	budgetAccounts, err := ListBudgetAccounts(ctx, db, today)
	if err != nil {
		return nil, fmt.Errorf("listing budget accounts: %w", err)
	}

	// 3. Fetch all categories for lookup
	allCategories, err := ListCategories(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("listing categories: %w", err)
	}
	categoriesByID := make(map[string]TransferTemplateCategory)
	for _, c := range allCategories {
		categoriesByID[c.ID] = c
	}

	// 4. Group transfer items by budget_category_id
	groupMap := make(map[string]*BudgetCategoryGroup)

	for _, t := range templatesWithAmount {
		if !t.Enabled || t.BudgetCategoryID == nil {
			continue
		}
		// Only include recurring templates
		if !strings.Contains(string(t.Recurrence), "*") {
			continue
		}
		amount := math.Abs(t.Amount)
		if amount == 0 {
			continue
		}
		catID := *t.BudgetCategoryID
		group := getOrCreateGroup(groupMap, catID, categoriesByID)
		group.Items = append(group.Items, BudgetItem{
			Name:   t.Name,
			Amount: amount,
			Source: "transfer",
		})
		group.Total += amount
	}

	// 5. Add interest items from budget accounts
	for _, acc := range budgetAccounts {
		if acc.BudgetCategoryID == nil || acc.GrowthModel == nil {
			continue
		}
		// Estimate monthly interest: balance * annual_rate / 12
		balance := 0.0
		if acc.LastSnapshot != nil {
			balance = acc.LastSnapshot.Balance.Mean()
		}
		annualRate := acc.GrowthModel.AnnualRate.Mean()
		monthlyInterest := math.Abs(balance * annualRate / 12.0)
		if monthlyInterest == 0 {
			continue
		}
		catID := *acc.BudgetCategoryID
		group := getOrCreateGroup(groupMap, catID, categoriesByID)
		group.Items = append(group.Items, BudgetItem{
			Name:   acc.Name + " (interest)",
			Amount: monthlyInterest,
			Source: "interest",
		})
		group.Total += monthlyInterest
	}

	// 6. Convert to sorted slice
	categories := make([]BudgetCategoryGroup, 0, len(groupMap))
	grandTotal := 0.0
	for _, group := range groupMap {
		categories = append(categories, *group)
		grandTotal += group.Total
	}
	sort.Slice(categories, func(i, j int) bool {
		return categories[i].Total > categories[j].Total
	})

	// 7. Build chart data
	chartEntries := make([]BudgetChartEntry, 0, len(categories))
	for _, cat := range categories {
		color := "#999999"
		if cat.Category.Color != nil {
			color = *cat.Category.Color
		}
		chartEntries = append(chartEntries, BudgetChartEntry{
			Name:  cat.Category.Name,
			Value: math.Round(cat.Total*100) / 100,
			Color: color,
		})
	}
	return &BudgetView{
		Categories: categories,
		GrandTotal: grandTotal,
		ChartData:  chartEntries,
	}, nil
}

func getOrCreateGroup(groupMap map[string]*BudgetCategoryGroup, catID string, categoriesByID map[string]TransferTemplateCategory) *BudgetCategoryGroup {
	group, exists := groupMap[catID]
	if !exists {
		cat := categoriesByID[catID]
		group = &BudgetCategoryGroup{
			Category: cat,
		}
		groupMap[catID] = group
	}
	return group
}
