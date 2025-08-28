package core

import (
	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/pefigo/internal/finance"
	"github.com/SimonSchneider/pefigo/internal/uncertain"
)

var staticEntities = []finance.Entity{
	{
		ID:   "checking",
		Name: "Checking",
		Snapshots: []finance.BalanceSnapshot{
			{Balance: uncertain.NewFixed(500), Date: date.Today().Add(-30)},
			{Balance: uncertain.NewFixed(1000), Date: date.Today()},
		},
	},
	{
		ID:   "savings",
		Name: "Savings",
		Snapshots: []finance.BalanceSnapshot{
			{Balance: uncertain.NewFixed(800_000), Date: date.Today().Add(-30)},
			{Balance: uncertain.NewFixed(810_000), Date: date.Today()},
		},
		//GrowthModel: &finance.LogNormalGrowth{
		//	AnnualRate:       uncertain.NewUniform(0.0, 0.02),
		//	AnnualVolatility: uncertain.NewFixed(0.01),
		//},
		GrowthModel: &finance.LogNormalGrowth{
			AnnualRate:       uncertain.NewUniform(0.04, 0.08),
			AnnualVolatility: uncertain.NewUniform(0.06, 0.12),
		},
	},
	{
		ID:   "realEstate",
		Name: "Real Estate",
		Snapshots: []finance.BalanceSnapshot{
			{Balance: uncertain.NewUniform(3_600_000, 3_900_000), Date: date.Today().Add(-1 * date.Year)},
			{Balance: uncertain.NewUniform(3_800_000, 4_100_000), Date: date.Today()},
		},
		GrowthModel: &finance.LogNormalGrowth{
			AnnualRate:       uncertain.NewUniform(0.03, 0.06),
			AnnualVolatility: uncertain.NewUniform(0.1, 0.2),
		},
	},
	//{
	//	ID:   "plantStocks",
	//	Name: "Plant stocks",
	//	Snapshots: []finance.BalanceSnapshot{
	//		{Balance: uncertain.NewFixed(2_437_000), Date: date.Today().Add(-1 * date.Year)},
	//	},
	//	GrowthModel: &finance.LogNormalGrowth{
	//		AnnualRate:       uncertain.NewUniform(0.2, 0.5),
	//		AnnualVolatility: uncertain.NewUniform(0.3, 0.5),
	//	},
	//},
	{
		ID:   "mortgage",
		Name: "Mortgage",
		BalanceLimit: finance.BalanceLimit{
			Upper: uncertain.NewFixed(0),
		},
		Snapshots: []finance.BalanceSnapshot{
			{Balance: uncertain.NewFixed(-1_300_000), Date: date.Today()},
		},
		GrowthModel: &finance.FixedGrowth{
			AnnualRate: uncertain.NewUniform(0.012, 0.045), // Negative growth for debt
		},
		CashFlow: &finance.CashFlowModel{
			Frequency:     "*-*-25",
			DestinationID: "checking", // Assume mortgage payments go to checking account
		},
	},
}

var staticTransfers = []finance.TransferTemplate{
	{
		ID:            "savings",
		Name:          "Savings Transfer",
		FromAccountID: "checking",
		ToAccountID:   "savings",
		AmountType:    finance.AmountPercent,
		AmountPercent: finance.TransferPercent{
			Percent: 0.2,
		},
		Priority:   0,
		Recurrence: "*-*-25",
		Enabled:    true,
	},
	{
		ID:            "extraMortgagePayment",
		Name:          "extraMortgage Transfer",
		FromAccountID: "checking",
		ToAccountID:   "mortgage",
		AmountType:    finance.AmountPercent,
		AmountPercent: finance.TransferPercent{
			Percent: 0.8,
		},
		Priority:   0,
		Recurrence: "*-*-25",
		Enabled:    true,
	},
	{
		ID:            "finalSavings",
		Name:          "Final Savings",
		FromAccountID: "checking",
		ToAccountID:   "savings",
		AmountType:    finance.AmountPercent,
		AmountPercent: finance.TransferPercent{
			Percent: 1,
		},
		Priority:   1,
		Recurrence: "*-*-25",
		Enabled:    true,
	},
	{
		ID:            "salary",
		Name:          "Salary",
		FromAccountID: "",
		ToAccountID:   "checking",
		AmountType:    finance.AmountFixed,
		AmountFixed: finance.TransferFixed{
			Amount: uncertain.NewFixed(60_000),
		},
		Priority:   2,
		Recurrence: "*-*-25",
		Enabled:    true,
	},
	{
		ID:            "fixedCosts",
		Name:          "Fixed Costs Transfer",
		FromAccountID: "checking",
		ToAccountID:   "",
		AmountType:    finance.AmountFixed,
		AmountFixed: finance.TransferFixed{
			Amount: uncertain.NewFixed(30_000),
		},
		Priority:   3,
		Recurrence: "*-*-25",
		Enabled:    true,
	},
	{
		ID:            "mortgagePayment",
		Name:          "Mortgage Payment",
		FromAccountID: "checking",
		ToAccountID:   "mortgage",
		AmountType:    finance.AmountFixed,
		AmountFixed: finance.TransferFixed{
			Amount: uncertain.NewFixed(10_000),
		},
		Priority:   4,
		Recurrence: "*-*-25",
		Enabled:    true,
	},
}
