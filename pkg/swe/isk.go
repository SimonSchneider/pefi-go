package swe

import (
	"time"

	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/pefigo/pkg/uncertain"
)

const iskTaxRate = 0.30

// ISKParams holds the yearly ISK tax parameters.
type ISKParams struct {
	SchablonRanta float64
	Fribelopp     float64
}

// ISKTax implements finance.TaxModel for Swedish ISK (Investeringssparkonto) accounts.
// Tax is calculated annually on Jan 1 based on quarterly account values and deposits.
type ISKTax struct {
	ParamsFunc func(date.Date) ISKParams

	quarterlyValues [4]uncertain.Value
	yearDeposits    uncertain.Value
	initialized     bool
}

func (t *ISKTax) Apply(ucfg *uncertain.Config, day date.Date, balance uncertain.Value, dayDeposits uncertain.Value) uncertain.Value {
	var tax uncertain.Value
	if day.Month() == time.January && day.Day() == 1 && t.initialized {
		tax = t.computeTax(ucfg, day)
		t.reset()
	}

	if !dayDeposits.Zero() {
		if t.yearDeposits.Zero() {
			t.yearDeposits = dayDeposits
		} else {
			t.yearDeposits = t.yearDeposits.Add(ucfg, dayDeposits)
		}
	}

	t.updateQuarter(day, balance)
	t.initialized = true

	return tax
}

func (t *ISKTax) computeTax(ucfg *uncertain.Config, day date.Date) uncertain.Value {
	q0 := t.quarterlyValues[0]
	q1 := t.quarterlyValues[1]
	q2 := t.quarterlyValues[2]
	q3 := t.quarterlyValues[3]
	deposits := t.yearDeposits

	// Look up params for the previous year (the year being taxed)
	prevYear := day.Add(-date.Day) // Dec 31 of previous year
	params := t.ParamsFunc(prevYear)
	schablonRanta := params.SchablonRanta
	fribelopp := params.Fribelopp

	return uncertain.NewMapped(func(cfg *uncertain.Config) float64 {
		qv0 := q0.Sample(cfg) - fribelopp
		qv1 := q1.Sample(cfg) - fribelopp
		qv2 := q2.Sample(cfg) - fribelopp
		qv3 := q3.Sample(cfg) - fribelopp
		if qv0 < 0 {
			qv0 = 0
		}
		if qv1 < 0 {
			qv1 = 0
		}
		if qv2 < 0 {
			qv2 = 0
		}
		if qv3 < 0 {
			qv3 = 0
		}
		dep := 0.0
		if !deposits.Zero() {
			dep = deposits.Sample(cfg)
		}
		kapitalunderlag := (qv0 + qv1 + qv2 + qv3 + dep) / 4.0
		taxAmount := kapitalunderlag * schablonRanta * iskTaxRate
		if taxAmount <= 0 {
			return 0
		}
		return taxAmount
	})
}

func (t *ISKTax) reset() {
	for i := range t.quarterlyValues {
		t.quarterlyValues[i] = uncertain.Value{}
	}
	t.yearDeposits = uncertain.Value{}
}

func (t *ISKTax) updateQuarter(day date.Date, balance uncertain.Value) {
	q := quarterIndex(day.Month())
	t.quarterlyValues[q] = balance
}

func quarterIndex(m time.Month) int {
	switch {
	case m <= time.March:
		return 0
	case m <= time.June:
		return 1
	case m <= time.September:
		return 2
	default:
		return 3
	}
}
