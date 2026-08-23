package site

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

// get is a small helper for the many plain GETs these tests make.
func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

var assetRefRE = regexp.MustCompile(`(?:href|src)="(/static/[^"]+)"`)

// Every asset the layout references must carry a ?v= fingerprint, otherwise it
// can only ever be served with revalidation instead of a long-lived cache.
func TestLayoutAssetsAreFingerprinted(t *testing.T) {
	h := newTestHandler(t)
	body := get(t, h, "/").Body.String()

	refs := assetRefRE.FindAllStringSubmatch(body, -1)
	if len(refs) == 0 {
		t.Fatal("no /static/ references found in the rendered page")
	}
	for _, m := range refs {
		ref := m[1]
		if !strings.Contains(ref, "?v=") {
			t.Errorf("asset reference %q has no ?v= fingerprint", ref)
		}
		if code := get(t, h, ref).Code; code != http.StatusOK {
			t.Errorf("asset reference %q serves %d, want 200", ref, code)
		}
	}
}

func TestStaticCacheHeaders(t *testing.T) {
	h := newTestHandler(t)

	// Discover the current fingerprint rather than hardcoding a hash.
	body := get(t, h, "/").Body.String()
	m := regexp.MustCompile(`href="(/static/css/style\.css\?v=[0-9a-f]+)"`).FindStringSubmatch(body)
	if m == nil {
		t.Fatal("stylesheet reference not found in the rendered page")
	}
	versioned := m[1]

	tests := []struct {
		name string
		path string
		want string
	}{
		{"versioned URL is immutable", versioned, "public, max-age=31536000, immutable"},
		{"unversioned URL revalidates", "/static/css/style.css", "public, no-cache"},
		{"stale version revalidates", "/static/css/style.css?v=deadbeef", "public, no-cache"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := get(t, h, tc.path)
			if got := rec.Header().Get("Cache-Control"); got != tc.want {
				t.Errorf("Cache-Control = %q, want %q", got, tc.want)
			}
			if rec.Header().Get("ETag") == "" {
				t.Error("ETag is missing")
			}
		})
	}
}

// Font filenames already carry a content hash, so they are immutable whether or
// not the request adds one — the stylesheet references them without a query.
func TestFontsAreImmutable(t *testing.T) {
	h := newTestHandler(t)
	css := get(t, h, "/static/css/style.css").Body.String()

	refs := regexp.MustCompile(`url\('(/static/fonts/[^']+)'\)`).FindAllStringSubmatch(css, -1)
	if len(refs) == 0 {
		t.Fatal("no font references found in the stylesheet")
	}
	for _, m := range refs {
		rec := get(t, h, m[1])
		if rec.Code != http.StatusOK {
			t.Errorf("%s serves %d, want 200", m[1], rec.Code)
		}
		if got := rec.Header().Get("Cache-Control"); !strings.Contains(got, "immutable") {
			t.Errorf("%s Cache-Control = %q, want immutable", m[1], got)
		}
		if got := rec.Header().Get("Content-Type"); got != "font/woff2" {
			t.Errorf("%s Content-Type = %q, want font/woff2", m[1], got)
		}
	}
}

// embed.FS files have a zero ModTime, so ETag is the only validator available.
// Without it every asset is refetched in full on every navigation.
func TestStaticRevalidationReturns304(t *testing.T) {
	h := newTestHandler(t)
	first := get(t, h, "/static/js/main.js")
	etag := first.Header().Get("ETag")
	if etag == "" {
		t.Fatal("no ETag on the first response")
	}

	req := httptest.NewRequest(http.MethodGet, "/static/js/main.js", nil)
	req.Header.Set("If-None-Match", etag)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotModified {
		t.Errorf("status = %d, want 304", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("304 returned %d bytes, want an empty body", rec.Body.Len())
	}
}

// Content types are declared explicitly because mime.TypeByExtension reads the
// Windows registry, where .js and .css are routinely wrong — and a mislabelled
// module is refused outright under X-Content-Type-Options: nosniff.
func TestStaticContentTypes(t *testing.T) {
	h := newTestHandler(t)
	tests := map[string]string{
		"/static/css/style.css":  "text/css; charset=utf-8",
		"/static/js/main.js":     "text/javascript; charset=utf-8",
		"/static/favicon.svg":    "image/svg+xml",
		"/static/favicon-32.png": "image/png",
		"/static/favicon.ico":    "image/x-icon",
	}
	for path, want := range tests {
		t.Run(path, func(t *testing.T) {
			if got := get(t, h, path).Header().Get("Content-Type"); got != want {
				t.Errorf("Content-Type = %q, want %q", got, want)
			}
		})
	}
}

func TestUnknownStaticAsset404s(t *testing.T) {
	h := newTestHandler(t)
	if code := get(t, h, "/static/nope.css").Code; code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", code)
	}
}

// Browsers probe /favicon.ico regardless of the link tags.
func TestFaviconRoutes(t *testing.T) {
	h := newTestHandler(t)
	for _, path := range []string{"/favicon.ico", "/static/favicon.svg", "/static/apple-touch-icon.png"} {
		t.Run(path, func(t *testing.T) {
			rec := get(t, h, path)
			if rec.Code != http.StatusOK {
				t.Errorf("status = %d, want 200", rec.Code)
			}
			if rec.Body.Len() == 0 {
				t.Error("empty body")
			}
		})
	}
}

// The whole point of self-hosting the fonts is that no visitor request reaches
// a third party. Guard it: a reintroduced CDN link would also need the CSP
// loosened, so this failing is the early warning.
func TestNoThirdPartyReferences(t *testing.T) {
	h := newTestHandler(t)
	banned := []string{"fonts.googleapis.com", "fonts.gstatic.com", "//cdn.", "unpkg.com", "jsdelivr"}

	for _, path := range []string{"/", "/experience", "/projects", "/skills", "/contact", "/static/css/style.css"} {
		t.Run(path, func(t *testing.T) {
			body := get(t, h, path).Body.String()
			for _, host := range banned {
				if strings.Contains(body, host) {
					t.Errorf("%s references third-party host %q", path, host)
				}
			}
		})
	}
}

func TestJSONLDIsValidAndDerivedFromCV(t *testing.T) {
	h := newTestHandler(t)
	body := get(t, h, "/").Body.String()

	m := regexp.MustCompile(`(?s)<script type="application/ld\+json">(.*?)</script>`).FindStringSubmatch(body)
	if m == nil {
		t.Fatal("no JSON-LD block in the rendered page")
	}
	raw := m[1]

	// json.Marshal escapes <, > and & to their \u form, so no CV value can
	// close the surrounding script element.
	if strings.ContainsAny(raw, "<>&") {
		t.Errorf("JSON-LD contains an unescaped HTML character: %q", raw)
	}

	var doc map[string]any
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		t.Fatalf("JSON-LD does not parse: %v", err)
	}
	for _, key := range []string{"@context", "@type", "name", "jobTitle", "url", "address", "sameAs", "knowsAbout"} {
		if _, ok := doc[key]; !ok {
			t.Errorf("JSON-LD is missing %q", key)
		}
	}
	if doc["@type"] != "Person" {
		t.Errorf("@type = %v, want Person", doc["@type"])
	}
	if doc["name"] != "Barry Prendergast" {
		t.Errorf("name = %v, want it taken from the CV data", doc["name"])
	}
}
