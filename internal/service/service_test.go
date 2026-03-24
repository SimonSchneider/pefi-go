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

func approxEqual(a, b, epsilon float64) bool {
	if a == 0 && b == 0 {
		return true
	}
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < epsilon
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

// ---- Net Segment Splitting (integration via ListAllTransferTemplates) ----

// seedTaxCache pre-populates the SQLite cache with tax rate and table data
// so the swe client doesn't hit the network.
func seedTaxCache(t *testing.T, svc *service.Service, kommun, forsamling, year string) {
	t.Helper()
	ctx := t.Context()
	cache := service.NewSQLiteCache(svc.DB())

	taxRateJSON := `[{"kommun":"` + kommun + `","församling":"` + forsamling + `","summa, exkl. kyrkoavgift":"31","summa, inkl. kyrkoavgift":"32","år":"` + year + `"}]`
	if err := cache.Set(ctx, "tax_rate:"+kommun+":"+forsamling+":"+year, taxRateJSON); err != nil {
		t.Fatalf("seeding tax rate cache: %v", err)
	}

	taxTableJSON := `[` +
		`{"inkomst fr.o.m.":"0","inkomst t.o.m.":"24999","kolumn 1":"0","tabellnr":"31","år":"` + year + `","antal dgr":"30B"},` +
		`{"inkomst fr.o.m.":"25000","inkomst t.o.m.":"49999","kolumn 1":"7500","tabellnr":"31","år":"` + year + `","antal dgr":"30B"},` +
		`{"inkomst fr.o.m.":"50000","inkomst t.o.m.":"99999","kolumn 1":"15000","tabellnr":"31","år":"` + year + `","antal dgr":"30B"}` +
		`]`
	if err := cache.Set(ctx, "tax_table:31:"+year, taxTableJSON); err != nil {
		t.Fatalf("seeding tax table cache: %v", err)
	}
}

func createGrossSalary(t *testing.T, svc *service.Service, name, kommun, forsamling string) service.Salary {
	t.Helper()
	sal, err := svc.UpsertSalary(t.Context(), service.Salary{
		Name:       name,
		Kommun:     kommun,
		Forsamling: forsamling,
		IsGross:    true,
		Enabled:    true,
		Recurrence: "*-*-25",
	})
	if err != nil {
		t.Fatalf("creating salary: %v", err)
	}
	return sal
}

func salaryTTsFromAll(t *testing.T, svc *service.Service, salaryID string) []service.TransferTemplate {
	t.Helper()
	all, err := svc.ListAllTransferTemplates(t.Context())
	if err != nil {
		t.Fatalf("ListAllTransferTemplates: %v", err)
	}
	var result []service.TransferTemplate
	for _, tt := range all {
		if tt.Source.EntityID == salaryID && tt.Source.Type == "salary" {
			result = append(result, tt)
		}
	}
	return result
}

func TestNetSegments_SplitsAtAdjustmentChange(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()
	seedTaxCache(t, svc, "STOCKHOLM", "TEST", "2025")

	sal := createGrossSalary(t, svc, "Test Salary", "STOCKHOLM", "TEST")

	if _, err := svc.UpsertSalaryAmount(ctx, service.SalaryAmount{
		SalaryID:  sal.ID,
		Amount:    newFixedValue(40000),
		StartDate: mustParseDate("2025-01-01"),
	}); err != nil {
		t.Fatalf("creating salary amount: %v", err)
	}

	if _, err := svc.UpsertInkomstbasbelopp(ctx, service.Inkomstbasbelopp{
		Amount:        76200,
		Prisbasbelopp: 57300,
		ValidFrom:     mustParseDate("2025-01-01"),
	}); err != nil {
		t.Fatalf("creating inkomstbasbelopp: %v", err)
	}

	if _, err := svc.UpsertSalaryAdjustment(ctx, service.SalaryAdjustment{
		SalaryID:            sal.ID,
		ValidFrom:           mustParseDate("2025-01-01"),
		VacationDaysPerYear: 25,
	}); err != nil {
		t.Fatalf("creating adjustment 1: %v", err)
	}
	if _, err := svc.UpsertSalaryAdjustment(ctx, service.SalaryAdjustment{
		SalaryID:             sal.ID,
		ValidFrom:            mustParseDate("2025-07-01"),
		VacationDaysPerYear:  25,
		SickDaysPerOccasion:  3,
		SickOccasionsPerYear: 4,
	}); err != nil {
		t.Fatalf("creating adjustment 2: %v", err)
	}

	netTTs := salaryTTsFromAll(t, svc, sal.ID)
	if len(netTTs) != 2 {
		t.Fatalf("expected 2 net TTs (split at adjustment change), got %d", len(netTTs))
	}
	if netTTs[0].StartDate != mustParseDate("2025-01-01") {
		t.Errorf("tt[0] StartDate = %v, want 2025-01-01", netTTs[0].StartDate)
	}
	if netTTs[0].EndDate == nil || *netTTs[0].EndDate != mustParseDate("2025-07-01") {
		t.Errorf("tt[0] EndDate = %v, want 2025-07-01", netTTs[0].EndDate)
	}
	if netTTs[1].StartDate != mustParseDate("2025-07-01") {
		t.Errorf("tt[1] StartDate = %v, want 2025-07-01", netTTs[1].StartDate)
	}
	if netTTs[1].EndDate != nil {
		t.Errorf("tt[1] EndDate should be nil, got %v", netTTs[1].EndDate)
	}
	// The second segment has sick deductions, so net should be lower
	if netTTs[1].AmountFixed.Mean() >= netTTs[0].AmountFixed.Mean() {
		t.Errorf("expected tt[1] (%v) < tt[0] (%v) due to sick deduction",
			netTTs[1].AmountFixed.Mean(), netTTs[0].AmountFixed.Mean())
	}
}

func TestNetSegments_SplitsAtPBBChange(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()
	seedTaxCache(t, svc, "STOCKHOLM", "TEST", "2025")

	sal := createGrossSalary(t, svc, "Test Salary", "STOCKHOLM", "TEST")

	// Use 55000 so annual (660k) exceeds both 10*PBB caps, making PBB
	// differences visible in the sick/VAB deduction calculations.
	if _, err := svc.UpsertSalaryAmount(ctx, service.SalaryAmount{
		SalaryID:  sal.ID,
		Amount:    newFixedValue(55000),
		StartDate: mustParseDate("2025-01-01"),
	}); err != nil {
		t.Fatalf("creating salary amount: %v", err)
	}

	if _, err := svc.UpsertInkomstbasbelopp(ctx, service.Inkomstbasbelopp{
		Amount:        76200,
		Prisbasbelopp: 52500,
		ValidFrom:     mustParseDate("2025-01-01"),
	}); err != nil {
		t.Fatalf("creating ibb 1: %v", err)
	}
	if _, err := svc.UpsertInkomstbasbelopp(ctx, service.Inkomstbasbelopp{
		Amount:        76200,
		Prisbasbelopp: 57300,
		ValidFrom:     mustParseDate("2025-07-01"),
	}); err != nil {
		t.Fatalf("creating ibb 2: %v", err)
	}

	if _, err := svc.UpsertSalaryAdjustment(ctx, service.SalaryAdjustment{
		SalaryID:             sal.ID,
		ValidFrom:            mustParseDate("2025-01-01"),
		SickDaysPerOccasion:  3,
		SickOccasionsPerYear: 4,
		VABDaysPerYear:       10,
	}); err != nil {
		t.Fatalf("creating adjustment: %v", err)
	}

	netTTs := salaryTTsFromAll(t, svc, sal.ID)
	if len(netTTs) != 2 {
		t.Fatalf("expected 2 net TTs (split at PBB change), got %d", len(netTTs))
	}
	if netTTs[0].StartDate != mustParseDate("2025-01-01") {
		t.Errorf("tt[0] StartDate = %v, want 2025-01-01", netTTs[0].StartDate)
	}
	if netTTs[1].StartDate != mustParseDate("2025-07-01") {
		t.Errorf("tt[1] StartDate = %v, want 2025-07-01", netTTs[1].StartDate)
	}
	// Higher PBB means higher cap → less deduction → higher net
	if netTTs[1].AmountFixed.Mean() <= netTTs[0].AmountFixed.Mean() {
		t.Errorf("expected tt[1] (%v) > tt[0] (%v) with higher PBB cap",
			netTTs[1].AmountFixed.Mean(), netTTs[0].AmountFixed.Mean())
	}
}

func TestNetSegments_SplitsAtAllBoundaries(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()
	seedTaxCache(t, svc, "STOCKHOLM", "TEST", "2025")
	seedTaxCache(t, svc, "STOCKHOLM", "TEST", "2026")

	sal := createGrossSalary(t, svc, "Test Salary", "STOCKHOLM", "TEST")

	if _, err := svc.UpsertSalaryAmount(ctx, service.SalaryAmount{
		SalaryID:  sal.ID,
		Amount:    newFixedValue(40000),
		StartDate: mustParseDate("2025-01-01"),
	}); err != nil {
		t.Fatalf("creating amount 1: %v", err)
	}
	if _, err := svc.UpsertSalaryAmount(ctx, service.SalaryAmount{
		SalaryID:  sal.ID,
		Amount:    newFixedValue(45000),
		StartDate: mustParseDate("2026-01-01"),
	}); err != nil {
		t.Fatalf("creating amount 2: %v", err)
	}

	if _, err := svc.UpsertInkomstbasbelopp(ctx, service.Inkomstbasbelopp{
		Amount:        76200,
		Prisbasbelopp: 52500,
		ValidFrom:     mustParseDate("2025-01-01"),
	}); err != nil {
		t.Fatalf("creating ibb 1: %v", err)
	}
	if _, err := svc.UpsertInkomstbasbelopp(ctx, service.Inkomstbasbelopp{
		Amount:        76200,
		Prisbasbelopp: 57300,
		ValidFrom:     mustParseDate("2025-04-01"),
	}); err != nil {
		t.Fatalf("creating ibb 2: %v", err)
	}

	if _, err := svc.UpsertSalaryAdjustment(ctx, service.SalaryAdjustment{
		SalaryID:             sal.ID,
		ValidFrom:            mustParseDate("2025-01-01"),
		VacationDaysPerYear:  25,
		SickDaysPerOccasion:  3,
		SickOccasionsPerYear: 4,
	}); err != nil {
		t.Fatalf("creating adjustment 1: %v", err)
	}
	if _, err := svc.UpsertSalaryAdjustment(ctx, service.SalaryAdjustment{
		SalaryID:             sal.ID,
		ValidFrom:            mustParseDate("2025-07-01"),
		VacationDaysPerYear:  25,
		SickDaysPerOccasion:  3,
		SickOccasionsPerYear: 6,
	}); err != nil {
		t.Fatalf("creating adjustment 2: %v", err)
	}

	netTTs := salaryTTsFromAll(t, svc, sal.ID)

	wantDates := []date.Date{
		mustParseDate("2025-01-01"),
		mustParseDate("2025-04-01"),
		mustParseDate("2025-07-01"),
		mustParseDate("2026-01-01"),
	}
	if len(netTTs) != len(wantDates) {
		t.Fatalf("expected %d net TTs (split at amount + PBB + adjustment boundaries), got %d", len(wantDates), len(netTTs))
	}
	for i, tt := range netTTs {
		if tt.StartDate != wantDates[i] {
			t.Errorf("tt[%d] StartDate = %v, want %v", i, tt.StartDate, wantDates[i])
		}
		if i < len(netTTs)-1 {
			if tt.EndDate == nil || *tt.EndDate != wantDates[i+1] {
				t.Errorf("tt[%d] EndDate = %v, want %v", i, tt.EndDate, wantDates[i+1])
			}
		} else {
			if tt.EndDate != nil {
				t.Errorf("tt[%d] EndDate should be nil, got %v", i, tt.EndDate)
			}
		}
	}
	// Last segment has higher salary (45k vs 40k), so net should be higher
	if netTTs[3].AmountFixed.Mean() <= netTTs[2].AmountFixed.Mean() {
		t.Errorf("expected tt[3] (%v) > tt[2] (%v) after salary increase",
			netTTs[3].AmountFixed.Mean(), netTTs[2].AmountFixed.Mean())
	}
}

func TestNetSegments_NoSplitWithoutAdjustmentsOrPBB(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()
	seedTaxCache(t, svc, "STOCKHOLM", "TEST", "2025")

	sal := createGrossSalary(t, svc, "Test Salary", "STOCKHOLM", "TEST")

	if _, err := svc.UpsertSalaryAmount(ctx, service.SalaryAmount{
		SalaryID:  sal.ID,
		Amount:    newFixedValue(40000),
		StartDate: mustParseDate("2025-01-01"),
	}); err != nil {
		t.Fatalf("creating salary amount: %v", err)
	}

	netTTs := salaryTTsFromAll(t, svc, sal.ID)
	if len(netTTs) != 1 {
		t.Fatalf("expected 1 net TT (no adjustments or PBB to split on), got %d", len(netTTs))
	}
	if netTTs[0].StartDate != mustParseDate("2025-01-01") {
		t.Errorf("tt[0] StartDate = %v, want 2025-01-01", netTTs[0].StartDate)
	}
	if netTTs[0].EndDate != nil {
		t.Errorf("tt[0] EndDate should be nil, got %v", netTTs[0].EndDate)
	}
}

// ---- Partial Parental Leave ----

func TestPartialParentalLeaveCRUD(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()

	sal, err := svc.UpsertSalary(ctx, service.Salary{Name: "Test", Enabled: true, Recurrence: "*-*-25"})
	if err != nil {
		t.Fatalf("creating salary: %v", err)
	}

	ppl, err := svc.UpsertPartialParentalLeave(ctx, service.PartialParentalLeave{
		SalaryID:               sal.ID,
		StartDate:              mustParseDate("2025-01-01"),
		EndDate:                mustParseDate("2026-01-01"),
		SjukDaysPerYear:        40,
		LagstaDaysPerYear:      10,
		SkippedWorkDaysPerYear: 50,
	})
	if err != nil {
		t.Fatalf("creating partial parental leave: %v", err)
	}
	if ppl.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if ppl.SjukDaysPerYear != 40 || ppl.LagstaDaysPerYear != 10 || ppl.SkippedWorkDaysPerYear != 50 {
		t.Fatalf("unexpected values: %+v", ppl)
	}
	if ppl.StartDate != mustParseDate("2025-01-01") || ppl.EndDate != mustParseDate("2026-01-01") {
		t.Fatalf("unexpected dates: start=%v end=%v", ppl.StartDate, ppl.EndDate)
	}

	list, err := svc.ListPartialParentalLeaves(ctx, sal.ID)
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1, got %d", len(list))
	}

	if err := svc.DeletePartialParentalLeave(ctx, ppl.ID); err != nil {
		t.Fatalf("deleting: %v", err)
	}
	list, err = svc.ListPartialParentalLeaves(ctx, sal.ID)
	if err != nil {
		t.Fatalf("listing after delete: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected 0 after delete, got %d", len(list))
	}
}

