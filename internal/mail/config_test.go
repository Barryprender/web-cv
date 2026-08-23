package mail

import (
	"errors"
	"strings"
	"testing"
)

// discardLogf swallows startup logging in tests that do not assert on it.
func discardLogf(string, ...any) {}

// collectLogf records what FromEnv reported.
func collectLogf(lines *[]string) func(string, ...any) {
	return func(format string, args ...any) {
		*lines = append(*lines, strings.NewReplacer("%s", "", "%d", "").Replace(format))
		_ = args
	}
}

const defaultTo = "barry@example.com"

func TestFromEnvWithNoConfiguration(t *testing.T) {
	// t.Setenv also isolates this test from the ambient environment, since it
	// forbids running in parallel.
	t.Setenv(EnvTransport, "")
	t.Setenv(EnvResendKey, "")
	t.Setenv(EnvPostmarkToken, "")
	t.Setenv(EnvFrom, "")
	t.Setenv(EnvTo, "")

	sender, err := FromEnv(discardLogf, defaultTo)
	if err != nil {
		t.Fatalf("FromEnv() = %v, want no error", err)
	}
	// Not an error at startup — but every send must fail, so nothing accepts a
	// message it cannot deliver.
	if err := sender.Send(t.Context(), validMessage()); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("Send() = %v, want ErrNotConfigured", err)
	}
}

// The log-only transport is for local development and must never be reachable
// by accident: only an explicit CONTACT_TRANSPORT=log selects it.
func TestFromEnvLogTransportIsExplicit(t *testing.T) {
	t.Setenv(EnvResendKey, "")
	t.Setenv(EnvPostmarkToken, "")
	t.Setenv(EnvFrom, "")

	t.Setenv(EnvTransport, "log")
	sender, err := FromEnv(discardLogf, defaultTo)
	if err != nil {
		t.Fatalf("FromEnv() = %v", err)
	}
	if _, ok := sender.(LogSender); !ok {
		t.Fatalf("got %T, want LogSender", sender)
	}

	// Anything else is not the log transport.
	for _, v := range []string{"", "logging", "true", "1"} {
		t.Setenv(EnvTransport, v)
		sender, err := FromEnv(discardLogf, defaultTo)
		if err != nil {
			t.Fatalf("FromEnv(%q) = %v", v, err)
		}
		if _, ok := sender.(LogSender); ok {
			t.Errorf("%s=%q selected the log transport", EnvTransport, v)
		}
	}
}

func TestFromEnvRegistersBothProviders(t *testing.T) {
	t.Setenv(EnvTransport, "")
	t.Setenv(EnvResendKey, "re_key")
	t.Setenv(EnvPostmarkToken, "pm_token")
	t.Setenv(EnvFrom, "cv@barrypre.com")
	t.Setenv(EnvTo, "")

	sender, err := FromEnv(discardLogf, defaultTo)
	if err != nil {
		t.Fatalf("FromEnv() = %v", err)
	}
	chain, ok := sender.(*Chain)
	if !ok {
		t.Fatalf("got %T, want *Chain", sender)
	}
	if got := chain.Senders(); got != 2 {
		t.Fatalf("registered %d providers, want 2", got)
	}
	// Resend first, then Postmark: the order messages are attempted in.
	if got, want := chain.Name(), "resend+postmark"; got != want {
		t.Errorf("Name() = %q, want %q", got, want)
	}
}

func TestFromEnvRegistersOneProvider(t *testing.T) {
	tests := []struct {
		name         string
		resend, post string
		want         string
	}{
		{"resend only", "re_key", "", "resend"},
		{"postmark only", "", "pm_token", "postmark"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(EnvTransport, "")
			t.Setenv(EnvResendKey, tc.resend)
			t.Setenv(EnvPostmarkToken, tc.post)
			t.Setenv(EnvFrom, "cv@barrypre.com")
			t.Setenv(EnvTo, "")

			sender, err := FromEnv(discardLogf, defaultTo)
			if err != nil {
				t.Fatalf("FromEnv() = %v", err)
			}
			chain := sender.(*Chain)
			if got := chain.Name(); got != tc.want {
				t.Errorf("Name() = %q, want %q", got, tc.want)
			}
		})
	}
}

