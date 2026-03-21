package core

import (
	"testing"

	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/pefigo/internal/uncertain"
)

func mustDate(s string) date.Date {
	d, err := date.ParseDate(s)
	if err != nil {
		panic(err)
	}
	return d
}

func round(d string, valuation, preMoneyShares, investment float64) InvestmentRound {
	return InvestmentRound{
		Date:           mustDate(d),
		Valuation:      valuation,
		PreMoneyShares: preMoneyShares,
		Investment:     investment,
	}
}

func change(d string, deltaShares, totalPrice float64) ShareChange {
	return ShareChange{
		Date:        mustDate(d),
		DeltaShares: deltaShares,
		TotalPrice:  totalPrice,
	}
}

var defaultSSA = StartupShareAccount{
	AccountID:               "test",
	TaxRate:                 0.25,
	ValuationDiscountFactor: 0.5,
}

func TestBuildStartupShareForecastState_ShareChangeAfterRound(t *testing.T) {
	ucfg := uncertain.NewConfig(1, 1)
	startDate := mustDate("2025-01-01")

	rounds := []InvestmentRound{
		round("2023-01-01", 1_000_000, 1_000, 100_000),
	}
	shareChanges := []ShareChange{
		change("2024-01-01", 100, 10_000),
	}

	result := buildStartupShareForecastState(ucfg, rounds, shareChanges, nil, defaultSSA, startDate)

	if len(result.Snapshots) < 2 {
		t.Fatalf("expected at least 2 snapshots (share change + startDate), got %d", len(result.Snapshots))
	}
	if result.Snapshots[0].Date != mustDate("2024-01-01") {
		t.Errorf("first snapshot date = %s, want 2024-01-01", result.Snapshots[0].Date)
	}
	if result.Snapshots[len(result.Snapshots)-1].Date != startDate {
		t.Errorf("last snapshot date = %s, want %s", result.Snapshots[len(result.Snapshots)-1].Date, startDate)
	}
	// Round date (2023-01-01) should be skipped since no shares owned then
	for _, s := range result.Snapshots {
		if s.Date == mustDate("2023-01-01") {
			t.Error("should not have snapshot at round date when no shares were owned")
		}
	}
}

func TestBuildStartupShareForecastState_MultipleShareChanges(t *testing.T) {
	ucfg := uncertain.NewConfig(1, 1)
	startDate := mustDate("2025-01-01")

	rounds := []InvestmentRound{
		round("2022-01-01", 1_000_000, 1_000, 100_000),
	}
	shareChanges := []ShareChange{
		change("2023-01-01", 50, 5_000),
		change("2024-01-01", 50, 5_000),
	}

	result := buildStartupShareForecastState(ucfg, rounds, shareChanges, nil, defaultSSA, startDate)

	// Should have snapshots at both share change dates + startDate
	if len(result.Snapshots) != 3 {
		t.Fatalf("expected 3 snapshots, got %d", len(result.Snapshots))
	}
	if result.Snapshots[0].Date != mustDate("2023-01-01") {
		t.Errorf("snapshot[0] date = %s, want 2023-01-01", result.Snapshots[0].Date)
	}
	if result.Snapshots[1].Date != mustDate("2024-01-01") {
		t.Errorf("snapshot[1] date = %s, want 2024-01-01", result.Snapshots[1].Date)
	}

	// Second snapshot should reflect more shares owned → higher balance
	if result.Snapshots[1].Balance.Mean() <= result.Snapshots[0].Balance.Mean() {
		t.Errorf("balance should increase with more shares: snap[0]=%f, snap[1]=%f",
			result.Snapshots[0].Balance.Mean(), result.Snapshots[1].Balance.Mean())
	}
}

func TestBuildStartupShareForecastState_MultipleRounds(t *testing.T) {
	ucfg := uncertain.NewConfig(1, 1)
	startDate := mustDate("2025-01-01")

	rounds := []InvestmentRound{
		round("2022-01-01", 1_000_000, 1_000, 100_000),
		round("2024-01-01", 5_000_000, 1_100, 500_000),
	}
	shareChanges := []ShareChange{
		change("2022-06-01", 100, 10_000),
	}

	result := buildStartupShareForecastState(ucfg, rounds, shareChanges, nil, defaultSSA, startDate)

	// Snapshots at: share change (2022-06-01), second round (2024-01-01), startDate
	dates := make(map[date.Date]float64)
	for _, s := range result.Snapshots {
		dates[s.Date] = s.Balance.Mean()
	}

	scBal, hasSC := dates[mustDate("2022-06-01")]
	r2Bal, hasR2 := dates[mustDate("2024-01-01")]
	if !hasSC {
		t.Error("missing snapshot at share change date 2022-06-01")
	}
	if !hasR2 {
		t.Error("missing snapshot at second round date 2024-01-01")
	}
	// After the higher-valuation round, the balance should be higher
	if hasSC && hasR2 && r2Bal <= scBal {
		t.Errorf("balance at round 2 (%f) should exceed balance at share change (%f) due to higher valuation", r2Bal, scBal)
	}
}

