package swe_test

import (
	"math"
	"testing"

	"github.com/SimonSchneider/pefigo/internal/swe"
)

func approxEqual(a, b, epsilon float64) bool {
	return math.Abs(a-b) < epsilon
}

func TestCalculateVacationPaySupplement(t *testing.T) {
	tests := []struct {
		name         string
		monthlyGross float64
		vacationDays float64
		want         float64
	}{
		{
			name:         "25 vacation days at 50000 gross",
			monthlyGross: 50000,
			vacationDays: 25,
			want:         50000 * 0.008 * 25 / 12,
		},
		{
			name:         "30 vacation days at 40000 gross",
			monthlyGross: 40000,
			vacationDays: 30,
			want:         40000 * 0.008 * 30 / 12,
		},
		{
			name:         "zero vacation days",
			monthlyGross: 50000,
			vacationDays: 0,
			want:         0,
		},
		{
			name:         "zero salary",
			monthlyGross: 0,
			vacationDays: 25,
			want:         0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := swe.CalculateVacationPaySupplement(tt.monthlyGross, tt.vacationDays)
			if got != tt.want {
				t.Errorf("CalculateVacationPaySupplement(%v, %v) = %v, want %v", tt.monthlyGross, tt.vacationDays, got, tt.want)
			}
		})
	}
}

func TestCalculateSickPayDeduction(t *testing.T) {
	const pbb = 57300.0

	tests := []struct {
		name              string
		monthlyGross      float64
		sickDaysPerOcc    float64
		sickOccsPerYear   float64
		prisbasbelopp     float64
		wantMonthlyApprox float64
	}{
		{
			name:              "zero occasions",
			monthlyGross:      50000,
			sickDaysPerOcc:    3,
			sickOccsPerYear:   0,
			prisbasbelopp:     pbb,
			wantMonthlyApprox: 0,
		},
		{
			name:              "zero sick days per occasion",
			monthlyGross:      50000,
			sickDaysPerOcc:    0,
			sickOccsPerYear:   4,
			prisbasbelopp:     pbb,
			wantMonthlyApprox: 0,
		},
		{
			name:            "salary below PBB cap, 4 occasions x 3 days",
			monthlyGross:    40000,
			sickDaysPerOcc:  3,
			sickOccsPerYear: 4,
			prisbasbelopp:   pbb,
			wantMonthlyApprox: func() float64 {
				annual := 40000.0 * 12
				daily := annual / 260
				// below cap: cappedDaily == daily
				karensDays := 4.0
				sjuklonDays := 4.0 * 2.0
				annualLoss := karensDays*daily + sjuklonDays*(daily-daily*0.8)
				return annualLoss / 12
			}(),
		},
		{
			name:            "salary above PBB cap",
			monthlyGross:    55000,
			sickDaysPerOcc:  3,
			sickOccsPerYear: 4,
			prisbasbelopp:   pbb,
			wantMonthlyApprox: func() float64 {
				annual := 55000.0 * 12
				capped := 10 * pbb
				daily := annual / 260
				cappedDaily := capped / 260
				karensDays := 4.0
				sjuklonDays := 4.0 * 2.0
				annualLoss := karensDays*daily + sjuklonDays*(daily-cappedDaily*0.8)
				return annualLoss / 12
			}(),
		},
		{
			name:            "1 day per occasion means only karensdag",
			monthlyGross:    50000,
			sickDaysPerOcc:  1,
			sickOccsPerYear: 6,
			prisbasbelopp:   pbb,
			wantMonthlyApprox: func() float64 {
				annual := 50000.0 * 12
				daily := annual / 260
				return 6 * daily / 12
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := swe.CalculateSickPayDeduction(tt.monthlyGross, tt.sickDaysPerOcc, tt.sickOccsPerYear, tt.prisbasbelopp)
			if !approxEqual(got, tt.wantMonthlyApprox, 0.01) {
				t.Errorf("got %v, want ~%v", got, tt.wantMonthlyApprox)
			}
		})
	}
}

