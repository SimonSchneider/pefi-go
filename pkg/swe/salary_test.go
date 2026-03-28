package swe_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SimonSchneider/pefigo/pkg/swe"
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

func yearAwareServers() (taxRateSrv, taxTableSrv *httptest.Server) {
	taxRates := []map[string]string{
		{
			"kommun":                   "STOCKHOLM",
			"församling":               "ADOLF FREDRIKS FÖRSAMLING",
			"summa, exkl. kyrkoavgift": "30.67",
			"summa, inkl. kyrkoavgift": "31.85",
			"år":                       "2025",
		},
		{
			"kommun":                   "STOCKHOLM",
			"församling":               "ADOLF FREDRIKS FÖRSAMLING",
			"summa, exkl. kyrkoavgift": "31.00",
			"summa, inkl. kyrkoavgift": "32.00",
			"år":                       "2026",
		},
	}

	taxTables := []map[string]string{
		{"inkomst fr.o.m.": "0", "inkomst t.o.m.": "24999", "kolumn 1": "0", "tabellnr": "31", "år": "2025", "antal dgr": "30B"},
		{"inkomst fr.o.m.": "25000", "inkomst t.o.m.": "49999", "kolumn 1": "7500", "tabellnr": "31", "år": "2025", "antal dgr": "30B"},
		{"inkomst fr.o.m.": "50000", "inkomst t.o.m.": "99999", "kolumn 1": "15000", "tabellnr": "31", "år": "2025", "antal dgr": "30B"},
		{"inkomst fr.o.m.": "0", "inkomst t.o.m.": "24999", "kolumn 1": "0", "tabellnr": "31", "år": "2026", "antal dgr": "30B"},
		{"inkomst fr.o.m.": "25000", "inkomst t.o.m.": "49999", "kolumn 1": "8000", "tabellnr": "31", "år": "2026", "antal dgr": "30B"},
		{"inkomst fr.o.m.": "50000", "inkomst t.o.m.": "99999", "kolumn 1": "16000", "tabellnr": "31", "år": "2026", "antal dgr": "30B"},
	}

	filterByYear := func(rows []map[string]string, year string) []map[string]string {
		var out []map[string]string
		for _, r := range rows {
			if r["år"] == year {
				out = append(out, r)
			}
		}
		return out
	}

	taxRateSrv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		year := r.URL.Query().Get("år")
		results := filterByYear(taxRates, year)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(swe.RowStoreResponse{ResultCount: len(results), Results: results})
	}))

	taxTableSrv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		year := r.URL.Query().Get("år")
		results := filterByYear(taxTables, year)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(swe.RowStoreResponse{ResultCount: len(results), Results: results})
	}))

	return taxRateSrv, taxTableSrv
}

func TestCalculateNetSalary(t *testing.T) {
	taxRateSrv, taxTableSrv := yearAwareServers()
	defer taxRateSrv.Close()
	defer taxTableSrv.Close()

	cache := newFakeCache()
	client := swe.NewClient(cache,
		swe.WithTaxRateURL(taxRateSrv.URL),
		swe.WithTaxTableURL(taxTableSrv.URL),
	)

	ctx := context.Background()

	t.Run("basic gross to net", func(t *testing.T) {
		result, err := client.CalculateNetSalary(ctx, swe.GrossSalaryInputWithAmount{
			GrossSalaryInput: swe.GrossSalaryInput{
				Kommun:       "STOCKHOLM",
				Forsamling:   "ADOLF FREDRIKS FÖRSAMLING",
				Year:         "2025",
				ChurchMember: false,
				Column:       1,
			},
			GrossMonthly: 40000,
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
		result, err := client.CalculateNetSalary(ctx, swe.GrossSalaryInputWithAmount{
			GrossSalaryInput: swe.GrossSalaryInput{
				Kommun:       "STOCKHOLM",
				Forsamling:   "ADOLF FREDRIKS FÖRSAMLING",
				Year:         "2025",
				ChurchMember: false,
				Column:       1,
			},
			GrossMonthly: 60000,
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

	t.Run("clamps future year to current year", func(t *testing.T) {
		// 2027 > current year (2026), clamps to 2026 which has data (tax=8000 for 40k bracket)
		result, err := client.NetSalaryCalculator(ctx, swe.GrossSalaryInput{
			Kommun:       "STOCKHOLM",
			Forsamling:   "ADOLF FREDRIKS FÖRSAMLING",
			Year:         "2027",
			ChurchMember: false,
			Column:       1,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		res, err := result(40000)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Tax != 8000 {
			t.Errorf("tax = %v, want 8000 (from 2026 via clamping)", res.Tax)
		}
	})

	t.Run("errors when no previous year data exists", func(t *testing.T) {
		// 2020 has no data, and no previous years have data either
		_, err := client.NetSalaryCalculator(ctx, swe.GrossSalaryInput{
			Kommun:       "STOCKHOLM",
			Forsamling:   "ADOLF FREDRIKS FÖRSAMLING",
			Year:         "2020",
			ChurchMember: false,
			Column:       1,
		})
		if err == nil {
			t.Fatal("expected error for year with no data and no fallback, got nil")
		}
	})
}

func TestNetSalaryCalculatorFallback(t *testing.T) {
	ctx := context.Background()

	t.Run("falls back to previous year when current year has no data", func(t *testing.T) {
		filterByYear := func(rows []map[string]string, year string) []map[string]string {
			var out []map[string]string
			for _, r := range rows {
				if r["år"] == year {
					out = append(out, r)
				}
			}
			return out
		}

		only2025Rates := []map[string]string{
			{
				"kommun":                   "STOCKHOLM",
				"församling":               "ADOLF FREDRIKS FÖRSAMLING",
				"summa, exkl. kyrkoavgift": "30.67",
				"summa, inkl. kyrkoavgift": "31.85",
				"år":                       "2025",
			},
		}
		only2025Tables := []map[string]string{
			{"inkomst fr.o.m.": "0", "inkomst t.o.m.": "24999", "kolumn 1": "0", "tabellnr": "31", "år": "2025", "antal dgr": "30B"},
			{"inkomst fr.o.m.": "25000", "inkomst t.o.m.": "49999", "kolumn 1": "7500", "tabellnr": "31", "år": "2025", "antal dgr": "30B"},
			{"inkomst fr.o.m.": "50000", "inkomst t.o.m.": "99999", "kolumn 1": "15000", "tabellnr": "31", "år": "2025", "antal dgr": "30B"},
		}

		rateSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			year := r.URL.Query().Get("år")
			results := filterByYear(only2025Rates, year)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(swe.RowStoreResponse{ResultCount: len(results), Results: results})
		}))
		defer rateSrv.Close()

		tableSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			year := r.URL.Query().Get("år")
			results := filterByYear(only2025Tables, year)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(swe.RowStoreResponse{ResultCount: len(results), Results: results})
		}))
		defer tableSrv.Close()

		fbClient := swe.NewClient(newFakeCache(),
			swe.WithTaxRateURL(rateSrv.URL),
			swe.WithTaxTableURL(tableSrv.URL),
		)

		// Request 2026 (current year); only 2025 exists, should fall back to 2025
		result, err := fbClient.NetSalaryCalculator(ctx, swe.GrossSalaryInput{
			Kommun:       "STOCKHOLM",
			Forsamling:   "ADOLF FREDRIKS FÖRSAMLING",
			Year:         "2026",
			ChurchMember: false,
			Column:       1,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		res, err := result(40000)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Tax != 7500 {
			t.Errorf("tax = %v, want 7500 (from 2025 fallback)", res.Tax)
		}
	})
}
