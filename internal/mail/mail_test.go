package mail

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func validMessage() Message {
	return Message{Subject: "hello", Text: "body", ReplyTo: "ada@example.com"}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		msg     Message
		wantErr bool
	}{
		{"ok", validMessage(), false},
		{"no reply-to is allowed", Message{Subject: "s", Text: "t"}, false},
		{"empty subject", Message{Text: "t"}, true},
		{"blank subject", Message{Subject: "   ", Text: "t"}, true},
		{"empty body", Message{Subject: "s"}, true},
		{"reply-to with a newline", Message{Subject: "s", Text: "t", ReplyTo: "a@b.com\r\nBcc: c@d.com"}, true},
		{"reply-to with a display name", Message{Subject: "s", Text: "t", ReplyTo: "Ada <a@b.com>"}, true},
		{"unparseable reply-to", Message{Subject: "s", Text: "t", ReplyTo: "not-an-address"}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.msg.Validate(); (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

// recordingSender counts calls and returns whatever it was told to.
type recordingSender struct {
	name  string
	err   error
	calls atomic.Int32
	delay time.Duration
}

func (r *recordingSender) Name() string { return r.name }

func (r *recordingSender) Send(ctx context.Context, _ Message) error {
	r.calls.Add(1)
	if r.delay > 0 {
		select {
		case <-time.After(r.delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return r.err
}

func TestChainStopsAtTheFirstSuccess(t *testing.T) {
	first := &recordingSender{name: "first"}
	second := &recordingSender{name: "second"}

	if err := NewChain(nil, first, second).Send(t.Context(), validMessage()); err != nil {
		t.Fatalf("Send() = %v, want nil", err)
	}
	if got := first.calls.Load(); got != 1 {
		t.Errorf("first provider called %d times, want 1", got)
	}
	// The second provider is the fallback, not a duplicate delivery.
	if got := second.calls.Load(); got != 0 {
		t.Errorf("second provider called %d times after the first succeeded, want 0", got)
	}
}

// The point of configuring two providers: one being down is not an outage.
func TestChainFallsThroughToTheSecond(t *testing.T) {
	first := &recordingSender{name: "first", err: errors.New("down")}
	second := &recordingSender{name: "second"}

	if err := NewChain(nil, first, second).Send(t.Context(), validMessage()); err != nil {
		t.Fatalf("Send() = %v, want nil — the second provider accepted it", err)
	}
	if got := second.calls.Load(); got != 1 {
		t.Errorf("second provider called %d times, want 1", got)
	}
}

func TestChainReportsEveryFailure(t *testing.T) {
	first := &recordingSender{name: "first", err: errors.New("first is down")}
	second := &recordingSender{name: "second", err: errors.New("second is down")}

	err := NewChain(nil, first, second).Send(t.Context(), validMessage())
	if err == nil {
		t.Fatal("Send() = nil, want an error when every provider failed")
	}
	// Both reasons, not just the last: diagnosing this from a log needs to
	// show whether one provider or the whole network was the problem.
	for _, want := range []string{"first is down", "second is down"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q is missing %q", err, want)
		}
	}
}

func TestChainWithNoProviders(t *testing.T) {
	err := NewChain(nil).Send(t.Context(), validMessage())
	if !errors.Is(err, ErrNotConfigured) {
		t.Errorf("Send() = %v, want ErrNotConfigured", err)
	}
}

func TestChainValidatesBeforeSending(t *testing.T) {
	s := &recordingSender{name: "s"}
	if err := NewChain(nil, s).Send(t.Context(), Message{Subject: "s"}); err == nil {
		t.Error("Send() = nil, want an error for a message with no body")
	}
	if got := s.calls.Load(); got != 0 {
		t.Errorf("provider called %d times with an invalid message, want 0", got)
	}
}

// A slow first provider must not eat the whole budget: the per-attempt deadline
// is what leaves the second one time to try.
func TestChainBoundsEachAttempt(t *testing.T) {
	slow := &recordingSender{name: "slow", delay: time.Hour}
	fast := &recordingSender{name: "fast"}

	chain := NewChain(nil, slow, fast)
	chain.attempt = 50 * time.Millisecond

	start := time.Now()
	if err := chain.Send(t.Context(), validMessage()); err != nil {
		t.Fatalf("Send() = %v, want nil", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("took %v; the per-attempt deadline did not apply", elapsed)
	}
	if got := fast.calls.Load(); got != 1 {
		t.Errorf("fallback called %d times, want 1", got)
	}
}

// A cancelled request means nobody is waiting for the answer.
func TestChainStopsWhenTheCallerGivesUp(t *testing.T) {
	first := &recordingSender{name: "first", err: errors.New("down")}
	second := &recordingSender{name: "second"}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if err := NewChain(nil, first, second).Send(ctx, validMessage()); err == nil {
		t.Error("Send() = nil, want an error on a cancelled context")
	}
	if got := second.calls.Load(); got != 0 {
		t.Errorf("second provider called %d times after cancellation, want 0", got)
	}
}

func TestChainName(t *testing.T) {
	chain := NewChain(nil, &recordingSender{name: "resend"}, &recordingSender{name: "postmark"})
	if got, want := chain.Name(), "resend+postmark"; got != want {
		t.Errorf("Name() = %q, want %q", got, want)
	}
	if got := chain.Senders(); got != 2 {
		t.Errorf("Senders() = %d, want 2", got)
	}
}

func TestLogSender(t *testing.T) {
	var lines []string
	s := LogSender{Logf: func(format string, args ...any) {
		lines = append(lines, format)
		_ = args
	}}
	if err := s.Send(t.Context(), validMessage()); err != nil {
		t.Fatalf("Send() = %v, want nil", err)
	}
	if len(lines) != 1 {
		t.Errorf("logged %d lines, want 1", len(lines))
	}
	// %q, so a submitted value cannot forge a log line.
	if !strings.Contains(lines[0], "%q") {
		t.Errorf("log format %q does not quote its values", lines[0])
	}
	if err := s.Send(t.Context(), Message{}); err == nil {
		t.Error("Send() = nil, want an error for an empty message")
	}
}

func TestOneLine(t *testing.T) {
	cases := map[string]string{
		"plain":          "plain",
		"a\r\nb":         "ab",
		"a\tb":           "a b",
		"a\x00\x1b[31mb": "a[31mb",
		"café —":         "café —", // non-ASCII is left alone
	}
	for in, want := range cases {
		if got := oneLine(in); got != want {
			t.Errorf("oneLine(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTruncateKeepsRunesWhole(t *testing.T) {
	// "é" is two bytes; cutting at 3 must not split it.
	if got := truncate("café", 4); got != "caf" {
		t.Errorf("truncate = %q, want %q", got, "caf")
	}
	if got := truncate("short", 100); got != "short" {
		t.Errorf("truncate should leave a short string alone, got %q", got)
	}
}

// providerTest exercises one provider against a stub of its API.
type providerTest struct {
	name    string
	build   func(endpoint string) Sender
	auth    func(*http.Request) string // returns the credential the request carried
	decode  func([]byte) (subject, text, replyTo string)
	okBody  string
	errBody string
	errCode int
}

func providerTests() []providerTest {
	return []providerTest{
		{
			name: "resend",
			build: func(endpoint string) Sender {
				s := NewResend("test-key", "cv@barrypre.com", "barry@example.com")
				s.Endpoint = endpoint
				return s
			},
			auth: func(r *http.Request) string {
				return strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			},
			decode: func(b []byte) (string, string, string) {
				var got resendRequest
				_ = json.Unmarshal(b, &got)
				return got.Subject, got.Text, got.ReplyTo
			},
			okBody:  `{"id":"abc"}`,
			errBody: `{"message":"domain is not verified","name":"validation_error"}`,
			errCode: http.StatusUnprocessableEntity,
		},
		{
			name: "postmark",
			build: func(endpoint string) Sender {
				s := NewPostmark("test-token", "cv@barrypre.com", "barry@example.com")
				s.Endpoint = endpoint
				return s
			},
			auth: func(r *http.Request) string {
				return r.Header.Get("X-Postmark-Server-Token")
			},
			decode: func(b []byte) (string, string, string) {
				var got postmarkRequest
				_ = json.Unmarshal(b, &got)
				return got.Subject, got.TextBody, got.ReplyTo
			},
			okBody:  `{"ErrorCode":0,"Message":"OK"}`,
			errBody: `{"ErrorCode":300,"Message":"Sender signature not confirmed"}`,
			errCode: http.StatusUnprocessableEntity,
		},
	}
}

func TestProviderSendsTheMessage(t *testing.T) {
	for _, tc := range providerTests() {
		t.Run(tc.name, func(t *testing.T) {
			var (
				gotAuth   string
				gotMethod string
				gotType   string
				body      []byte
			)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotAuth = tc.auth(r)
				gotMethod = r.Method
				gotType = r.Header.Get("Content-Type")
				body, _ = io.ReadAll(r.Body)
				w.Header().Set("Content-Type", "application/json")
				io.WriteString(w, tc.okBody)
			}))
			defer srv.Close()

			msg := Message{Subject: "hello", Text: "the body", ReplyTo: "ada@example.com"}
			if err := tc.build(srv.URL).Send(t.Context(), msg); err != nil {
				t.Fatalf("Send() = %v, want nil", err)
			}

			if gotMethod != http.MethodPost {
				t.Errorf("method = %q, want POST", gotMethod)
			}
			if gotType != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", gotType)
			}
			if gotAuth == "" {
				t.Error("request carried no credential")
			}
			subject, text, replyTo := tc.decode(body)
			if subject != "hello" || text != "the body" || replyTo != "ada@example.com" {
				t.Errorf("sent subject=%q text=%q replyTo=%q", subject, text, replyTo)
			}
			// Never HTML: the body is a stranger's prose.
			if strings.Contains(string(body), "Html") || strings.Contains(string(body), `"html"`) {
				t.Errorf("request contains an HTML body field:\n%s", body)
			}
		})
	}
}

func TestProviderReportsAPIErrors(t *testing.T) {
	for _, tc := range providerTests() {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.errCode)
				io.WriteString(w, tc.errBody)
			}))
			defer srv.Close()

			err := tc.build(srv.URL).Send(t.Context(), validMessage())
			if err == nil {
				t.Fatal("Send() = nil, want an error")
			}
			// The provider's explanation belongs in the log, so it has to
			// survive into the error.
			if !strings.Contains(err.Error(), "not verified") && !strings.Contains(err.Error(), "not confirmed") {
				t.Errorf("error %q does not carry the provider's reason", err)
			}
		})
	}
}