func TestCalculateVABDeduction(t *testing.T) {
	const pbb = 57300.0

	tests := []struct {
		name              string
		monthlyGross      float64
		vabDaysPerYear    float64
		prisbasbelopp     float64
		wantMonthlyApprox float64
	}{
		{
			name:              "zero VAB days",
			monthlyGross:      50000,
			vabDaysPerYear:    0,
			prisbasbelopp:     pbb,
			wantMonthlyApprox: 0,
		},
		{
			name:           "10 VAB days, salary below SGI cap",
			monthlyGross:   40000,
			vabDaysPerYear: 10,
			prisbasbelopp:  pbb,
			wantMonthlyApprox: func() float64 {
				annual := 40000.0 * 12
				dailySalary := annual / 260
				salaryLoss := 10 * dailySalary
				sgi := annual * 0.97 // below 10*PBB
				dailyFK := sgi * 0.80 / 365
				fkComp := 10 * dailyFK
				return (salaryLoss - fkComp) / 12
			}(),
		},
		{
			name:           "10 VAB days, salary above SGI cap",
			monthlyGross:   55000,
			vabDaysPerYear: 10,
			prisbasbelopp:  pbb,
			wantMonthlyApprox: func() float64 {
				annual := 55000.0 * 12
				dailySalary := annual / 260
				salaryLoss := 10 * dailySalary
				sgi := 10 * pbb // capped
				dailyFK := sgi * 0.80 / 365
				fkComp := 10 * dailyFK
				return (salaryLoss - fkComp) / 12
			}(),
		},
		{
			name:           "FK compensation is less than salary for high earner",
			monthlyGross:   60000,
			vabDaysPerYear: 20,
			prisbasbelopp:  pbb,
			wantMonthlyApprox: func() float64 {
				annual := 60000.0 * 12
				dailySalary := annual / 260
				salaryLoss := 20 * dailySalary
				sgi := 10 * pbb // capped
				dailyFK := sgi * 0.80 / 365
				fkComp := 20 * dailyFK
				return (salaryLoss - fkComp) / 12
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := swe.CalculateVABDeduction(tt.monthlyGross, tt.vabDaysPerYear, tt.prisbasbelopp)
			if !approxEqual(got, tt.wantMonthlyApprox, 0.01) {
				t.Errorf("got %v, want ~%v", got, tt.wantMonthlyApprox)
			}
		})
	}
}

func TestCalculateVABDeduction_positive(t *testing.T) {
	got := swe.CalculateVABDeduction(40000, 10, 57300)
	if got <= 0 {
		t.Errorf("expected positive deduction, got %v", got)
	}
}

func TestAdjustGrossSalary(t *testing.T) {
	const pbb = 57300.0
	const gross = 50000.0

	t.Run("no adjustments", func(t *testing.T) {
		got := swe.AdjustGrossSalary(gross, swe.SalaryAdjustmentParams{})
		if got != gross {
			t.Errorf("got %v, want %v", got, gross)
		}
	})

	t.Run("only vacation", func(t *testing.T) {
		params := swe.SalaryAdjustmentParams{YearlyVacationDays: 25}
		got := swe.AdjustGrossSalary(gross, params)
		wantVacation := swe.CalculateVacationPaySupplement(gross, 25)
		if !approxEqual(got, gross+wantVacation, 0.01) {
			t.Errorf("got %v, want ~%v", got, gross+wantVacation)
		}
	})

	t.Run("all adjustments combined", func(t *testing.T) {
		params := swe.SalaryAdjustmentParams{
			YearlyVacationDays:   25,
			SickDaysPerOccasion:  3,
			SickOccasionsPerYear: 4,
			VABDaysPerYear:       10,
			Prisbasbelopp:        pbb,
		}
		got := swe.AdjustGrossSalary(gross, params)
		vacation := swe.CalculateVacationPaySupplement(gross, 25)
		sick := swe.CalculateSickPayDeduction(gross, 3, 4, pbb)
		vab := swe.CalculateVABDeduction(gross, 10, pbb)
		want := gross + vacation - sick - vab
		if !approxEqual(got, want, 0.01) {
			t.Errorf("got %v, want ~%v", got, want)
		}
	})

	t.Run("result is less than gross when sick and VAB dominate", func(t *testing.T) {
		params := swe.SalaryAdjustmentParams{
			SickDaysPerOccasion:  5,
			SickOccasionsPerYear: 12,
			VABDaysPerYear:       30,
			Prisbasbelopp:        pbb,
		}
		got := swe.AdjustGrossSalary(gross, params)
		if got >= gross {
			t.Errorf("expected adjusted (%v) < gross (%v)", got, gross)
		}
	})
}
