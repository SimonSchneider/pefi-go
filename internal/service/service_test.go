package service_test

import (
	"testing"

	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/pefigo"
	"github.com/SimonSchneider/pefigo/internal/service"
	"github.com/SimonSchneider/pefigo/internal/swe"
	"github.com/SimonSchneider/pefigo/internal/uncertain"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func newFixedValue(v float64) uncertain.Value {
	return uncertain.NewFixed(v)
}

func mustParseDate(s string) date.Date {
	d, err := date.ParseDate(s)
	if err != nil {
		panic(err)
	}
	return d
}

func newTestService(t *testing.T) *service.Service {
	t.Helper()
	db, err := service.GetMigratedDB(t.Context(), pefigo.StaticEmbeddedFS, "static/migrations", ":memory:")
	if err != nil {
		t.Fatalf("failed to create test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return service.New(db)
}

// ---- Account Type CRUD ----

func TestAccountTypeCRUD(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()

	at, err := svc.UpsertAccountType(ctx, service.AccountTypeInput{Name: "Savings", Color: "#00ff00"})
	if err != nil {
		t.Fatalf("create account type: %v", err)
	}
	if at.Name != "Savings" || at.Color != "#00ff00" || at.ID == "" {
		t.Fatalf("unexpected account type: %+v", at)
	}

	got, err := svc.GetAccountType(ctx, at.ID)
	if err != nil {
		t.Fatalf("get account type: %v", err)
	}
	if got.Name != "Savings" {
		t.Fatalf("expected Savings, got %s", got.Name)
	}

	at2, err := svc.UpsertAccountType(ctx, service.AccountTypeInput{ID: at.ID, Name: "Updated", Color: "#ff0000"})
	if err != nil {
		t.Fatalf("update account type: %v", err)
	}
	if at2.Name != "Updated" || at2.ID != at.ID {
		t.Fatalf("unexpected updated account type: %+v", at2)
	}

	list, err := svc.ListAccountTypes(ctx)
	if err != nil {
		t.Fatalf("list account types: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 account type, got %d", len(list))
	}

	if err := svc.DeleteAccountType(ctx, at.ID); err != nil {
		t.Fatalf("delete account type: %v", err)
	}
	list, err = svc.ListAccountTypes(ctx)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected 0 account types after delete, got %d", len(list))
	}
}

// ---- Category CRUD ----

func TestCategoryCRUD(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()

	color := "#abcdef"
	cat, err := svc.UpsertCategory(ctx, service.TransferTemplateCategoryInput{Name: "Housing", Color: &color})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}
	if cat.Name != "Housing" || cat.ID == "" {
		t.Fatalf("unexpected category: %+v", cat)
	}

	got, err := svc.GetCategory(ctx, cat.ID)
	if err != nil {
		t.Fatalf("get category: %v", err)
	}
	if got.Name != "Housing" {
		t.Fatalf("expected Housing, got %s", got.Name)
	}

	list, err := svc.ListCategories(ctx)
	if err != nil {
		t.Fatalf("list categories: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 category, got %d", len(list))
	}

	if err := svc.DeleteCategory(ctx, cat.ID); err != nil {
		t.Fatalf("delete category: %v", err)
	}
}

// ---- Account CRUD ----

func TestAccountCRUD(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()

	at, err := svc.UpsertAccountType(ctx, service.AccountTypeInput{Name: "Checking"})
	if err != nil {
		t.Fatalf("create account type: %v", err)
	}

	acc, err := svc.UpsertAccount(ctx, service.AccountInput{Name: "My Account", TypeID: at.ID})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	if acc.Name != "My Account" || acc.ID == "" || acc.TypeID != at.ID {
		t.Fatalf("unexpected account: %+v", acc)
	}

	got, err := svc.GetAccount(ctx, acc.ID)
	if err != nil {
		t.Fatalf("get account: %v", err)
	}
	if got.Name != "My Account" {
		t.Fatalf("expected My Account, got %s", got.Name)
	}

	acc2, err := svc.UpsertAccount(ctx, service.AccountInput{ID: acc.ID, Name: "Renamed Account", TypeID: at.ID})
	if err != nil {
		t.Fatalf("update account: %v", err)
	}
	if acc2.Name != "Renamed Account" {
		t.Fatalf("expected Renamed Account, got %s", acc2.Name)
	}

	list, err := svc.ListAccounts(ctx)
	if err != nil {
		t.Fatalf("list accounts: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 account, got %d", len(list))
	}

	if err := svc.DeleteAccount(ctx, acc.ID); err != nil {
		t.Fatalf("delete account: %v", err)
	}
	list, err = svc.ListAccounts(ctx)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected 0 accounts after delete, got %d", len(list))
	}
}

// ---- Special Dates CRUD ----

func TestSpecialDateCRUD(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()

	sd, err := svc.UpsertSpecialDate(ctx, service.SpecialDateInput{Name: "Christmas", Date: mustParseDate("2025-12-25"), Color: "#ff0000"})
	if err != nil {
		t.Fatalf("create special date: %v", err)
	}
	if sd.Name != "Christmas" || sd.ID == "" {
		t.Fatalf("unexpected special date: %+v", sd)
	}

	got, err := svc.GetSpecialDate(ctx, sd.ID)
	if err != nil {
		t.Fatalf("get special date: %v", err)
	}
	if got.Name != "Christmas" {
		t.Fatalf("expected Christmas, got %s", got.Name)
	}

	list, err := svc.ListSpecialDates(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1, got %d", len(list))
	}

	if err := svc.DeleteSpecialDate(ctx, sd.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

// ---- Snapshot CRUD ----

func TestSnapshotCRUD(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()

	acc, err := svc.UpsertAccount(ctx, service.AccountInput{Name: "Test"})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}

	d := mustParseDate("2025-01-01")
	snap, err := svc.UpsertAccountSnapshot(ctx, acc.ID, service.AccountSnapshotInput{
		Date:    d,
		Balance: newFixedValue(1000),
	})
	if err != nil {
		t.Fatalf("upsert snapshot: %v", err)
	}
	if snap.AccountID != acc.ID || snap.Balance.Mean() != 1000 {
		t.Fatalf("unexpected snapshot: %+v", snap)
	}

	snaps, err := svc.ListAccountSnapshots(ctx, acc.ID)
	if err != nil {
		t.Fatalf("list snapshots: %v", err)
	}
	if len(snaps) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(snaps))
	}

	if err := svc.DeleteAccountSnapshot(ctx, acc.ID, d); err != nil {
		t.Fatalf("delete snapshot: %v", err)
	}
	snaps, err = svc.ListAccountSnapshots(ctx, acc.ID)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(snaps) != 0 {
		t.Fatalf("expected 0 after delete, got %d", len(snaps))
	}
}

// ---- Transfer Template CRUD ----

func TestTransferTemplateCRUD(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()

	acc1, _ := svc.UpsertAccount(ctx, service.AccountInput{Name: "From"})
	acc2, _ := svc.UpsertAccount(ctx, service.AccountInput{Name: "To"})

	tt, err := svc.UpsertTransferTemplate(ctx, service.TransferTemplate{
		Name:          "Rent",
		FromAccountID: acc1.ID,
		ToAccountID:   acc2.ID,
		AmountType:    "fixed",
		AmountFixed:   newFixedValue(1500),
		Recurrence:    "*-*-01",
		StartDate:     mustParseDate("2025-01-01"),
		Enabled:       true,
	})
	if err != nil {
		t.Fatalf("create transfer template: %v", err)
	}
	if tt.Name != "Rent" || tt.ID == "" {
		t.Fatalf("unexpected template: %+v", tt)
	}

	got, err := svc.GetTransferTemplate(ctx, tt.ID)
	if err != nil {
		t.Fatalf("get template: %v", err)
	}
	if got.Name != "Rent" {
		t.Fatalf("expected Rent, got %s", got.Name)
	}

	dup, err := svc.DuplicateTransferTemplate(ctx, tt.ID)
	if err != nil {
		t.Fatalf("duplicate template: %v", err)
	}
	if dup.ID == tt.ID {
		t.Fatal("duplicate should have a different ID")
	}

	list, err := svc.ListTransferTemplates(ctx)
	if err != nil {
		t.Fatalf("list templates: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 templates, got %d", len(list))
	}

	if err := svc.DeleteTransferTemplate(ctx, dup.ID); err != nil {
		t.Fatalf("delete template: %v", err)
	}
	if err := svc.DeleteTransferTemplate(ctx, tt.ID); err != nil {
		t.Fatalf("delete template: %v", err)
	}
}

