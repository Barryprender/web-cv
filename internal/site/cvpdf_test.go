package site

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"barrypre.com/webcv/internal/cvpdf"
	"barrypre.com/webcv/internal/data"
)

// The PDF is generated at build time and committed, so nothing regenerates it
// when internal/data changes. This is the check that catches a forgotten
// `go generate ./internal/site` before it ships a CV that disagrees with the
// site it is downloaded from.
func TestPDFIsCurrent(t *testing.T) {
	// Every language, because a translation edited without regenerating would
	// otherwise ship a Spanish PDF that disagrees with the Spanish site.
	for _, lang := range data.Langs {
		t.Run(string(lang), func(t *testing.T) {
			asset := cvPDFAssetFor(lang)
			committed, err := staticFS.ReadFile("static/" + asset)
			if err != nil {
				t.Fatalf("read the committed PDF: %v", err)
			}
			if !bytes.Equal(committed, cvpdf.Build(lang)) {
				t.Errorf("static/%s is out of date with internal/data.\n"+
					"Regenerate it with: go generate ./internal/site", asset)
			}
		})
	}
}

// Each language must be served its own file. Serving one document from both
// URLs is the failure this catches, and it would otherwise look fine.
func TestEachLanguageServesItsOwnPDF(t *testing.T) {
	handler := newTestHandler(t)
	seen := make(map[string]data.Lang, len(data.Langs))

	for _, lang := range data.Langs {
		path := cvPDFPathFor(lang)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s = %d, want 200", path, rec.Code)
		}

		body := rec.Body.String()
		if other, ok := seen[body]; ok {
			t.Errorf("%s and %s serve identical bytes", other, lang)
		}
		seen[body] = lang

		if !bytes.Equal([]byte(body), cvpdf.Build(lang)) {
			t.Errorf("%s does not serve the %s document", path, lang)
		}
	}
}

func TestCVPDFRoute(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestHandler(t).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, cvPDFPath, nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s = %d, want 200", cvPDFPath, rec.Code)
	}
	// Served as octet-stream it would be unopenable in the browser's viewer,
	// and under nosniff nothing would recover the type.
	if got, want := rec.Header().Get("Content-Type"), "application/pdf"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
	if !bytes.HasPrefix(rec.Body.Bytes(), []byte("%PDF-")) {
		t.Error("body is not a PDF")
	}
}

func TestCVPDFDisposition(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestHandler(t).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, cvPDFPath, nil))

	got := rec.Header().Get("Content-Disposition")
	// inline, not attachment: the viewer opens it, and the download attribute
	// on the link is what saves it.
	if !strings.HasPrefix(got, "inline;") {
		t.Errorf("Content-Disposition = %q, want an inline disposition", got)
	}
	if !strings.Contains(got, `filename="Barry-Prendergast-CV.pdf"`) {
		t.Errorf("Content-Disposition = %q, want the CV filename", got)
	}
}

func TestCVPDFRevalidates(t *testing.T) {
	handler := newTestHandler(t)

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, cvPDFPath, nil))

	etag := first.Header().Get("ETag")
	if etag == "" {
		t.Fatal("no ETag on the PDF; every visit would refetch the whole file")
	}
	// Without a content hash in the URL the response must revalidate rather
	// than be cached immutably — otherwise an updated CV would never reach
	// anyone who had already downloaded the old one.
	if got := first.Header().Get("Cache-Control"); got != "public, no-cache" {
		t.Errorf("Cache-Control = %q, want %q", got, "public, no-cache")
	}

	req := httptest.NewRequest(http.MethodGet, cvPDFPath, nil)
	req.Header.Set("If-None-Match", etag)
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, req)

	if second.Code != http.StatusNotModified {
		t.Errorf("conditional GET = %d, want 304", second.Code)
	}
	if second.Body.Len() != 0 {
		t.Errorf("304 carried %d bytes of body", second.Body.Len())
	}
}

// The download is a plain GET of a static file. Nothing about it may become a
// second way to make the server do work.
func TestCVPDFRejectsNonGET(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		rec := httptest.NewRecorder()
		newTestHandler(t).ServeHTTP(rec, httptest.NewRequest(method, cvPDFPath, nil))
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s %s = %d, want 405", method, cvPDFPath, rec.Code)
		}
	}
}

func TestPagesLinkToTheCV(t *testing.T) {
	// The download is worthless if no page offers it. Both entry points that
	// a visitor plausibly looks at must carry the link.
	for _, lang := range data.Langs {
		for _, route := range []string{"/", "/contact"} {
			path := localPath(lang, route)
			want := cvPDFPathFor(lang)

			rec := httptest.NewRecorder()
			newTestHandler(t).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
			body := rec.Body.String()
			// The language-local PDF, not the English one: a Spanish page that
			// hands over an English CV is the drift worth catching.
			if !strings.Contains(body, `href="`+want+`"`) {
				t.Errorf("%s does not link to %s", path, want)
			}
			if !strings.Contains(body, "download") {
				t.Errorf("%s links to the CV without a download attribute", path)
			}
		}
	}
}

func TestSafeFilename(t *testing.T) {
	cases := map[string]string{
		"Barry Prendergast":   "Barry-Prendergast",
		`Bad" name`:           "Bad-name",
		"Name\r\nInjected: x": "Name-Injected-x",
		"  padded  ":          "padded",
		"José Núñez":          "Jos-N-ez", // accented letters are dropped, never emitted raw
	}
	for in, want := range cases {
		if got := safeFilename(in); got != want {
			t.Errorf("safeFilename(%q) = %q, want %q", in, got, want)
		}
	}
}

// A quote or a newline reaching the header would let a name break out of the
// quoted parameter. safeFilename is what stops it; this pins that.
func TestCVPDFFilenameIsHeaderSafe(t *testing.T) {
	if strings.ContainsAny(cvPDFFilename, "\"\\\r\n;") {
		t.Errorf("filename %q carries a character that would break the header", cvPDFFilename)
	}
}
