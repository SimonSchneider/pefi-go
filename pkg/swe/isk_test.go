package swe_test

import (
	"testing"
	"time"

	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/pefigo/pkg/swe"
	"github.com/SimonSchneider/pefigo/pkg/uncertain"
)

func TestISKTax_FixedBalance(t *testing.T) {
	ucfg := uncertain.NewConfig(time.Now().UnixMilli(), 1)
	isk := &swe.ISKTax{
		ParamsFunc: func(d date.Date) swe.ISKParams { return swe.ISKParams{SchablonRanta: 0.0125} },
	}

	balance := uncertain.NewFixed(1_000_000)
	start := mustParseDate("2000-01-01")
	end := mustParseDate("2001-01-02")
	var totalTax float64
	for day := range date.Iter(start, end, date.Day) {
		tax := isk.Apply(ucfg, day, balance, uncertain.NewFixed(0))
		totalTax += tax.Mean()
	}
	// Expected: 1_000_000 * 4 (quarterly values all 1M) + 0 deposits / 4 = 1_000_000
	// Tax: 1_000_000 * 0.0125 * 0.30 = 3750
	expected := 3750.0
	if totalTax < expected*0.99 || totalTax > expected*1.01 {
		t.Errorf("total ISK tax = %f, expected %f", totalTax, expected)
	}
}

func TestISKTax_MultiYear(t *testing.T) {
	ucfg := uncertain.NewConfig(time.Now().UnixMilli(), 1)
	isk := &swe.ISKTax{
		ParamsFunc: func(d date.Date) swe.ISKParams { return swe.ISKParams{SchablonRanta: 0.0125} },
	}

	balance := uncertain.NewFixed(1_000_000)
	start := mustParseDate("2000-01-01")
	end := mustParseDate("2003-01-02")
	var totalTax float64
	for day := range date.Iter(start, end, date.Day) {
		tax := isk.Apply(ucfg, day, balance, uncertain.NewFixed(0))
		totalTax += tax.Mean()
	}
	// 3 years of tax: 3750 * 3 = 11250
	expected := 11250.0
	if totalTax < expected*0.99 || totalTax > expected*1.01 {
		t.Errorf("total ISK tax over 3 years = %f, expected %f", totalTax, expected)
	}
}

func TestISKTax_VaryingSchablonRanta(t *testing.T) {
	ucfg := uncertain.NewConfig(time.Now().UnixMilli(), 1)
	isk := &swe.ISKTax{
		ParamsFunc: func(d date.Date) swe.ISKParams {
			if d.Year() <= 2000 {
				return swe.ISKParams{SchablonRanta: 0.0125}
			}
			return swe.ISKParams{SchablonRanta: 0.0200}
		},
	}

	balance := uncertain.NewFixed(1_000_000)
	start := mustParseDate("2000-01-01")
	end := mustParseDate("2002-01-02")
	var totalTax float64
	for day := range date.Iter(start, end, date.Day) {
		tax := isk.Apply(ucfg, day, balance, uncertain.NewFixed(0))
		totalTax += tax.Mean()
	}
	// Year 2000 (tax on 2001-01-01): 1M * 0.0125 * 0.30 = 3750
	// Year 2001 (tax on 2002-01-01): 1M * 0.0200 * 0.30 = 6000
	expected := 9750.0
	if totalTax < expected*0.99 || totalTax > expected*1.01 {
		t.Errorf("total ISK tax with varying rates = %f, expected %f", totalTax, expected)
	}
}

func TestISKTax_WithDeposits(t *testing.T) {
	ucfg := uncertain.NewConfig(time.Now().UnixMilli(), 1)
	isk := &swe.ISKTax{
		ParamsFunc: func(d date.Date) swe.ISKParams { return swe.ISKParams{SchablonRanta: 0.0125} },
	}

	balance := uncertain.NewFixed(1_000_000)
	start := mustParseDate("2000-01-01")
	end := mustParseDate("2001-01-02")
	var totalTax float64
	for day := range date.Iter(start, end, date.Day) {
		deposit := uncertain.NewFixed(0)
		if day.Day() == 15 {
			deposit = uncertain.NewFixed(10_000)
		}
		tax := isk.Apply(ucfg, day, balance, deposit)
		totalTax += tax.Mean()
	}
	// kapitalunderlag = (Q1 + Q2 + Q3 + Q4 + deposits) / 4
	// Q1-Q4 all = 1M (balance doesn't change in this test), deposits = 12 * 10k = 120k
	// kapitalunderlag = (4*1M + 120k) / 4 = 1_030_000
	// Tax: 1_030_000 * 0.0125 * 0.30 = 3862.5
	expected := 3862.5
	if totalTax < expected*0.99 || totalTax > expected*1.01 {
		t.Errorf("total ISK tax with deposits = %f, expected %f", totalTax, expected)
	}
}

