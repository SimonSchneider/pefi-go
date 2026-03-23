package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/goslu/sid"
	"github.com/SimonSchneider/pefigo/internal/pdb"
	"github.com/SimonSchneider/pefigo/internal/swe"
	"github.com/SimonSchneider/pefigo/internal/ui"
	"github.com/SimonSchneider/pefigo/internal/uncertain"
)

type PensionSegment struct {
	StartDate date.Date
	EndDate   *date.Date
	Pension   uncertain.Value
}

type Salary struct {
	ID               string
	Name             string
	ToAccountID      string
	PensionAccountID string
	Priority         int64
	Recurrence       date.Cron
	BudgetCategoryID *string
	Enabled          bool
	Kommun           string
	Forsamling       string
	ChurchMember     bool
	IsGross          bool
	Amounts          []SalaryAmount
	// PensionSegments is populated by the service layer when IsGross is true.
	// Segments are split at the union of salary-amount and IBB change dates.
	PensionSegments []PensionSegment
}

type SalaryAmount struct {
	ID        string
	SalaryID  string
	Amount    uncertain.Value
	StartDate date.Date
	// Net is populated by the service layer for gross salaries.
	// It is an uncertain.Value derived from Amount via tax computation.
	Net uncertain.Value
}

func (s Salary) GenerateTransferTemplates() []TransferTemplate {
	sorted := make([]SalaryAmount, len(s.Amounts))
	copy(sorted, s.Amounts)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].StartDate < sorted[j].StartDate
	})

	source := TransferTemplateSource{
		Type:     "salary",
		EntityID: s.ID,
		Label:    s.Name,
		EditURL:  "/salaries/" + s.ID + "/edit",
	}

	var templates []TransferTemplate

	// Net salary TTs: one per salary amount period
	for i, amt := range sorted {
		var endDate *date.Date
		if i+1 < len(sorted) {
			ed := sorted[i+1].StartDate
			endDate = &ed
		}

		amountFixed := amt.Amount
		if s.IsGross && amt.Net.Valid() {
			amountFixed = amt.Net
		}

		templates = append(templates, TransferTemplate{
			ID:               "salary:" + amt.ID,
			Name:             s.Name,
			FromAccountID:    "",
			ToAccountID:      s.ToAccountID,
			AmountType:       "fixed",
			AmountFixed:      amountFixed,
			Priority:         s.Priority,
			Recurrence:       s.Recurrence,
			StartDate:        amt.StartDate,
			EndDate:          endDate,
			Enabled:          s.Enabled,
			BudgetCategoryID: s.BudgetCategoryID,
			Source:           source,
		})
	}

	// Pension TTs: split at both salary-amount and IBB change boundaries
	if s.IsGross && s.PensionAccountID != "" {
		for i, seg := range s.PensionSegments {
			templates = append(templates, TransferTemplate{
				ID:               fmt.Sprintf("salary-pension:%s:%d", s.ID, i),
				Name:             s.Name + " (pension)",
				FromAccountID:    "",
				ToAccountID:      s.PensionAccountID,
				AmountType:       "fixed",
				AmountFixed:      seg.Pension,
				Priority:         s.Priority,
				Recurrence:       s.Recurrence,
				StartDate:        seg.StartDate,
				EndDate:          seg.EndDate,
				Enabled:          s.Enabled,
				BudgetCategoryID: s.BudgetCategoryID,
				Source:           source,
			})
		}
	}

	return templates
}

