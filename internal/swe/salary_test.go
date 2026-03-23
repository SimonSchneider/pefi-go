package swe_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SimonSchneider/pefigo/internal/swe"
)

func grossToNetServers() (taxRateSrv, taxTableSrv *httptest.Server) {
	taxRateSrv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := swe.RowStoreResponse{
			ResultCount: 1,
			Results: []map[string]string{
				{
					"kommun":                   "STOCKHOLM",
					"församling":               "ADOLF FREDRIKS FÖRSAMLING",
					"summa, exkl. kyrkoavgift": "30.67",
					"summa, inkl. kyrkoavgift": "31.85",
					"år":                       "2025",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))

	taxTableSrv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := swe.RowStoreResponse{
			ResultCount: 3,
			Results: []map[string]string{
				{"inkomst fr.o.m.": "0", "inkomst t.o.m.": "24999", "kolumn 1": "0", "tabellnr": "31", "år": "2025", "antal dgr": "30B"},
				{"inkomst fr.o.m.": "25000", "inkomst t.o.m.": "49999", "kolumn 1": "7500", "tabellnr": "31", "år": "2025", "antal dgr": "30B"},
				{"inkomst fr.o.m.": "50000", "inkomst t.o.m.": "99999", "kolumn 1": "15000", "tabellnr": "31", "år": "2025", "antal dgr": "30B"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))

	return taxRateSrv, taxTableSrv
}

func TestCalculateNetSalary(t *testing.T) {
	taxRateSrv, taxTableSrv := grossToNetServers()
	defer taxRateSrv.Close()
	defer taxTableSrv.Close()

	cache := newFakeCache()
	client := swe.NewClient(cache,
		swe.WithTaxRateURL(taxRateSrv.URL),
		swe.WithTaxTableURL(taxTableSrv.URL),
	)

	ctx := context.Background()

	t.Run("basic gross to net", func(t *testing.T) {
		result, err := client.CalculateNetSalary(ctx, swe.GrossSalaryInput{
			GrossMonthly: 40000,
			Kommun:       "STOCKHOLM",
			Forsamling:   "ADOLF FREDRIKS FÖRSAMLING",
			Year:         "2025",
			ChurchMember: false,
			Column:       1,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.GrossMonthly != 40000 {
			t.Errorf("gross = %v, want 40000", result.GrossMonthly)
		}
		if result.Tax != 7500 {
			t.Errorf("tax = %v, want 7500", result.Tax)
		}
		if result.NetMonthly != 32500 {
			t.Errorf("net = %v, want 32500", result.NetMonthly)
		}
	})

	t.Run("higher salary bracket", func(t *testing.T) {
		result, err := client.CalculateNetSalary(ctx, swe.GrossSalaryInput{
			GrossMonthly: 60000,
			Kommun:       "STOCKHOLM",
			Forsamling:   "ADOLF FREDRIKS FÖRSAMLING",
			Year:         "2025",
			ChurchMember: false,
			Column:       1,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Tax != 15000 {
			t.Errorf("tax = %v, want 15000", result.Tax)
		}
		if result.NetMonthly != 45000 {
			t.Errorf("net = %v, want 45000", result.NetMonthly)
		}
	})
}
