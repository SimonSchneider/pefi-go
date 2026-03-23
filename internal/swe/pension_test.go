package swe_test

import (
	"math"
	"testing"

	"github.com/SimonSchneider/pefigo/internal/swe"
)

func TestCalculateITP1Pension(t *testing.T) {
	const ibb = 76200.0
	cutoffYearly := ibb * 7.5  // 571500
	cutoffMonthly := cutoffYearly / 12 // 47625

	tests := []struct {
		name          string
		grossMonthly  float64
		ibb           float64
		wantPension   float64
	}{
		{
			name:         "zero salary",
			grossMonthly: 0,
			ibb:          ibb,
			wantPension:  0,
		},
		{
			name:         "below cutoff",
			grossMonthly: 40000,
			ibb:          ibb,
			wantPension:  40000 * 0.045,
		},
		{
			name:         "at cutoff exactly",
			grossMonthly: cutoffMonthly,
			ibb:          ibb,
			wantPension:  cutoffMonthly * 0.045,
		},
		{
			name:         "above cutoff",
			grossMonthly: 50000,
			ibb:          ibb,
			wantPension:  cutoffMonthly*0.045 + (50000-cutoffMonthly)*0.3,
		},
		{
			name:         "well above cutoff",
			grossMonthly: 100000,
			ibb:          ibb,
			wantPension:  cutoffMonthly*0.045 + (100000-cutoffMonthly)*0.3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := swe.CalculateITP1Pension(tt.grossMonthly, tt.ibb)
			if math.Abs(got-tt.wantPension) > 0.01 {
				t.Errorf("CalculateITP1Pension(%v, %v) = %v, want %v", tt.grossMonthly, tt.ibb, got, tt.wantPension)
			}
		})
	}
}
