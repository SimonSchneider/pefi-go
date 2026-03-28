package swe

type SalaryBreakdown struct {
	GrossMonthly             float64
	VacationSupplement       float64
	SickPayDeduction         float64
	VABDeduction             float64
	PartialParentalDeduction float64
	AdjustedGross            float64
	Tax                      float64
	NetMonthly               float64
	IsFullParentalLeave      bool
	FKSjukCompensation       float64
}

// CalculateSalaryBreakdown produces an itemized breakdown of a normal (non-full-
// parental-leave) monthly salary. The taxFunc should return the monthly tax for
// a given adjusted gross.
func CalculateSalaryBreakdown(
	grossMonthly float64,
	adj SalaryAdjustmentParams,
	pplSjukDays, pplLagstaDays, pplSkippedDays, prisbasbelopp float64,
	taxFunc func(float64) (float64, error),
) SalaryBreakdown {
	vacation := CalculateVacationPaySupplement(grossMonthly, adj.YearlyVacationDays)
	sick := CalculateSickPayDeduction(grossMonthly, adj.SickDaysPerOccasion, adj.SickOccasionsPerYear, adj.Prisbasbelopp)
	vab := CalculateVABDeduction(grossMonthly, adj.VABDaysPerYear, adj.Prisbasbelopp)
	ppl := CalculatePartialParentalLeaveDeduction(grossMonthly, pplSjukDays, pplLagstaDays, pplSkippedDays, prisbasbelopp)

	adjusted := grossMonthly + vacation - sick - vab - ppl
	tax, _ := taxFunc(adjusted)

	return SalaryBreakdown{
		GrossMonthly:             grossMonthly,
		VacationSupplement:       vacation,
		SickPayDeduction:         sick,
		VABDeduction:             vab,
		PartialParentalDeduction: ppl,
		AdjustedGross:            adjusted,
		Tax:                      tax,
		NetMonthly:               adjusted - tax,
	}
}

// CalculateFullParentalLeaveBreakdown produces a breakdown for a period where
// the employee is on full parental leave (no employer salary, FK compensation only).
func CalculateFullParentalLeaveBreakdown(grossMonthly, sjukDaysPerWeek, prisbasbelopp float64) SalaryBreakdown {
	sjukComp := CalculateFullParentalLeaveCompensation(grossMonthly, sjukDaysPerWeek, prisbasbelopp)
	return SalaryBreakdown{
		GrossMonthly:        grossMonthly,
		IsFullParentalLeave: true,
		FKSjukCompensation:  sjukComp,
		NetMonthly:          sjukComp,
	}
}
