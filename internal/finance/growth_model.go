package finance

import (
	"math"
	"sort"

	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/pefigo/internal/uncertain"
)

type GrowthModel interface {
	StartsOn() date.Date // Returns the start date of the growth model
	IsActiveOn(date date.Date) bool
	Apply(ucfg *uncertain.Config, day date.Date, totalBalance uncertain.Value) (delta uncertain.Value)
}

type TimeFrameGrowth struct {
	StartDate date.Date
	EndDate   *date.Date // optional
}

func (i *TimeFrameGrowth) StartsOn() date.Date {
	return i.StartDate
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

type GrowthCombined struct {
	Growths []GrowthModel
}

func NewGrowthCombined(growths ...GrowthModel) *GrowthCombined {
	// Sort growths by start date
	sort.Slice(growths, func(i, j int) bool {
		return growths[i].StartsOn().Before(growths[j].StartsOn())
	})
	return &GrowthCombined{Growths: growths}
}

func (g *GrowthCombined) StartsOn() date.Date {
	if len(g.Growths) == 0 {
		return date.Date(0) // Return zero date if no growths are defined
	}
	return g.Growths[0].StartsOn()
}

func (g *GrowthCombined) IsActiveOn(day date.Date) bool {
	for _, gr := range g.Growths {
		if gr.IsActiveOn(day) {
			return true
		}
	}
	return false
}

func (g *GrowthCombined) Apply(ucfg *uncertain.Config, day date.Date, totalBalance uncertain.Value) uncertain.Value {
	// find the first growth that is active on the given day
	i, found := sort.Find(len(g.Growths), func(i int) int {
		if g.Growths[i].StartsOn().After(day) {
			return 1 // This growth starts after the day, so we skip it
		}
		if g.Growths[i].IsActiveOn(day) {
			return 0 // This growth is active on the day
		}
		return -1 // This growth is not active on the day
	})
	if !found {
		return uncertain.NewFixed(0.0) // No growth applicable
	}
	gr := g.Growths[i].Apply(ucfg, day, totalBalance)
	return gr
}

var _ GrowthModel = &GrowthCombined{}
var _ GrowthModel = &FixedGrowth{}
var _ GrowthModel = &LogNormalGrowth{}

type FixedGrowth struct {
	TimeFrameGrowth
	AnnualRate uncertain.Value // Annual growth rate, e.g. 0.05 for 5%
}

func (i *FixedGrowth) Apply(ucfg *uncertain.Config, day date.Date, totalBalance uncertain.Value) uncertain.Value {
	dailyGrowthFactor := i.AnnualRate.Add(ucfg, uncertain.NewFixed(1)).Pow(ucfg, uncertain.NewFixed(1.0/365.0))
	return totalBalance.Mul(ucfg, dailyGrowthFactor.Sub(ucfg, uncertain.NewFixed(1)))
}

type LogNormalGrowth struct {
	TimeFrameGrowth
	AnnualRate       uncertain.Value
	AnnualVolatility uncertain.Value // Optional, can be used for more complex models
}

func (i *LogNormalGrowth) Apply(ucfg *uncertain.Config, day date.Date, totalBalance uncertain.Value) uncertain.Value {
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
