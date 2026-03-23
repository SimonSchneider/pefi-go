package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/goslu/sid"
	"github.com/SimonSchneider/pefigo/internal/pdb"
	"github.com/SimonSchneider/pefigo/internal/ui"
	"github.com/SimonSchneider/pefigo/internal/uncertain"
)

type Salary struct {
	ID               string
	Name             string
	ToAccountID      string
	Priority         int64
	Recurrence       date.Cron
	BudgetCategoryID *string
	Enabled          bool
	Amounts          []SalaryAmount
}

type SalaryAmount struct {
	ID        string
	SalaryID  string
	Amount    uncertain.Value
	StartDate date.Date
}

func (s Salary) GenerateTransferTemplates() []TransferTemplate {
	sorted := make([]SalaryAmount, len(s.Amounts))
	copy(sorted, s.Amounts)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].StartDate < sorted[j].StartDate
	})
	var templates []TransferTemplate
	for i, amt := range sorted {
		var endDate *date.Date
		if i+1 < len(sorted) {
			ed := sorted[i+1].StartDate
			endDate = &ed
		}
		templates = append(templates, TransferTemplate{
			ID:               "salary:" + amt.ID,
			Name:             s.Name,
			FromAccountID:    "",
			ToAccountID:      s.ToAccountID,
			AmountType:       "fixed",
			AmountFixed:      amt.Amount,
			Priority:         s.Priority,
			Recurrence:       s.Recurrence,
			StartDate:        amt.StartDate,
			EndDate:          endDate,
			Enabled:          s.Enabled,
			BudgetCategoryID: s.BudgetCategoryID,
			Source: TransferTemplateSource{
				Type:     "salary",
				EntityID: s.ID,
				Label:    s.Name,
				EditURL:  "/salaries/" + s.ID + "/edit",
			},
		})
	}
	return templates
}

func salaryFromDB(s pdb.Salary) Salary {
	return Salary{
		ID:               s.ID,
		Name:             s.Name,
		ToAccountID:      ui.OrDefault(s.ToAccountID),
		Priority:         s.Priority,
		Recurrence:       date.Cron(s.Recurrence),
		BudgetCategoryID: s.BudgetCategoryID,
		Enabled:          s.Enabled,
	}
}

func salaryAmountFromDB(a pdb.SalaryAmount) (SalaryAmount, error) {
	var amount uncertain.Value
	if err := amount.Decode(a.Amount); err != nil {
		return SalaryAmount{}, fmt.Errorf("decoding salary amount: %w", err)
	}
	return SalaryAmount{
		ID:        a.ID,
		SalaryID:  a.SalaryID,
		Amount:    amount,
		StartDate: date.Date(a.StartDate),
	}, nil
}

func (s *Service) UpsertSalary(ctx context.Context, inp Salary) (Salary, error) {
	if inp.ID == "" {
		inp.ID = sid.MustNewString(32)
	}
	now := time.Now().Unix()
	sal, err := pdb.New(s.db).UpsertSalary(ctx, pdb.UpsertSalaryParams{
		ID:               inp.ID,
		Name:             inp.Name,
		ToAccountID:      ui.WithDefaultNull(inp.ToAccountID),
		Priority:         inp.Priority,
		Recurrence:       string(inp.Recurrence),
		BudgetCategoryID: inp.BudgetCategoryID,
		Enabled:          inp.Enabled,
		CreatedAt:        now,
		UpdatedAt:        now,
	})
	if err != nil {
		return Salary{}, fmt.Errorf("upserting salary: %w", err)
	}
	return salaryFromDB(sal), nil
}

func (s *Service) ListSalaries(ctx context.Context) ([]Salary, error) {
	rows, err := pdb.New(s.db).ListSalaries(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing salaries: %w", err)
	}
	salaries := make([]Salary, 0, len(rows))
	for _, r := range rows {
		salaries = append(salaries, salaryFromDB(r))
	}
	return salaries, nil
}

func (s *Service) GetSalary(ctx context.Context, id string) (Salary, error) {
	sal, err := pdb.New(s.db).GetSalary(ctx, id)
	if err != nil {
		return Salary{}, fmt.Errorf("getting salary: %w", err)
	}
	result := salaryFromDB(sal)
	amounts, err := s.ListSalaryAmounts(ctx, id)
	if err != nil {
		return Salary{}, fmt.Errorf("listing salary amounts: %w", err)
	}
	result.Amounts = amounts
	return result, nil
}

func (s *Service) DeleteSalary(ctx context.Context, id string) error {
	if err := pdb.New(s.db).DeleteSalary(ctx, id); err != nil {
		return fmt.Errorf("deleting salary: %w", err)
	}
	return nil
}

