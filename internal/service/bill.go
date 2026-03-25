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

type BillAccount struct {
	ID            string
	Name          string
	FromAccountID string
	Recurrence    date.Cron
	Priority      int64
	Enabled       bool
	Bills         []Bill
}

type Bill struct {
	ID               string
	BillAccountID    string
	Name             string
	BudgetCategoryID *string
	Enabled          bool
	Notes            string
	URL              string
	Amounts          []BillAmount
}

type BillAmount struct {
	ID        string
	BillID    string
	Amount    uncertain.Value
	StartDate date.Date
	EndDate   *date.Date
}

func (b Bill) CurrentAmount() float64 {
	today := date.Today()
	var current *BillAmount
	for i := range b.Amounts {
		if b.Amounts[i].StartDate <= today {
			if b.Amounts[i].EndDate != nil && *b.Amounts[i].EndDate <= today {
				continue
			}
			if current == nil || b.Amounts[i].StartDate > current.StartDate {
				current = &b.Amounts[i]
			}
		}
	}
	if current == nil {
		return 0
	}
	return current.Amount.Mean()
}

func (ba BillAmount) GetStartDateString() string {
	if ba.ID == "" {
		return ""
	}
	return ba.StartDate.String()
}

func (ba BillAmount) GetEndDateString() string {
	if ba.ID == "" || ba.EndDate == nil {
		return ""
	}
	return ba.EndDate.String()
}

func (ba BillAmount) GetAmountString() string {
	if ba.ID == "" {
		return ""
	}
	return ba.Amount.SimpleEncode()
}

func (b Bill) GenerateTransferTemplates(ba BillAccount) []TransferTemplate {
	if len(b.Amounts) == 0 {
		return nil
	}

	source := TransferTemplateSource{
		Type:     "bill",
		EntityID: b.ID,
		Label:    b.Name,
		EditURL:  "/bills/" + ba.ID + "/edit",
	}

	enabled := ba.Enabled && b.Enabled

	sorted := make([]BillAmount, len(b.Amounts))
	copy(sorted, b.Amounts)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].StartDate < sorted[j].StartDate
	})

	var templates []TransferTemplate
	for i, amt := range sorted {
		endDate := amt.EndDate
		if i+1 < len(sorted) {
			nextStart := sorted[i+1].StartDate
			if endDate == nil || *endDate > nextStart {
				endDate = &nextStart
			}
		}
		templates = append(templates, TransferTemplate{
			ID:               fmt.Sprintf("bill:%s:%d", b.ID, i),
			Name:             b.Name,
			FromAccountID:    ba.FromAccountID,
			ToAccountID:      "",
			AmountType:       "fixed",
			AmountFixed:      amt.Amount,
			Priority:         ba.Priority,
			Recurrence:       ba.Recurrence,
			StartDate:        amt.StartDate,
			EndDate:          endDate,
			Enabled:          enabled,
			BudgetCategoryID: b.BudgetCategoryID,
			Source:           source,
		})
	}

	return templates
}

func billAccountFromDB(ba pdb.BillAccount) BillAccount {
	return BillAccount{
		ID:            ba.ID,
		Name:          ba.Name,
		FromAccountID: ui.OrDefault(ba.FromAccountID),
		Recurrence:    date.Cron(ba.Recurrence),
		Priority:      ba.Priority,
		Enabled:       ba.Enabled,
	}
}

func billFromDB(b pdb.Bill) Bill {
	return Bill{
		ID:               b.ID,
		BillAccountID:    b.BillAccountID,
		Name:             b.Name,
		BudgetCategoryID: b.BudgetCategoryID,
		Enabled:          b.Enabled,
		Notes:            b.Notes,
		URL:              b.Url,
	}
}

func billAmountFromDB(a pdb.BillAmount) (BillAmount, error) {
	var amount uncertain.Value
	if err := amount.Decode(a.Amount); err != nil {
		return BillAmount{}, fmt.Errorf("decoding bill amount: %w", err)
	}
	var endDate *date.Date
	if a.EndDate != nil {
		d := date.Date(*a.EndDate)
		endDate = &d
	}
	return BillAmount{
		ID:        a.ID,
		BillID:    a.BillID,
		Amount:    amount,
		StartDate: date.Date(a.StartDate),
		EndDate:   endDate,
	}, nil
}

