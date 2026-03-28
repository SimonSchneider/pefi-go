package swe

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

const (
	defaultTaxRateURL  = "https://skatteverket.entryscape.net/rowstore/dataset/c67b320b-ffee-4876-b073-dd9236cd2a99"
	defaultTaxTableURL = "https://skatteverket.entryscape.net/rowstore/dataset/88320397-5c32-4c16-ae79-d36d95b17b95"
)

type RowStoreResponse struct {
	ResultCount int                 `json:"resultCount"`
	Results     []map[string]string `json:"results"`
	Next        string              `json:"next,omitempty"`
}

type Client struct {
	cache       Cache
	httpClient  *http.Client
	taxRateURL  string
	taxTableURL string
}

type ClientOption func(*Client)

func WithTaxRateURL(u string) ClientOption {
	return func(c *Client) { c.taxRateURL = u }
}

func WithTaxTableURL(u string) ClientOption {
	return func(c *Client) { c.taxTableURL = u }
}

func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) { c.httpClient = hc }
}

func NewClient(cache Cache, opts ...ClientOption) *Client {
	c := &Client{
		cache:       cache,
		httpClient:  http.DefaultClient,
		taxRateURL:  defaultTaxRateURL,
		taxTableURL: defaultTaxTableURL,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

func (c *Client) fetchAllPages(ctx context.Context, baseURL string, params url.Values) ([]map[string]string, error) {
	var allResults []map[string]string
	params.Set("_limit", "500")
	offset := 0

	for {
		params.Set("_offset", strconv.Itoa(offset))
		reqURL := baseURL + "?" + params.Encode()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetching %s: %w", reqURL, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, reqURL)
		}

		var result RowStoreResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("decoding response: %w", err)
		}

		allResults = append(allResults, result.Results...)

		if len(allResults) >= result.ResultCount || result.Next == "" {
			break
		}
		offset += 500
	}

	return allResults, nil
}

func (c *Client) fetchCached(ctx context.Context, cacheKey, baseURL string, params url.Values) ([]map[string]string, error) {
	if raw, ok, err := c.cache.Get(ctx, cacheKey); err != nil {
		return nil, fmt.Errorf("cache get: %w", err)
	} else if ok {
		var results []map[string]string
		if err := json.Unmarshal([]byte(raw), &results); err != nil {
			return nil, fmt.Errorf("decoding cached data: %w", err)
		}
		return results, nil
	}

	results, err := c.fetchAllPages(ctx, baseURL, params)
	if err != nil {
		return nil, err
	}

	raw, err := json.Marshal(results)
	if err != nil {
		return nil, fmt.Errorf("encoding for cache: %w", err)
	}
	if err := c.cache.Set(ctx, cacheKey, string(raw)); err != nil {
		return nil, fmt.Errorf("cache set: %w", err)
	}

	return results, nil
}

// GetTaxTableNumber resolves the tax table number for a kommun/forsamling/year.
// It fetches from the Skatteverket tax rate API and rounds the total rate to the nearest integer.
func (c *Client) GetTaxTableNumber(ctx context.Context, kommun, forsamling, year string, churchMember bool) (int, error) {
	cacheKey := fmt.Sprintf("tax_rate:%s:%s:%s", kommun, forsamling, year)
	params := url.Values{}
	params.Set("kommun", kommun)
	params.Set("församling", forsamling)
	params.Set("år", year)

	results, err := c.fetchCached(ctx, cacheKey, c.taxRateURL, params)
	if err != nil {
		return 0, fmt.Errorf("fetching tax rates: %w", err)
	}
	if len(results) == 0 {
		return 0, fmt.Errorf("no tax rate found for %s/%s/%s", kommun, forsamling, year)
	}

	row := results[0]
	field := "summa, exkl. kyrkoavgift"
	if churchMember {
		field = "summa, inkl. kyrkoavgift"
	}

	rateStr, ok := row[field]
	if !ok {
		return 0, fmt.Errorf("field %q not found in response", field)
	}

	rate, err := strconv.ParseFloat(strings.ReplaceAll(rateStr, ",", "."), 64)
	if err != nil {
		return 0, fmt.Errorf("parsing rate %q: %w", rateStr, err)
	}

	return int(math.Round(rate)), nil
}