// Postmark answers 200 with a non-zero ErrorCode for some refusals. Trusting
// the status alone would report a message delivered that went nowhere.
func TestPostmarkTreatsErrorCodeAsFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"ErrorCode":406,"Message":"Inactive recipient"}`)
	}))
	defer srv.Close()

	s := NewPostmark("tok", "a@b.com", "c@d.com")
	s.Endpoint = srv.URL

	err := s.Send(t.Context(), validMessage())
	if err == nil {
		t.Fatal("Send() = nil, want an error — a 200 with ErrorCode 406 is not a delivery")
	}
	if !strings.Contains(err.Error(), "Inactive recipient") {
		t.Errorf("error %q does not name the reason", err)
	}
}

func TestPostmarkSetsMessageStream(t *testing.T) {
	var body []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		io.WriteString(w, `{"ErrorCode":0}`)
	}))
	defer srv.Close()

	s := NewPostmark("tok", "a@b.com", "c@d.com")
	s.Endpoint = srv.URL
	if err := s.Send(t.Context(), validMessage()); err != nil {
		t.Fatalf("Send() = %v", err)
	}
	if !strings.Contains(string(body), `"MessageStream":"outbound"`) {
		t.Errorf("request does not set the outbound stream:\n%s", body)
	}
}

// Every request carries a credential, so a redirect must not be followed — that
// is how a token ends up at a host it was never issued for.
func TestProviderRefusesRedirects(t *testing.T) {
	for _, tc := range providerTests() {
		t.Run(tc.name, func(t *testing.T) {
			var elsewhere atomic.Int32
			sink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				elsewhere.Add(1)
				io.WriteString(w, `{"ErrorCode":0,"id":"x"}`)
			}))
			defer sink.Close()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, sink.URL, http.StatusTemporaryRedirect)
			}))
			defer srv.Close()

			if err := tc.build(srv.URL).Send(t.Context(), validMessage()); err == nil {
				t.Error("Send() = nil, want the redirect refused")
			}
			if got := elsewhere.Load(); got != 0 {
				t.Errorf("the credential was replayed at the redirect target %d time(s)", got)
			}
		})
	}
}

func TestProviderHonoursContextCancellation(t *testing.T) {
	for _, tc := range providerTests() {
		t.Run(tc.name, func(t *testing.T) {
			release := make(chan struct{})
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				<-release
			}))
			defer srv.Close()
			defer close(release)

			ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
			defer cancel()

			if err := tc.build(srv.URL).Send(ctx, validMessage()); err == nil {
				t.Error("Send() = nil, want a timeout error")
			}
		})
	}
}

func TestProviderValidatesBeforeCalling(t *testing.T) {
	for _, tc := range providerTests() {
		t.Run(tc.name, func(t *testing.T) {
			var called atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				called.Add(1)
			}))
			defer srv.Close()

			bad := Message{Subject: "s", Text: "t", ReplyTo: "a@b.com\r\nBcc: victim@example.com"}
			if err := tc.build(srv.URL).Send(t.Context(), bad); err == nil {
				t.Error("Send() = nil, want the header injection refused")
			}
			if got := called.Load(); got != 0 {
				t.Errorf("provider was called %d times with an invalid message", got)
			}
		})
	}
}
