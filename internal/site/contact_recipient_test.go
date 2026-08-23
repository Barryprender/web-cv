package site

import (
	"strings"
	"testing"

	"barrypre.com/webcv/internal/data"
	"barrypre.com/webcv/internal/mail"
)

// Where contact messages go is a fact about this deployment, not a detail of
// the mail package, so it is pinned here.
//
// cmd/server passes data.Me.Contact.Email into mail.FromEnv as the default
// recipient, which means the address on the contact page and the address the
// form delivers to are the same value and cannot drift apart. This fails if
// that address is ever changed without it being deliberate.
const contactRecipient = "barryprendergast78@gmail.com"

func TestContactMessagesGoToTheRightAddress(t *testing.T) {
	if got := data.Me.Contact.Email; got != contactRecipient {
		t.Errorf("contact address = %q, want %q\n"+
			"cmd/server passes this to mail.FromEnv as the default recipient, "+
			"so changing it changes where the contact form delivers.", got, contactRecipient)
	}
}

// The default has to survive being handed to the mail package, since an unset
// CONTACT_TO is the normal deployment.
func TestDefaultRecipientReachesTheProvider(t *testing.T) {
	t.Setenv(mail.EnvTransport, "")
	t.Setenv(mail.EnvResendKey, "re_test_key")
	t.Setenv(mail.EnvPostmarkToken, "pm_test_token")
	t.Setenv(mail.EnvFrom, "cv@barrypre.com")
	t.Setenv(mail.EnvTo, "") // unset: the default is what applies

	sender, err := mail.FromEnv(func(string, ...any) {}, data.Me.Contact.Email)
	if err != nil {
		t.Fatalf("FromEnv() = %v", err)
	}
	chain, ok := sender.(*mail.Chain)
	if !ok {
		t.Fatalf("got %T, want *mail.Chain", sender)
	}
	// Both providers must be aimed at the same mailbox; a failover that
	// delivered somewhere else would be worse than not failing over.
	for _, to := range chain.Recipients() {
		if to != contactRecipient {
			t.Errorf("a provider is addressed to %q, want %q", to, contactRecipient)
		}
	}
	if got := len(chain.Recipients()); got != 2 {
		t.Errorf("checked %d providers, want 2", got)
	}
}

// The address is a real mailbox as far as the mail package is concerned — a
// decorated or malformed one would be refused at startup.
func TestContactRecipientIsDeliverable(t *testing.T) {
	msg := mail.Message{Subject: "s", Text: "t", ReplyTo: data.Me.Contact.Email}
	if err := msg.Validate(); err != nil {
		t.Errorf("the contact address is not a bare deliverable address: %v", err)
	}
	if !strings.Contains(data.Me.Contact.Email, "@") {
		t.Error("the contact address has no domain")
	}
}