// ---- Growth Model CRUD ----

func TestGrowthModelCRUD(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()

	acc, _ := svc.UpsertAccount(ctx, service.AccountInput{Name: "Savings"})

	gm, err := svc.UpsertAccountGrowthModel(ctx, service.AccountGrowthModelInput{
		AccountID:        acc.ID,
		Type:             "fixed",
		AnnualRate:       newFixedValue(0.05),
		AnnualVolatility: newFixedValue(0),
		StartDate:        mustParseDate("2025-01-01"),
	})
	if err != nil {
		t.Fatalf("create growth model: %v", err)
	}
	if gm.AccountID != acc.ID || gm.Type != "fixed" {
		t.Fatalf("unexpected growth model: %+v", gm)
	}

	list, err := svc.ListAccountGrowthModels(ctx, acc.ID)
	if err != nil {
		t.Fatalf("list growth models: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 growth model, got %d", len(list))
	}

	if err := svc.DeleteAccountGrowthModel(ctx, gm.ID); err != nil {
		t.Fatalf("delete growth model: %v", err)
	}
}

// ---- Dashboard Data Assembly ----

func TestGetDashboardData(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()

	at, _ := svc.UpsertAccountType(ctx, service.AccountTypeInput{Name: "Savings", Color: "#00ff00"})
	acc, _ := svc.UpsertAccount(ctx, service.AccountInput{Name: "My Savings", TypeID: at.ID})
	svc.UpsertAccountSnapshot(ctx, acc.ID, service.AccountSnapshotInput{
		Date:    mustParseDate("2025-01-01"),
		Balance: newFixedValue(5000),
	})

	dashboard, err := svc.GetDashboardData(ctx)
	if err != nil {
		t.Fatalf("get dashboard data: %v", err)
	}
	if dashboard.TotalBalance != 5000 {
		t.Fatalf("expected total balance 5000, got %f", dashboard.TotalBalance)
	}
	if dashboard.TotalAssets != 5000 {
		t.Fatalf("expected total assets 5000, got %f", dashboard.TotalAssets)
	}
	if len(dashboard.AccountTypeGroups) != 1 {
		t.Fatalf("expected 1 account type group, got %d", len(dashboard.AccountTypeGroups))
	}
	if dashboard.AccountTypeGroups[0].AccountType.Name != "Savings" {
		t.Fatalf("expected Savings group, got %s", dashboard.AccountTypeGroups[0].AccountType.Name)
	}
}

// ---- Budget Data Assembly ----

func TestGetBudgetData(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()

	color := "#ff0000"
	cat, _ := svc.UpsertCategory(ctx, service.TransferTemplateCategoryInput{Name: "Housing", Color: &color})
	acc1, _ := svc.UpsertAccount(ctx, service.AccountInput{Name: "Checking"})
	acc2, _ := svc.UpsertAccount(ctx, service.AccountInput{Name: "Landlord"})

	svc.UpsertTransferTemplate(ctx, service.TransferTemplate{
		Name:             "Rent",
		FromAccountID:    acc1.ID,
		ToAccountID:      acc2.ID,
		AmountType:       "fixed",
		AmountFixed:      newFixedValue(1500),
		Recurrence:       "*-*-01",
		StartDate:        mustParseDate("2020-01-01"),
		Enabled:          true,
		BudgetCategoryID: &cat.ID,
	})

	budget, err := svc.GetBudgetData(ctx)
	if err != nil {
		t.Fatalf("get budget data: %v", err)
	}
	if budget.GrandTotal != 1500 {
		t.Fatalf("expected grand total 1500, got %f", budget.GrandTotal)
	}
	if len(budget.Categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(budget.Categories))
	}
	if budget.Categories[0].Category.Name != "Housing" {
		t.Fatalf("expected Housing category, got %s", budget.Categories[0].Category.Name)
	}
}

// ---- Categories Page Data ----

func TestGetCategoriesPageData(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()

	at, err := svc.UpsertAccountType(ctx, service.AccountTypeInput{Name: "Savings", Color: "#00ff00"})
	if err != nil {
		t.Fatalf("create account type: %v", err)
	}

	color := "#abcdef"
	cat, err := svc.UpsertCategory(ctx, service.TransferTemplateCategoryInput{Name: "Housing", Color: &color})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}

	view, err := svc.GetCategoriesPageData(ctx)
	if err != nil {
		t.Fatalf("get categories page data: %v", err)
	}
	if len(view.AccountTypes) != 1 {
		t.Fatalf("expected 1 account type, got %d", len(view.AccountTypes))
	}
	if view.AccountTypes[0].Name != at.Name {
		t.Fatalf("expected account type %s, got %s", at.Name, view.AccountTypes[0].Name)
	}
	if len(view.Categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(view.Categories))
	}
	if view.Categories[0].Name != cat.Name {
		t.Fatalf("expected category %s, got %s", cat.Name, view.Categories[0].Name)
	}
}

func TestGetCategoriesPageDataEmpty(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()

	view, err := svc.GetCategoriesPageData(ctx)
	if err != nil {
		t.Fatalf("get categories page data: %v", err)
	}
	if len(view.AccountTypes) != 0 {
		t.Fatalf("expected 0 account types, got %d", len(view.AccountTypes))
	}
	if len(view.Categories) != 0 {
		t.Fatalf("expected 0 categories, got %d", len(view.Categories))
	}
}

// ---- Transfer Chart Data ----

func TestGetTransferChartData_NodeColorsFromAccountTypes(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()

	at1, _ := svc.UpsertAccountType(ctx, service.AccountTypeInput{Name: "Savings", Color: "#00ff00"})
	at2, _ := svc.UpsertAccountType(ctx, service.AccountTypeInput{Name: "Checking", Color: "#0000ff"})

	acc1, _ := svc.UpsertAccount(ctx, service.AccountInput{Name: "My Savings", TypeID: at1.ID})
	acc2, _ := svc.UpsertAccount(ctx, service.AccountInput{Name: "My Checking", TypeID: at2.ID})

	svc.UpsertTransferTemplate(ctx, service.TransferTemplate{
		Name:          "Salary",
		FromAccountID: "",
		ToAccountID:   acc1.ID,
		AmountType:    "fixed",
		AmountFixed:   newFixedValue(5000),
		Recurrence:    "*-*-25",
		StartDate:     mustParseDate("2020-01-01"),
		Enabled:       true,
	})
	svc.UpsertTransferTemplate(ctx, service.TransferTemplate{
		Name:          "Transfer",
		FromAccountID: acc1.ID,
		ToAccountID:   acc2.ID,
		AmountType:    "fixed",
		AmountFixed:   newFixedValue(2000),
		Recurrence:    "*-*-01",
		StartDate:     mustParseDate("2020-01-01"),
		Enabled:       true,
	})
	svc.UpsertTransferTemplate(ctx, service.TransferTemplate{
		Name:          "Groceries",
		FromAccountID: acc2.ID,
		ToAccountID:   "",
		AmountType:    "fixed",
		AmountFixed:   newFixedValue(500),
		Recurrence:    "*-*-01",
		StartDate:     mustParseDate("2020-01-01"),
		Enabled:       true,
	})

	t.Run("group by account uses account type color in itemStyle", func(t *testing.T) {
		data, err := svc.GetTransferChartData(ctx, service.GroupByAccount)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		nodeColors := make(map[string]string)
		for _, node := range data.Data {
			if node.ItemStyle != nil {
				nodeColors[node.Name] = node.ItemStyle.Color
			}
		}
		if nodeColors["My Savings"] != "#00ff00" {
			t.Errorf("expected My Savings color #00ff00, got %q", nodeColors["My Savings"])
		}
		if nodeColors["My Checking"] != "#0000ff" {
			t.Errorf("expected My Checking color #0000ff, got %q", nodeColors["My Checking"])
		}
		if nodeColors["Income"] != "#388E3C" {
			t.Errorf("expected Income color #388E3C, got %q", nodeColors["Income"])
		}
	})

	t.Run("group by account_type uses account type color in itemStyle", func(t *testing.T) {
		data, err := svc.GetTransferChartData(ctx, service.GroupByAccountType)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		nodeColors := make(map[string]string)
		for _, node := range data.Data {
			if node.ItemStyle != nil {
				nodeColors[node.Name] = node.ItemStyle.Color
			}
		}
		if nodeColors["Savings"] != "#00ff00" {
			t.Errorf("expected Savings color #00ff00, got %q", nodeColors["Savings"])
		}
		if nodeColors["Checking"] != "#0000ff" {
			t.Errorf("expected Checking color #0000ff, got %q", nodeColors["Checking"])
		}
		if nodeColors["Expenses"] != "#D32F2F" {
			t.Errorf("expected Expenses color #D32F2F, got %q", nodeColors["Expenses"])
		}
	})
}

