package swe

import "math"

const LagstaNivaPerDay = 180.0

// CalculatePartialParentalLeaveDeduction returns the monthly salary deduction
// for partial parental leave. The parent works most days but takes some
// föräldraledighet days per year. Salary is lost for skipped work days;
// Försäkringskassan compensates sjukpenning days at ~80% of SGI and
// lägstanivå days at a fixed 180 kr/day.
func CalculatePartialParentalLeaveDeduction(monthlyGross, sjukDaysPerYear, lagstaDaysPerYear, skippedWorkDaysPerYear, prisbasbelopp float64) float64 {
	if skippedWorkDaysPerYear == 0 && sjukDaysPerYear == 0 && lagstaDaysPerYear == 0 {
		return 0
	}
	annualSalary := monthlyGross * 12
	dailySalary := annualSalary / WorkingDaysPerYear
	salaryLoss := skippedWorkDaysPerYear * dailySalary

	sgi := math.Min(annualSalary*0.97, 10*prisbasbelopp)
	sjukComp := sjukDaysPerYear * sgi * 0.80 / 365
	lagstaComp := lagstaDaysPerYear * LagstaNivaPerDay

	return (salaryLoss - sjukComp - lagstaComp) / 12
}

// CalculateFullParentalLeaveCompensation returns the monthly compensation
// received during full (100%) parental leave. No salary is paid by the
// employer; instead Försäkringskassan pays föräldrapenning at
// sjukpenningnivå (~80% of SGI) for sjukDaysPerWeek days per week.
func CalculateFullParentalLeaveCompensation(monthlyGross, sjukDaysPerWeek, prisbasbelopp float64) float64 {
	annualSalary := monthlyGross * 12
	sgi := math.Min(annualSalary*0.97, 10*prisbasbelopp)
	sjukDailyRate := sgi * 0.80 / 365

	sjukDaysPerYear := sjukDaysPerWeek / 7 * 365
	return sjukDaysPerYear * sjukDailyRate / 12
}
