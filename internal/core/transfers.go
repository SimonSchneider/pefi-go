package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/goslu/srvu"
	"github.com/SimonSchneider/goslu/static/shttp"
	"github.com/SimonSchneider/pefigo/internal/finance"
	"github.com/SimonSchneider/pefigo/internal/uncertain"
)

type TransfersView struct {
	day date.Date

	TransferTemplatesThatNeedAmount []TransferTemplateWithAmount
	TransferTemplates               []TransferTemplate

	transfersReady      bool
	incompleteTransfers bool
	accounts            map[string]Account
	transfers           []Transfer
}

type Transfer struct {
	FromAccountID string
	ToAccountID   string
	Amount        float64
}

type TransfersViewInput struct {
	day date.Date
}

func (i *TransfersViewInput) FromForm(r *http.Request) error {
	return shttp.Parse(&i.day, date.ParseDate, r.FormValue("day"), date.Today())
}

func (i *TransfersViewInput) ToView() *TransfersView {
	return &TransfersView{day: i.day}
}

func NewTransfersView(day date.Date) *TransfersView {
	return &TransfersView{day: day}
}

func TransfersPage(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var inp TransfersViewInput
		if err := inp.FromForm(r); err != nil {
			return fmt.Errorf("parsing transfers view input: %w", err)
		}
		view := inp.ToView()
		allTransferTemplates, err := ListTransferTemplatesWithChildren(ctx, db)
		if err != nil {
			return fmt.Errorf("listing transfer templates: %w", err)
		}
		view.transfersReady = false
		for _, t := range allTransferTemplates {
			if t.EndDate == nil || t.EndDate.After(inp.day) {
				if t.StartDate.Before(inp.day) && t.Recurrence.Matches(inp.day) {
					if t.AmountType == "fixed" && t.AmountFixed.Distribution != uncertain.DistFixed {
						amount, err := shttp.ParseFloat(r.FormValue("amount_" + t.ID))
						if err == nil {
							view.transfersReady = true
						}
						view.TransferTemplatesThatNeedAmount = append(view.TransferTemplatesThatNeedAmount, TransferTemplateWithAmount{TransferTemplate: t, Amount: amount})
					} else {
						view.TransferTemplates = append(view.TransferTemplates, t)
					}
				}
			}
		}
		if view.transfersReady {
			ucfg := uncertain.NewConfig(time.Now().UnixMilli(), 1)
			transfers := make([]finance.TransferTemplate, 0)
			entities := make([]finance.Entity, 0)
			accounts, err := ListAccountsDetailed(ctx, db, inp.day)
			if err != nil {
				return fmt.Errorf("listing accounts: %w", err)
			}
			view.accounts = make(map[string]Account)
			for _, a := range accounts {
				// Skip startup share accounts
				if a.StartupShareAccount != nil {
					continue
				}
				// Skip accounts without snapshots
				if a.LastSnapshot == nil {
					continue
				}

				view.accounts[a.ID] = a.Account
				ul := uncertain.Value{}
				if a.BalanceUpperLimit != nil {
					ul = uncertain.NewFixed(*a.BalanceUpperLimit)
				}
				entity := finance.Entity{
					ID:   a.ID,
					Name: a.Name,
					BalanceLimit: finance.BalanceLimit{
						Upper: ul,
					},
					Snapshots: []finance.BalanceSnapshot{a.LastSnapshot.ToFinance()},
				}
				if a.GrowthModel != nil {
					entity.GrowthModel = GrowthModels([]GrowthModel{*a.GrowthModel}).ToFinance()
				}
				entities = append(entities, entity)
			}

			for _, t := range view.TransferTemplatesThatNeedAmount {
				ft := t.TransferTemplate.ToFinanceTransferTemplate()
				ft.AmountFixed.Amount = uncertain.NewFixed(t.Amount)
				transfers = append(transfers, ft)
			}
			for _, t := range view.TransferTemplates {
				transfers = append(transfers, t.ToFinanceTransferTemplate())
			}

			transferRecorder := finance.TransferRecorderFunc(func(sourceAccountID, destinationAccountID string, day date.Date, amount uncertain.Value) error {
				if amount.Distribution != uncertain.DistFixed {
					view.incompleteTransfers = true
					return nil
				}
				view.transfers = append(view.transfers, Transfer{
					FromAccountID: sourceAccountID,
					ToAccountID:   destinationAccountID,
					Amount:        amount.Fixed.Value,
				})
				return nil
			})
			if err := finance.RunPrediction(ctx, ucfg, view.day, view.day.Add(1*date.Day), date.Cron(view.day.String()), entities, transfers, finance.CompositeRecorder{TransferRecorder: transferRecorder}); err != nil {
				return fmt.Errorf("running prediction for SSE: %w", err)
			}
		}
		view.transfers = SimplifyTransfers(view.transfers)
		return NewView(ctx, w, r).Render(Page("Transfers", PageTransfers(view)))
	})
}