func TestNetSegments_SplitsAtPartialParentalLeaveChange(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()
	seedTaxCache(t, svc, "STOCKHOLM", "TEST", "2025")

	sal := createGrossSalary(t, svc, "Test Salary", "STOCKHOLM", "TEST")

	if _, err := svc.UpsertSalaryAmount(ctx, service.SalaryAmount{
		SalaryID:  sal.ID,
		Amount:    newFixedValue(40000),
		StartDate: mustParseDate("2025-01-01"),
	}); err != nil {
		t.Fatalf("creating salary amount: %v", err)
	}

	if _, err := svc.UpsertInkomstbasbelopp(ctx, service.Inkomstbasbelopp{
		Amount:        76200,
		Prisbasbelopp: 57300,
		ValidFrom:     mustParseDate("2025-01-01"),
	}); err != nil {
		t.Fatalf("creating inkomstbasbelopp: %v", err)
	}

	// No parental leave initially — just 1 segment
	netTTs := salaryTTsFromAll(t, svc, sal.ID)
	if len(netTTs) != 1 {
		t.Fatalf("expected 1 net TT before parental leave, got %d", len(netTTs))
	}
	baseNet := netTTs[0].AmountFixed.Mean()

	// Add partial parental leave from mid-year to end of year
	if _, err := svc.UpsertPartialParentalLeave(ctx, service.PartialParentalLeave{
		SalaryID:               sal.ID,
		StartDate:              mustParseDate("2025-07-01"),
		EndDate:                mustParseDate("2026-01-01"),
		SjukDaysPerYear:        40,
		LagstaDaysPerYear:      10,
		SkippedWorkDaysPerYear: 50,
	}); err != nil {
		t.Fatalf("creating partial parental leave: %v", err)
	}

	netTTs = salaryTTsFromAll(t, svc, sal.ID)
	// Should split: [2025-01-01, 2025-07-01) normal, [2025-07-01, 2026-01-01) with deduction, [2026-01-01, ...) normal
	if len(netTTs) != 3 {
		t.Fatalf("expected 3 net TTs (before/during/after partial leave), got %d", len(netTTs))
	}
	if netTTs[0].StartDate != mustParseDate("2025-01-01") {
		t.Errorf("tt[0] StartDate = %v, want 2025-01-01", netTTs[0].StartDate)
	}
	if netTTs[0].EndDate == nil || *netTTs[0].EndDate != mustParseDate("2025-07-01") {
		t.Errorf("tt[0] EndDate = %v, want 2025-07-01", netTTs[0].EndDate)
	}
	if netTTs[1].StartDate != mustParseDate("2025-07-01") {
		t.Errorf("tt[1] StartDate = %v, want 2025-07-01", netTTs[1].StartDate)
	}
	if netTTs[1].EndDate == nil || *netTTs[1].EndDate != mustParseDate("2026-01-01") {
		t.Errorf("tt[1] EndDate = %v, want 2026-01-01", netTTs[1].EndDate)
	}
	if netTTs[2].StartDate != mustParseDate("2026-01-01") {
		t.Errorf("tt[2] StartDate = %v, want 2026-01-01", netTTs[2].StartDate)
	}

	// First segment should be unchanged (no parental leave active)
	if !approxEqual(netTTs[0].AmountFixed.Mean(), baseNet, 0.01) {
		t.Errorf("tt[0] net = %v, want ~%v (unchanged)", netTTs[0].AmountFixed.Mean(), baseNet)
	}
	// Second segment should be lower due to parental leave deduction
	if netTTs[1].AmountFixed.Mean() >= netTTs[0].AmountFixed.Mean() {
		t.Errorf("expected tt[1] (%v) < tt[0] (%v) due to parental leave deduction",
			netTTs[1].AmountFixed.Mean(), netTTs[0].AmountFixed.Mean())
	}
	// Third segment (after leave ends) should return to normal
	if !approxEqual(netTTs[2].AmountFixed.Mean(), baseNet, 1.0) {
		t.Errorf("tt[2] net = %v, want ~%v (back to normal)", netTTs[2].AmountFixed.Mean(), baseNet)
	}
}