func salaryFromDB(s pdb.Salary) Salary {
	return Salary{
		ID:               s.ID,
		Name:             s.Name,
		ToAccountID:      ui.OrDefault(s.ToAccountID),
		PensionAccountID: ui.OrDefault(s.PensionAccountID),
		Priority:         s.Priority,
		Recurrence:       date.Cron(s.Recurrence),
		BudgetCategoryID: s.BudgetCategoryID,
		Enabled:          s.Enabled,
		Kommun:           s.Kommun,
		Forsamling:       s.Forsamling,
		ChurchMember:     s.ChurchMember,
		IsGross:          s.IsGross,
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
		PensionAccountID: ui.WithDefaultNull(inp.PensionAccountID),
		Priority:         inp.Priority,
		Recurrence:       string(inp.Recurrence),
		BudgetCategoryID: inp.BudgetCategoryID,
		Enabled:          inp.Enabled,
		Kommun:           inp.Kommun,
		Forsamling:       inp.Forsamling,
		ChurchMember:     inp.ChurchMember,
		IsGross:          inp.IsGross,
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

	ibbs, err := s.ListInkomstbasbelopp(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing inkomstbasbelopp: %w", err)
	}

	var templates []TransferTemplate
	for _, sal := range salaries {
		salary := salaryFromDB(sal)
		salary.Amounts = amountsBySalary[salary.ID]

		if salary.IsGross && salary.Kommun != "" && salary.Forsamling != "" {
			s.populateNetAmounts(ctx, &salary)
			salary.PensionSegments = s.computePensionSegments(ctx, salary, ibbs)
		}

		templates = append(templates, salary.GenerateTransferTemplates()...)
	}
	return templates, nil
}


// populateNetAmounts sets amt.Net on each salary amount as a mapped uncertain.Value
// that derives net salary from the gross amount via cached tax lookups.
func (s *Service) populateNetAmounts(ctx context.Context, sal *Salary) {
	for i := range sal.Amounts {
		amt := &sal.Amounts[i]
		year := strings.SplitN(amt.StartDate.String(), "-", 2)[0]

		// Pre-warm the cache by doing one lookup with the mean value.
		// This ensures the tax table is cached for subsequent samples.
		s.sweClient.CalculateNetSalary(ctx, swe.GrossSalaryInput{
			GrossMonthly: amt.Amount.Mean(),
			Kommun:       sal.Kommun,
			Forsamling:   sal.Forsamling,
			Year:         year,
			ChurchMember: sal.ChurchMember,
			Column:       1,
		})

		grossAmount := amt.Amount
		kommun := sal.Kommun
		forsamling := sal.Forsamling
		churchMember := sal.ChurchMember
		sweClient := s.sweClient

		amt.Net = uncertain.NewMapped(func(cfg *uncertain.Config) float64 {
			gross := grossAmount.Sample(cfg)
			res, err := sweClient.CalculateNetSalary(context.Background(), swe.GrossSalaryInput{
				GrossMonthly: gross,
				Kommun:       kommun,
				Forsamling:   forsamling,
				Year:         year,
				ChurchMember: churchMember,
				Column:       1,
			})
			if err != nil {
				return gross
			}
			return res.NetMonthly
		})
	}
}

// computePensionSegments builds pension segments split at the union of
// salary-amount and IBB change-point dates. Each segment's Pension is a
// mapped uncertain.Value that derives pension from the gross amount.
func (s *Service) computePensionSegments(_ context.Context, sal Salary, ibbs []Inkomstbasbelopp) []PensionSegment {
	if len(sal.Amounts) == 0 {
		return nil
	}

	sorted := make([]SalaryAmount, len(sal.Amounts))
	copy(sorted, sal.Amounts)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].StartDate < sorted[j].StartDate
	})

	dateSet := make(map[date.Date]struct{})
	for _, amt := range sorted {
		dateSet[amt.StartDate] = struct{}{}
	}
	for _, ibb := range ibbs {
		if ibb.ValidFrom >= sorted[0].StartDate {
			dateSet[ibb.ValidFrom] = struct{}{}
		}
	}

	dates := make([]date.Date, 0, len(dateSet))
	for d := range dateSet {
		dates = append(dates, d)
	}
	sort.Slice(dates, func(i, j int) bool { return dates[i] < dates[j] })

	var segments []PensionSegment
	for i, d := range dates {
		grossAmount := activeSalaryAmountAt(sorted, d)
		if grossAmount == nil {
			continue
		}
		ibb := activeIBBAt(ibbs, d)
		if ibb == 0 {
			continue
		}

		gross := *grossAmount
		ibbVal := ibb
		pension := uncertain.NewMapped(func(cfg *uncertain.Config) float64 {
			return swe.CalculateITP1Pension(gross.Sample(cfg), ibbVal)
		})

		var endDate *date.Date
		if i+1 < len(dates) {
			ed := dates[i+1]
			endDate = &ed
		}

		segments = append(segments, PensionSegment{
			StartDate: d,
			EndDate:   endDate,
			Pension:   pension,
		})
	}
	return segments
}

// activeSalaryAmountAt returns the uncertain.Value gross salary active at a given date.
func activeSalaryAmountAt(sorted []SalaryAmount, d date.Date) *uncertain.Value {
	var active *uncertain.Value
	for i := range sorted {
		if sorted[i].StartDate <= d {
			active = &sorted[i].Amount
		}
	}
	return active
}
