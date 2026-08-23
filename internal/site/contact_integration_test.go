package site

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"barrypre.com/webcv/internal/mail"
)

// These exercise the whole path in one piece — form POST, handler, chain, real
// provider client, HTTP wire format — against a stub standing in for the
// provider's API. The unit tests either side of this cover the handler with a
// fake sender and the providers with a fake handler; only this covers the seam
// where the two meet.

// fakeProvider stands in for one of the real APIs.
type fakeProvider struct {
	server   *httptest.Server
	hits     atomic.Int32
	lastBody atomic.Value // string
}

// newFakeProvider serves status with body for every request.
func newFakeProvider(status int, body string) *fakeProvider {
	f := &fakeProvider{}
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.hits.Add(1)
		raw, _ := io.ReadAll(r.Body)
		f.lastBody.Store(string(raw))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		io.WriteString(w, body)
	}))
	return f
}

func (f *fakeProvider) close() { f.server.Close() }
func (f *fakeProvider) body() string {
	s, _ := f.lastBody.Load().(string)
	return s
}

const (
	resendOK     = `{"id":"fake-id"}`
	resendFail   = `{"message":"domain not verified","name":"validation_error"}`
	postmarkOK   = `{"ErrorCode":0,"Message":"OK"}`
	postmarkFail = `{"ErrorCode":300,"Message":"sender signature not confirmed"}`
	fakeFromAddr = "cv@barrypre.com"
	fakeToAddr   = "barry@example.com"
)

// buildChain wires both real provider clients at the given stub endpoints.
func buildChain(resendURL, postmarkURL string) *mail.Chain {
	r := mail.NewResend("re_test_key", fakeFromAddr, fakeToAddr)
	r.Endpoint = resendURL
	p := mail.NewPostmark("pm_test_token", fakeFromAddr, fakeToAddr)
	p.Endpoint = postmarkURL
	return mail.NewChain(log.Printf, r, p)
}

func TestEndToEndDeliveryThroughResend(t *testing.T) {
	captureLog(t)

	resend := newFakeProvider(http.StatusOK, resendOK)
	defer resend.close()
	postmark := newFakeProvider(http.StatusOK, postmarkOK)
	defer postmark.close()

	h := NewHandler(2026, buildChain(resend.server.URL, postmark.server.URL))

	contentType, body := multipartFormBody(t, map[string]string{
		"name":    "Ada Lovelace",
		"email":   "ada@example.com",
		"message": "Interested in the Go work.",
	})
	rec := postContact(t, h, contentType, body, "application/json")

	if rec.Code != http.StatusOK {
		t.Fatalf("POST /contact = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
	if got := resend.hits.Load(); got != 1 {
		t.Errorf("Resend called %d times, want 1", got)
	}
	// Resend accepted it, so Postmark must not also send a copy.
	if got := postmark.hits.Load(); got != 0 {
		t.Errorf("Postmark called %d times after Resend succeeded, want 0", got)
	}

	var sent struct {
		From    string   `json:"from"`
		To      []string `json:"to"`
		Subject string   `json:"subject"`
		Text    string   `json:"text"`
		ReplyTo string   `json:"reply_to"`
	}
	if err := json.Unmarshal([]byte(resend.body()), &sent); err != nil {
		t.Fatalf("provider received unparseable JSON: %v\n%s", err, resend.body())
	}
	if sent.From != fakeFromAddr {
		t.Errorf("from = %q, want %q", sent.From, fakeFromAddr)
	}
	if len(sent.To) != 1 || sent.To[0] != fakeToAddr {
		t.Errorf("to = %v, want [%q]", sent.To, fakeToAddr)
	}
	if sent.ReplyTo != "ada@example.com" {
		t.Errorf("reply_to = %q, want the visitor's address", sent.ReplyTo)
	}
	if !strings.Contains(sent.Text, "Interested in the Go work.") {
		t.Errorf("body does not carry the message:\n%s", sent.Text)
	}
}

// The reason for configuring two providers: Resend refusing is not an outage.
func TestEndToEndFailsOverToPostmark(t *testing.T) {
	captureLog(t)

	resend := newFakeProvider(http.StatusUnprocessableEntity, resendFail)
	defer resend.close()
	postmark := newFakeProvider(http.StatusOK, postmarkOK)
	defer postmark.close()

	h := NewHandler(2026, buildChain(resend.server.URL, postmark.server.URL))

	contentType, body := multipartFormBody(t, validFields())
	rec := postContact(t, h, contentType, body, "application/json")

	if rec.Code != http.StatusOK {
		t.Fatalf("POST /contact = %d, want 200 — Postmark accepted it", rec.Code)
	}
	if got := postmark.hits.Load(); got != 1 {
		t.Fatalf("Postmark called %d times, want 1", got)
	}
	if !strings.Contains(postmark.body(), `"MessageStream":"outbound"`) {
		t.Errorf("Postmark request is missing the message stream:\n%s", postmark.body())
	}
}

// Only when both refuse does the visitor hear about it.
func TestEndToEndReportsFailureWhenBothProvidersRefuse(t *testing.T) {
	buf := captureLog(t)

	resend := newFakeProvider(http.StatusUnprocessableEntity, resendFail)
	defer resend.close()
	postmark := newFakeProvider(http.StatusUnprocessableEntity, postmarkFail)
	defer postmark.close()

	h := NewHandler(2026, buildChain(resend.server.URL, postmark.server.URL))

	contentType, body := multipartFormBody(t, map[string]string{
		"name":    "Ada",
		"email":   "ada@example.com",
		"message": "a message worth keeping",
	})
	rec := postContact(t, h, contentType, body, "application/json")

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("POST /contact = %d, want 502", rec.Code)
	}
	if got := resend.hits.Load(); got != 1 {
		t.Errorf("Resend called %d times, want 1", got)
	}
	if got := postmark.hits.Load(); got != 1 {
		t.Errorf("Postmark called %d times, want 1", got)
	}

	logged := buf.String()
	// The undelivered message has to be recoverable from the log — the visitor
	// has just been told it did not arrive.
	if !strings.Contains(logged, "a message worth keeping") {
		t.Errorf("the undelivered message was not logged:\n%s", logged)
	}
	// And both providers' reasons have to be there to diagnose it.
	for _, want := range []string{"domain not verified", "sender signature not confirmed"} {
		if !strings.Contains(logged, want) {
			t.Errorf("log is missing %q:\n%s", want, logged)
		}
	}

	// None of that reaches the visitor.
	var got struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.OK {
		t.Error("ok = true for a message that was never delivered")
	}
	for _, leak := range []string{"domain not verified", "sender signature", "re_test_key", "pm_test_token"} {
		if strings.Contains(got.Error, leak) {
			t.Errorf("response leaks %q: %q", leak, got.Error)
		}
	}
}

// A credential must never appear in a log line, on any path.
func TestEndToEndDoesNotLogCredentials(t *testing.T) {
	buf := captureLog(t)

	resend := newFakeProvider(http.StatusUnauthorized, `{"message":"invalid api key"}`)
	defer resend.close()
	postmark := newFakeProvider(http.StatusUnauthorized, `{"ErrorCode":10,"Message":"bad token"}`)
	defer postmark.close()

	h := NewHandler(2026, buildChain(resend.server.URL, postmark.server.URL))

	contentType, body := multipartFormBody(t, validFields())
	postContact(t, h, contentType, body, "application/json")

	for _, secret := range []string{"re_test_key", "pm_test_token"} {
		if strings.Contains(buf.String(), secret) {
			t.Errorf("log leaks %q:\n%s", secret, buf.String())
		}
	}
}