// ---- Full Parental Leave ----

func TestFullParentalLeaveCRUD(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()

	sal, err := svc.UpsertSalary(ctx, service.Salary{Name: "Test", Enabled: true, Recurrence: "*-*-25"})
	if err != nil {
		t.Fatalf("creating salary: %v", err)
	}

	fpl, err := svc.UpsertFullParentalLeave(ctx, service.FullParentalLeave{
		SalaryID:        sal.ID,
		StartDate:       mustParseDate("2025-06-01"),
		EndDate:         mustParseDate("2026-06-01"),
		SjukDaysPerWeek: 5,
	})
	if err != nil {
		t.Fatalf("creating full parental leave: %v", err)
	}
	if fpl.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if fpl.SjukDaysPerWeek != 5 {
		t.Fatalf("unexpected sjuk days: %v", fpl.SjukDaysPerWeek)
	}

	list, err := svc.ListFullParentalLeaves(ctx, sal.ID)
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1, got %d", len(list))
	}

	if err := svc.DeleteFullParentalLeave(ctx, fpl.ID); err != nil {
		t.Fatalf("deleting: %v", err)
	}
	list, err = svc.ListFullParentalLeaves(ctx, sal.ID)
	if err != nil {
		t.Fatalf("listing after delete: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected 0 after delete, got %d", len(list))
	}
}

