package swe

import (
	"context"
	"fmt"
)

type GrossSalaryInput struct {
	GrossMonthly float64
	Kommun       string
	Forsamling   string
	Year         string
	ChurchMember bool
	Column       int
}

type NetSalaryResult struct {
	GrossMonthly float64
	Tax          float64
	NetMonthly   float64
}

// CalculateNetSalary computes net salary from gross input using cached tax table lookups.
func (c *Client) CalculateNetSalary(ctx context.Context, input GrossSalaryInput) (*NetSalaryResult, error) {
	tableNumber, err := c.GetTaxTableNumber(ctx, input.Kommun, input.Forsamling, input.Year, input.ChurchMember)
	if err != nil {
		return nil, fmt.Errorf("getting tax table number: %w", err)
	}

	tax, err := c.LookupTax(ctx, tableNumber, input.Year, input.GrossMonthly, input.Column)
	if err != nil {
		return nil, fmt.Errorf("looking up tax: %w", err)
	}

	return &NetSalaryResult{
		GrossMonthly: input.GrossMonthly,
		Tax:          tax,
		NetMonthly:   input.GrossMonthly - tax,
	}, nil
}
