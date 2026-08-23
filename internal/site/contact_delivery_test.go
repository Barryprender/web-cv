package site

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"barrypre.com/webcv/internal/data"
	"barrypre.com/webcv/internal/mail"
)

// stubSender accepts everything and records what it was given.
type stubSender struct {
	mu   sync.Mutex
	sent []mail.Message
}

func (*stubSender) Name() string { return "stub" }

func (s *stubSender) Send(_ context.Context, m mail.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sent = append(s.sent, m)
	return nil
}

func (s *stubSender) messages() []mail.Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]mail.Message(nil), s.sent...)
}

// failingSender stands in for a provider that is refusing messages.
type failingSender struct{}

func (failingSender) Name() string { return "failing" }

func (failingSender) Send(context.Context, mail.Message) error {
	return errors.New("provider said no: account suspended")
}

// blockingSender never returns until its context is done, which is what a
// provider that has stopped answering looks like from here.
type blockingSender struct{}

func (blockingSender) Name() string { return "blocking" }

func (blockingSender) Send(ctx context.Context, _ mail.Message) error {
	<-ctx.Done()
	return ctx.Err()
}

func TestContactDeliversTheSubmission(t *testing.T) {
	captureLog(t)
	sender := &stubSender{}
	h := NewHandler(2026, sender)

	fields := map[string]string{
		"name":    "Ada Lovelace",
		"email":   "ada@example.com",
		"message": "I would like to talk about the analytical engine.",
	}
	contentType, body := multipartFormBody(t, fields)
	rec := postContact(t, h, contentType, body, "application/json")

	if rec.Code != http.StatusOK {
		t.Fatalf("POST /contact = %d, want 200", rec.Code)
	}

	sent := sender.messages()
	if len(sent) != 1 {
		t.Fatalf("sent %d messages, want 1", len(sent))
	}
	got := sent[0]

	// Reply-To is the whole point: replying from the mailbox has to reach the
	// visitor, not the site's own sending identity.
	if got.ReplyTo != "ada@example.com" {
		t.Errorf("ReplyTo = %q, want the visitor's address", got.ReplyTo)
	}
	if !strings.Contains(got.Subject, "Ada Lovelace") {
		t.Errorf("Subject = %q, want the sender's name in it", got.Subject)
	}
	for _, want := range []string{"Ada Lovelace", "ada@example.com", "analytical engine"} {
		if !strings.Contains(got.Text, want) {
			t.Errorf("body is missing %q:\n%s", want, got.Text)
		}
	}
	if err := got.Validate(); err != nil {
		t.Errorf("composed a message the mail package rejects: %v", err)
	}
}

// The visitor is told when delivery fails. Reporting success for a message that
// went nowhere is the failure mode this whole path exists to avoid.
func TestContactReportsDeliveryFailure(t *testing.T) {
	captureLog(t)
	h := NewHandler(2026, failingSender{})

	contentType, body := multipartFormBody(t, validFields())
	rec := postContact(t, h, contentType, body, "application/json")

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("POST /contact = %d, want 502", rec.Code)
	}
	got := rec.Body.String()
	if !strings.Contains(got, data.Me.Contact.Email) {
		t.Errorf("failure response does not offer the direct address:\n%s", got)
	}
	// The provider's own words must not reach the visitor.
	if strings.Contains(got, "account suspended") {
		t.Errorf("provider detail leaked to the visitor:\n%s", got)
	}
}

// Without JS the result comes back as a redirect, so the failure has to survive
// the round trip as its own status rather than collapsing into "error".
func TestContactFailureRedirectsWithFailedStatus(t *testing.T) {
	captureLog(t)
	h := NewHandler(2026, failingSender{})

	contentType, body := multipartFormBody(t, validFields())
	rec := postContact(t, h, contentType, body, "text/html")

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("no-JS POST = %d, want 303", rec.Code)
	}
	if got, want := rec.Header().Get("Location"), "/contact?status=failed#form-status"; got != want {
		t.Errorf("Location = %q, want %q", got, want)
	}
}

func TestContactPageRendersDeliveryFailure(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestHandler(t).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/contact?status=failed", nil))

	body := rec.Body.String()
	if !strings.Contains(body, "something is wrong on my end") {
		t.Errorf("failed status is not rendered:\n%s", body)
	}
	// The whole point of this status is offering a route that does not depend
	// on the form working.
	if !strings.Contains(body, "mailto:"+data.Me.Contact.Email) {
		t.Error("failure copy does not offer a mailto fallback")
	}
}