// ---- Transfer Simplification ----

func TestSimplifyTransfers(t *testing.T) {
	transfers := []service.Transfer{
		{FromAccountID: "a", ToAccountID: "b", Amount: 100},
		{FromAccountID: "b", ToAccountID: "a", Amount: 30},
		{FromAccountID: "a", ToAccountID: "c", Amount: 50},
	}
	result := service.SimplifyTransfers(transfers)
	if len(result) != 2 {
		t.Fatalf("expected 2 simplified transfers, got %d: %+v", len(result), result)
	}
	for _, tr := range result {
		if tr.FromAccountID == "a" && tr.ToAccountID == "b" && tr.Amount != 70 {
			t.Fatalf("expected A->B = 70, got %f", tr.Amount)
		}
		if tr.FromAccountID == "a" && tr.ToAccountID == "c" && tr.Amount != 50 {
			t.Fatalf("expected A->C = 50, got %f", tr.Amount)
		}
	}
}

func TestSimplifyTransfersRemovesSelfAndExternal(t *testing.T) {
	transfers := []service.Transfer{
		{FromAccountID: "a", ToAccountID: "a", Amount: 100},
		{FromAccountID: "", ToAccountID: "b", Amount: 50},
		{FromAccountID: "c", ToAccountID: "", Amount: 30},
		{FromAccountID: "a", ToAccountID: "b", Amount: 200},
	}
	result := service.SimplifyTransfers(transfers)
	if len(result) != 1 {
		t.Fatalf("expected 1 simplified transfer, got %d: %+v", len(result), result)
	}
	if result[0].Amount != 200 {
		t.Fatalf("expected amount 200, got %f", result[0].Amount)
	}
}

// ---- Accounts Detailed View ----

func TestListAccountsDetailed(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()

	at, _ := svc.UpsertAccountType(ctx, service.AccountTypeInput{Name: "Bank"})
	acc, _ := svc.UpsertAccount(ctx, service.AccountInput{Name: "Savings", TypeID: at.ID})
	svc.UpsertAccountSnapshot(ctx, acc.ID, service.AccountSnapshotInput{
		Date:    mustParseDate("2025-01-01"),
		Balance: newFixedValue(3000),
	})
	svc.UpsertAccountGrowthModel(ctx, service.AccountGrowthModelInput{
		AccountID:        acc.ID,
		Type:             "fixed",
		AnnualRate:       newFixedValue(0.03),
		AnnualVolatility: newFixedValue(0),
		StartDate:        mustParseDate("2025-01-01"),
	})

	detailed, err := svc.ListAccountsDetailed(ctx, mustParseDate("2025-03-15"))
	if err != nil {
		t.Fatalf("list detailed: %v", err)
	}
	if len(detailed) != 1 {
		t.Fatalf("expected 1 detailed account, got %d", len(detailed))
	}
	if detailed[0].LastSnapshot == nil {
		t.Fatal("expected non-nil last snapshot")
	}
	if detailed[0].GrowthModel == nil {
		t.Fatal("expected non-nil growth model")
	}
}

// ---- Transfer Template Child Amount Computation ----

func TestGetTransferTemplatesPageData_ChildAmountsComputed(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()

	pastEnd := mustParseDate("2020-12-31")
	parent, err := svc.UpsertTransferTemplate(ctx, service.TransferTemplate{
		Name:        "Rent",
		AmountType:  "fixed",
		AmountFixed: newFixedValue(1500),
		Priority:    1,
		Recurrence:  "*-*-01",
		StartDate:   mustParseDate("2020-01-01"),
		EndDate:     &pastEnd,
		Enabled:     true,
	})
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}

	_, err = svc.UpsertTransferTemplate(ctx, service.TransferTemplate{
		Name:             "Rent",
		AmountType:       "fixed",
		AmountFixed:      newFixedValue(2000),
		Priority:         1,
		Recurrence:       "*-*-01",
		StartDate:        mustParseDate("2020-01-01"),
		Enabled:          true,
		ParentTemplateID: &parent.ID,
	})
	if err != nil {
		t.Fatalf("create child: %v", err)
	}

	view, err := svc.GetTransferTemplatesPageData(ctx)
	if err != nil {
		t.Fatalf("GetTransferTemplatesPageData: %v", err)
	}

	if len(view.TransferTemplates) != 1 {
		t.Fatalf("expected 1 root template, got %d", len(view.TransferTemplates))
	}

	parentTpl := view.TransferTemplates[0]
	if parentTpl.Amount != 0 {
		t.Errorf("inactive parent: expected amount 0, got %f", parentTpl.Amount)
	}

	if len(parentTpl.ChildTemplates) != 1 {
		t.Fatalf("expected 1 child template, got %d", len(parentTpl.ChildTemplates))
	}

	childWithAmount := view.GetChildWithAmount(parentTpl.ChildTemplates[0])
	if childWithAmount.Amount != 2000 {
		t.Errorf("active child: expected amount 2000, got %f", childWithAmount.Amount)
	}
}

func TestGetTransferTemplatesPageData_ActiveChildContributesToMonthlyIncome(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()

	acc, err := svc.UpsertAccount(ctx, service.AccountInput{Name: "Checking"})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}

	pastEnd := mustParseDate("2020-12-31")
	parent, err := svc.UpsertTransferTemplate(ctx, service.TransferTemplate{
		Name:        "Salary",
		ToAccountID: acc.ID,
		AmountType:  "fixed",
		AmountFixed: newFixedValue(4000),
		Priority:    1,
		Recurrence:  "*-*-25",
		StartDate:   mustParseDate("2020-01-01"),
		EndDate:     &pastEnd,
		Enabled:     true,
	})
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}

	_, err = svc.UpsertTransferTemplate(ctx, service.TransferTemplate{
		Name:             "Salary",
		ToAccountID:      acc.ID,
		AmountType:       "fixed",
		AmountFixed:      newFixedValue(5000),
		Priority:         1,
		Recurrence:       "*-*-25",
		StartDate:        mustParseDate("2020-01-01"),
		Enabled:          true,
		ParentTemplateID: &parent.ID,
	})
	if err != nil {
		t.Fatalf("create child: %v", err)
	}

	view, err := svc.GetTransferTemplatesPageData(ctx)
	if err != nil {
		t.Fatalf("GetTransferTemplatesPageData: %v", err)
	}

	if view.MonthlyIncome != 5000 {
		t.Errorf("expected monthly income 5000 from active child, got %f", view.MonthlyIncome)
	}
}

func TestComputeTransfersView_IncludesActiveChildOfInactiveParent(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()

	acc1, _ := svc.UpsertAccount(ctx, service.AccountInput{Name: "Checking"})
	acc2, _ := svc.UpsertAccount(ctx, service.AccountInput{Name: "Savings"})

	pastEnd := mustParseDate("2024-12-31")
	parent, err := svc.UpsertTransferTemplate(ctx, service.TransferTemplate{
		Name:          "Rent",
		FromAccountID: acc1.ID,
		ToAccountID:   acc2.ID,
		AmountType:    "fixed",
		AmountFixed:   newFixedValue(1500),
		Priority:      1,
		Recurrence:    "*-*-01",
		StartDate:     mustParseDate("2020-01-01"),
		EndDate:       &pastEnd,
		Enabled:       true,
	})
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}

	child, err := svc.UpsertTransferTemplate(ctx, service.TransferTemplate{
		Name:             "Rent",
		FromAccountID:    acc1.ID,
		ToAccountID:      acc2.ID,
		AmountType:       "fixed",
		AmountFixed:      newFixedValue(2000),
		Priority:         1,
		Recurrence:       "*-*-01",
		StartDate:        mustParseDate("2025-01-01"),
		Enabled:          true,
		ParentTemplateID: &parent.ID,
	})
	if err != nil {
		t.Fatalf("create child: %v", err)
	}

	day := mustParseDate("2025-06-01")
	view, err := svc.ComputeTransfersView(ctx, day, nil)
	if err != nil {
		t.Fatalf("ComputeTransfersView: %v", err)
	}

	found := false
	for _, tt := range view.TransferTemplates {
		if tt.ID == child.ID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("active child template %s not found in TransfersView; parent-child should be a visual grouping only", child.ID)
	}
}

