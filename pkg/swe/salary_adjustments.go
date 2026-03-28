package swe

import "math"

const (
	WorkingDaysPerYear  = 260
	WorkingHoursPerYear = 2080
)

// CalculateVacationPaySupplement returns the monthly vacation pay supplement
// (semesterdagstillägg). Each vacation day adds 0.8% of monthly salary,
// averaged over 12 months.
func CalculateVacationPaySupplement(monthlyGross float64, yearlyVacationDays float64) float64 {
	return monthlyGross * 0.008 * yearlyVacationDays / 12
}

// CalculateSickPayDeduction returns the monthly salary deduction due to sick leave.
// Sjuklön rules: 80% of salary (capped at 10*PBB annually), 1 karensdag per occasion.
func CalculateSickPayDeduction(monthlyGross, sickDaysPerOccasion, sickOccasionsPerYear, prisbasbelopp float64) float64 {
	if sickOccasionsPerYear == 0 || sickDaysPerOccasion == 0 {
		return 0
	}
	annualSalary := monthlyGross * 12
	cappedAnnual := math.Min(annualSalary, 10*prisbasbelopp)
	dailySalary := annualSalary / WorkingDaysPerYear
	cappedDaily := cappedAnnual / WorkingDaysPerYear

	karensDays := sickOccasionsPerYear
	sjuklonDays := sickOccasionsPerYear * (sickDaysPerOccasion - 1)

	annualLoss := karensDays*dailySalary + sjuklonDays*(dailySalary-cappedDaily*0.8)
	return annualLoss / 12
}

// CalculateVABDeduction returns the monthly net salary deduction due to VAB.
// The employer doesn't pay during VAB, but Försäkringskassan compensates at
// ~80% of SGI (with 3% schablonavdrag, capped at 10*PBB).
func CalculateVABDeduction(monthlyGross, vabDaysPerYear, prisbasbelopp float64) float64 {
	if vabDaysPerYear == 0 {
		return 0
	}
	annualSalary := monthlyGross * 12
	dailySalary := annualSalary / WorkingDaysPerYear
	annualSalaryLoss := vabDaysPerYear * dailySalary

	sgi := math.Min(annualSalary*0.97, 10*prisbasbelopp)
	dailyFK := sgi * 0.80 / 365
	annualFKCompensation := vabDaysPerYear * dailyFK

	return (annualSalaryLoss - annualFKCompensation) / 12
}

type SalaryAdjustmentParams struct {
	YearlyVacationDays   float64
	SickDaysPerOccasion  float64
	SickOccasionsPerYear float64
	VABDaysPerYear       float64
	Prisbasbelopp        float64
}

// AdjustGrossSalary applies vacation supplement, sick pay deduction, and VAB
// deduction to a monthly gross salary, returning the adjusted monthly gross.
func AdjustGrossSalary(monthlyGross float64, params SalaryAdjustmentParams) float64 {
	vacation := CalculateVacationPaySupplement(monthlyGross, params.YearlyVacationDays)
	sick := CalculateSickPayDeduction(monthlyGross, params.SickDaysPerOccasion, params.SickOccasionsPerYear, params.Prisbasbelopp)
	vab := CalculateVABDeduction(monthlyGross, params.VABDaysPerYear, params.Prisbasbelopp)
	return monthlyGross + vacation - sick - vab
}
