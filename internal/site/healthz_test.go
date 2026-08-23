package site

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The platform restarts a machine whose health check fails, so this endpoint
// existing and staying cheap is a deployment concern, not a cosmetic one.
func TestHealthz(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestHandler(t).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /healthz = %d, want 200", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != "ok" {
		t.Errorf("body = %q, want %q", got, "ok")
	}
	// Never cached: a cached probe answer would keep reporting healthy after
	// the process stopped being so.
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}
}

// The probe must not depend on the email providers. Wiring it to them would
// take the whole site down whenever Resend had a bad minute, and the site
// serves every page without touching them.
func TestHealthzIgnoresMailConfiguration(t *testing.T) {
	rec := httptest.NewRecorder()
	NewHandler(2026, nil).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("GET /healthz with no mail sender = %d, want 200", rec.Code)
	}
}

// It is a probe, not a page: it must not appear in the sitemap.
func TestHealthzIsNotInTheSitemap(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestHandler(t).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil))

	if strings.Contains(rec.Body.String(), "healthz") {
		t.Error("sitemap lists the health endpoint")
	}
}