func TestFormStatusAcceptsFailed(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/contact?status=failed", nil)
	if got := formStatus(r); got != "failed" {
		t.Errorf("formStatus() = %q, want %q", got, "failed")
	}
}

// An unconfigured deployment must refuse the message rather than accept one it
// cannot deliver. A nil sender is the shape that mistake takes in code.
func TestContactWithNoSenderConfigured(t *testing.T) {
	captureLog(t)
	h := NewHandler(2026, nil)

	contentType, body := multipartFormBody(t, validFields())
	rec := postContact(t, h, contentType, body, "application/json")

	if rec.Code != http.StatusBadGateway {
		t.Errorf("POST with no sender = %d, want 502", rec.Code)
	}
}

// A provider that stops answering must not hold the request past the server's
// write deadline; the visitor gets a failure instead of a dropped connection.
func TestContactTimesOutOnAHangingProvider(t *testing.T) {
	captureLog(t)
	h := NewHandler(2026, blockingSender{})

	contentType, body := multipartFormBody(t, validFields())

	done := make(chan int, 1)
	go func() {
		done <- postContact(t, h, contentType, body, "application/json").Code
	}()

	select {
	case code := <-done:
		if code != http.StatusBadGateway {
			t.Errorf("hanging provider = %d, want 502", code)
		}
	case <-time.After(sendTimeout + 3*time.Second):
		t.Fatal("request outlived the send timeout")
	}
}

// A malformed address would become an unusable Reply-To, and it is the
// visitor's to fix — so it is a 400 telling them to check it, not a 502
// blaming the site.
func TestContactRejectsMalformedEmail(t *testing.T) {
	captureLog(t)
	sender := &stubSender{}
	h := NewHandler(2026, sender)

	for _, addr := range []string{
		"not-an-address",
		"a@b@c.com",
		"Ada <ada@example.com>", // a display name is not a form field value
		"ada@example.com\r\nBcc: victim@example.com",
		"@example.com",
	} {
		fields := validFields()
		fields["email"] = addr
		contentType, body := multipartFormBody(t, fields)
		rec := postContact(t, h, contentType, body, "application/json")

		if rec.Code != http.StatusBadRequest {
			t.Errorf("email %q = %d, want 400", addr, rec.Code)
		}
	}
	if n := len(sender.messages()); n != 0 {
		t.Errorf("%d malformed submissions reached the sender", n)
	}
}

func TestContactAcceptsOrdinaryAddresses(t *testing.T) {
	captureLog(t)
	for _, addr := range []string{
		"ada@example.com",
		"ada.lovelace+cv@sub.example.co.uk",
		"a@b.io",
	} {
		sender := &stubSender{}
		h := NewHandler(2026, sender)

		fields := validFields()
		fields["email"] = addr
		contentType, body := multipartFormBody(t, fields)
		rec := postContact(t, h, contentType, body, "application/json")

		if rec.Code != http.StatusOK {
			t.Errorf("email %q = %d, want 200", addr, rec.Code)
		}
	}
}

// Delivery is attempted only after validation and the rate limiter, so a
// refused submission never costs a provider call.
func TestContactDoesNotSendRejectedSubmissions(t *testing.T) {
	captureLog(t)
	sender := &stubSender{}
	h := NewHandler(2026, sender)

	contentType, body := multipartFormBody(t, map[string]string{"name": "", "email": "", "message": ""})
	if rec := postContact(t, h, contentType, body, "application/json"); rec.Code != http.StatusBadRequest {
		t.Fatalf("empty submission = %d, want 400", rec.Code)
	}
	if n := len(sender.messages()); n != 0 {
		t.Errorf("an invalid submission reached the sender %d time(s)", n)
	}
}

func TestComposeContactEmailTruncates(t *testing.T) {
	msg := contactMessage{
		Name:    strings.Repeat("n", 500),
		Email:   "ada@example.com",
		Message: strings.Repeat("m", 40_000),
	}
	got := composeContactEmail(msg, time.Now())

	if len(got.Subject) > 200 {
		t.Errorf("subject is %d bytes; the name should be truncated", len(got.Subject))
	}
	// The 64 KB the endpoint accepts is not what gets handed to a provider.
	// The allowance above the 16 KB cap is the surrounding template text.
	if len(got.Text) > (16<<10)+500 {
		t.Errorf("email body is %d bytes; the message should be truncated to 16 KB", len(got.Text))
	}
}
