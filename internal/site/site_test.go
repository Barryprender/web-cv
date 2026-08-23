package site

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"barrypre.com/webcv/internal/data"
)

// newTestHandler builds a fresh handler per test. Each one carries its own rate
// limiter, so one test cannot exhaust another test's budget.
func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	return NewHandler(2026, &stubSender{})
}

// captureLog redirects the standard logger for the duration of a test.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	flags := log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(io.Discard)
		log.SetFlags(flags)
	})
	return &buf
}

// urlencodedBody builds a body the way a plain no-JS form POST would.
func urlencodedBody(fields map[string]string) (string, io.Reader) {
	form := url.Values{}
	for k, v := range fields {
		form.Set(k, v)
	}
	return "application/x-www-form-urlencoded", strings.NewReader(form.Encode())
}

// multipartFormBody builds a body the way fetch + FormData would have.
func multipartFormBody(t *testing.T, fields map[string]string) (string, io.Reader) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		if err := w.WriteField(k, v); err != nil {
			t.Fatalf("write field %q: %v", k, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	return w.FormDataContentType(), &buf
}

// postContact sends one submission and returns the recorded response.
func postContact(t *testing.T, h http.Handler, contentType string, body io.Reader, accept string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/contact", body)
	req.Header.Set("Content-Type", contentType)
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func validFields() map[string]string {
	return map[string]string{"name": "Ada", "email": "ada@example.com", "message": "hello"}
}

func TestPagesRender(t *testing.T) {
	tests := []struct {
		path      string
		wantTitle string
		wantBody  string
	}{
		{"/", "Barry Prendergast — Senior Full-Stack Engineer", "Barry Prendergast"},
		{"/experience", "Experience — Barry Prendergast", "Quality Compusoft"},
		{"/projects", "Projects — Barry Prendergast", "Archway Orthotics Portal"},
		{"/skills", "Skills — Barry Prendergast", "Frontend"},
		{"/contact", "Contact — Barry Prendergast", "form-status"},
	}
	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			h := newTestHandler(t)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
			if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
				t.Errorf("Content-Type = %q, want text/html", ct)
			}
			body := rec.Body.String()
			if !strings.Contains(body, "<title>"+tc.wantTitle+"</title>") {
				t.Errorf("missing title %q", tc.wantTitle)
			}
			if !strings.Contains(body, tc.wantBody) {
				t.Errorf("body does not mention %q", tc.wantBody)
			}
		})
	}
}

