package site

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeadersOnEveryRoute(t *testing.T) {
	want := map[string]string{
		"X-Content-Type-Options":       "nosniff",
		"Referrer-Policy":              "strict-origin-when-cross-origin",
		"X-Frame-Options":              "DENY",
		"Cross-Origin-Opener-Policy":   "same-origin",
		"Cross-Origin-Resource-Policy": "same-origin",
	}
	for _, path := range []string{"/", "/experience", "/projects", "/skills", "/contact", "/static/css/style.css"} {
		t.Run(path, func(t *testing.T) {
			h := newTestHandler(t)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))

			for header, value := range want {
				if got := rec.Header().Get(header); got != value {
					t.Errorf("%s = %q, want %q", header, got, value)
				}
			}
			if rec.Header().Get("Content-Security-Policy") == "" {
				t.Error("Content-Security-Policy is missing")
			}
			if rec.Header().Get("Permissions-Policy") == "" {
				t.Error("Permissions-Policy is missing")
			}
		})
	}
}

// The policy has to stay strict. Neither scripts nor styles are ever inline, so
// nothing needs an 'unsafe-' escape hatch.
func TestContentSecurityPolicyDirectives(t *testing.T) {
	h := newTestHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	csp := rec.Header().Get("Content-Security-Policy")

	for _, directive := range []string{
		"default-src 'none'",
		"script-src 'self'",
		"form-action 'self'",
		"base-uri 'none'",
		"frame-ancestors 'none'",
		"object-src 'none'",
	} {
		if !strings.Contains(csp, directive) {
			t.Errorf("CSP is missing %q\ngot: %s", directive, csp)
		}
	}

	// No directive may carry an 'unsafe-' token. Inline styles in a template
	// would force 'unsafe-inline' back into style-src and weaken the policy for
	// every page, so this guards the whole header rather than one directive.
	for part := range strings.SplitSeq(csp, ";") {
		part = strings.TrimSpace(part)
		if strings.Contains(part, "unsafe") {
			t.Errorf("CSP directive has been loosened: %q", part)
		}
	}
}

// HSTS is only safe over HTTPS. Behind a TLS-terminating proxy r.TLS is nil, so
// that deployment opts in explicitly rather than this trusting a header any
// client can set.
func TestHSTSIsOptIn(t *testing.T) {
	t.Run("absent over plain HTTP", func(t *testing.T) {
		h := newTestHandler(t)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

		if got := rec.Header().Get("Strict-Transport-Security"); got != "" {
			t.Errorf("Strict-Transport-Security = %q, want it absent", got)
		}
	})

	t.Run("not set by a forwarded header", func(t *testing.T) {
		h := newTestHandler(t)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Forwarded-Proto", "https")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if got := rec.Header().Get("Strict-Transport-Security"); got != "" {
			t.Errorf("Strict-Transport-Security = %q, want a spoofable header to be ignored", got)
		}
	})

	t.Run("present with FORCE_HSTS", func(t *testing.T) {
		t.Setenv("FORCE_HSTS", "1")
		h := newTestHandler(t) // env is read when the handler is built
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

		if got := rec.Header().Get("Strict-Transport-Security"); !strings.Contains(got, "max-age=") {
			t.Errorf("Strict-Transport-Security = %q, want a max-age", got)
		}
	})
}

func TestCSRFRejectsCrossOriginPosts(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		wantCode int
	}{
		{"cross-site", map[string]string{"Sec-Fetch-Site": "cross-site"}, http.StatusForbidden},
		{"same-site subdomain", map[string]string{"Sec-Fetch-Site": "same-site"}, http.StatusForbidden},
		{"foreign origin", map[string]string{"Origin": "https://evil.example"}, http.StatusForbidden},
		{"same-origin", map[string]string{"Sec-Fetch-Site": "same-origin"}, http.StatusOK},
		{"origin matching host", map[string]string{"Origin": "http://example.com"}, http.StatusOK},
		{"no browser headers", nil, http.StatusOK},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			captureLog(t)
			h := newTestHandler(t)
			contentType, body := urlencodedBody(validFields())

			req := httptest.NewRequest(http.MethodPost, "/contact", body)
			req.Header.Set("Content-Type", contentType)
			req.Header.Set("Accept", "application/json")
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tc.wantCode {
				t.Errorf("status = %d, want %d (body %s)", rec.Code, tc.wantCode, rec.Body.String())
			}
		})
	}
}

// Safe methods must keep working cross-origin — nothing on this site changes
// state on a GET.
func TestCSRFLeavesSafeMethodsAlone(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestContactIsRateLimited(t *testing.T) {
	captureLog(t)
	h := newTestHandler(t)

	// The handler is built with a burst of 5.
	for i := 1; i <= 5; i++ {
		contentType, body := urlencodedBody(validFields())
		rec := postContact(t, h, contentType, body, "application/json")
		if rec.Code != http.StatusOK {
			t.Fatalf("submission %d = %d, want 200", i, rec.Code)
		}
	}

	contentType, body := urlencodedBody(validFields())
	rec := postContact(t, h, contentType, body, "application/json")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("submission past the burst = %d, want 429", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("429 response is missing Retry-After")
	}
}

// Only the contact POST spends from the budget; browsing must never be limited.
func TestPagesAreNotRateLimited(t *testing.T) {
	h := newTestHandler(t)
	for i := range 20 {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("page request %d = %d, want 200", i+1, rec.Code)
		}
	}
	for i := range 20 {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/static/css/style.css", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("asset request %d = %d, want 200", i+1, rec.Code)
		}
	}
}
