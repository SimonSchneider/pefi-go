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
	"github.com/SimonSchneider/pefigo/internal/pdb"
	"github.com/SimonSchneider/pefigo/internal/uncertain"
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

// SnapshotHistorySeries is one series (account type) for the snapshot history grouped bar chart.
type SnapshotHistorySeries struct {
	Name  string    `json:"name"`
	Color string    `json:"color"`
	Data  []float64 `json:"data"`
}

// SnapshotHistoryChartData is the data for the dashboard snapshot history grouped bar chart.
type SnapshotHistoryChartData struct {
	Dates  []string                `json:"dates"`
	Series []SnapshotHistorySeries `json:"series"`
}

type DashboardView struct {
	TotalBalance         float64
	TotalAssets          float64
	TotalLiabilities     float64
	Budget               *BudgetView
	AccountTypeGroups    []AccountTypeGroup
	AccountChartData     []AccountTypeChartEntry
	SnapshotHistoryChart SnapshotHistoryChartData
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

	snapshotHistory, err := buildSnapshotHistoryChart(ctx, db, accountTypes)
	if err != nil {
		return nil, fmt.Errorf("building snapshot history chart: %w", err)
	}

	return &DashboardView{
		TotalBalance:         totalBalance,
		TotalAssets:          totalAssets,
		TotalLiabilities:     totalLiabilities,
		Budget:               budget,
		AccountTypeGroups:    groups,
		AccountChartData:     chartData,
		SnapshotHistoryChart: snapshotHistory,
	}, nil
}

func buildSnapshotHistoryChart(ctx context.Context, db *sql.DB, accountTypes []AccountType) (SnapshotHistoryChartData, error) {
	rows, err := pdb.New(db).ListSnapshotHistoryWithType(ctx)
	if err != nil {
		return SnapshotHistoryChartData{}, err
	}
	type TypeKey struct {
		Date   int64
		TypeID string
	}
	typeKey := func(date int64, typeID string) TypeKey { return TypeKey{date, typeID} }
	sumByDateAndType := make(map[TypeKey]float64)
	dateSet := make(map[int64]struct{})
	for _, r := range rows {
		var v uncertain.Value
		if err := v.Decode(r.Balance); err != nil {
			return SnapshotHistoryChartData{}, fmt.Errorf("decoding balance: %w", err)
		}
		key := typeKey(r.Date, r.TypeID)
		sumByDateAndType[key] += v.Mean()
		dateSet[r.Date] = struct{}{}
	}
	dates := make([]int64, 0, len(dateSet))
	for d := range dateSet {
		dates = append(dates, d)
	}
	sort.Slice(dates, func(i, j int) bool { return dates[i] < dates[j] })

	// Limit to at most snapshotHistoryMaxBars, evenly spaced with latest always last
	const snapshotHistoryMaxBars = 24
	dates = downsampleDates(dates, snapshotHistoryMaxBars)

	typeByName := make(map[string]AccountType)
	for _, at := range accountTypes {
		typeByName[at.ID] = at
	}
	// Include untyped accounts (type_id "") in the chart if they have any balance
	for _, d := range dates {
		if sumByDateAndType[typeKey(d, "")] != 0 {
			if _, ok := typeByName[""]; !ok {
				typeByName[""] = AccountType{ID: "", Name: "Uncategorized", Color: "#999999"}
			}
			break
		}
	}
	typeIDsOrdered := make([]string, 0, len(accountTypes)+1)
	for _, at := range accountTypes {
		typeIDsOrdered = append(typeIDsOrdered, at.ID)
	}
	if _, ok := typeByName[""]; ok {
		typeIDsOrdered = append(typeIDsOrdered, "")
	}
	dateLabels := make([]string, len(dates))
	for i, d := range dates {
		dateLabels[i] = date.Date(d).String()
	}
	series := make([]SnapshotHistorySeries, 0)
	for _, typeID := range typeIDsOrdered {
		at := typeByName[typeID]
		color := at.Color
		if color == "" {
			color = "#999999"
		}
		data := make([]float64, len(dates))
		hasAny := false
		for i, d := range dates {
			v := sumByDateAndType[typeKey(d, typeID)]
			data[i] = math.Round(v*100) / 100
			if v != 0 {
				hasAny = true
			}
		}
		if hasAny {
			series = append(series, SnapshotHistorySeries{
				Name:  at.Name,
				Color: color,
				Data:  data,
			})
		}
	}
	return SnapshotHistoryChartData{
		Dates:  dateLabels,
		Series: series,
	}, nil
}

// downsampleDates returns at most maxBars dates from dates, evenly spaced,
// with the first being the oldest and the last being the latest (most recent).
func downsampleDates(dates []int64, maxBars int) []int64 {
	if len(dates) <= maxBars || maxBars <= 1 {
		return dates
	}
	out := make([]int64, 0, maxBars)
	n := len(dates)
	for i := 0; i < maxBars; i++ {
		idx := (n - 1) * i / (maxBars - 1)
		out = append(out, dates[idx])
	}
	return out
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
