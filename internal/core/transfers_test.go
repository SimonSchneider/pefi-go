package core

import (
	"testing"
)

func TestSimplifyTransfers(t *testing.T) {
	transfers := []Transfer{
		{FromAccountID: "1", ToAccountID: "2", Amount: 100},
		{FromAccountID: "2", ToAccountID: "1", Amount: 200},
		{FromAccountID: "1", ToAccountID: "3", Amount: 300},
		{FromAccountID: "1", ToAccountID: "2", Amount: 400},
		{FromAccountID: "1", ToAccountID: "1", Amount: 500},
		{FromAccountID: "1", ToAccountID: "4", Amount: 600},
		{FromAccountID: "4", ToAccountID: "1", Amount: 700},
	}
	simplified := SimplifyTransfers(transfers)

	t.Logf("simplified: %+v", simplified)
	if len(simplified) != 3 {
		t.Fatalf("expected 2 transfers, got %d", len(simplified))
	}
	if simplified[0].FromAccountID != "1" {
		t.Errorf("expected from account ID 1, got %s", simplified[0].FromAccountID)
	}
	if simplified[0].ToAccountID != "2" {
		t.Errorf("expected to account ID 2, got %s", simplified[0].ToAccountID)
	}
	if simplified[0].Amount != 300 {
		t.Errorf("expected amount 300, got %f", simplified[0].Amount)
	}
	if simplified[1].FromAccountID != "1" {
		t.Errorf("expected from account ID 1, got %s", simplified[1].FromAccountID)
	}
	if simplified[1].ToAccountID != "3" {
		t.Errorf("expected to account ID 3, got %s", simplified[1].ToAccountID)
	}
	if simplified[1].Amount != 300 {
		t.Errorf("expected amount 300, got %f", simplified[1].Amount)
	}
	if simplified[2].FromAccountID != "4" {
		t.Errorf("expected from account ID 4, got %s", simplified[2].FromAccountID)
	}
	if simplified[2].ToAccountID != "1" {
		t.Errorf("expected to account ID 1, got %s", simplified[2].ToAccountID)
	}
	if simplified[2].Amount != 100 {
		t.Errorf("expected amount 100, got %f", simplified[2].Amount)
	}
}
