package swe

import (
	"context"
	"fmt"
	"strconv"
)

type GrossSalaryInput struct {
	Kommun       string
	Forsamling   string
	Year         string
	ChurchMember bool
	Column       int
}

type GrossSalaryInputWithAmount struct {
	GrossSalaryInput
	GrossMonthly float64
}

type NetSalaryResult struct {
	GrossMonthly float64
	Tax          float64
	NetMonthly   float64
}

const maxYearFallback = 5

func (c *Client) NetSalaryCalculator(ctx context.Context, input GrossSalaryInput) (func(grossMonthly float64) (*NetSalaryResult, error), error) {
	year, err := strconv.Atoi(input.Year)
	if err != nil {
		return nil, fmt.Errorf("parsing year %q: %w", input.Year, err)
	}

	var tableNumber int
	var taxLookup func(float64) (float64, error)
	var lastErr error

	for attempts := 0; attempts <= maxYearFallback; attempts++ {
		tryYear := strconv.Itoa(year - attempts)
		tableNumber, lastErr = c.GetTaxTableNumber(ctx, input.Kommun, input.Forsamling, tryYear, input.ChurchMember)
		if lastErr != nil {
			continue
		}
		taxLookup, lastErr = c.NewTaxLookup(ctx, tableNumber, tryYear, input.Column)
		if lastErr != nil {
			continue
		}
		break
	}
	if lastErr != nil {
		return nil, fmt.Errorf("no tax data available for %s or previous %d years: %w", input.Year, maxYearFallback, lastErr)
	}

	return func(grossMonthly float64) (*NetSalaryResult, error) {
		tax, err := taxLookup(grossMonthly)
		if err != nil {
			return nil, fmt.Errorf("looking up tax: %w", err)
		}
		return &NetSalaryResult{
			GrossMonthly: grossMonthly,
			Tax:          tax,
			NetMonthly:   grossMonthly - tax,
		}, nil
	}, nil
}

// CalculateNetSalary computes net salary from gross input using cached tax table lookups.
func (c *Client) CalculateNetSalary(ctx context.Context, input GrossSalaryInputWithAmount) (*NetSalaryResult, error) {
	calculator, err := c.NetSalaryCalculator(ctx, input.GrossSalaryInput)
	if err != nil {
		return nil, fmt.Errorf("creating net salary calculator: %w", err)
	}
	return calculator(input.GrossMonthly)
}
