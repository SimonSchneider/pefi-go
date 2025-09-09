package finance

import (
	"time"

	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/pefigo/internal/uncertain"
)

type ISKTaxModel struct {
	TemplatePercentage uncertain.Value
	TaxRate            uncertain.Value
	DestinationID      string

	accruedBalance  uncertain.Value
	accruedDeposits uncertain.Value

	tax uncertain.Value
}

func (m *ISKTaxModel) Apply(ucfg *uncertain.Config, day date.Date, balance uncertain.Value) {
	t := day.ToStdTime()
	if m.isRecordingDate(t) {
		m.accruedBalance = balance.Add(ucfg, m.accruedBalance)
	}
	if m.IsTaxDay(t) {
		capitalValue := m.accruedBalance.Add(ucfg, m.accruedDeposits).Mul(ucfg, uncertain.NewFixed(0.25))
		templateValue := capitalValue.Mul(ucfg, m.TemplatePercentage)
		m.tax = templateValue.Mul(ucfg, m.TaxRate)

		m.accruedBalance = uncertain.NewFixed(0.0)
		m.accruedDeposits = uncertain.NewFixed(0.0)
	}
}

func (m *ISKTaxModel) ApplyDeposit(ucfg *uncertain.Config, deposit uncertain.Value) {
	m.accruedDeposits = deposit.Add(ucfg, m.accruedDeposits)
}

func (m *ISKTaxModel) IsTaxDay(stdTime time.Time) bool {
	return stdTime.Month() == time.December && stdTime.Day() == 31
}

func (m *ISKTaxModel) GetAndResetTax() uncertain.Value {
	tax := m.tax
	m.tax = uncertain.NewFixed(0.0)
	return tax
}

func (m *ISKTaxModel) isRecordingDate(stdTime time.Time) bool {
	return ((stdTime.Month() == time.January && stdTime.Day() == 1) ||
		(stdTime.Month() == time.April && stdTime.Day() == 1) ||
		(stdTime.Month() == time.July && stdTime.Day() == 1) ||
		(stdTime.Month() == time.October && stdTime.Day() == 1))
}