func TestBuildStartupShareForecastState_LatestRoundUsedForGrowthModel(t *testing.T) {
	ucfg := uncertain.NewConfig(1, 1)
	startDate := mustDate("2025-01-01")

	rounds := []InvestmentRound{
		round("2022-01-01", 1_000_000, 1_000, 100_000),
		round("2024-01-01", 10_000_000, 1_100, 1_000_000),
	}
	shareChanges := []ShareChange{
		change("2022-06-01", 100, 10_000),
	}

	result := buildStartupShareForecastState(ucfg, rounds, shareChanges, nil, defaultSSA, startDate)

	// Growth model should use the latest round's post-money valuation (10M + 1M = 11M)
	gotValuation := result.GrowthModel.Valuation.Mean()
	expectedValuation := 11_000_000.0
	if gotValuation != expectedValuation {
		t.Errorf("growth model valuation = %f, want %f (should use latest round)", gotValuation, expectedValuation)
	}

	// Post-money shares for second round: 1100 + (1_000_000 / (10_000_000/1100)) = 1100 + 110 = 1210
	gotShares := result.GrowthModel.TotalShares.Mean()
	expectedShares := 1210.0
	if gotShares != expectedShares {
		t.Errorf("growth model total shares = %f, want %f (should use latest round)", gotShares, expectedShares)
	}
}

func TestBuildStartupShareForecastState_SingleRoundNoShareChanges(t *testing.T) {
	ucfg := uncertain.NewConfig(1, 1)
	startDate := mustDate("2025-01-01")

	rounds := []InvestmentRound{
		round("2023-01-01", 1_000_000, 1_000, 100_000),
	}

	result := buildStartupShareForecastState(ucfg, rounds, nil, nil, defaultSSA, startDate)

	// Only the startDate snapshot (round date skipped because 0 shares owned)
	if len(result.Snapshots) != 1 {
		t.Fatalf("expected 1 snapshot (startDate only, no shares owned), got %d", len(result.Snapshots))
	}
	if result.Snapshots[0].Date != startDate {
		t.Errorf("snapshot date = %s, want %s", result.Snapshots[0].Date, startDate)
	}
}

func TestBuildStartupShareForecastState_ShareChangeBeforeAnyRound(t *testing.T) {
	ucfg := uncertain.NewConfig(1, 1)
	startDate := mustDate("2025-01-01")

	rounds := []InvestmentRound{
		round("2024-01-01", 1_000_000, 1_000, 100_000),
	}
	shareChanges := []ShareChange{
		change("2023-01-01", 100, 10_000),
	}

	result := buildStartupShareForecastState(ucfg, rounds, shareChanges, nil, defaultSSA, startDate)

	// Share change at 2023-01-01 has no round on or before it → skipped.
	// Round at 2024-01-01 has shares owned → included.
	// Plus startDate.
	dates := make(map[date.Date]bool)
	for _, s := range result.Snapshots {
		dates[s.Date] = true
	}
	if dates[mustDate("2023-01-01")] {
		t.Error("should not have snapshot at share change date before any round")
	}
	if !dates[mustDate("2024-01-01")] {
		t.Error("should have snapshot at round date where shares are owned")
	}
}

func TestBuildStartupShareForecastState_StartDateAlwaysPresent(t *testing.T) {
	ucfg := uncertain.NewConfig(1, 1)
	startDate := mustDate("2025-01-01")

	rounds := []InvestmentRound{
		round("2023-01-01", 1_000_000, 1_000, 100_000),
	}
	shareChanges := []ShareChange{
		change("2023-06-01", 100, 10_000),
	}

	result := buildStartupShareForecastState(ucfg, rounds, shareChanges, nil, defaultSSA, startDate)

	lastSnap := result.Snapshots[len(result.Snapshots)-1]
	if lastSnap.Date != startDate {
		t.Errorf("last snapshot should be at startDate %s, got %s", startDate, lastSnap.Date)
	}
}

func TestBuildStartupShareForecastState_SnapshotsSortedByDate(t *testing.T) {
	ucfg := uncertain.NewConfig(1, 1)
	startDate := mustDate("2025-01-01")

	rounds := []InvestmentRound{
		round("2022-01-01", 1_000_000, 1_000, 100_000),
		round("2024-01-01", 5_000_000, 1_100, 500_000),
	}
	shareChanges := []ShareChange{
		change("2022-06-01", 50, 5_000),
		change("2023-06-01", 50, 5_000),
	}

	result := buildStartupShareForecastState(ucfg, rounds, shareChanges, nil, defaultSSA, startDate)

	for i := 1; i < len(result.Snapshots); i++ {
		if result.Snapshots[i].Date < result.Snapshots[i-1].Date {
			t.Errorf("snapshots not sorted: [%d]=%s > [%d]=%s",
				i-1, result.Snapshots[i-1].Date, i, result.Snapshots[i].Date)
		}
	}
}

func TestBuildStartupShareForecastState_ValuationUsesCorrectRoundPerDate(t *testing.T) {
	ucfg := uncertain.NewConfig(1, 1)
	startDate := mustDate("2025-01-01")

	rounds := []InvestmentRound{
		round("2022-01-01", 1_000_000, 1_000, 100_000),
		round("2024-01-01", 10_000_000, 1_100, 1_000_000),
	}
	shareChanges := []ShareChange{
		change("2022-06-01", 100, 10_000),
	}

	result := buildStartupShareForecastState(ucfg, rounds, shareChanges, nil, defaultSSA, startDate)

	balances := make(map[date.Date]float64)
	for _, s := range result.Snapshots {
		balances[s.Date] = s.Balance.Mean()
	}

	scBal := balances[mustDate("2022-06-01")]
	r2Bal := balances[mustDate("2024-01-01")]

	// Share change date uses round 1 valuation (1.1M post-money), round 2 uses 11M.
	// With same shares, the ratio should reflect the ~10x valuation increase.
	if r2Bal <= scBal*2 {
		t.Errorf("round 2 balance (%f) should be much larger than share change balance (%f) due to 10x valuation", r2Bal, scBal)
	}
}