// The root pattern is registered as "/{$}". Without that it is a subtree
// wildcard and answers for every unmatched path.
func TestRootDoesNotSwallowUnknownPaths(t *testing.T) {
	h := newTestHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/no-such-page", nil))

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestExperiencePageRendersEveryJob(t *testing.T) {
	h := newTestHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/experience", nil))

	// Derived from the data, so reordering or renaming an entry does not
	// silently drop it from the page without failing here.
	body := rec.Body.String()
	for _, job := range data.Me.Jobs {
		if !strings.Contains(body, job.Company) {
			t.Errorf("experience page is missing %q", job.Company)
		}
	}
	if len(data.Me.Jobs) < 8 {
		t.Errorf("only %d entries in the timeline", len(data.Me.Jobs))
	}
}

func TestStaticAssetsServed(t *testing.T) {
	for _, path := range []string{
		"/static/css/style.css",
		"/static/js/main.js",
		"/static/js/components/cv-nav.js",
		"/static/js/components/cv-theme-toggle.js",
		"/static/js/components/cv-timeline.js",
		"/static/js/components/cv-contact-form.js",
	} {
		t.Run(path, func(t *testing.T) {
			h := newTestHandler(t)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
			if rec.Code != http.StatusOK {
				t.Errorf("status = %d, want 200", rec.Code)
			}
		})
	}
}

// main.js imports each component by path. A rename on one side only 404s, and
// because it is a module graph that takes down every component, not just the
// renamed one — which is how all four silently stopped registering once.
func TestComponentImportsResolve(t *testing.T) {
	h := newTestHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/static/js/main.js", nil))

	imports := 0
	for line := range strings.SplitSeq(rec.Body.String(), "\n") {
		_, rest, found := strings.Cut(line, `import './`)
		if !found {
			continue
		}
		path, _, _ := strings.Cut(rest, `'`)
		imports++

		assetRec := httptest.NewRecorder()
		h.ServeHTTP(assetRec, httptest.NewRequest(http.MethodGet, "/static/js/"+path, nil))
		if assetRec.Code != http.StatusOK {
			t.Errorf("main.js imports %q which serves %d, want 200", path, assetRec.Code)
		}
	}
	if imports == 0 {
		t.Fatal("found no imports in main.js — the scan is not matching anything")
	}
}

func TestContactAcceptsBothEncodings(t *testing.T) {
	tests := []struct {
		name string
		body func(*testing.T) (string, io.Reader)
	}{
		{"urlencoded", func(*testing.T) (string, io.Reader) { return urlencodedBody(validFields()) }},
		{"multipart", func(t *testing.T) (string, io.Reader) { return multipartFormBody(t, validFields()) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			captureLog(t)
			h := newTestHandler(t)
			contentType, body := tc.body(t)
			rec := postContact(t, h, contentType, body, "application/json")

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
			}
			var got struct {
				OK    bool   `json:"ok"`
				Error string `json:"error"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if !got.OK {
				t.Errorf("ok = false, error = %q", got.Error)
			}
		})
	}
}

func TestContactRejectsIncompleteSubmission(t *testing.T) {
	captureLog(t)
	h := newTestHandler(t)
	contentType, body := urlencodedBody(map[string]string{"name": "", "email": "", "message": ""})
	rec := postContact(t, h, contentType, body, "application/json")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	// The message must name what is missing, not restate that everything is
	// required.
	if !strings.Contains(rec.Body.String(), "Please enter your name, your email address and your message.") {
		t.Errorf("body = %s, want the missing fields named", rec.Body.String())
	}
}

// An oversized body must be reported as too large, not misreported as an empty
// submission. ParseMultipartForm discards the inner ParseForm error once the
// body turns out not to be multipart, which is exactly how that regression arose.
func TestContactRejectsOversizedBody(t *testing.T) {
	fields := validFields()
	fields["message"] = strings.Repeat("x", maxContactBody+1024)

	tests := []struct {
		name string
		body func(*testing.T) (string, io.Reader)
	}{
		{"urlencoded", func(*testing.T) (string, io.Reader) { return urlencodedBody(fields) }},
		{"multipart", func(t *testing.T) (string, io.Reader) { return multipartFormBody(t, fields) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			captureLog(t)
			h := newTestHandler(t)
			contentType, body := tc.body(t)
			rec := postContact(t, h, contentType, body, "application/json")

			if rec.Code != http.StatusRequestEntityTooLarge {
				t.Fatalf("status = %d, want 413 (body %s)", rec.Code, rec.Body.String())
			}
		})
	}
}

// Without JS the browser does a plain POST, so the result has to survive a
// redirect. The fragment is what makes the message visible on arrival.
func TestContactRedirectsWithoutJSON(t *testing.T) {
	tests := []struct {
		name   string
		fields map[string]string
		want   string
	}{
		{"success", validFields(), "/contact?status=sent#form-status"},
		{"failure", map[string]string{"name": "", "email": "", "message": ""}, "/contact?status=error#form-status"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			captureLog(t)
			h := newTestHandler(t)
			contentType, body := urlencodedBody(tc.fields)
			rec := postContact(t, h, contentType, body, "")

			if rec.Code != http.StatusSeeOther {
				t.Fatalf("status = %d, want 303", rec.Code)
			}
			if got := rec.Header().Get("Location"); got != tc.want {
				t.Errorf("Location = %q, want %q", got, tc.want)
			}
		})
	}
}

// A submitted value must not be able to forge log lines or smuggle terminal
// escapes into whatever reads them.
//
// Only the failure path logs the submission, so that is the path this exercises
// — a delivered message is never written to the log at all.
func TestContactFailureLogIsInjectionSafe(t *testing.T) {
	buf := captureLog(t)
	h := NewHandler(2026, failingSender{})

	contentType, body := multipartFormBody(t, map[string]string{
		"name":    "Evil\n2026/01/01 00:00:00 FORGED ADMIN LOGIN",
		"email":   "e@example.com",
		"message": "line1\nline2\x1b[31mRED",
	})
	postContact(t, h, contentType, body, "application/json")

	logged := buf.String()
	if got := strings.Count(strings.TrimRight(logged, "\n"), "\n"); got != 0 {
		t.Errorf("log spans %d extra lines, want a single line:\n%s", got, logged)
	}
	if strings.ContainsRune(logged, '\x1b') {
		t.Error("log contains a raw escape character")
	}
	for _, want := range []string{`\n`, `\x1b`} {
		if !strings.Contains(logged, want) {
			t.Errorf("log is missing escaped %q:\n%s", want, logged)
		}
	}
}

func TestContactFailureLogIsTruncated(t *testing.T) {
	buf := captureLog(t)
	h := NewHandler(2026, failingSender{})

	fields := validFields()
	fields["message"] = strings.Repeat("z", 5000)
	contentType, body := multipartFormBody(t, fields)
	postContact(t, h, contentType, body, "application/json")

	if n := strings.Count(buf.String(), "z"); n > 1000 {
		t.Errorf("logged %d message characters, want the field truncated to 1000", n)
	}
}

// A delivered message is in a mailbox; keeping a second copy in the log is an
// unnecessary place for a stranger's words to sit.
func TestContactSuccessDoesNotLogTheMessage(t *testing.T) {
	buf := captureLog(t)
	h := newTestHandler(t)

	fields := validFields()
	fields["message"] = "a-very-distinctive-body"
	contentType, body := multipartFormBody(t, fields)
	postContact(t, h, contentType, body, "application/json")

	if strings.Contains(buf.String(), "a-very-distinctive-body") {
		t.Errorf("a delivered message was written to the log:\n%s", buf.String())
	}
}

func TestFormStatusWhitelist(t *testing.T) {
	tests := []struct{ query, want string }{
		{"", ""},
		{"?status=sent", "sent"},
		{"?status=error", "error"},
		{"?status=%3Cscript%3E", ""},
		{"?status=anything-else", ""},
	}
	for _, tc := range tests {
		t.Run(tc.query, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/contact"+tc.query, nil)
			if got := formStatus(r); got != tc.want {
				t.Errorf("formStatus() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestContactPageRendersStatus(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		want     string
		wantNone string
	}{
		{"sent", "?status=sent", "Message sent", ""},
		{"error", "?status=error", "That did not send", ""},
		{"busy", "?status=busy", "Too many messages", ""},
		{"none", "", "", "Message sent"},
		{"injected", "?status=%3Cscript%3E", "", "<script>"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newTestHandler(t)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/contact"+tc.query, nil))

			body := rec.Body.String()
			if tc.want != "" && !strings.Contains(body, tc.want) {
				t.Errorf("page does not contain %q", tc.want)
			}
			if tc.wantNone != "" && strings.Contains(body, tc.wantNone) {
				t.Errorf("page unexpectedly contains %q", tc.wantNone)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name string
		in   string
		n    int
		want string
	}{
		{"shorter than limit", "abc", 10, "abc"},
		{"exactly the limit", "abc", 3, "abc"},
		{"longer than limit", "abcdef", 3, "abc"},
		{"does not split a rune", "aé", 2, "a"}, // é is two bytes
		{"zero limit", "abc", 0, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := truncate(tc.in, tc.n); got != tc.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tc.in, tc.n, got, tc.want)
			}
		})
	}
}