// ---- Salary CRUD ----

func TestSalaryCRUD(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()

	acc, _ := svc.UpsertAccount(ctx, service.AccountInput{Name: "Checking"})

	sal, err := svc.UpsertSalary(ctx, service.Salary{
		Name:        "Acme Corp",
		ToAccountID: acc.ID,
		Priority:    0,
		Recurrence:  "*-*-25",
		Enabled:     true,
	})
	if err != nil {
		t.Fatalf("create salary: %v", err)
	}
	if sal.Name != "Acme Corp" || sal.ID == "" {
		t.Fatalf("unexpected salary: %+v", sal)
	}

	got, err := svc.GetSalary(ctx, sal.ID)
	if err != nil {
		t.Fatalf("get salary: %v", err)
	}
	if got.Name != "Acme Corp" || got.ToAccountID != acc.ID {
		t.Fatalf("unexpected salary: %+v", got)
	}

	list, err := svc.ListSalaries(ctx)
	if err != nil {
		t.Fatalf("list salaries: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 salary, got %d", len(list))
	}

	if err := svc.DeleteSalary(ctx, sal.ID); err != nil {
		t.Fatalf("delete salary: %v", err)
	}
	list, _ = svc.ListSalaries(ctx)
	if len(list) != 0 {
		t.Fatalf("expected 0 salaries after delete, got %d", len(list))
	}
}

func TestSalaryAmountCRUD(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()

	sal, _ := svc.UpsertSalary(ctx, service.Salary{
		Name:    "Acme Corp",
		Enabled: true,
	})

	amt, err := svc.UpsertSalaryAmount(ctx, service.SalaryAmount{
		SalaryID:  sal.ID,
		Amount:    newFixedValue(30000),
		StartDate: mustParseDate("2025-01-01"),
	})
	if err != nil {
		t.Fatalf("create salary amount: %v", err)
	}
	if amt.Amount.Mean() != 30000 || amt.ID == "" {
		t.Fatalf("unexpected salary amount: %+v", amt)
	}

	amounts, err := svc.ListSalaryAmounts(ctx, sal.ID)
	if err != nil {
		t.Fatalf("list salary amounts: %v", err)
	}
	if len(amounts) != 1 {
		t.Fatalf("expected 1 amount, got %d", len(amounts))
	}

	if err := svc.DeleteSalaryAmount(ctx, amt.ID); err != nil {
		t.Fatalf("delete salary amount: %v", err)
	}
	amounts, _ = svc.ListSalaryAmounts(ctx, sal.ID)
	if len(amounts) != 0 {
		t.Fatalf("expected 0 amounts after delete, got %d", len(amounts))
	}
}

func TestSalaryAmountCascadeDelete(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()

	sal, _ := svc.UpsertSalary(ctx, service.Salary{
		Name:    "Acme Corp",
		Enabled: true,
	})
	svc.UpsertSalaryAmount(ctx, service.SalaryAmount{
		SalaryID:  sal.ID,
		Amount:    newFixedValue(30000),
		StartDate: mustParseDate("2025-01-01"),
	})

	if err := svc.DeleteSalary(ctx, sal.ID); err != nil {
		t.Fatalf("delete salary: %v", err)
	}
	amounts, _ := svc.ListSalaryAmounts(ctx, sal.ID)
	if len(amounts) != 0 {
		t.Fatalf("expected cascade delete of amounts, got %d", len(amounts))
	}
}

// ---- Salary Transfer Template Generation (Gross) ----

func TestSalaryGenerateTransferTemplates_GrossSalary(t *testing.T) {
	sal := service.Salary{
		ID:               "sal1",
		Name:             "Acme Corp",
		ToAccountID:      "acc1",
		PensionAccountID: "pension1",
		Priority:         0,
		Recurrence:       "*-*-25",
		Enabled:          true,
		IsGross:          true,
		Amounts: []service.SalaryAmount{
			{ID: "amt1", Amount: newFixedValue(50000), StartDate: mustParseDate("2025-01-01")},
		},
		NetSegments: []service.NetSalarySegment{
			{StartDate: mustParseDate("2025-01-01"), Net: newFixedValue(35000)},
		},
		PensionSegments: []service.PensionSegment{
			{StartDate: mustParseDate("2025-01-01"), Pension: newFixedValue(2500)},
		},
	}

	templates := sal.GenerateTransferTemplates()
	if len(templates) != 2 {
		t.Fatalf("expected 2 templates (net + pension), got %d", len(templates))
	}

	var netTpl, pensionTpl service.TransferTemplate
	for _, tt := range templates {
		if tt.ToAccountID == "acc1" {
			netTpl = tt
		} else if tt.ToAccountID == "pension1" {
			pensionTpl = tt
		}
	}

	if netTpl.AmountFixed.Mean() != 35000 {
		t.Errorf("net template amount = %v, want 35000", netTpl.AmountFixed.Mean())
	}
	if netTpl.FromAccountID != "" {
		t.Errorf("net template FromAccountID should be empty, got %s", netTpl.FromAccountID)
	}

	if pensionTpl.AmountFixed.Mean() != 2500 {
		t.Errorf("pension template amount = %v, want 2500", pensionTpl.AmountFixed.Mean())
	}
	if pensionTpl.FromAccountID != "" {
		t.Errorf("pension template FromAccountID should be empty, got %s", pensionTpl.FromAccountID)
	}
	if pensionTpl.ToAccountID != "pension1" {
		t.Errorf("pension template ToAccountID = %s, want pension1", pensionTpl.ToAccountID)
	}
}

func TestSalaryGenerateTransferTemplates_GrossNoPensionAccount(t *testing.T) {
	sal := service.Salary{
		ID:          "sal1",
		Name:        "Acme Corp",
		ToAccountID: "acc1",
		Priority:    0,
		Recurrence:  "*-*-25",
		Enabled:     true,
		IsGross:     true,
		Amounts: []service.SalaryAmount{
			{ID: "amt1", Amount: newFixedValue(50000), StartDate: mustParseDate("2025-01-01")},
		},
		NetSegments: []service.NetSalarySegment{
			{StartDate: mustParseDate("2025-01-01"), Net: newFixedValue(35000)},
		},
		PensionSegments: []service.PensionSegment{
			{StartDate: mustParseDate("2025-01-01"), Pension: newFixedValue(2500)},
		},
	}

	templates := sal.GenerateTransferTemplates()
	if len(templates) != 1 {
		t.Fatalf("expected 1 template (net only, no pension account), got %d", len(templates))
	}
	if templates[0].AmountFixed.Mean() != 35000 {
		t.Errorf("net template amount = %v, want 35000", templates[0].AmountFixed.Mean())
	}
}