func (s *Service) UpsertBillAccount(ctx context.Context, inp BillAccount) (BillAccount, error) {
	if inp.ID == "" {
		inp.ID = sid.MustNewString(32)
	}
	now := time.Now().Unix()
	ba, err := pdb.New(s.db).UpsertBillAccount(ctx, pdb.UpsertBillAccountParams{
		ID:            inp.ID,
		Name:          inp.Name,
		FromAccountID: ui.WithDefaultNull(inp.FromAccountID),
		Recurrence:    string(inp.Recurrence),
		Priority:      inp.Priority,
		Enabled:       inp.Enabled,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	if err != nil {
		return BillAccount{}, fmt.Errorf("upserting bill account: %w", err)
	}
	return billAccountFromDB(ba), nil
}

func (s *Service) GetBillAccount(ctx context.Context, id string) (BillAccount, error) {
	ba, err := pdb.New(s.db).GetBillAccount(ctx, id)
	if err != nil {
		return BillAccount{}, fmt.Errorf("getting bill account: %w", err)
	}
	return billAccountFromDB(ba), nil
}

func (s *Service) ListBillAccounts(ctx context.Context) ([]BillAccount, error) {
	rows, err := pdb.New(s.db).ListBillAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing bill accounts: %w", err)
	}
	result := make([]BillAccount, len(rows))
	for i, r := range rows {
		result[i] = billAccountFromDB(r)
	}
	return result, nil
}

func (s *Service) DeleteBillAccount(ctx context.Context, id string) error {
	if err := pdb.New(s.db).DeleteBillAccount(ctx, id); err != nil {
		return fmt.Errorf("deleting bill account: %w", err)
	}
	return nil
}

func (s *Service) UpsertBill(ctx context.Context, inp Bill) (Bill, error) {
	if inp.ID == "" {
		inp.ID = sid.MustNewString(32)
	}
	now := time.Now().Unix()
	b, err := pdb.New(s.db).UpsertBill(ctx, pdb.UpsertBillParams{
		ID:               inp.ID,
		BillAccountID:    inp.BillAccountID,
		Name:             inp.Name,
		BudgetCategoryID: inp.BudgetCategoryID,
		Enabled:          inp.Enabled,
		Notes:            inp.Notes,
		Url:              inp.URL,
		CreatedAt:        now,
		UpdatedAt:        now,
	})
	if err != nil {
		return Bill{}, fmt.Errorf("upserting bill: %w", err)
	}
	return billFromDB(b), nil
}

func (s *Service) GetBill(ctx context.Context, id string) (Bill, error) {
	b, err := pdb.New(s.db).GetBill(ctx, id)
	if err != nil {
		return Bill{}, fmt.Errorf("getting bill: %w", err)
	}
	return billFromDB(b), nil
}

func (s *Service) ListBills(ctx context.Context, billAccountID string) ([]Bill, error) {
	rows, err := pdb.New(s.db).ListBills(ctx, billAccountID)
	if err != nil {
		return nil, fmt.Errorf("listing bills: %w", err)
	}
	result := make([]Bill, len(rows))
	for i, r := range rows {
		result[i] = billFromDB(r)
	}
	return result, nil
}

func (s *Service) DeleteBill(ctx context.Context, id string) error {
	if err := pdb.New(s.db).DeleteBill(ctx, id); err != nil {
		return fmt.Errorf("deleting bill: %w", err)
	}
	return nil
}

func (s *Service) UpsertBillAmount(ctx context.Context, inp BillAmount) (BillAmount, error) {
	if inp.ID == "" {
		inp.ID = sid.MustNewString(32)
	}
	encoded, err := inp.Amount.Encode()
	if err != nil {
		return BillAmount{}, fmt.Errorf("encoding bill amount: %w", err)
	}
	var endDate *int64
	if inp.EndDate != nil {
		d := int64(*inp.EndDate)
		endDate = &d
	}
	now := time.Now().Unix()
	a, err := pdb.New(s.db).UpsertBillAmount(ctx, pdb.UpsertBillAmountParams{
		ID:        inp.ID,
		BillID:    inp.BillID,
		Amount:    encoded,
		StartDate: int64(inp.StartDate),
		EndDate:   endDate,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return BillAmount{}, fmt.Errorf("upserting bill amount: %w", err)
	}
	return billAmountFromDB(a)
}

func (s *Service) ListBillAmounts(ctx context.Context, billID string) ([]BillAmount, error) {
	rows, err := pdb.New(s.db).ListBillAmounts(ctx, billID)
	if err != nil {
		return nil, fmt.Errorf("listing bill amounts: %w", err)
	}
	var result []BillAmount
	for _, r := range rows {
		a, err := billAmountFromDB(r)
		if err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, nil
}

func (s *Service) DeleteBillAmount(ctx context.Context, id string) error {
	if err := pdb.New(s.db).DeleteBillAmount(ctx, id); err != nil {
		return fmt.Errorf("deleting bill amount: %w", err)
	}
	return nil
}

type BillAccountEditView struct {
	BillAccount BillAccount
	Accounts    []Account
	Categories  []TransferTemplateCategory
}

func (v BillAccountEditView) IsEdit() bool {
	return v.BillAccount.ID != ""
}

type BillEditView struct {
	Bill           Bill
	BillAccount    BillAccount
	Categories     []TransferTemplateCategory
}

func (v BillEditView) IsEdit() bool {
	return v.Bill.ID != ""
}

func (s *Service) GetBillAccountsPageData(ctx context.Context) ([]BillAccount, error) {
	accounts, err := s.ListBillAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing bill accounts: %w", err)
	}
	allBills, err := pdb.New(s.db).ListAllBills(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing all bills: %w", err)
	}
	allAmounts, err := pdb.New(s.db).ListAllBillAmounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing all bill amounts: %w", err)
	}
	amountsByBill := make(map[string][]BillAmount)
	for _, a := range allAmounts {
		parsed, err := billAmountFromDB(a)
		if err != nil {
			return nil, err
		}
		amountsByBill[a.BillID] = append(amountsByBill[a.BillID], parsed)
	}
	billsByAccount := make(map[string][]Bill)
	for _, b := range allBills {
		bill := billFromDB(b)
		bill.Amounts = amountsByBill[b.ID]
		billsByAccount[b.BillAccountID] = append(billsByAccount[b.BillAccountID], bill)
	}
	for i := range accounts {
		accounts[i].Bills = billsByAccount[accounts[i].ID]
	}
	return accounts, nil
}

func (s *Service) GetBillAccountNewPageData(ctx context.Context) (*BillAccountEditView, error) {
	accs, err := s.ListAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing accounts: %w", err)
	}
	categories, err := s.ListCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing categories: %w", err)
	}
	return &BillAccountEditView{
		Accounts:   accs,
		Categories: categories,
	}, nil
}

