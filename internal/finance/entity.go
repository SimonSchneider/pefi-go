package finance

import (
	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/pefigo/internal/uncertain"
)

type CompoundingPeriod string

const (
	Daily   CompoundingPeriod = "daily"
	Monthly CompoundingPeriod = "monthly"
	Yearly  CompoundingPeriod = "yearly"
)

type FinancialEntity struct {
	ID   string
	Name string

	Balance  uncertain.Value
	Growth   GrowthModel
	CashFlow *CashFlowModel
}

type GrowthModel interface {
	GrowthRate(at date.Date) uncertain.Value
	Compounding() CompoundingPeriod
	Apply(balance uncertain.Value, on date.Date) uncertain.Value
}

type CashFlowModel struct {
	Frequency               date.Cron
	TargetFinancialEntityID string
}

type FinancialEntityPred struct {
	FinancialEntity

	accruedInterest uncertain.Value
	snapshots       []FinancialEntitySnapshot
}

func RunPrediction(rawEntities []FinancialEntity, start, end date.Date) ([]FinancialEntityPred, error) {
	entities := make([]FinancialEntityPred, len(rawEntities))
	for i, entity := range entities {
		entities[i] = FinancialEntityPred{}
	}
	return results, nil
}