func TestSalaryGenerateTransferTemplates_GrossWithIBBChange(t *testing.T) {
	// One salary amount spanning two IBB values:
	// Salary: 50000 from 2025-01-01
	// IBB: 76200 from 2025-01-01, 80000 from 2025-07-01
	// Expected: 1 net TT, 2 pension TTs (split at IBB boundary)

	ibb1 := 76200.0
	ibb2 := 80000.0
	gross := 50000.0
	pension1 := swe.CalculateITP1Pension(gross, ibb1)
	pension2 := swe.CalculateITP1Pension(gross, ibb2)

	ibbChangeDate := mustParseDate("2025-07-01")

	sal := service.Salary{
		ID:               "sal1",
		Name:             "Acme Corp",
		ToAccountID:      "acc1",
		PensionAccountID: "pension1",
		Priority:         0,
		Recurrence:       "*-*-25",
		Enabled:          true,
		IsGross:          true,
		Amounts: []service.SalaryAmount{
			{ID: "amt1", Amount: newFixedValue(gross), StartDate: mustParseDate("2025-01-01")},
		},
		NetSegments: []service.NetSalarySegment{
			{StartDate: mustParseDate("2025-01-01"), Net: newFixedValue(35000)},
		},
		PensionSegments: []service.PensionSegment{
			{StartDate: mustParseDate("2025-01-01"), EndDate: &ibbChangeDate, Pension: newFixedValue(pension1)},
			{StartDate: mustParseDate("2025-07-01"), Pension: newFixedValue(pension2)},
		},
	}

	templates := sal.GenerateTransferTemplates()

	var netTTs, pensionTTs []service.TransferTemplate
	for _, tt := range templates {
		if tt.ToAccountID == "acc1" {
			netTTs = append(netTTs, tt)
		} else if tt.ToAccountID == "pension1" {
			pensionTTs = append(pensionTTs, tt)
		}
	}

	if len(netTTs) != 1 {
		t.Fatalf("expected 1 net TT, got %d", len(netTTs))
	}
	if netTTs[0].AmountFixed.Mean() != 35000 {
		t.Errorf("net amount = %v, want 35000", netTTs[0].AmountFixed.Mean())
	}

	if len(pensionTTs) != 2 {
		t.Fatalf("expected 2 pension TTs (split at IBB change), got %d", len(pensionTTs))
	}
	if pensionTTs[0].AmountFixed.Mean() != pension1 {
		t.Errorf("pension[0] = %v, want %v", pensionTTs[0].AmountFixed.Mean(), pension1)
	}
	if pensionTTs[0].EndDate == nil || *pensionTTs[0].EndDate != ibbChangeDate {
		t.Errorf("pension[0] EndDate = %v, want %v", pensionTTs[0].EndDate, ibbChangeDate)
	}
	if pensionTTs[1].AmountFixed.Mean() != pension2 {
		t.Errorf("pension[1] = %v, want %v", pensionTTs[1].AmountFixed.Mean(), pension2)
	}
	if pensionTTs[1].EndDate != nil {
		t.Errorf("pension[1] EndDate should be nil, got %v", pensionTTs[1].EndDate)
	}
}

func TestSalaryGenerateTransferTemplates_GrossMultipleAmountsAndIBB(t *testing.T) {
	// Salary: 40000 from 2025-01-01, 45000 from 2026-01-01
	// IBB: 76200 from 2025-01-01, 80000 from 2025-07-01
	// Expected pension segments:
	//   [2025-01-01, 2025-07-01) -> pension(40000, 76200)
	//   [2025-07-01, 2026-01-01) -> pension(40000, 80000)
	//   [2026-01-01, nil)        -> pension(45000, 80000)

	ibb1 := 76200.0
	ibb2 := 80000.0
	d1 := mustParseDate("2025-01-01")
	d2 := mustParseDate("2025-07-01")
	d3 := mustParseDate("2026-01-01")
	p1 := swe.CalculateITP1Pension(40000, ibb1)
	p2 := swe.CalculateITP1Pension(40000, ibb2)
	p3 := swe.CalculateITP1Pension(45000, ibb2)

	sal := service.Salary{
		ID:               "sal1",
		Name:             "Acme Corp",
		ToAccountID:      "acc1",
		PensionAccountID: "pension1",
		Recurrence:       "*-*-25",
		Enabled:          true,
		IsGross:          true,
		Amounts: []service.SalaryAmount{
			{ID: "amt1", Amount: newFixedValue(40000), StartDate: d1},
			{ID: "amt2", Amount: newFixedValue(45000), StartDate: d3},
		},
		NetSegments: []service.NetSalarySegment{
			{StartDate: d1, EndDate: &d3, Net: newFixedValue(32500)},
			{StartDate: d3, Net: newFixedValue(37500)},
		},
		PensionSegments: []service.PensionSegment{
			{StartDate: d1, EndDate: &d2, Pension: newFixedValue(p1)},
			{StartDate: d2, EndDate: &d3, Pension: newFixedValue(p2)},
			{StartDate: d3, Pension: newFixedValue(p3)},
		},
	}

	templates := sal.GenerateTransferTemplates()

	var netTTs, pensionTTs []service.TransferTemplate
	for _, tt := range templates {
		if tt.ToAccountID == "acc1" {
			netTTs = append(netTTs, tt)
		} else if tt.ToAccountID == "pension1" {
			pensionTTs = append(pensionTTs, tt)
		}
	}

	if len(netTTs) != 2 {
		t.Fatalf("expected 2 net TTs, got %d", len(netTTs))
	}
	if len(pensionTTs) != 3 {
		t.Fatalf("expected 3 pension TTs, got %d", len(pensionTTs))
	}
	if pensionTTs[0].AmountFixed.Mean() != p1 {
		t.Errorf("pension[0] = %v, want %v", pensionTTs[0].AmountFixed.Mean(), p1)
	}
	if pensionTTs[1].AmountFixed.Mean() != p2 {
		t.Errorf("pension[1] = %v, want %v", pensionTTs[1].AmountFixed.Mean(), p2)
	}
	if pensionTTs[2].AmountFixed.Mean() != p3 {
		t.Errorf("pension[2] = %v, want %v", pensionTTs[2].AmountFixed.Mean(), p3)
	}
}

// ---- BuildNetSegments (date-union splitting logic) ----

// stubNetCalculator returns a calculator that applies adjustments and returns
// a fixed net value = AdjustGrossSalary(gross.Mean(), params).
// This lets us test the splitting logic without a real tax API.
func stubNetCalculator(gross uncertain.Value, adjParams swe.SalaryAdjustmentParams, _ date.Date) (uncertain.Value, error) {
	adjusted := swe.AdjustGrossSalary(gross.Mean(), adjParams)
	return uncertain.NewFixed(adjusted), nil
}

func TestBuildNetSegments_SplitsAtAdjustmentChange(t *testing.T) {
	d1 := mustParseDate("2025-01-01")
	d2 := mustParseDate("2025-07-01")
	pbb := 57300.0

	sal := service.Salary{
		Amounts: []service.SalaryAmount{
			{ID: "amt1", Amount: newFixedValue(50000), StartDate: d1},
		},
		Adjustments: []service.SalaryAdjustment{
			{ValidFrom: d1, VacationDaysPerYear: 25},
			{ValidFrom: d2, VacationDaysPerYear: 30, SickDaysPerOccasion: 3, SickOccasionsPerYear: 4},
		},
	}
	ibbs := []service.Inkomstbasbelopp{
		{ValidFrom: d1, Prisbasbelopp: pbb},
	}

	segments, err := service.BuildNetSegments(sal, ibbs, stubNetCalculator)
	if err != nil {
		t.Fatalf("BuildNetSegments: %v", err)
	}

	if len(segments) != 2 {
		t.Fatalf("expected 2 segments (split at adjustment change), got %d", len(segments))
	}

	if segments[0].StartDate != d1 {
		t.Errorf("seg[0] StartDate = %v, want %v", segments[0].StartDate, d1)
	}
	if segments[0].EndDate == nil || *segments[0].EndDate != d2 {
		t.Errorf("seg[0] EndDate = %v, want %v", segments[0].EndDate, d2)
	}
	if segments[1].StartDate != d2 {
		t.Errorf("seg[1] StartDate = %v, want %v", segments[1].StartDate, d2)
	}
	if segments[1].EndDate != nil {
		t.Errorf("seg[1] EndDate should be nil, got %v", segments[1].EndDate)
	}

	// With different adjustments, the net values should differ
	if segments[0].Net.Mean() == segments[1].Net.Mean() {
		t.Errorf("expected different net values for different adjustments, both = %v", segments[0].Net.Mean())
	}
}

