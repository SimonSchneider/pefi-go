package core

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"sort"

	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/goslu/srvu"
)

type AccountTypeGroup struct {
	AccountType  AccountType
	Accounts     []AccountDetailed
	TotalBalance float64
}

type AccountTypeChartEntry struct {
	Name     string              `json:"name"`
	Color    string              `json:"color"`
	Accounts []AccountChartEntry `json:"accounts"`
}

type AccountChartEntry struct {
	Name    string  `json:"name"`
	Balance float64 `json:"balance"`
}

type DashboardView struct {
	TotalBalance      float64
	TotalAssets       float64
	TotalLiabilities  float64
	Budget            *BudgetView
	AccountTypeGroups []AccountTypeGroup
	AccountChartData  []AccountTypeChartEntry
}

func DashboardPage(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		view, err := computeDashboardView(ctx, db)
		if err != nil {
			return fmt.Errorf("computing dashboard view: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Dashboard", PageDashboard(view)))
	})
}

func computeDashboardView(ctx context.Context, db *sql.DB) (*DashboardView, error) {
	today := date.Today()

	budget, err := computeBudgetView(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("computing budget view: %w", err)
	}

	accounts, err := ListAccountsDetailed(ctx, db, today)
	if err != nil {
		return nil, fmt.Errorf("listing accounts: %w", err)
	}

	accountTypes, err := ListAccountTypes(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("listing account types: %w", err)
	}

	var totalBalance, totalAssets, totalLiabilities float64
	for _, acc := range accounts {
		if acc.LastSnapshot != nil {
			bal := acc.LastSnapshot.Balance.Mean()
			totalBalance += bal
			if bal > 0 {
				totalAssets += bal
			} else {
				totalLiabilities += bal
			}
		}
	}

	groups := groupAccountsByType(accounts, accountTypes)
	chartData := buildAccountChartData(groups)

	return &DashboardView{
		TotalBalance:      totalBalance,
		TotalAssets:       totalAssets,
		TotalLiabilities:  totalLiabilities,
		Budget:            budget,
		AccountTypeGroups: groups,
		AccountChartData:  chartData,
	}, nil
}

func groupAccountsByType(accounts []AccountDetailed, accountTypes []AccountType) []AccountTypeGroup {
	typeMap := make(map[string]*AccountTypeGroup)
	for _, at := range accountTypes {
		typeMap[at.ID] = &AccountTypeGroup{
			AccountType: at,
		}
	}

	for _, acc := range accounts {
		group, exists := typeMap[acc.TypeID]
		if !exists {
			continue
		}
		group.Accounts = append(group.Accounts, acc)
		if acc.LastSnapshot != nil {
			group.TotalBalance += acc.LastSnapshot.Balance.Mean()
		}
	}

	groups := make([]AccountTypeGroup, 0, len(typeMap))
	for _, group := range typeMap {
		if len(group.Accounts) > 0 {
			groups = append(groups, *group)
		}
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].TotalBalance > groups[j].TotalBalance
	})
	return groups
}

func buildAccountChartData(groups []AccountTypeGroup) []AccountTypeChartEntry {
	entries := make([]AccountTypeChartEntry, 0, len(groups))
	for _, group := range groups {
		if group.TotalBalance == 0 {
			continue
		}
		color := group.AccountType.Color
		if color == "" {
			color = "#999999"
		}
		accs := make([]AccountChartEntry, 0, len(group.Accounts))
		for _, acc := range group.Accounts {
			bal := 0.0
			if acc.LastSnapshot != nil {
				bal = math.Round(acc.LastSnapshot.Balance.Mean()*100) / 100
			}
			accs = append(accs, AccountChartEntry{
				Name:    acc.Name,
				Balance: bal,
			})
		}
		entries = append(entries, AccountTypeChartEntry{
			Name:     group.AccountType.Name,
			Color:    color,
			Accounts: accs,
		})
	}
	return entries
}
