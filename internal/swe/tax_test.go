package swe_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SimonSchneider/pefigo/internal/swe"
)

type fakeCache struct {
	data map[string]string
}

func newFakeCache() *fakeCache {
	return &fakeCache{data: make(map[string]string)}
}

func (c *fakeCache) Get(_ context.Context, key string) (string, bool, error) {
	v, ok := c.data[key]
	return v, ok, nil
}

func (c *fakeCache) Set(_ context.Context, key string, value string) error {
	c.data[key] = value
	return nil
}

func taxRateServer() *httptest.Server {
	allRows := []map[string]string{
		{
			"kommun":                   "STOCKHOLM",
			"församling":               "ADOLF FREDRIKS FÖRSAMLING",
			"summa, exkl. kyrkoavgift": "30.67",
			"summa, inkl. kyrkoavgift": "31.85",
			"år":                       "2025",
		},
		{
			"kommun":                   "STOCKHOLM",
			"församling":               "KATARINA FÖRSAMLING",
			"summa, exkl. kyrkoavgift": "30.67",
			"summa, inkl. kyrkoavgift": "31.72",
			"år":                       "2025",
		},
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		filtered := allRows
		if år := q.Get("år"); år != "" {
			var out []map[string]string
			for _, row := range filtered {
				if row["år"] == år {
					out = append(out, row)
				}
			}
			filtered = out
		}
		if kommun := q.Get("kommun"); kommun != "" {
			var out []map[string]string
			for _, row := range filtered {
				if row["kommun"] == kommun {
					out = append(out, row)
				}
			}
			filtered = out
		}
		if forsamling := q.Get("församling"); forsamling != "" {
			var out []map[string]string
			for _, row := range filtered {
				if row["församling"] == forsamling {
					out = append(out, row)
				}
			}
			filtered = out
		}

		resp := swe.RowStoreResponse{
			ResultCount: len(filtered),
			Results:     filtered,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
}

func skattetabellServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		tabellnr := q.Get("tabellnr")
		år := q.Get("år")

		resp := swe.RowStoreResponse{
			ResultCount: 0,
			Results:     []map[string]string{},
		}

		if tabellnr == "31" && år == "2025" {
			resp.ResultCount = 3
			resp.Results = []map[string]string{
				{"inkomst fr.o.m.": "0", "inkomst t.o.m.": "24999", "kolumn 1": "0", "tabellnr": "31", "år": "2025", "antal dgr": "30B"},
				{"inkomst fr.o.m.": "25000", "inkomst t.o.m.": "49999", "kolumn 1": "7500", "tabellnr": "31", "år": "2025", "antal dgr": "30B"},
				{"inkomst fr.o.m.": "50000", "inkomst t.o.m.": "99999", "kolumn 1": "15000", "tabellnr": "31", "år": "2025", "antal dgr": "30B"},
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
}

func TestGetTaxTableNumber(t *testing.T) {
	srv := taxRateServer()
	defer srv.Close()

	cache := newFakeCache()
	client := swe.NewClient(cache, swe.WithTaxRateURL(srv.URL))

	ctx := context.Background()

	t.Run("excl church", func(t *testing.T) {
		got, err := client.GetTaxTableNumber(ctx, "STOCKHOLM", "ADOLF FREDRIKS FÖRSAMLING", "2025", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != 31 {
			t.Errorf("got table %d, want 31", got)
		}
	})

	t.Run("incl church", func(t *testing.T) {
		got, err := client.GetTaxTableNumber(ctx, "STOCKHOLM", "ADOLF FREDRIKS FÖRSAMLING", "2025", true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != 32 {
			t.Errorf("got table %d, want 32", got)
		}
	})
}

func TestLookupTax(t *testing.T) {
	srv := skattetabellServer()
	defer srv.Close()

	cache := newFakeCache()
	client := swe.NewClient(cache, swe.WithTaxTableURL(srv.URL))

	ctx := context.Background()

	t.Run("bracket 25000-49999", func(t *testing.T) {
		got, err := client.LookupTax(ctx, 31, "2025", 40000, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != 7500 {
			t.Errorf("got tax %v, want 7500", got)
		}
	})

	t.Run("bracket 50000-99999", func(t *testing.T) {
		got, err := client.LookupTax(ctx, 31, "2025", 60000, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != 15000 {
			t.Errorf("got tax %v, want 15000", got)
		}
	})

	t.Run("no matching bracket", func(t *testing.T) {
		_, err := client.LookupTax(ctx, 31, "2025", 200000, 1)
		if err == nil {
			t.Fatal("expected error for income outside brackets")
		}
	})
}

func TestListKommuner(t *testing.T) {
	srv := taxRateServer()
	defer srv.Close()

	cache := newFakeCache()
	client := swe.NewClient(cache, swe.WithTaxRateURL(srv.URL))

	ctx := context.Background()

	kommuner, err := client.ListKommuner(ctx, "2025")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(kommuner) != 1 {
		t.Fatalf("expected 1 kommun, got %d", len(kommuner))
	}
	if kommuner[0] != "STOCKHOLM" {
		t.Errorf("expected STOCKHOLM, got %s", kommuner[0])
	}
}

func TestListForsamlingar(t *testing.T) {
	srv := taxRateServer()
	defer srv.Close()

	cache := newFakeCache()
	client := swe.NewClient(cache, swe.WithTaxRateURL(srv.URL))

	ctx := context.Background()

	forsamlingar, err := client.ListForsamlingar(ctx, "STOCKHOLM", "2025")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(forsamlingar) != 2 {
		t.Fatalf("expected 2 forsamlingar, got %d", len(forsamlingar))
	}
	if forsamlingar[0] != "ADOLF FREDRIKS FÖRSAMLING" {
		t.Errorf("expected ADOLF FREDRIKS FÖRSAMLING first, got %s", forsamlingar[0])
	}
	if forsamlingar[1] != "KATARINA FÖRSAMLING" {
		t.Errorf("expected KATARINA FÖRSAMLING second, got %s", forsamlingar[1])
	}
}

func TestCachingBehavior(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := swe.RowStoreResponse{
			ResultCount: 1,
			Results: []map[string]string{
				{
					"kommun":                    "STOCKHOLM",
					"församling":                "TEST",
					"summa, exkl. kyrkoavgift":  "30.67",
					"summa, inkl. kyrkoavgift":  "31.85",
					"år":                        "2025",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	cache := newFakeCache()
	client := swe.NewClient(cache, swe.WithTaxRateURL(srv.URL))
	ctx := context.Background()

	client.GetTaxTableNumber(ctx, "STOCKHOLM", "TEST", "2025", false)
	client.GetTaxTableNumber(ctx, "STOCKHOLM", "TEST", "2025", false)

	if callCount != 1 {
		t.Errorf("expected 1 API call (second should hit cache), got %d", callCount)
	}
}
