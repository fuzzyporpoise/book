package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWebsiteTitle(t *testing.T) {
	tests := []struct {
		name                    string
		status                  int
		body                    string
		wantTitle               string
		wantErr                 bool
		wantErrTitleUnavailable bool
		wantErrNotFound         bool
	}{
		{
			name:      "OK with title",
			status:    http.StatusOK,
			body:      `<html><head><title>  Hello   World  </title></head></html>`,
			wantTitle: "Hello World",
			wantErr:   false,
		},
		{
			name:                    "OK with empty title",
			status:                  http.StatusOK,
			body:                    `<html><head><title>   </title></head></html>`,
			wantTitle:               "",
			wantErr:                 true,
			wantErrTitleUnavailable: true,
		},
		{
			name:                    "OK with missing title tag",
			status:                  http.StatusOK,
			body:                    `<html><head></head><body>hi</body></html>`,
			wantTitle:               "",
			wantErr:                 true,
			wantErrTitleUnavailable: true,
		},
		{
			name:                    "Forbidden",
			status:                  http.StatusForbidden,
			body:                    "",
			wantTitle:               "",
			wantErr:                 true,
			wantErrTitleUnavailable: true,
		},
		{
			name:            "NotFound",
			status:          http.StatusNotFound,
			body:            "",
			wantTitle:       "",
			wantErr:         true,
			wantErrNotFound: true,
		},
		{
			name:      "ServerError",
			status:    http.StatusInternalServerError,
			body:      "",
			wantTitle: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				fmt.Fprint(w, tt.body)
			}))
			defer server.Close()

			gotTitle, gotErr := WebsiteTitle(context.Background(), server.URL)
			if gotTitle != tt.wantTitle {
				t.Errorf("title mismatch: got %q, want %q", gotTitle, tt.wantTitle)
			}
			if tt.wantErr && gotErr == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantErr && gotErr != nil {
				t.Errorf("unexpected error: %v", gotErr)
			}
			if tt.wantErrTitleUnavailable && !errors.Is(gotErr, ErrTitleUnavailable) {
				t.Errorf("expected ErrTitleUnavailable, got %v", gotErr)
			}
			if tt.wantErrNotFound && !errors.Is(gotErr, ErrNotFound) {
				t.Errorf("expected ErrNotFound, got %v", gotErr)
			}
		})
	}
}

type recordingTransport struct {
	base http.RoundTripper
	used bool
}

func (rt *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.used = true
	return rt.base.RoundTrip(req)
}

func TestWebsiteTitleWithClientUsesCallerClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><head><title>Custom Client</title></head></html>`)
	}))
	defer server.Close()

	transport := &recordingTransport{base: http.DefaultTransport}
	client := &http.Client{Transport: transport}

	gotTitle, err := WebsiteTitleWithClient(context.Background(), client, server.URL)
	if err != nil {
		t.Fatalf("WebsiteTitleWithClient: %v", err)
	}
	if gotTitle != "Custom Client" {
		t.Errorf("title = %q, want %q", gotTitle, "Custom Client")
	}
	if !transport.used {
		t.Error("caller-supplied client was not used for the request")
	}
}

func TestWebsiteTitleWithClientHonorsCallerTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		fmt.Fprint(w, `<html><head><title>Slow</title></head></html>`)
	}))
	defer server.Close()

	client := &http.Client{Timeout: 50 * time.Millisecond}
	if _, err := WebsiteTitleWithClient(context.Background(), client, server.URL); err == nil {
		t.Fatal("expected a timeout error from the caller-supplied client, got nil")
	}
}

func TestWebsiteTitleWithClientNilFallsBackToDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><head><title>Default Client</title></head></html>`)
	}))
	defer server.Close()

	gotTitle, err := WebsiteTitleWithClient(context.Background(), nil, server.URL)
	if err != nil {
		t.Fatalf("WebsiteTitleWithClient(nil): %v", err)
	}
	if gotTitle != "Default Client" {
		t.Errorf("title = %q, want %q", gotTitle, "Default Client")
	}
}
