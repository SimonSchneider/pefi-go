package finance

import (
	"fmt"
	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/pefigo/internal/uncertain"
)

type ConcreteTransfers struct {
	ID     string
	Amount float64
}

func handleTransfer(balances map[string]float64, t TransferTemplate, day date.Date, overrideAmount float64, recorder TransferRecorder) {
	// This function would handle the transfer logic, e.g., updating balances, recording transfers, etc.
	// For simulation purposes, we will just print the transfer details.
	var amount float64
	if overrideAmount > 0 {
		amount = overrideAmount
	} else if t.AmountType == AmountPercent {
		if t.FromAccountID != "" {
			amount = t.AmountPercent.Percent * balances[t.FromAccountID]
		}
	} else {
		amount = t.AmountFixed.Amount.Mean()
	}
	if t.FromAccountID != "" {
		balances[t.FromAccountID] -= amount
	}
	if t.ToAccountID != "" {
		balances[t.ToAccountID] += amount
	}
	if recorder != nil {
		if err := recorder.OnTransfer(t.FromAccountID, t.ToAccountID, day, uncertain.NewFixed(amount)); err != nil {
			fmt.Printf("Error recording transfer: %s\n", err)
		}
	}
}

func initBalance(balances map[string]float64, keys ...string) map[string]float64 {
	for _, key := range keys {
		if key == "" {
			continue // Skip empty keys
		}
		if _, exists := balances[key]; !exists {
			balances[key] = 0 // Initialize balance for the key if it doesn't exist
		}
	}
	return balances
}

func RunSimulation(given []ConcreteTransfers, transfers []TransferTemplate, day date.Date, recorder TransferRecorder) {
	cts := make(map[string]ConcreteTransfers)
	for _, g := range given {
		cts[g.ID] = g
	}
	balances := make(map[string]float64)
	fmt.Printf("Running Simulation with %+v transfers\n", cts)
	// support priority balances (should consolidate with simulation logic)
	for _, t := range transfers {
		if !t.Enabled || t.EffectiveFrom.After(day) || (t.EffectiveTo != nil && t.EffectiveTo.Before(day)) {
			fmt.Printf("Skipping transfer: %s\n", t.ID)
			continue // Skip transfers that are not active on the given day
		}
		balances = initBalance(balances, t.FromAccountID, t.ToAccountID)
		if ct, ok := cts[t.ID]; ok {
			handleTransfer(balances, t, day, ct.Amount, recorder)
			continue
		}
		if t.FromAccountID == "" || t.ToAccountID == "" {
			fmt.Printf("Skipping transfer: %s\n", t.ID)
			// Skip non internal transfers
		}
		if t.AmountType == AmountFixed && t.AmountFixed.Amount.Distribution != uncertain.DistFixed {
			fmt.Printf("Skipping transfer with non-fixed amount: %s\n", t.ID)
			// Skip transfers with non-fixed amounts
			continue
		}
		if t.AmountFixed.Amount.Zero() && t.AmountPercent.Percent == 0 {
			// Skip transfers with zero amount
			continue
		}
		handleTransfer(balances, t, day, 0, recorder)
	}
	fmt.Printf("Simulation with %+v transfers\n%+v\n", cts, balances)
}
