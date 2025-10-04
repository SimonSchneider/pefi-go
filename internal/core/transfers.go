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
		allTransferTemplates, err := ListTransferTemplates(ctx, db)
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
				view.accounts[a.ID] = a.Account
				ul := uncertain.Value{}
				if a.BalanceUpperLimit != nil {
					ul = uncertain.NewFixed(*a.BalanceUpperLimit)
				}
				entities = append(entities, finance.Entity{
					ID:   a.ID,
					Name: a.Name,
					BalanceLimit: finance.BalanceLimit{
						Upper: ul,
					},
					Snapshots: []finance.BalanceSnapshot{a.LastSnapshot.ToFinance()},
				})
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

func TransferChartPage(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return NewView(ctx, w, r).Render(Page("Transfers Chart", PageTransfersChart()))
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
	type TransferChartLink struct {
		Source string  `json:"source"`
		Target string  `json:"target"`
		Value  float64 `json:"value"`
		Label  string  `json:"label"`
	}
	type TransferChartDataEnvelope struct {
		Data  []TransferChartData `json:"data"`
		Links []TransferChartLink `json:"links"`
	}

	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
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

		chartData := make([]TransferChartData, 0, len(accounts))
		for _, a := range accounts {
			at := accountTypesById[a.TypeID]
			chartData = append(chartData, TransferChartData{Name: a.Name, Label: a.Name, ItemStyle: ItemStyle{Color: at.Color}})
		}
		chartData = append(chartData, TransferChartData{Name: "Income", Label: "Income", ItemStyle: ItemStyle{Color: "#388E3C"}})
		chartData = append(chartData, TransferChartData{Name: "Expenses", Label: "Expenses", ItemStyle: ItemStyle{Color: "#D32F2F"}})

		accountsById := KeyBy(accounts, func(a Account) string { return a.ID })
		chartLinks := make([]TransferChartLink, 0, len(ttsWithAmounts))
		for _, t := range ttsWithAmounts {
			if strings.Contains(string(t.Name), "Matkort") {
				continue
			}
			if t.Amount > 0 && t.Enabled && strings.Contains(string(t.Recurrence), "*") {
				link := TransferChartLink{Source: t.FromAccountID, Target: t.ToAccountID, Label: t.Name, Value: t.Amount}
				if t.FromAccountID == "" {
					link.Source = "Income"
				} else {
					link.Source = accountsById[t.FromAccountID].Name
				}
				if t.ToAccountID == "" {
					link.Target = "Expenses"
				} else {
					link.Target = accountsById[t.ToAccountID].Name
				}
				chartLinks = append(chartLinks, link)
			}
		}

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
