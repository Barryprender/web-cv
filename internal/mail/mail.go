// Package mail delivers the contact form's messages through a transactional
// email API.
//
// Two providers are implemented, Resend and Postmark, behind one Sender
// interface. Both are plain JSON over net/http, so supporting them costs no
// dependency. Configure either or both: with both configured, Chain tries them
// in order and only reports failure when every one has failed, which is the
// point of carrying two.
//
// Nothing here retries a failed send on its own. A contact form has a person
// waiting on the response, and a silent retry loop would hold the request open
// past the server's write timeout; the visitor is told instead, and they can
// resubmit or use the address on the page.
package mail

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strings"
	"time"
	"unicode"
)

// Message is one outbound email. From and To live on the provider, since they
// are configuration rather than anything the sender of the form chooses.
type Message struct {
	Subject string

	// Text is the plain-text part and is always required. It is what a
	// text-only client shows, and a message with no text part scores worse
	// with spam filters — so HTML never replaces it, only accompanies it.
	Text string

	// HTML is the optional rich part. Providers send both when it is set, and
	// the client picks. Whatever builds this must escape per context —
	// html/template does, string concatenation does not.
	HTML string

	// ReplyTo is the visitor's own address, so replying from the mailbox goes
	// back to them rather than to the site's sending identity. It is validated
	// before use — see Validate.
	ReplyTo string
}

// Sender delivers a message. Implementations must be safe for concurrent use.
type Sender interface {
	// Send delivers m, honouring ctx for cancellation and deadlines.
	Send(ctx context.Context, m Message) error

	// Name identifies the provider in logs. It must never include credentials.
	Name() string
}

// ErrNotConfigured reports that no provider was configured. It is separate from
// a delivery failure: one is a deployment mistake to fix, the other is a
// provider having a bad day.
var ErrNotConfigured = errors.New("mail: no provider configured")

// Validate reports whether the message can be sent as it stands.
//
// The address check matters beyond politeness: it becomes the Reply-To header
// on an email built from data a stranger submitted. Both providers take JSON,
// which escapes a newline rather than passing it through, but an address is
// still the one field here that ends up in a header, so it is parsed properly
// and rejected if it carries anything a header cannot hold.
func (m Message) Validate() error {
	if strings.TrimSpace(m.Subject) == "" {
		return errors.New("mail: empty subject")
	}
	if strings.TrimSpace(m.Text) == "" {
		return errors.New("mail: empty body")
	}
	if m.ReplyTo == "" {
		return nil // optional
	}
	if strings.ContainsAny(m.ReplyTo, "\r\n") {
		return errors.New("mail: reply-to contains a line break")
	}
	addr, err := mail.ParseAddress(m.ReplyTo)
	if err != nil {
		return fmt.Errorf("mail: unparseable reply-to: %w", err)
	}
	// ParseAddress accepts a display name ("Name <a@b>"); only the address
	// part is wanted here, so anything decorated is refused rather than
	// silently reinterpreted.
	if addr.Name != "" || addr.Address != m.ReplyTo {
		return errors.New("mail: reply-to must be a bare address")
	}
	return nil
}

// Chain sends through the first provider that accepts the message.
type Chain struct {
	senders []Sender

	// attempt bounds each individual provider. The whole chain is still
	// bounded by the caller's context, so a slow first provider cannot eat the
	// entire budget and leave the second no time to try.
	attempt time.Duration

	// logf receives one line per failed provider. Left nil, failures are only
	// reported through the returned error.
	logf func(format string, args ...any)
}

// NewChain builds a Chain over senders, in the order they should be tried.
func NewChain(logf func(string, ...any), senders ...Sender) *Chain {
	return &Chain{senders: senders, attempt: 4 * time.Second, logf: logf}
}

// Name lists the providers in the order they are tried.
func (c *Chain) Name() string {
	names := make([]string, len(c.senders))
	for i, s := range c.senders {
		names[i] = s.Name()
	}
	return strings.Join(names, "+")
}

// Senders reports how many providers are configured.
func (c *Chain) Senders() int { return len(c.senders) }

// Recipients lists the address each provider is aimed at, so a caller can check
// that a failover would deliver to the same mailbox as the primary. A provider
// that does not report one is skipped.
func (c *Chain) Recipients() []string {
	var out []string
	for _, s := range c.senders {
		if r, ok := s.(interface{ Recipient() string }); ok {
			out = append(out, r.Recipient())
		}
	}
	return out
}

// Send tries each provider in turn and returns nil as soon as one succeeds.
//
// If every provider fails, the errors are joined so the log shows why each one
// did rather than only the last. The caller must not put that error in front of
// a visitor: it can carry provider detail that is nobody else's business.
func (c *Chain) Send(ctx context.Context, m Message) error {
	if len(c.senders) == 0 {
		return ErrNotConfigured
	}
	if err := m.Validate(); err != nil {
		return err
	}

	var failures []error
	for _, s := range c.senders {
		// A cancelled request means the visitor is gone; stop rather than
		// spending the next provider's attempt on a response nobody will read.
		if err := ctx.Err(); err != nil {
			failures = append(failures, err)
			break
		}

		attemptCtx, cancel := context.WithTimeout(ctx, c.attempt)
		err := s.Send(attemptCtx, m)
		cancel()
		if err == nil {
			return nil
		}
		if c.logf != nil {
			c.logf("mail: %s failed: %v", s.Name(), err)
		}
		failures = append(failures, fmt.Errorf("%s: %w", s.Name(), err))
	}
	return errors.Join(failures...)
}

// LogSender writes messages to a log instead of delivering them. It exists for
// local development, where there are no API keys and no wish to send real mail.
//
// It is never selected by accident: config.go only builds one when
// CONTACT_TRANSPORT is set to "log" explicitly, so a production deployment
// missing its credentials fails loudly rather than quietly dropping messages
// into a log file.
type LogSender struct {
	Logf func(format string, args ...any)
}

func (LogSender) Name() string { return "log" }

func (l LogSender) Send(_ context.Context, m Message) error {
	if err := m.Validate(); err != nil {
		return err
	}
	// %q escapes newlines and control characters, so a submitted value cannot
	// forge extra log lines or smuggle terminal escapes into whatever reads them.
	l.Logf("mail: would send subject=%q reply-to=%q body=%q", m.Subject, m.ReplyTo, m.Text)
	return nil
}

// newHTTPClient builds the client the providers share.
//
// Redirects are refused outright. Every request here carries an API
// credential, and the default client would replay it against whatever host a
// redirect named — the classic way a token ends up somewhere it was never
// meant to go. These are JSON APIs at fixed endpoints; they have no business
// redirecting.
func newHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, _ []*http.Request) error {
			return fmt.Errorf("refusing redirect to %s", req.URL.Host)
		},
	}
}

// oneLine strips control characters, so a value that reaches a header cannot
// carry a line break into it.
func oneLine(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\t' {
			return ' '
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}

// truncate shortens s to at most n bytes without splitting a rune.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !isRuneStart(s[n]) {
		n--
	}
	return s[:n]
}

func isRuneStart(b byte) bool { return b&0xC0 != 0x80 }