// Simplify transfers
// - by removing transfers that are not internal
// - merges transfers that have the same from and to account (regardless of direction) by summing the amounts
func SimplifyTransfers(transfers []Transfer) []Transfer {
	// Filter out self-transfers and external transfers
	simplified := make([]Transfer, 0)
	for _, t := range transfers {
		if t.FromAccountID != "" && t.ToAccountID != "" && t.FromAccountID != t.ToAccountID {
			simplified = append(simplified, t)
		}
	}

	type Key struct {
		FromAccountID string
		ToAccountID   string
	}
	netAmounts := make(map[Key]float64)
	for _, t := range simplified {
		key := Key{FromAccountID: t.FromAccountID, ToAccountID: t.ToAccountID}
		reverseKey := Key{FromAccountID: t.ToAccountID, ToAccountID: t.FromAccountID}

		if reverseAmount, exists := netAmounts[reverseKey]; exists {
			netAmount := reverseAmount - t.Amount
			if netAmount > 0 {
				netAmounts[reverseKey] = netAmount
			} else if netAmount < 0 {
				delete(netAmounts, reverseKey)
				netAmounts[key] = -netAmount
			} else {
				delete(netAmounts, reverseKey)
			}
		} else {
			// Add or update the amount for this direction
			netAmounts[key] += t.Amount
		}
	}

	result := make([]Transfer, 0)

	for key, amount := range netAmounts {
		if amount != 0 {
			result = append(result, Transfer{
				FromAccountID: key.FromAccountID,
				ToAccountID:   key.ToAccountID,
				Amount:        amount,
			})
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].FromAccountID == result[j].FromAccountID {
			return result[i].ToAccountID < result[j].ToAccountID
		}
		return result[i].FromAccountID < result[j].FromAccountID
	})

	return result
}

// TransferChartLink represents a link in the Sankey chart
type TransferChartLink struct {
	Source string  `json:"source"`
	Target string  `json:"target"`
	Value  float64 `json:"value"`
	Label  string  `json:"label"`
}

// SimplifyChartLinks removes cycles by netting opposite transfers
// - merges transfers that have the same source and target (regardless of direction) by summing the amounts
// - if Account A → Account B has amount X and Account B → Account A has amount Y, keeps only the net direction with amount |X - Y|
func SimplifyChartLinks(links []TransferChartLink) []TransferChartLink {
	type Key struct {
		Source string
		Target string
	}
	type LinkInfo struct {
		Value float64
		Label string
	}

	netLinks := make(map[Key]LinkInfo)
	for _, link := range links {
		// Skip self-loops
		if link.Source == link.Target {
			continue
		}

		key := Key{Source: link.Source, Target: link.Target}
		reverseKey := Key{Source: link.Target, Target: link.Source}

		if reverseInfo, exists := netLinks[reverseKey]; exists {
			// We have a reverse link, net them
			netAmount := reverseInfo.Value - link.Value
			if netAmount > 0 {
				// Keep reverse direction, update amount
				netLinks[reverseKey] = LinkInfo{
					Value: netAmount,
					Label: reverseInfo.Label, // Keep the original label
				}
			} else if netAmount < 0 {
				// Switch to forward direction
				delete(netLinks, reverseKey)
				netLinks[key] = LinkInfo{
					Value: -netAmount,
					Label: link.Label, // Use the new label
				}
			} else {
				// Net to zero, remove both
				delete(netLinks, reverseKey)
			}
		} else {
			// No reverse link exists, add or update the amount for this direction
			if existing, exists := netLinks[key]; exists {
				netLinks[key] = LinkInfo{
					Value: existing.Value + link.Value,
					Label: existing.Label + ", " + link.Label, // Combine labels
				}
			} else {
				netLinks[key] = LinkInfo{
					Value: link.Value,
					Label: link.Label,
				}
			}
		}
	}

	result := make([]TransferChartLink, 0, len(netLinks))
	seenPairs := make(map[string]bool) // "source:target" -> exists
	for key, info := range netLinks {
		// Only include links with positive value and ensure no self-loops
		if info.Value > 0 && key.Source != key.Target {
			// Check if reverse pair already exists
			reversePair := key.Target + ":" + key.Source
			if seenPairs[reversePair] {
				// Reverse link already added, skip this one (shouldn't happen, but be safe)
				continue
			}
			pair := key.Source + ":" + key.Target
			seenPairs[pair] = true
			result = append(result, TransferChartLink{
				Source: key.Source,
				Target: key.Target,
				Value:  info.Value,
				Label:  info.Label,
			})
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Source == result[j].Source {
			return result[i].Target < result[j].Target
		}
		return result[i].Source < result[j].Source
	})

	return result
}

