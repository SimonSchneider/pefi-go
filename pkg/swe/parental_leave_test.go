package swe_test

import (
	"testing"

	"github.com/SimonSchneider/pefigo/pkg/swe"
)

func TestCalculatePartialParentalLeaveDeduction(t *testing.T) {
	const pbb = 57300.0

	tests := []struct {
		name              string
		monthlyGross      float64
		sjukDays          float64
		lagstaDays        float64
		skippedWorkDays   float64
		prisbasbelopp     float64
		wantMonthlyApprox float64
	}{
		{
			name:              "zero days returns zero",
			monthlyGross:      50000,
			sjukDays:          0,
			lagstaDays:        0,
			skippedWorkDays:   0,
			prisbasbelopp:     pbb,
			wantMonthlyApprox: 0,
		},
		{
			name:            "below SGI cap, mixed days",
			monthlyGross:    40000,
			sjukDays:        40,
			lagstaDays:      10,
			skippedWorkDays: 50,
			prisbasbelopp:   pbb,
			wantMonthlyApprox: func() float64 {
				annual := 40000.0 * 12
				dailySalary := annual / 260
				salaryLoss := 50 * dailySalary
				sgi := annual * 0.97
				sjukComp := 40 * sgi * 0.80 / 365
				lagstaComp := 10 * 180.0
				return (salaryLoss - sjukComp - lagstaComp) / 12
			}(),
		},
		{
			name:            "above SGI cap",
			monthlyGross:    55000,
			sjukDays:        40,
			lagstaDays:      10,
			skippedWorkDays: 50,
			prisbasbelopp:   pbb,
			wantMonthlyApprox: func() float64 {
				annual := 55000.0 * 12
				dailySalary := annual / 260
				salaryLoss := 50 * dailySalary
				sgi := 10 * pbb
				sjukComp := 40 * sgi * 0.80 / 365
				lagstaComp := 10 * 180.0
				return (salaryLoss - sjukComp - lagstaComp) / 12
			}(),
		},
		{
			name:            "only sjukpenning days, no lagsta",
			monthlyGross:    40000,
			sjukDays:        52,
			lagstaDays:      0,
			skippedWorkDays: 52,
			prisbasbelopp:   pbb,
			wantMonthlyApprox: func() float64 {
				annual := 40000.0 * 12
				dailySalary := annual / 260
				salaryLoss := 52 * dailySalary
				sgi := annual * 0.97
				sjukComp := 52 * sgi * 0.80 / 365
				return (salaryLoss - sjukComp) / 12
			}(),
		},
		{
			name:            "only lagsta days, no sjuk",
			monthlyGross:    40000,
			sjukDays:        0,
			lagstaDays:      52,
			skippedWorkDays: 52,
			prisbasbelopp:   pbb,
			wantMonthlyApprox: func() float64 {
				annual := 40000.0 * 12
				dailySalary := annual / 260
				salaryLoss := 52 * dailySalary
				lagstaComp := 52 * 180.0
				return (salaryLoss - lagstaComp) / 12
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := swe.CalculatePartialParentalLeaveDeduction(tt.monthlyGross, tt.sjukDays, tt.lagstaDays, tt.skippedWorkDays, tt.prisbasbelopp)
			if !approxEqual(got, tt.wantMonthlyApprox, 0.01) {
				t.Errorf("got %v, want ~%v", got, tt.wantMonthlyApprox)
			}
		})
	}
}

func TestCalculatePartialParentalLeaveDeduction_positive(t *testing.T) {
	got := swe.CalculatePartialParentalLeaveDeduction(40000, 40, 10, 50, 57300)
	if got <= 0 {
		t.Errorf("expected positive deduction, got %v", got)
	}
}

func TestCalculateFullParentalLeaveCompensation(t *testing.T) {
	const pbb = 57300.0

	tests := []struct {
		name              string
		monthlyGross      float64
		sjukDaysPerWeek   float64
		prisbasbelopp     float64
		wantMonthlyApprox float64
	}{
		{
			name:            "7 sjuk days/week, below SGI cap",
			monthlyGross:    40000,
			sjukDaysPerWeek: 7,
			prisbasbelopp:   pbb,
			wantMonthlyApprox: func() float64 {
				annual := 40000.0 * 12
				sgi := annual * 0.97
				sjukDaily := sgi * 0.80 / 365
				return 7 * 365 / 7 * sjukDaily / 12
			}(),
		},
		{
			name:              "0 sjuk days/week returns zero",
			monthlyGross:      40000,
			sjukDaysPerWeek:   0,
			prisbasbelopp:     pbb,
			wantMonthlyApprox: 0,
		},
		{
			name:            "4 sjuk days/week, only sjukpenning paid",
			monthlyGross:    40000,
			sjukDaysPerWeek: 4,
			prisbasbelopp:   pbb,
			wantMonthlyApprox: func() float64 {
				annual := 40000.0 * 12
				sgi := annual * 0.97
				sjukDaily := sgi * 0.80 / 365
				sjukDays := 4.0 / 7 * 365
				return sjukDays * sjukDaily / 12
			}(),
		},
		{
			name:            "above SGI cap, 7 sjuk days",
			monthlyGross:    55000,
			sjukDaysPerWeek: 7,
			prisbasbelopp:   pbb,
			wantMonthlyApprox: func() float64 {
				sgi := 10 * pbb
				sjukDaily := sgi * 0.80 / 365
				return 365 * sjukDaily / 12
			}(),
		},
		{
			name:            "above SGI cap, 4 sjuk days",
			monthlyGross:    55000,
			sjukDaysPerWeek: 4,
			prisbasbelopp:   pbb,
			wantMonthlyApprox: func() float64 {
				sgi := 10 * pbb
				sjukDaily := sgi * 0.80 / 365
				sjukDays := 4.0 / 7 * 365
				return sjukDays * sjukDaily / 12
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := swe.CalculateFullParentalLeaveCompensation(tt.monthlyGross, tt.sjukDaysPerWeek, tt.prisbasbelopp)
			if !approxEqual(got, tt.wantMonthlyApprox, 0.01) {
				t.Errorf("got %v, want ~%v", got, tt.wantMonthlyApprox)
			}
		})
	}
}
