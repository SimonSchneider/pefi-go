package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/SimonSchneider/pefigo/internal/pdb"
)

func ExtractDomain(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return ""
	}
	return u.Hostname()
}

func (s *Service) GetCachedFavicon(ctx context.Context, domain string) ([]byte, string, error) {
	row, err := pdb.New(s.db).GetFavicon(ctx, domain)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", fmt.Errorf("favicon not cached for %s: %w", domain, err)
		}
		return nil, "", fmt.Errorf("getting favicon: %w", err)
	}
	return row.IconData, row.ContentType, nil
}

func (s *Service) UpsertFavicon(ctx context.Context, domain string, iconData []byte, contentType string) error {
	return pdb.New(s.db).UpsertFavicon(ctx, pdb.UpsertFaviconParams{
		Domain:      domain,
		IconData:    iconData,
		ContentType: contentType,
		CreatedAt:   time.Now().Unix(),
	})
}

func (s *Service) GetOrFetchFavicon(ctx context.Context, domain string) ([]byte, string, error) {
	data, ct, err := s.GetCachedFavicon(ctx, domain)
	if err == nil {
		return data, ct, nil
	}

	data, ct, err = fetchFavicon(ctx, domain)
	if err != nil {
		return nil, "", fmt.Errorf("fetching favicon for %s: %w", domain, err)
	}

	if upsertErr := s.UpsertFavicon(ctx, domain, data, ct); upsertErr != nil {
		return nil, "", fmt.Errorf("caching favicon for %s: %w", domain, upsertErr)
	}

	return data, ct, nil
}

func fetchFavicon(ctx context.Context, domain string) ([]byte, string, error) {
	faviconURL := fmt.Sprintf("https://www.google.com/s2/favicons?domain=%s&sz=256", url.QueryEscape(domain))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, faviconURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("creating request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("reading response: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/png"
	}

	return body, contentType, nil
}
