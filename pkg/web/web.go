// Package web provides the small network helpers used when managing
// bookmarks: fetching a page's title and opening a URL in the system browser.
package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// ErrTitleUnavailable is returned when a page title cannot be fetched
// automatically (e.g. HTTP 403 or an empty <title> tag). Callers should
// prompt the user to enter a title manually.
var ErrTitleUnavailable = errors.New("web: couldn't fetch title")

// ErrNotFound is returned when the page responds with HTTP 404.
var ErrNotFound = errors.New("web: not found")

// OpenURL opens the given URL in the default browser.
func OpenURL(url string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}

// WebsiteTitle fetches and extracts the page title from a URL. The request is
// bound to ctx and capped at 10 seconds.
func WebsiteTitle(ctx context.Context, url string) (string, error) {
	return WebsiteTitleWithClient(ctx, nil, url)
}

// WebsiteTitleWithClient is WebsiteTitle with a caller-supplied HTTP client,
// letting daemons and library consumers control the transport, timeout, retry
// policy, and cookie jar. A nil client falls back to the default 10-second
// client. The request remains bound to ctx for cancellation.
func WebsiteTitleWithClient(ctx context.Context, client *http.Client, url string) (string, error) {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = res.Body.Close() }()

	switch res.StatusCode {
	case http.StatusOK:
		doc, err := goquery.NewDocumentFromReader(res.Body)
		if err != nil {
			return "", err
		}
		title := strings.Join(strings.Fields(strings.TrimSpace(doc.Find("title").Text())), " ")
		if title == "" {
			return "", ErrTitleUnavailable
		}
		return title, nil
	case http.StatusForbidden:
		return "", ErrTitleUnavailable
	case http.StatusNotFound:
		return "", fmt.Errorf("fetch title: %w: %s", ErrNotFound, url)
	default:
		return "", fmt.Errorf("fetch title: unexpected status %d", res.StatusCode)
	}
}