func TestBuildNetSegments_SplitsAtPBBChange(t *testing.T) {
	d1 := mustParseDate("2025-01-01")
	d2 := mustParseDate("2025-07-01")

	sal := service.Salary{
		Amounts: []service.SalaryAmount{
			{ID: "amt1", Amount: newFixedValue(55000), StartDate: d1},
		},
		Adjustments: []service.SalaryAdjustment{
			{ValidFrom: d1, SickDaysPerOccasion: 3, SickOccasionsPerYear: 4, VABDaysPerYear: 10},
		},
	}
	ibbs := []service.Inkomstbasbelopp{
		{ValidFrom: d1, Prisbasbelopp: 52500},
		{ValidFrom: d2, Prisbasbelopp: 57300},
	}

	segments, err := service.BuildNetSegments(sal, ibbs, stubNetCalculator)
	if err != nil {
		t.Fatalf("BuildNetSegments: %v", err)
	}

	if len(segments) != 2 {
		t.Fatalf("expected 2 segments (split at PBB change), got %d", len(segments))
	}
	if segments[0].StartDate != d1 {
		t.Errorf("seg[0] StartDate = %v, want %v", segments[0].StartDate, d1)
	}
	if segments[1].StartDate != d2 {
		t.Errorf("seg[1] StartDate = %v, want %v", segments[1].StartDate, d2)
	}

	// PBB affects sick/VAB caps, so net should differ
	if segments[0].Net.Mean() == segments[1].Net.Mean() {
		t.Errorf("expected different net values for different PBB, both = %v", segments[0].Net.Mean())
	}
}

func TestBuildNetSegments_SplitsAtAllBoundaries(t *testing.T) {
	d1 := mustParseDate("2025-01-01")
	d2 := mustParseDate("2025-04-01") // PBB change
	d3 := mustParseDate("2025-07-01") // adjustment change
	d4 := mustParseDate("2026-01-01") // salary amount change

	sal := service.Salary{
		Amounts: []service.SalaryAmount{
			{ID: "amt1", Amount: newFixedValue(50000), StartDate: d1},
			{ID: "amt2", Amount: newFixedValue(55000), StartDate: d4},
		},
		Adjustments: []service.SalaryAdjustment{
			{ValidFrom: d1, VacationDaysPerYear: 25, SickDaysPerOccasion: 3, SickOccasionsPerYear: 4},
			{ValidFrom: d3, VacationDaysPerYear: 25, SickDaysPerOccasion: 3, SickOccasionsPerYear: 6},
		},
	}
	ibbs := []service.Inkomstbasbelopp{
		{ValidFrom: d1, Prisbasbelopp: 52500},
		{ValidFrom: d2, Prisbasbelopp: 57300},
	}

	segments, err := service.BuildNetSegments(sal, ibbs, stubNetCalculator)
	if err != nil {
		t.Fatalf("BuildNetSegments: %v", err)
	}

	if len(segments) != 4 {
		t.Fatalf("expected 4 segments (amount + PBB + adjustment + amount changes), got %d", len(segments))
	}

	wantDates := []date.Date{d1, d2, d3, d4}
	for i, seg := range segments {
		if seg.StartDate != wantDates[i] {
			t.Errorf("seg[%d] StartDate = %v, want %v", i, seg.StartDate, wantDates[i])
		}
		if i < len(segments)-1 {
			if seg.EndDate == nil || *seg.EndDate != wantDates[i+1] {
				t.Errorf("seg[%d] EndDate = %v, want %v", i, seg.EndDate, wantDates[i+1])
			}
		} else {
			if seg.EndDate != nil {
				t.Errorf("seg[%d] EndDate should be nil, got %v", i, seg.EndDate)
			}
		}
	}

	// Salary amount changes at d4 (50k -> 55k), so seg[3] should have higher net
	if segments[3].Net.Mean() <= segments[2].Net.Mean() {
		t.Errorf("expected seg[3] (%v) > seg[2] (%v) after salary increase", segments[3].Net.Mean(), segments[2].Net.Mean())
	}
}

func TestBuildNetSegments_IgnoresDatesBeforeFirstAmount(t *testing.T) {
	d1 := mustParseDate("2025-06-01")
	dBefore := mustParseDate("2025-01-01")

	sal := service.Salary{
		Amounts: []service.SalaryAmount{
			{ID: "amt1", Amount: newFixedValue(50000), StartDate: d1},
		},
		Adjustments: []service.SalaryAdjustment{
			{ValidFrom: dBefore, VacationDaysPerYear: 25},
		},
	}
	ibbs := []service.Inkomstbasbelopp{
		{ValidFrom: dBefore, Prisbasbelopp: 57300},
	}

	segments, err := service.BuildNetSegments(sal, ibbs, stubNetCalculator)
	if err != nil {
		t.Fatalf("BuildNetSegments: %v", err)
	}

	if len(segments) != 1 {
		t.Fatalf("expected 1 segment (dates before first amount ignored), got %d", len(segments))
	}
	if segments[0].StartDate != d1 {
		t.Errorf("seg[0] StartDate = %v, want %v", segments[0].StartDate, d1)
	}
}

func TestBuildNetSegments_EmptyAmounts(t *testing.T) {
	sal := service.Salary{}
	segments, err := service.BuildNetSegments(sal, nil, stubNetCalculator)
	if err != nil {
		t.Fatalf("BuildNetSegments: %v", err)
	}
	if segments != nil {
		t.Fatalf("expected nil segments for empty amounts, got %d", len(segments))
	}
}

func TestBuildNetSegments_NoAdjustmentsOrPBB(t *testing.T) {
	d1 := mustParseDate("2025-01-01")
	d2 := mustParseDate("2026-01-01")

	sal := service.Salary{
		Amounts: []service.SalaryAmount{
			{ID: "amt1", Amount: newFixedValue(40000), StartDate: d1},
			{ID: "amt2", Amount: newFixedValue(45000), StartDate: d2},
		},
	}

	segments, err := service.BuildNetSegments(sal, nil, stubNetCalculator)
	if err != nil {
		t.Fatalf("BuildNetSegments: %v", err)
	}

	if len(segments) != 2 {
		t.Fatalf("expected 2 segments (one per salary amount), got %d", len(segments))
	}
	if segments[0].StartDate != d1 {
		t.Errorf("seg[0] StartDate = %v, want %v", segments[0].StartDate, d1)
	}
	if segments[1].StartDate != d2 {
		t.Errorf("seg[1] StartDate = %v, want %v", segments[1].StartDate, d2)
	}
	// Without adjustments, net should equal gross
	if segments[0].Net.Mean() != 40000 {
		t.Errorf("seg[0] Net = %v, want 40000 (no adjustments)", segments[0].Net.Mean())
	}
	if segments[1].Net.Mean() != 45000 {
		t.Errorf("seg[1] Net = %v, want 45000 (no adjustments)", segments[1].Net.Mean())
	}
}

func TestSalaryGenerateTransferTemplates_GrossNetSegmentsSplitAtAdjustmentChange(t *testing.T) {
	// One salary amount, but adjustment changes mid-year.
	// Net segments should produce 2 net TTs with different amounts.
	d1 := mustParseDate("2025-01-01")
	d2 := mustParseDate("2025-07-01")

	sal := service.Salary{
		ID:               "sal1",
		Name:             "Test",
		ToAccountID:      "acc1",
		PensionAccountID: "pension1",
		Recurrence:       "*-*-25",
		Enabled:          true,
		IsGross:          true,
		Amounts: []service.SalaryAmount{
			{ID: "amt1", Amount: newFixedValue(50000), StartDate: d1},
		},
		NetSegments: []service.NetSalarySegment{
			{StartDate: d1, EndDate: &d2, Net: newFixedValue(36000)},
			{StartDate: d2, Net: newFixedValue(35000)},
		},
		PensionSegments: []service.PensionSegment{
			{StartDate: d1, Pension: newFixedValue(2500)},
		},
	}

	templates := sal.GenerateTransferTemplates()

	var netTTs []service.TransferTemplate
	for _, tt := range templates {
		if tt.ToAccountID == "acc1" {
			netTTs = append(netTTs, tt)
		}
	}

	if len(netTTs) != 2 {
		t.Fatalf("expected 2 net TTs (split at adjustment change), got %d", len(netTTs))
	}
	if netTTs[0].StartDate != d1 {
		t.Errorf("net[0] StartDate = %v, want %v", netTTs[0].StartDate, d1)
	}
	if netTTs[0].EndDate == nil || *netTTs[0].EndDate != d2 {
		t.Errorf("net[0] EndDate = %v, want %v", netTTs[0].EndDate, d2)
	}
	if netTTs[0].AmountFixed.Mean() != 36000 {
		t.Errorf("net[0] amount = %v, want 36000", netTTs[0].AmountFixed.Mean())
	}
	if netTTs[1].StartDate != d2 {
		t.Errorf("net[1] StartDate = %v, want %v", netTTs[1].StartDate, d2)
	}
	if netTTs[1].EndDate != nil {
		t.Errorf("net[1] EndDate should be nil, got %v", netTTs[1].EndDate)
	}
	if netTTs[1].AmountFixed.Mean() != 35000 {
		t.Errorf("net[1] amount = %v, want 35000", netTTs[1].AmountFixed.Mean())
	}
}