func (s *Service) GetBillAccountEditPageData(ctx context.Context, id string) (*BillAccountEditView, error) {
	ba, err := s.GetBillAccount(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting bill account: %w", err)
	}
	bills, err := s.ListBills(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("listing bills: %w", err)
	}
	allAmounts, err := pdb.New(s.db).ListAllBillAmounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing bill amounts: %w", err)
	}
	amountsByBill := make(map[string][]BillAmount)
	for _, a := range allAmounts {
		parsed, err := billAmountFromDB(a)
		if err != nil {
			return nil, err
		}
		amountsByBill[a.BillID] = append(amountsByBill[a.BillID], parsed)
	}
	for i := range bills {
		bills[i].Amounts = amountsByBill[bills[i].ID]
	}
	ba.Bills = bills
	accs, err := s.ListAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing accounts: %w", err)
	}
	categories, err := s.ListCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing categories: %w", err)
	}
	return &BillAccountEditView{
		BillAccount: ba,
		Accounts:    accs,
		Categories:  categories,
	}, nil
}

func (s *Service) GetBillEditPageData(ctx context.Context, billID string) (*BillEditView, error) {
	bill, err := s.GetBill(ctx, billID)
	if err != nil {
		return nil, fmt.Errorf("getting bill: %w", err)
	}
	amounts, err := s.ListBillAmounts(ctx, billID)
	if err != nil {
		return nil, fmt.Errorf("listing bill amounts: %w", err)
	}
	bill.Amounts = amounts
	ba, err := s.GetBillAccount(ctx, bill.BillAccountID)
	if err != nil {
		return nil, fmt.Errorf("getting bill account: %w", err)
	}
	categories, err := s.ListCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing categories: %w", err)
	}
	return &BillEditView{
		Bill:        bill,
		BillAccount: ba,
		Categories:  categories,
	}, nil
}

func (s *Service) generateBillTransferTemplates(ctx context.Context) ([]TransferTemplate, error) {
	billAccounts, err := pdb.New(s.db).ListBillAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing bill accounts: %w", err)
	}
	allBills, err := pdb.New(s.db).ListAllBills(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing all bills: %w", err)
	}
	allAmounts, err := pdb.New(s.db).ListAllBillAmounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing all bill amounts: %w", err)
	}

	amountsByBill := make(map[string][]BillAmount)
	for _, a := range allAmounts {
		parsed, err := billAmountFromDB(a)
		if err != nil {
			return nil, err
		}
		amountsByBill[a.BillID] = append(amountsByBill[a.BillID], parsed)
	}

	billsByAccount := make(map[string][]Bill)
	for _, b := range allBills {
		bill := billFromDB(b)
		bill.Amounts = amountsByBill[b.ID]
		billsByAccount[b.BillAccountID] = append(billsByAccount[b.BillAccountID], bill)
	}

	var templates []TransferTemplate
	for _, ba := range billAccounts {
		account := billAccountFromDB(ba)
		for _, bill := range billsByAccount[ba.ID] {
			templates = append(templates, bill.GenerateTransferTemplates(account)...)
		}
	}
	return templates, nil
}