type TransferChartGroupBy string

const (
	GroupByAccount     TransferChartGroupBy = "account"
	GroupByAccountType TransferChartGroupBy = "account_type"
)

func ParseTransferChartGroupBy(val string) (TransferChartGroupBy, error) {
	switch val {
	case "account":
		return GroupByAccount, nil
	case "account_type":
		return GroupByAccountType, nil
	default:
		return GroupByAccount, fmt.Errorf("invalid group by: %s", val)
	}
}

func TransferChartPage(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var groupBy TransferChartGroupBy
		if err := shttp.Parse(&groupBy, ParseTransferChartGroupBy, r.FormValue("group_by"), GroupByAccount); err != nil {
			return fmt.Errorf("parsing group_by: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Transfers Chart", PageTransfersChart(groupBy)))
	})
}

func TransferChartData(db *sql.DB) http.Handler {
	type ItemStyle struct {
		Color string `json:"color"`
	}
	type TransferChartData struct {
		Name      string    `json:"name"`
		Label     string    `json:"label"`
		ItemStyle ItemStyle `json:"itemStyle"`
	}
	type TransferChartDataEnvelope struct {
		Data  []TransferChartData `json:"data"`
		Links []TransferChartLink `json:"links"`
	}

	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var groupBy TransferChartGroupBy
		if err := shttp.Parse(&groupBy, ParseTransferChartGroupBy, r.FormValue("group_by"), GroupByAccount); err != nil {
			return fmt.Errorf("parsing group_by: %w", err)
		}

		transfersTemplates, err := ListTransferTemplates(ctx, db)
		if err != nil {
			return fmt.Errorf("listing transfer templates: %w", err)
		}
		accounts, err := ListAccounts(ctx, db)
		if err != nil {
			return fmt.Errorf("listing accounts: %w", err)
		}
		accountTypes, err := ListAccountTypes(ctx, db)
		if err != nil {
			return fmt.Errorf("listing account types: %w", err)
		}
		accountTypesById := KeyBy(accountTypes, func(a AccountType) string { return a.ID })
		ttsWithAmounts := makeTransferTemplatesWithAmount(transfersTemplates, date.Today())

		accountsById := KeyBy(accounts, func(a Account) string { return a.ID })

		// Handle accounts without type for grouping
		if groupBy == GroupByAccountType {
			for _, a := range accounts {
				if a.TypeID == "" {
					accountTypesById[""] = AccountType{
						ID:    "",
						Name:  "unknown",
						Color: "",
					}
					break
				}
			}
		}

		var chartData []TransferChartData
		var accountToNodeName map[string]string

		if groupBy == GroupByAccountType {
			// Create nodes for account types
			accountTypeSet := make(map[string]AccountType)
			for _, a := range accounts {
				typeID := a.TypeID
				if typeID == "" {
					typeID = ""
				}
				if at, exists := accountTypesById[typeID]; exists {
					accountTypeSet[typeID] = at
				} else {
					accountTypeSet[""] = AccountType{ID: "", Name: "unknown", Color: ""}
				}
			}

			chartData = make([]TransferChartData, 0, len(accountTypeSet))
			for _, at := range accountTypeSet {
				chartData = append(chartData, TransferChartData{
					Name:      at.Name,
					Label:     at.Name,
					ItemStyle: ItemStyle{Color: at.Color},
				})
			}

			// Map account IDs to their account type names
			accountToNodeName = make(map[string]string, len(accounts))
			for _, a := range accounts {
				typeID := a.TypeID
				if typeID == "" {
					typeID = ""
				}
				if at, exists := accountTypesById[typeID]; exists {
					accountToNodeName[a.ID] = at.Name
				} else {
					accountToNodeName[a.ID] = "unknown"
				}
			}
		} else {
			// Create nodes for individual accounts
			chartData = make([]TransferChartData, 0, len(accounts))
			for _, a := range accounts {
				typeID := a.TypeID
				if typeID == "" {
					typeID = ""
				}
				at, exists := accountTypesById[typeID]
				if !exists {
					at = AccountType{ID: "", Name: "unknown", Color: ""}
				}
				chartData = append(chartData, TransferChartData{
					Name:      a.Name,
					Label:     a.Name,
					ItemStyle: ItemStyle{Color: at.Color},
				})
			}

			// Map account IDs to their account names
			accountToNodeName = make(map[string]string, len(accounts))
			for _, a := range accounts {
				accountToNodeName[a.ID] = a.Name
			}
		}

		// Always add Income and Expenses nodes
		chartData = append(chartData, TransferChartData{Name: "Income", Label: "Income", ItemStyle: ItemStyle{Color: "#388E3C"}})
		chartData = append(chartData, TransferChartData{Name: "Expenses", Label: "Expenses", ItemStyle: ItemStyle{Color: "#D32F2F"}})

		// Build chart links
		// First, aggregate links by source/target when grouping by account type
		type LinkKey struct {
			Source string
			Target string
		}
		aggregatedLinks := make(map[LinkKey]struct {
			Value float64
			Label string
		})

		for _, t := range ttsWithAmounts {
			if strings.Contains(string(t.Name), "Matkort") {
				continue
			}
			if t.Amount > 0 && t.Enabled && strings.Contains(string(t.Recurrence), "*") {
				var source, target string
				if t.FromAccountID == "" {
					source = "Income"
				} else {
					if nodeName, exists := accountToNodeName[t.FromAccountID]; exists {
						source = nodeName
					} else {
						// Fallback to account name if not found in map
						if acc, exists := accountsById[t.FromAccountID]; exists {
							source = acc.Name
						}
					}
				}
				if t.ToAccountID == "" {
					target = "Expenses"
				} else {
					if nodeName, exists := accountToNodeName[t.ToAccountID]; exists {
						target = nodeName
					} else {
						// Fallback to account name if not found in map
						if acc, exists := accountsById[t.ToAccountID]; exists {
							target = acc.Name
						}
					}
				}

				// Skip if source or target is empty (shouldn't happen, but be safe)
				if source == "" || target == "" {
					continue
				}

				// Skip self-loops (source == target)
				if source == target {
					continue
				}

				key := LinkKey{Source: source, Target: target}
				if existing, exists := aggregatedLinks[key]; exists {
					aggregatedLinks[key] = struct {
						Value float64
						Label string
					}{
						Value: existing.Value + t.Amount,
						Label: existing.Label + ", " + t.Name,
					}
				} else {
					aggregatedLinks[key] = struct {
						Value float64
						Label string
					}{
						Value: t.Amount,
						Label: t.Name,
					}
				}
			}
		}

		// Convert aggregated links to TransferChartLink slice
		chartLinks := make([]TransferChartLink, 0, len(aggregatedLinks))
		for key, info := range aggregatedLinks {
			// Double-check: skip self-loops
			if key.Source != key.Target && info.Value > 0 {
				chartLinks = append(chartLinks, TransferChartLink{
					Source: key.Source,
					Target: key.Target,
					Value:  info.Value,
					Label:  info.Label,
				})
			}
		}

		// Remove cycles by netting opposite transfers
		chartLinks = SimplifyChartLinks(chartLinks)

		data := TransferChartDataEnvelope{
			Data:  chartData,
			Links: chartLinks,
		}
		if err := json.NewEncoder(w).Encode(data); err != nil {
			return fmt.Errorf("encoding JSON: %w", err)
		}
		return nil
	})
}