func TestNetSegments_FullParentalLeaveOverridesNetSalary(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()
	seedTaxCache(t, svc, "STOCKHOLM", "TEST", "2025")
	seedTaxCache(t, svc, "STOCKHOLM", "TEST", "2026")

	sal := createGrossSalary(t, svc, "Test Salary", "STOCKHOLM", "TEST")

	if _, err := svc.UpsertSalaryAmount(ctx, service.SalaryAmount{
		SalaryID:  sal.ID,
		Amount:    newFixedValue(40000),
		StartDate: mustParseDate("2025-01-01"),
	}); err != nil {
		t.Fatalf("creating salary amount: %v", err)
	}

	if _, err := svc.UpsertInkomstbasbelopp(ctx, service.Inkomstbasbelopp{
		Amount:        76200,
		Prisbasbelopp: 57300,
		ValidFrom:     mustParseDate("2025-01-01"),
	}); err != nil {
		t.Fatalf("creating inkomstbasbelopp: %v", err)
	}

	// Add full parental leave from June 2025 to Jan 2026
	if _, err := svc.UpsertFullParentalLeave(ctx, service.FullParentalLeave{
		SalaryID:        sal.ID,
		StartDate:       mustParseDate("2025-06-01"),
		EndDate:         mustParseDate("2026-01-01"),
		SjukDaysPerWeek: 5,
	}); err != nil {
		t.Fatalf("creating full parental leave: %v", err)
	}

	netTTs := salaryTTsFromAll(t, svc, sal.ID)
	// Should split: [2025-01-01, 2025-06-01) normal, [2025-06-01, 2026-01-01) FK compensation, [2026-01-01, ...) normal
	if len(netTTs) != 3 {
		t.Fatalf("expected 3 net TTs (before/during/after leave), got %d", len(netTTs))
	}
	if netTTs[0].StartDate != mustParseDate("2025-01-01") {
		t.Errorf("tt[0] StartDate = %v, want 2025-01-01", netTTs[0].StartDate)
	}
	if netTTs[1].StartDate != mustParseDate("2025-06-01") {
		t.Errorf("tt[1] StartDate = %v, want 2025-06-01", netTTs[1].StartDate)
	}
	if netTTs[2].StartDate != mustParseDate("2026-01-01") {
		t.Errorf("tt[2] StartDate = %v, want 2026-01-01", netTTs[2].StartDate)
	}

	// During leave, compensation should be lower than normal net salary
	normalNet := netTTs[0].AmountFixed.Mean()
	leaveComp := netTTs[1].AmountFixed.Mean()
	afterNet := netTTs[2].AmountFixed.Mean()
	if leaveComp >= normalNet {
		t.Errorf("expected leave compensation (%v) < normal net (%v)", leaveComp, normalNet)
	}
	// FK compensation should be positive
	if leaveComp <= 0 {
		t.Errorf("expected positive FK compensation, got %v", leaveComp)
	}
	// After leave, should return to normal net
	if !approxEqual(afterNet, normalNet, 1.0) {
		t.Errorf("expected after-leave net (%v) ~= normal net (%v)", afterNet, normalNet)
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