func TestISKTax_QuarterlyAveraging(t *testing.T) {
	ucfg := uncertain.NewConfig(time.Now().UnixMilli(), 1)
	isk := &swe.ISKTax{
		ParamsFunc: func(d date.Date) swe.ISKParams { return swe.ISKParams{SchablonRanta: 0.0125} },
	}

	start := mustParseDate("2000-01-01")
	end := mustParseDate("2001-01-02")
	var totalTax float64
	for day := range date.Iter(start, end, date.Day) {
		var balance uncertain.Value
		switch {
		case day.Month() <= 3:
			balance = uncertain.NewFixed(100_000)
		case day.Month() <= 6:
			balance = uncertain.NewFixed(200_000)
		case day.Month() <= 9:
			balance = uncertain.NewFixed(300_000)
		default:
			balance = uncertain.NewFixed(400_000)
		}
		tax := isk.Apply(ucfg, day, balance, uncertain.NewFixed(0))
		totalTax += tax.Mean()
	}
	// Q1 final value (March 31) = 100k, Q2 (June 30) = 200k, Q3 (Sep 30) = 300k, Q4 (Dec 31) = 400k
	// kapitalunderlag = (100k + 200k + 300k + 400k + 0) / 4 = 250k
	// Tax: 250k * 0.0125 * 0.30 = 937.5
	expected := 937.5
	if totalTax < expected*0.99 || totalTax > expected*1.01 {
		t.Errorf("total ISK tax with quarterly averaging = %f, expected %f", totalTax, expected)
	}
}

func TestISKTax_Fribelopp(t *testing.T) {
	ucfg := uncertain.NewConfig(time.Now().UnixMilli(), 1)
	isk := &swe.ISKTax{
		ParamsFunc: func(d date.Date) swe.ISKParams {
			return swe.ISKParams{SchablonRanta: 0.0125, Fribelopp: 300}
		},
	}

	balance := uncertain.NewFixed(1_000_000)
	start := mustParseDate("2000-01-01")
	end := mustParseDate("2001-01-02")
	var totalTax float64
	for day := range date.Iter(start, end, date.Day) {
		tax := isk.Apply(ucfg, day, balance, uncertain.NewFixed(0))
		totalTax += tax.Mean()
	}
	// Each quarter value = 1M, deduct 300 from each: (999700 + 999700 + 999700 + 999700 + 0) / 4 = 999700
	// Tax: 999700 * 0.0125 * 0.30 = 3748.875
	expected := 3748.875
	if totalTax < expected*0.99 || totalTax > expected*1.01 {
		t.Errorf("total ISK tax with fribelopp = %f, expected %f", totalTax, expected)
	}
}

func TestISKTax_FribeloppReducesToZero(t *testing.T) {
	ucfg := uncertain.NewConfig(time.Now().UnixMilli(), 1)
	isk := &swe.ISKTax{
		ParamsFunc: func(d date.Date) swe.ISKParams {
			return swe.ISKParams{SchablonRanta: 0.0125, Fribelopp: 500}
		},
	}

	balance := uncertain.NewFixed(200)
	start := mustParseDate("2000-01-01")
	end := mustParseDate("2001-01-02")
	var totalTax float64
	for day := range date.Iter(start, end, date.Day) {
		tax := isk.Apply(ucfg, day, balance, uncertain.NewFixed(0))
		totalTax += tax.Mean()
	}
	// Each quarter = 200, deduct 500 -> each goes to 0 (clamped)
	// kapitalunderlag = 0, no tax
	if totalTax != 0 {
		t.Errorf("expected 0 tax when fribelopp exceeds balance, got %f", totalTax)
	}
}

func mustParseDate(s string) date.Date {
	d, err := date.ParseDate(s)
	if err != nil {
		panic(err)
	}
	return d
}