func TestSalaryGenerateTransferTemplates_GrossNetSegmentsSplitAtPBBChange(t *testing.T) {
	// One salary amount, one adjustment, but PBB changes mid-year.
	// This affects sick/VAB caps, producing 2 net TTs.
	d1 := mustParseDate("2025-01-01")
	d2 := mustParseDate("2025-07-01")

	sal := service.Salary{
		ID:          "sal1",
		Name:        "Test",
		ToAccountID: "acc1",
		Recurrence:  "*-*-25",
		Enabled:     true,
		IsGross:     true,
		Amounts: []service.SalaryAmount{
			{ID: "amt1", Amount: newFixedValue(50000), StartDate: d1},
		},
		NetSegments: []service.NetSalarySegment{
			{StartDate: d1, EndDate: &d2, Net: newFixedValue(35500)},
			{StartDate: d2, Net: newFixedValue(35200)},
		},
	}

	templates := sal.GenerateTransferTemplates()

	if len(templates) != 2 {
		t.Fatalf("expected 2 templates (2 net segments, no pension), got %d", len(templates))
	}
	if templates[0].AmountFixed.Mean() != 35500 {
		t.Errorf("net[0] = %v, want 35500", templates[0].AmountFixed.Mean())
	}
	if templates[0].EndDate == nil || *templates[0].EndDate != d2 {
		t.Errorf("net[0] EndDate = %v, want %v", templates[0].EndDate, d2)
	}
	if templates[1].AmountFixed.Mean() != 35200 {
		t.Errorf("net[1] = %v, want 35200", templates[1].AmountFixed.Mean())
	}
	if templates[1].EndDate != nil {
		t.Errorf("net[1] EndDate should be nil, got %v", templates[1].EndDate)
	}
}

func TestSalaryGenerateTransferTemplates_GrossNetSegmentsSplitAtMultipleBoundaries(t *testing.T) {
	// Two salary amounts + adjustment change + PBB change at different dates.
	// Should produce 4 net TTs at the union of all change dates.
	d1 := mustParseDate("2025-01-01")
	d2 := mustParseDate("2025-04-01") // PBB changes
	d3 := mustParseDate("2025-07-01") // adjustment changes
	d4 := mustParseDate("2026-01-01") // salary amount changes

	sal := service.Salary{
		ID:               "sal1",
		Name:             "Test",
		ToAccountID:      "acc1",
		PensionAccountID: "pension1",
		Recurrence:       "*-*-25",
		Enabled:          true,
		IsGross:          true,
		Amounts: []service.SalaryAmount{
			{ID: "amt1", Amount: newFixedValue(40000), StartDate: d1},
			{ID: "amt2", Amount: newFixedValue(45000), StartDate: d4},
		},
		NetSegments: []service.NetSalarySegment{
			{StartDate: d1, EndDate: &d2, Net: newFixedValue(30000)},
			{StartDate: d2, EndDate: &d3, Net: newFixedValue(30100)},
			{StartDate: d3, EndDate: &d4, Net: newFixedValue(29800)},
			{StartDate: d4, Net: newFixedValue(33500)},
		},
		PensionSegments: []service.PensionSegment{
			{StartDate: d1, Pension: newFixedValue(2000)},
		},
	}

	templates := sal.GenerateTransferTemplates()

	var netTTs, pensionTTs []service.TransferTemplate
	for _, tt := range templates {
		if tt.ToAccountID == "acc1" {
			netTTs = append(netTTs, tt)
		} else if tt.ToAccountID == "pension1" {
			pensionTTs = append(pensionTTs, tt)
		}
	}

	if len(netTTs) != 4 {
		t.Fatalf("expected 4 net TTs (split at PBB, adjustment, and salary changes), got %d", len(netTTs))
	}
	if netTTs[0].StartDate != d1 {
		t.Errorf("net[0] StartDate = %v, want %v", netTTs[0].StartDate, d1)
	}
	if netTTs[0].AmountFixed.Mean() != 30000 {
		t.Errorf("net[0] = %v, want 30000", netTTs[0].AmountFixed.Mean())
	}
	if netTTs[1].StartDate != d2 {
		t.Errorf("net[1] StartDate = %v, want %v", netTTs[1].StartDate, d2)
	}
	if netTTs[2].StartDate != d3 {
		t.Errorf("net[2] StartDate = %v, want %v", netTTs[2].StartDate, d3)
	}
	if netTTs[3].StartDate != d4 {
		t.Errorf("net[3] StartDate = %v, want %v", netTTs[3].StartDate, d4)
	}
	if netTTs[3].EndDate != nil {
		t.Errorf("net[3] EndDate should be nil, got %v", netTTs[3].EndDate)
	}
	if netTTs[3].AmountFixed.Mean() != 33500 {
		t.Errorf("net[3] = %v, want 33500", netTTs[3].AmountFixed.Mean())
	}

	if len(pensionTTs) != 1 {
		t.Fatalf("expected 1 pension TT, got %d", len(pensionTTs))
	}
}

// ---- Salary Transfer Template Generation ----

func TestSalaryGenerateTransferTemplates_SingleAmount(t *testing.T) {
	sal := service.Salary{
		ID:          "sal1",
		Name:        "Acme Corp",
		ToAccountID: "acc1",
		Priority:    0,
		Recurrence:  "*-*-25",
		Enabled:     true,
		Amounts: []service.SalaryAmount{
			{ID: "amt1", Amount: newFixedValue(30000), StartDate: mustParseDate("2025-01-01")},
		},
	}

	templates := sal.GenerateTransferTemplates()
	if len(templates) != 1 {
		t.Fatalf("expected 1 template, got %d", len(templates))
	}

	tt := templates[0]
	if tt.ID != "salary:amt1" {
		t.Errorf("expected ID salary:amt1, got %s", tt.ID)
	}
	if tt.Name != "Acme Corp" {
		t.Errorf("expected name Acme Corp, got %s", tt.Name)
	}
	if tt.FromAccountID != "" {
		t.Errorf("expected empty FromAccountID, got %s", tt.FromAccountID)
	}
	if tt.ToAccountID != "acc1" {
		t.Errorf("expected ToAccountID acc1, got %s", tt.ToAccountID)
	}
	if tt.AmountFixed.Mean() != 30000 {
		t.Errorf("expected amount 30000, got %f", tt.AmountFixed.Mean())
	}
	if tt.EndDate != nil {
		t.Errorf("expected nil EndDate for single amount, got %v", tt.EndDate)
	}
	if !tt.Source.IsGenerated() {
		t.Error("expected Source.IsGenerated() to be true")
	}
	if tt.Source.Type != "salary" {
		t.Errorf("expected source type salary, got %s", tt.Source.Type)
	}
	if tt.Source.EntityID != "sal1" {
		t.Errorf("expected source entity ID sal1, got %s", tt.Source.EntityID)
	}
}

func TestSalaryGenerateTransferTemplates_MultipleAmounts(t *testing.T) {
	sal := service.Salary{
		ID:          "sal1",
		Name:        "Acme Corp",
		ToAccountID: "acc1",
		Priority:    0,
		Recurrence:  "*-*-25",
		Enabled:     true,
		Amounts: []service.SalaryAmount{
			{ID: "amt2", Amount: newFixedValue(35000), StartDate: mustParseDate("2026-01-01")},
			{ID: "amt1", Amount: newFixedValue(30000), StartDate: mustParseDate("2025-01-01")},
		},
	}

	templates := sal.GenerateTransferTemplates()
	if len(templates) != 2 {
		t.Fatalf("expected 2 templates, got %d", len(templates))
	}

	first := templates[0]
	if first.AmountFixed.Mean() != 30000 {
		t.Errorf("first template: expected amount 30000, got %f", first.AmountFixed.Mean())
	}
	if first.EndDate == nil {
		t.Fatal("first template: expected non-nil EndDate")
	}
	if *first.EndDate != mustParseDate("2026-01-01") {
		t.Errorf("first template: expected EndDate 2026-01-01, got %v", *first.EndDate)
	}

	second := templates[1]
	if second.AmountFixed.Mean() != 35000 {
		t.Errorf("second template: expected amount 35000, got %f", second.AmountFixed.Mean())
	}
	if second.EndDate != nil {
		t.Errorf("second template: expected nil EndDate, got %v", second.EndDate)
	}
}

