package service_test

import (
	"context"
	"testing"

	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/pefigo"
	"github.com/SimonSchneider/pefigo/internal/service"
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
	db, err := service.GetMigratedDB(context.Background(), pefigo.StaticEmbeddedFS, "static/migrations", ":memory:")
	if err != nil {
		t.Fatalf("failed to create test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return service.New(db)
}

// ---- Account Type CRUD ----

func TestAccountTypeCRUD(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

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
	ctx := context.Background()

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
	ctx := context.Background()

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
	ctx := context.Background()

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
	ctx := context.Background()

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
	ctx := context.Background()

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
	ctx := context.Background()

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
	ctx := context.Background()

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
	ctx := context.Background()

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
	ctx := context.Background()

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
	ctx := context.Background()

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
	ctx := context.Background()

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
	ctx := context.Background()

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
