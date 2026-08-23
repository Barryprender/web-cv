package mail

import (
	"fmt"
	"os"
	"strings"
)

// Environment variables read by FromEnv.
const (
	EnvResendKey     = "RESEND_API_KEY"
	EnvPostmarkToken = "POSTMARK_SERVER_TOKEN"
	EnvFrom          = "CONTACT_FROM"
	EnvTo            = "CONTACT_TO"
	EnvTransport     = "CONTACT_TRANSPORT"
)

// FromEnv builds the contact form's sender from the environment.
//
// Providers are tried in the order they are registered here: Resend first, then
// Postmark. Configure both and a message only fails once both have refused it.
//
// The rules, in order:
//
//   - CONTACT_TRANSPORT=log short-circuits everything and logs instead of
//     sending. This is for local development, and it is deliberately explicit:
//     a deployment that forgets its API keys must not silently fall into it.
//   - Otherwise every provider with credentials is registered. CONTACT_FROM is
//     required, since neither provider will send without a verified sender, and
//     CONTACT_TO defaults to defaultTo.
//   - With no credentials at all, the returned Sender fails every send with
//     ErrNotConfigured. Nothing accepts a message it cannot deliver.
//
// The returned error describes a misconfiguration worth refusing to start over
// — credentials present but no sender address, say — rather than the ordinary
// case of nothing being configured.
func FromEnv(logf func(string, ...any), defaultTo string) (Sender, error) {
	if strings.EqualFold(os.Getenv(EnvTransport), "log") {
		logf("mail: %s=log — contact messages will be logged, not sent", EnvTransport)
		return LogSender{Logf: logf}, nil
	}

	var (
		resendKey   = strings.TrimSpace(os.Getenv(EnvResendKey))
		postmarkTok = strings.TrimSpace(os.Getenv(EnvPostmarkToken))
		from        = strings.TrimSpace(os.Getenv(EnvFrom))
		to          = strings.TrimSpace(os.Getenv(EnvTo))
		configured  = resendKey != "" || postmarkTok != ""
	)
	if to == "" {
		to = defaultTo
	}

	if !configured {
		logf("mail: no provider configured (%s / %s unset) — the contact form will "+
			"report a delivery failure to visitors. Set %s=log for local development.",
			EnvResendKey, EnvPostmarkToken, EnvTransport)
		return NewChain(logf), nil // an empty chain fails every send with ErrNotConfigured
	}

	// A misconfigured address is caught here rather than on the first
	// submission, so it surfaces at deploy time instead of losing a message.
	if from == "" {
		return nil, fmt.Errorf("mail: %s is set but %s is empty", credentialNames(resendKey, postmarkTok), EnvFrom)
	}
	if err := (Message{Subject: "x", Text: "x", ReplyTo: to}).Validate(); err != nil {
		return nil, fmt.Errorf("mail: %s is not a bare email address: %w", EnvTo, err)
	}

	var senders []Sender
	if resendKey != "" {
		senders = append(senders, NewResend(resendKey, from, to))
	}
	if postmarkTok != "" {
		senders = append(senders, NewPostmark(postmarkTok, from, to))
	}

	chain := NewChain(logf, senders...)
	logf("mail: sending contact messages via %s to %s", chain.Name(), to)
	return chain, nil
}

// credentialNames lists which credentials were supplied, for an error message
// that says what was actually found. The values themselves are never included.
func credentialNames(resendKey, postmarkToken string) string {
	var names []string
	if resendKey != "" {
		names = append(names, EnvResendKey)
	}
	if postmarkToken != "" {
		names = append(names, EnvPostmarkToken)
	}
	return strings.Join(names, " and ")
}
