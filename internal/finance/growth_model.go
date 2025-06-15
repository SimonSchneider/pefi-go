package finance

import (
	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/pefigo/internal/uncertain"
	"math"
)

type GrowthModel interface {
	IsActiveOn(date date.Date) bool
	Apply(ucfg *uncertain.Config, totalBalance uncertain.Value) (delta uncertain.Value)
}

type TimeFrameGrowth struct {
	StartDate date.Date
	EndDate   *date.Date // optional

}

func (i *TimeFrameGrowth) IsActiveOn(day date.Date) bool {
	// Check if the interest model applies to the given day
	if i.StartDate.After(day) {
		return false
	}
	if i.EndDate != nil && i.EndDate.Before(day) {
		return false
	}
	return true
}

type FixedGrowth struct {
	TimeFrameGrowth
	AnnualRate uncertain.Value // Annual growth rate, e.g. 0.05 for 5%
}

func (i *FixedGrowth) Apply(ucfg *uncertain.Config, totalBalance uncertain.Value) uncertain.Value {
	dailyGrowthFactor := i.AnnualRate.Add(ucfg, uncertain.NewFixed(1)).Pow(ucfg, uncertain.NewFixed(1.0/365.0))
	return totalBalance.Mul(ucfg, dailyGrowthFactor.Sub(ucfg, uncertain.NewFixed(1)))
}

type LogNormalGrowth struct {
	TimeFrameGrowth
	AnnualRate       uncertain.Value
	AnnualVolatility uncertain.Value // Optional, can be used for more complex models
}

func (i *LogNormalGrowth) Apply(ucfg *uncertain.Config, totalBalance uncertain.Value) uncertain.Value {
	dailyMu := i.AnnualRate.Mul(ucfg, uncertain.NewFixed(1.0/365.0))
	dailySigma := i.AnnualVolatility.Mul(ucfg, uncertain.NewFixed(1.0/math.Sqrt(365)))
	if !i.AnnualVolatility.Valid() {
		dailySigma = uncertain.NewFixed(0.0) // If no volatility is set, use 0
	}

	// Daily log return is normally distributed: N(dailyMu, dailySigma)
	dailyLogReturn := uncertain.NewMapped(
		func(cfg *uncertain.Config) float64 {
			mu := dailyMu.Sample(cfg)
			sigma := dailySigma.Sample(cfg)
			return cfg.RNG.NormFloat64()*sigma + mu
		},
	)

	// Convert to growth factor: exp(log_return) - 1
	dailyGrowth := dailyLogReturn.Exp().Sub(ucfg, uncertain.NewFixed(1))

	return totalBalance.Mul(ucfg, dailyGrowth)
}
