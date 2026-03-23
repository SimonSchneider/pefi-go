package swe

import "math"

// CalculateITP1Pension computes the monthly ITP1 pension contribution.
// Below 7.5 inkomstbasbelopp (IBB): 4.5% of salary.
// Above 7.5 IBB: 30% of the excess.
func CalculateITP1Pension(grossMonthlySalary, inkomstbasbelopp float64) float64 {
	cutoffMonthly := inkomstbasbelopp * 7.5 / 12
	belowCutoff := math.Min(grossMonthlySalary, cutoffMonthly)
	aboveCutoff := math.Max(0, grossMonthlySalary-cutoffMonthly)
	return belowCutoff*0.045 + aboveCutoff*0.3
}