func TestSalaryGenerateTransferTemplates_DisabledSalary(t *testing.T) {
	sal := service.Salary{
		ID:      "sal1",
		Name:    "Acme Corp",
		Enabled: false,
		Amounts: []service.SalaryAmount{
			{ID: "amt1", Amount: newFixedValue(30000), StartDate: mustParseDate("2025-01-01")},
		},
	}

	templates := sal.GenerateTransferTemplates()
	if len(templates) != 1 {
		t.Fatalf("expected 1 template, got %d", len(templates))
	}
	if templates[0].Enabled {
		t.Error("expected template to be disabled when salary is disabled")
	}
}

func TestSalaryGenerateTransferTemplates_NoAmounts(t *testing.T) {
	sal := service.Salary{
		ID:      "sal1",
		Name:    "Acme Corp",
		Enabled: true,
	}

	templates := sal.GenerateTransferTemplates()
	if len(templates) != 0 {
		t.Fatalf("expected 0 templates for salary with no amounts, got %d", len(templates))
	}
}

// ---- ListAllTransferTemplates integration ----

func TestListAllTransferTemplates_MergesDBAndSalary(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()

	acc, _ := svc.UpsertAccount(ctx, service.AccountInput{Name: "Checking"})

	svc.UpsertTransferTemplate(ctx, service.TransferTemplate{
		Name:          "Rent",
		FromAccountID: acc.ID,
		AmountType:    "fixed",
		AmountFixed:   newFixedValue(1500),
		Recurrence:    "*-*-01",
		StartDate:     mustParseDate("2025-01-01"),
		Enabled:       true,
	})

	sal, _ := svc.UpsertSalary(ctx, service.Salary{
		Name:        "Acme Corp",
		ToAccountID: acc.ID,
		Priority:    0,
		Recurrence:  "*-*-25",
		Enabled:     true,
	})
	svc.UpsertSalaryAmount(ctx, service.SalaryAmount{
		SalaryID:  sal.ID,
		Amount:    newFixedValue(30000),
		StartDate: mustParseDate("2025-01-01"),
	})

	all, err := svc.ListAllTransferTemplates(ctx)
	if err != nil {
		t.Fatalf("ListAllTransferTemplates: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 templates (1 DB + 1 salary), got %d", len(all))
	}

	var dbCount, salaryCount int
	for _, tt := range all {
		if tt.Source.IsGenerated() {
			salaryCount++
			if tt.Source.Type != "salary" {
				t.Errorf("expected source type salary, got %s", tt.Source.Type)
			}
		} else {
			dbCount++
		}
	}
	if dbCount != 1 || salaryCount != 1 {
		t.Fatalf("expected 1 DB + 1 salary, got %d DB + %d salary", dbCount, salaryCount)
	}
}

func TestListAllTransferTemplatesWithChildren_IncludesSalaryTemplates(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()

	sal, _ := svc.UpsertSalary(ctx, service.Salary{
		Name:    "Acme Corp",
		Enabled: true,
	})
	svc.UpsertSalaryAmount(ctx, service.SalaryAmount{
		SalaryID:  sal.ID,
		Amount:    newFixedValue(30000),
		StartDate: mustParseDate("2025-01-01"),
	})

	all, err := svc.ListAllTransferTemplatesWithChildren(ctx)
	if err != nil {
		t.Fatalf("ListAllTransferTemplatesWithChildren: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 template, got %d", len(all))
	}
	if !all[0].Source.IsGenerated() {
		t.Error("expected salary-generated template")
	}
}

func TestGetBudgetData_IncludesSalaryTemplates(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()

	color := "#00ff00"
	cat, _ := svc.UpsertCategory(ctx, service.TransferTemplateCategoryInput{Name: "Income", Color: &color})
	acc, _ := svc.UpsertAccount(ctx, service.AccountInput{Name: "Checking"})

	sal, _ := svc.UpsertSalary(ctx, service.Salary{
		Name:             "Acme Corp",
		ToAccountID:      acc.ID,
		Priority:         0,
		Recurrence:       "*-*-25",
		BudgetCategoryID: &cat.ID,
		Enabled:          true,
	})
	svc.UpsertSalaryAmount(ctx, service.SalaryAmount{
		SalaryID:  sal.ID,
		Amount:    newFixedValue(30000),
		StartDate: mustParseDate("2020-01-01"),
	})

	budget, err := svc.GetBudgetData(ctx)
	if err != nil {
		t.Fatalf("GetBudgetData: %v", err)
	}
	if budget.GrandTotal != 30000 {
		t.Fatalf("expected grand total 30000, got %f", budget.GrandTotal)
	}
}

// ---- SQLite Cache ----

func TestSQLiteCacheGetSetRoundtrip(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()

	cache := service.NewSQLiteCache(svc.DB())

	_, ok, err := cache.Get(ctx, "missing-key")
	if err != nil {
		t.Fatalf("get missing key: %v", err)
	}
	if ok {
		t.Fatal("expected ok=false for missing key")
	}

	if err := cache.Set(ctx, "test-key", `{"some":"data"}`); err != nil {
		t.Fatalf("set: %v", err)
	}

	val, ok, err := cache.Get(ctx, "test-key")
	if err != nil {
		t.Fatalf("get after set: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true after set")
	}
	if val != `{"some":"data"}` {
		t.Fatalf("unexpected value: %s", val)
	}

	if err := cache.Set(ctx, "test-key", `{"updated":"value"}`); err != nil {
		t.Fatalf("overwrite: %v", err)
	}

	val, _, _ = cache.Get(ctx, "test-key")
	if val != `{"updated":"value"}` {
		t.Fatalf("expected updated value, got: %s", val)
	}
}

// ---- Inkomstbasbelopp CRUD ----

func TestInkomstbasbeloppCRUD(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()

	ibb, err := svc.UpsertInkomstbasbelopp(ctx, service.Inkomstbasbelopp{
		Amount:    76200,
		ValidFrom: mustParseDate("2025-01-01"),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if ibb.Amount != 76200 || ibb.ID == "" {
		t.Fatalf("unexpected: %+v", ibb)
	}

	got, err := svc.GetInkomstbasbelopp(ctx, ibb.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Amount != 76200 || got.ValidFrom != mustParseDate("2025-01-01") {
		t.Fatalf("unexpected get: %+v", got)
	}

	ibb2, err := svc.UpsertInkomstbasbelopp(ctx, service.Inkomstbasbelopp{
		ID:        ibb.ID,
		Amount:    80000,
		ValidFrom: mustParseDate("2025-01-01"),
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if ibb2.Amount != 80000 || ibb2.ID != ibb.ID {
		t.Fatalf("unexpected update: %+v", ibb2)
	}

	list, err := svc.ListInkomstbasbelopp(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1, got %d", len(list))
	}

	if err := svc.DeleteInkomstbasbelopp(ctx, ibb.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	list, _ = svc.ListInkomstbasbelopp(ctx)
	if len(list) != 0 {
		t.Fatalf("expected 0 after delete, got %d", len(list))
	}
}

func TestInkomstbasbeloppOrdering(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()

	svc.UpsertInkomstbasbelopp(ctx, service.Inkomstbasbelopp{Amount: 80000, ValidFrom: mustParseDate("2026-01-01")})
	svc.UpsertInkomstbasbelopp(ctx, service.Inkomstbasbelopp{Amount: 76200, ValidFrom: mustParseDate("2025-01-01")})
	svc.UpsertInkomstbasbelopp(ctx, service.Inkomstbasbelopp{Amount: 84000, ValidFrom: mustParseDate("2027-01-01")})

	list, err := svc.ListInkomstbasbelopp(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3, got %d", len(list))
	}
	if list[0].Amount != 76200 || list[1].Amount != 80000 || list[2].Amount != 84000 {
		t.Fatalf("unexpected order: %v, %v, %v", list[0].Amount, list[1].Amount, list[2].Amount)
	}
}