type taxBracket struct {
	From float64
	To   float64
	Tax  float64
}

// NewTaxLookup fetches and parses the tax table once, returning a fast lookup
// function that does a binary search over sorted brackets.
func (c *Client) NewTaxLookup(ctx context.Context, tableNumber int, year string, column int) (func(grossMonthly float64) (float64, error), error) {
	tabellnr := strconv.Itoa(tableNumber)
	cacheKey := fmt.Sprintf("tax_table:%s:%s", tabellnr, year)
	params := url.Values{}
	params.Set("tabellnr", tabellnr)
	params.Set("år", year)
	params.Set("antal dgr", "30B")

	results, err := c.fetchCached(ctx, cacheKey, c.taxTableURL, params)
	if err != nil {
		return nil, fmt.Errorf("fetching tax table: %w", err)
	}

	colKey := fmt.Sprintf("kolumn %d", column)
	brackets := make([]taxBracket, 0, len(results))
	for _, row := range results {
		from, err := strconv.ParseFloat(row["inkomst fr.o.m."], 64)
		if err != nil {
			continue
		}
		to, err := strconv.ParseFloat(row["inkomst t.o.m."], 64)
		if err != nil {
			continue
		}
		taxStr, ok := row[colKey]
		if !ok || taxStr == "" {
			continue
		}
		tax, err := strconv.ParseFloat(taxStr, 64)
		if err != nil {
			continue
		}
		brackets = append(brackets, taxBracket{From: from, To: to, Tax: tax})
	}

	sort.Slice(brackets, func(i, j int) bool { return brackets[i].From < brackets[j].From })

	return func(grossMonthly float64) (float64, error) {
		i := sort.Search(len(brackets), func(i int) bool { return brackets[i].To >= grossMonthly })
		if i < len(brackets) && grossMonthly >= brackets[i].From && grossMonthly <= brackets[i].To {
			return brackets[i].Tax, nil
		}
		return 0, fmt.Errorf("no matching bracket for income %.0f in table %d/%s", grossMonthly, tableNumber, year)
	}, nil
}

// LookupTax finds the tax amount for a given table number, year, gross monthly income, and column.
func (c *Client) LookupTax(ctx context.Context, tableNumber int, year string, grossMonthly float64, column int) (float64, error) {
	lookup, err := c.NewTaxLookup(ctx, tableNumber, year, column)
	if err != nil {
		return 0, err
	}
	return lookup(grossMonthly)
}

// ListKommuner returns all distinct kommun names for a given year, sorted alphabetically.
func (c *Client) ListKommuner(ctx context.Context, year string) ([]string, error) {
	cacheKey := fmt.Sprintf("kommuner:%s", year)
	params := url.Values{}
	params.Set("år", year)

	results, err := c.fetchCached(ctx, cacheKey, c.taxRateURL, params)
	if err != nil {
		return nil, fmt.Errorf("fetching kommuner: %w", err)
	}

	seen := make(map[string]struct{})
	var kommuner []string
	for _, row := range results {
		k := row["kommun"]
		if k == "" {
			continue
		}
		if _, ok := seen[k]; !ok {
			seen[k] = struct{}{}
			kommuner = append(kommuner, k)
		}
	}
	sort.Strings(kommuner)
	return kommuner, nil
}

// ListForsamlingar returns all distinct forsamling names for a given kommun and year, sorted alphabetically.
func (c *Client) ListForsamlingar(ctx context.Context, kommun, year string) ([]string, error) {
	cacheKey := fmt.Sprintf("forsamlingar:%s:%s", kommun, year)
	params := url.Values{}
	params.Set("kommun", kommun)
	params.Set("år", year)

	results, err := c.fetchCached(ctx, cacheKey, c.taxRateURL, params)
	if err != nil {
		return nil, fmt.Errorf("fetching forsamlingar: %w", err)
	}

	seen := make(map[string]struct{})
	var forsamlingar []string
	for _, row := range results {
		f := row["församling"]
		if f == "" {
			continue
		}
		if _, ok := seen[f]; !ok {
			seen[f] = struct{}{}
			forsamlingar = append(forsamlingar, f)
		}
	}
	sort.Strings(forsamlingar)
	return forsamlingar, nil
}