// Credentials without a sender address is a deployment that would fail on its
// first real submission. Better to refuse to start.
func TestFromEnvRequiresFrom(t *testing.T) {
	t.Setenv(EnvTransport, "")
	t.Setenv(EnvResendKey, "re_key")
	t.Setenv(EnvPostmarkToken, "")
	t.Setenv(EnvFrom, "")
	t.Setenv(EnvTo, "")

	_, err := FromEnv(discardLogf, defaultTo)
	if err == nil {
		t.Fatal("FromEnv() = nil, want an error when the sender address is missing")
	}
	if !strings.Contains(err.Error(), EnvFrom) {
		t.Errorf("error %q does not name %s", err, EnvFrom)
	}
	// It must name which credential it found, and never the value.
	if !strings.Contains(err.Error(), EnvResendKey) {
		t.Errorf("error %q does not say which credential was set", err)
	}
	if strings.Contains(err.Error(), "re_key") {
		t.Errorf("error leaks the credential: %q", err)
	}
}

func TestFromEnvRejectsAMalformedRecipient(t *testing.T) {
	t.Setenv(EnvTransport, "")
	t.Setenv(EnvResendKey, "re_key")
	t.Setenv(EnvPostmarkToken, "")
	t.Setenv(EnvFrom, "cv@barrypre.com")
	t.Setenv(EnvTo, "Barry <barry@example.com>")

	if _, err := FromEnv(discardLogf, defaultTo); err == nil {
		t.Error("FromEnv() = nil, want a decorated recipient rejected")
	}
}

func TestFromEnvUsesTheDefaultRecipient(t *testing.T) {
	t.Setenv(EnvTransport, "")
	t.Setenv(EnvResendKey, "re_key")
	t.Setenv(EnvPostmarkToken, "")
	t.Setenv(EnvFrom, "cv@barrypre.com")
	t.Setenv(EnvTo, "")

	sender, err := FromEnv(discardLogf, defaultTo)
	if err != nil {
		t.Fatalf("FromEnv() = %v", err)
	}
	resend := sender.(*Chain).senders[0].(*Resend)
	if resend.To != defaultTo {
		t.Errorf("To = %q, want the default %q", resend.To, defaultTo)
	}
}

// An unconfigured deployment has to say so at startup, since the symptom
// otherwise only appears when a visitor's message is refused.
func TestFromEnvWarnsWhenUnconfigured(t *testing.T) {
	t.Setenv(EnvTransport, "")
	t.Setenv(EnvResendKey, "")
	t.Setenv(EnvPostmarkToken, "")
	t.Setenv(EnvFrom, "")

	var lines []string
	if _, err := FromEnv(collectLogf(&lines), defaultTo); err != nil {
		t.Fatalf("FromEnv() = %v", err)
	}
	if len(lines) == 0 {
		t.Fatal("FromEnv logged nothing about being unconfigured")
	}
	if !strings.Contains(lines[0], "no provider configured") {
		t.Errorf("startup log does not warn clearly: %q", lines[0])
	}
}

// Credentials must never reach a log line, not even the configured one.
func TestFromEnvDoesNotLogCredentials(t *testing.T) {
	t.Setenv(EnvTransport, "")
	t.Setenv(EnvResendKey, "re_super_secret")
	t.Setenv(EnvPostmarkToken, "pm_super_secret")
	t.Setenv(EnvFrom, "cv@barrypre.com")
	t.Setenv(EnvTo, "")

	var logged strings.Builder
	logf := func(format string, args ...any) {
		logged.WriteString(format)
		for _, a := range args {
			if s, ok := a.(string); ok {
				logged.WriteString(s)
			}
		}
	}
	if _, err := FromEnv(logf, defaultTo); err != nil {
		t.Fatalf("FromEnv() = %v", err)
	}
	for _, secret := range []string{"re_super_secret", "pm_super_secret"} {
		if strings.Contains(logged.String(), secret) {
			t.Errorf("startup log leaks %q:\n%s", secret, logged.String())
		}
	}
}

// Whitespace around a value pasted into a secrets manager must not make a
// configured provider look unconfigured.
func TestFromEnvTrimsWhitespace(t *testing.T) {
	t.Setenv(EnvTransport, "")
	t.Setenv(EnvResendKey, "  re_key\n")
	t.Setenv(EnvPostmarkToken, "")
	t.Setenv(EnvFrom, "  cv@barrypre.com  ")
	t.Setenv(EnvTo, "")

	sender, err := FromEnv(discardLogf, defaultTo)
	if err != nil {
		t.Fatalf("FromEnv() = %v", err)
	}
	resend := sender.(*Chain).senders[0].(*Resend)
	if resend.APIKey != "re_key" {
		t.Errorf("APIKey = %q, want it trimmed", resend.APIKey)
	}
	if resend.From != "cv@barrypre.com" {
		t.Errorf("From = %q, want it trimmed", resend.From)
	}
}
