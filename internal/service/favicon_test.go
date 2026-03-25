package service_test

import (
	"bytes"
	"testing"

	"github.com/SimonSchneider/pefigo/internal/service"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{name: "https with path", url: "https://netflix.com/account", want: "netflix.com"},
		{name: "http", url: "http://example.com", want: "example.com"},
		{name: "https with port", url: "https://example.com:8080/path", want: "example.com"},
		{name: "subdomain", url: "https://www.spotify.com/premium", want: "www.spotify.com"},
		{name: "empty string", url: "", want: ""},
		{name: "invalid url", url: "not-a-url", want: ""},
		{name: "just domain no scheme", url: "netflix.com", want: ""},
		{name: "https with query", url: "https://hbo.com/shows?id=123", want: "hbo.com"},
		{name: "trailing slash", url: "https://disney.com/", want: "disney.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.ExtractDomain(tt.url)
			if got != tt.want {
				t.Errorf("ExtractDomain(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestExtractCompanyName(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{name: "https with path", url: "https://netflix.com/account", want: "Netflix"},
		{name: "app subdomain", url: "https://app.spotify.com/premium", want: "Spotify"},
		{name: "www+app subdomain", url: "https://www.app.spotify.com", want: "Spotify"},
		{name: "www subdomain", url: "https://www.hbo.com/shows", want: "Hbo"},
		{name: "bare domain", url: "https://hbo.com/shows", want: "Hbo"},
		{name: "empty string", url: "", want: ""},
		{name: "no scheme", url: "netflix.com", want: ""},
		{name: "single segment domain", url: "https://localhost/path", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.ExtractCompanyName(tt.url)
			if got != tt.want {
				t.Errorf("ExtractCompanyName(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestFaviconCacheRoundtrip(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()

	iconData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A} // PNG header bytes
	contentType := "image/png"

	if err := svc.UpsertFavicon(ctx, "netflix.com", iconData, contentType); err != nil {
		t.Fatalf("UpsertFavicon: %v", err)
	}

	gotData, gotCT, err := svc.GetCachedFavicon(ctx, "netflix.com")
	if err != nil {
		t.Fatalf("GetCachedFavicon: %v", err)
	}
	if !bytes.Equal(gotData, iconData) {
		t.Errorf("icon data mismatch: got %v, want %v", gotData, iconData)
	}
	if gotCT != contentType {
		t.Errorf("content type mismatch: got %q, want %q", gotCT, contentType)
	}

	_, _, err = svc.GetCachedFavicon(ctx, "unknown.com")
	if err == nil {
		t.Fatal("expected error for missing domain, got nil")
	}
}

func TestFaviconCacheDeduplication(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()

	iconData := []byte{0x89, 0x50, 0x4E, 0x47}

	if err := svc.UpsertFavicon(ctx, "netflix.com", iconData, "image/png"); err != nil {
		t.Fatalf("first UpsertFavicon: %v", err)
	}

	updatedData := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	if err := svc.UpsertFavicon(ctx, "netflix.com", updatedData, "image/jpeg"); err != nil {
		t.Fatalf("second UpsertFavicon: %v", err)
	}

	gotData, gotCT, err := svc.GetCachedFavicon(ctx, "netflix.com")
	if err != nil {
		t.Fatalf("GetCachedFavicon: %v", err)
	}
	if !bytes.Equal(gotData, updatedData) {
		t.Errorf("expected updated data, got %v", gotData)
	}
	if gotCT != "image/jpeg" {
		t.Errorf("expected image/jpeg, got %q", gotCT)
	}
}

func TestGetOrFetchFavicon_CacheHit(t *testing.T) {
	svc := newTestService(t)
	ctx := t.Context()

	iconData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	if err := svc.UpsertFavicon(ctx, "cached.com", iconData, "image/png"); err != nil {
		t.Fatalf("seeding cache: %v", err)
	}

	gotData, gotCT, err := svc.GetOrFetchFavicon(ctx, "cached.com")
	if err != nil {
		t.Fatalf("GetOrFetchFavicon: %v", err)
	}
	if !bytes.Equal(gotData, iconData) {
		t.Errorf("expected cached data, got %v", gotData)
	}
	if gotCT != "image/png" {
		t.Errorf("expected image/png, got %q", gotCT)
	}
}
