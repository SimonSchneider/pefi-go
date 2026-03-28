package swe

import (
	"context"
	"fmt"
	"strconv"
	"time"
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

func (c *Client) NetSalaryCalculator(ctx context.Context, input GrossSalaryInput) (func(grossMonthly float64) (*NetSalaryResult, error), error) {
	year, err := strconv.Atoi(input.Year)
	if err != nil {
		return nil, fmt.Errorf("parsing year %q: %w", input.Year, err)
	}

	effectiveYear := min(year, time.Now().Year())

	var tableNumber int
	var taxLookup func(float64) (float64, error)

	tryYear := strconv.Itoa(effectiveYear)
	tableNumber, err = c.GetTaxTableNumber(ctx, input.Kommun, input.Forsamling, tryYear, input.ChurchMember)
	if err != nil {
		tryYear = strconv.Itoa(effectiveYear - 1)
		tableNumber, err = c.GetTaxTableNumber(ctx, input.Kommun, input.Forsamling, tryYear, input.ChurchMember)
		if err != nil {
			return nil, fmt.Errorf("no tax data available for %d or %d: %w", effectiveYear, effectiveYear-1, err)
		}
	}
	taxLookup, err = c.NewTaxLookup(ctx, tableNumber, tryYear, input.Column)
	if err != nil {
		return nil, fmt.Errorf("fetching tax table %d for %s: %w", tableNumber, tryYear, err)
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