func (s *Service) UpsertSalaryAmount(ctx context.Context, inp SalaryAmount) (SalaryAmount, error) {
	if inp.ID == "" {
		inp.ID = sid.MustNewString(32)
	}
	encoded, err := inp.Amount.Encode()
	if err != nil {
		return SalaryAmount{}, fmt.Errorf("encoding amount: %w", err)
	}
	now := time.Now().Unix()
	a, err := pdb.New(s.db).UpsertSalaryAmount(ctx, pdb.UpsertSalaryAmountParams{
		ID:        inp.ID,
		SalaryID:  inp.SalaryID,
		Amount:    encoded,
		StartDate: int64(inp.StartDate),
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return SalaryAmount{}, fmt.Errorf("upserting salary amount: %w", err)
	}
	return salaryAmountFromDB(a)
}

func (s *Service) ListSalaryAmounts(ctx context.Context, salaryID string) ([]SalaryAmount, error) {
	rows, err := pdb.New(s.db).ListSalaryAmounts(ctx, salaryID)
	if err != nil {
		return nil, fmt.Errorf("listing salary amounts: %w", err)
	}
	amounts := make([]SalaryAmount, 0, len(rows))
	for _, r := range rows {
		a, err := salaryAmountFromDB(r)
		if err != nil {
			return nil, err
		}
		amounts = append(amounts, a)
	}
	return amounts, nil
}

func (s *Service) DeleteSalaryAmount(ctx context.Context, id string) error {
	if err := pdb.New(s.db).DeleteSalaryAmount(ctx, id); err != nil {
		return fmt.Errorf("deleting salary amount: %w", err)
	}
	return nil
}

func (sa SalaryAmount) GetStartDateString() string {
	if sa.ID == "" {
		return ""
	}
	return sa.StartDate.String()
}

func (sa SalaryAmount) GetAmountString() string {
	if sa.ID == "" {
		return ""
	}
	return sa.Amount.SimpleEncode()
}

func (s Salary) CurrentAmount() float64 {
	today := date.Today()
	var current *SalaryAmount
	for i := range s.Amounts {
		if s.Amounts[i].StartDate <= today {
			if current == nil || s.Amounts[i].StartDate > current.StartDate {
				current = &s.Amounts[i]
			}
		}
	}
	if current == nil {
		return 0
	}
	return current.Amount.Mean()
}

type SalaryEditView struct {
	Salary     Salary
	Accounts   []Account
	Categories []TransferTemplateCategory
}

func (v SalaryEditView) IsEdit() bool {
	return v.Salary.ID != ""
}

func (s *Service) GetSalariesPageData(ctx context.Context) ([]Salary, error) {
	salaries, err := s.ListSalaries(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing salaries: %w", err)
	}
	allAmounts, err := pdb.New(s.db).ListAllSalaryAmounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing salary amounts: %w", err)
	}
	amountsBySalary := make(map[string][]SalaryAmount)
	for _, a := range allAmounts {
		parsed, err := salaryAmountFromDB(a)
		if err != nil {
			return nil, err
		}
		amountsBySalary[a.SalaryID] = append(amountsBySalary[a.SalaryID], parsed)
	}
	for i := range salaries {
		salaries[i].Amounts = amountsBySalary[salaries[i].ID]
	}
	return salaries, nil
}

func (s *Service) GetSalaryNewPageData(ctx context.Context) (*SalaryEditView, error) {
	accs, err := s.ListAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing accounts: %w", err)
	}
	categories, err := s.ListCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing categories: %w", err)
	}
	return &SalaryEditView{
		Accounts:   accs,
		Categories: categories,
	}, nil
}

func (s *Service) GetSalaryEditPageData(ctx context.Context, id string) (*SalaryEditView, error) {
	sal, err := s.GetSalary(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting salary: %w", err)
	}
	accs, err := s.ListAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing accounts: %w", err)
	}
	categories, err := s.ListCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing categories: %w", err)
	}
	return &SalaryEditView{
		Salary:     sal,
		Accounts:   accs,
		Categories: categories,
	}, nil
}

func (s *Service) generateSalaryTransferTemplates(ctx context.Context) ([]TransferTemplate, error) {
	salaries, err := pdb.New(s.db).ListSalaries(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing salaries: %w", err)
	}
	allAmounts, err := pdb.New(s.db).ListAllSalaryAmounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing salary amounts: %w", err)
	}
	amountsBySalary := make(map[string][]SalaryAmount)
	for _, a := range allAmounts {
		parsed, err := salaryAmountFromDB(a)
		if err != nil {
			return nil, err
		}
		amountsBySalary[a.SalaryID] = append(amountsBySalary[a.SalaryID], parsed)
	}
	var templates []TransferTemplate
	for _, sal := range salaries {
		s := salaryFromDB(sal)
		s.Amounts = amountsBySalary[s.ID]
		templates = append(templates, s.GenerateTransferTemplates()...)
	}
	return templates, nil
}
