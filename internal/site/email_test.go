package site

import (
	"html"
	"strings"
	"testing"
	"time"
)

func sampleTime() time.Time {
	return time.Date(2026, 8, 23, 14, 32, 0, 0, time.UTC)
}

func compose(t *testing.T, name, email, message string) (text, htmlPart string) {
	t.Helper()
	m := composeContactEmail(contactMessage{Name: name, Email: email, Message: message}, sampleTime())
	if m.Text == "" {
		t.Fatal("no plain-text part was built")
	}
	return m.Text, m.HTML
}

// Both parts, always. HTML alone leaves text-only clients with an empty message
// and reads as a spam signal to filters.
func TestEmailIsMultipart(t *testing.T) {
	text, htmlPart := compose(t, "Ada Lovelace", "ada@example.com", "Hello there.")

	if !strings.Contains(text, "Hello there.") {
		t.Errorf("text part is missing the message:\n%s", text)
	}
	if !strings.HasPrefix(strings.TrimSpace(htmlPart), "<!doctype html>") {
		t.Errorf("HTML part is not a document:\n%s", htmlPart[:min(120, len(htmlPart))])
	}
	for _, want := range []string{"Ada Lovelace", "ada@example.com", "Hello there."} {
		if !strings.Contains(htmlPart, want) {
			t.Errorf("HTML part is missing %q", want)
		}
	}
}

// The reason sending HTML is safe at all. If this ever fails, stop shipping.
func TestEmailEscapesInjectedMarkup(t *testing.T) {
	const (
		evilName = `<script>alert(1)</script>`
		evilBody = `<img src=x onerror="alert(2)"> and </div><script>alert(3)</script>`
	)
	_, htmlPart := compose(t, evilName, "ada@example.com", evilBody)

	// No executable markup survives anywhere in the document.
	for _, forbidden := range []string{
		"<script>alert(1)</script>",
		"<img src=x onerror",
		"<script>alert(3)</script>",
	} {
		if strings.Contains(htmlPart, forbidden) {
			t.Errorf("unescaped markup in the HTML part: %q", forbidden)
		}
	}
	// It is present, just inert.
	if !strings.Contains(htmlPart, "&lt;script&gt;") {
		t.Errorf("the injected tag was dropped rather than escaped:\n%s", htmlPart)
	}
	// And it round-trips: the reader sees exactly what was typed.
	if got := html.UnescapeString(htmlPart); !strings.Contains(got, evilName) {
		t.Error("escaping altered the content the reader sees")
	}
}

// The address lands in an href, which is a different escaping context from the
// body — a quote breaking out there would be an injection the body test misses.
func TestEmailEscapesTheMailtoLink(t *testing.T) {
	// Deliberately not a valid address; validEmail rejects this before the
	// handler ever composes, so this checks the template's own defence.
	_, htmlPart := compose(t, "Ada", `a@b.com" onmouseover="alert(1)`, "hi")

	if strings.Contains(htmlPart, `onmouseover="alert(1)"`) {
		t.Errorf("the attribute context was escaped out of:\n%s", htmlPart)
	}
	if strings.Contains(htmlPart, `href="mailto:a@b.com" o`) {
		t.Error("the href attribute was terminated early by the input")
	}
}

// Newlines have to survive without the template rewriting the string, since
// editing text after escaping is how injections get reintroduced.
func TestEmailPreservesNewlinesWithoutRewriting(t *testing.T) {
	_, htmlPart := compose(t, "Ada", "ada@example.com", "line one\nline two\n\nline four")

	if !strings.Contains(htmlPart, "white-space:pre-wrap") {
		t.Error("the message block does not preserve whitespace")
	}
	// No manual newline-to-<br> substitution.
	if strings.Contains(htmlPart, "line one<br>") {
		t.Error("newlines were rewritten as markup rather than preserved by CSS")
	}
	if !strings.Contains(htmlPart, "line one\nline two") {
		t.Error("the message's own line breaks were lost")
	}
}

func TestEmailSubjectAndReplyTo(t *testing.T) {
	m := composeContactEmail(
		contactMessage{Name: "Ada Lovelace", Email: "ada@example.com", Message: "hi"}, sampleTime())

	if !strings.Contains(m.Subject, siteHost) {
		t.Errorf("Subject = %q, want the site host in it", m.Subject)
	}
	if !strings.Contains(m.Subject, "Ada Lovelace") {
		t.Errorf("Subject = %q, want the sender's name", m.Subject)
	}
	if m.ReplyTo != "ada@example.com" {
		t.Errorf("ReplyTo = %q", m.ReplyTo)
	}
	// A subject cannot carry a line break into a header.
	if strings.ContainsAny(m.Subject, "\r\n") {
		t.Error("subject contains a line break")
	}
}

func TestEmailReplyButtonUsesTheFirstName(t *testing.T) {
	_, htmlPart := compose(t, "Ada Lovelace", "ada@example.com", "hi")
	if !strings.Contains(htmlPart, "reply to Ada") {
		t.Errorf("reply button is not labelled with the first name:\n%s", htmlPart)
	}
}

func TestFirstName(t *testing.T) {
	cases := map[string]string{
		"Ada Lovelace":     "Ada",
		"Ada":              "Ada",
		"  Ada  Lovelace ": "Ada",
		"":                 "sender",
		"   ":              "sender",
	}
	for in, want := range cases {
		if got := firstName(in); got != want {
			t.Errorf("firstName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPreviewIsOneFlatLine(t *testing.T) {
	got := preview("line one\nline two\t\tspaced")
	if strings.ContainsAny(got, "\n\t") {
		t.Errorf("preview carries whitespace control characters: %q", got)
	}
	if got != "line one line two spaced" {
		t.Errorf("preview = %q", got)
	}

	long := preview(strings.Repeat("x", 500))
	if len([]rune(long)) > previewBytes+1 {
		t.Errorf("preview is %d runes, want it truncated", len([]rune(long)))
	}
	if !strings.HasSuffix(long, "…") {
		t.Error("a truncated preview should show an ellipsis")
	}
}

// The endpoint accepts 64 KB; a provider has no business receiving all of it.
func TestEmailTruncatesLongFields(t *testing.T) {
	m := composeContactEmail(contactMessage{
		Name:    strings.Repeat("n", 1000),
		Email:   "ada@example.com",
		Message: strings.Repeat("m", 40_000),
	}, sampleTime())

	if len(m.Text) > maxMessageBytes+1000 {
		t.Errorf("text part is %d bytes, want the message truncated", len(m.Text))
	}
	if strings.Count(m.HTML, "m") > maxMessageBytes+2000 {
		t.Error("the HTML part did not truncate the message")
	}
}

// The timestamp is rendered in the reader's timezone, which needs the embedded
// tzdata: distroless carries no zoneinfo, so a missing import would silently
// fall back to UTC in production and nowhere else.
func TestEmailTimestampIsLocalised(t *testing.T) {
	if localTZ == time.UTC {
		t.Fatal("Europe/Madrid did not load; the time/tzdata import is missing")
	}
	m := composeContactEmail(
		contactMessage{Name: "Ada", Email: "a@b.com", Message: "hi"}, sampleTime())

	// 14:32 UTC in August is 16:32 in Madrid (CEST).
	if !strings.Contains(m.Text, "16:32") {
		t.Errorf("timestamp not converted to Madrid time:\n%s", m.Text)
	}
	if !strings.Contains(m.Text, "23 August 2026") {
		t.Errorf("date is missing or misformatted:\n%s", m.Text)
	}
}

// Every colour must be inline as well as in the <style> block: Gmail strips
// much of the block, and a colour that lives only there renders as the client's
// default.
func TestEmailStylesAreInline(t *testing.T) {
	_, htmlPart := compose(t, "Ada", "ada@example.com", "hi")

	for _, token := range []string{"#FBF8F4", "#142A41", "#1D7D3E", "#D4D8DD"} {
		if !strings.Contains(htmlPart, "style=") || !strings.Contains(htmlPart, token) {
			t.Errorf("palette colour %s is not present inline", token)
		}
	}
	// No external stylesheet: mail clients do not fetch them.
	if strings.Contains(htmlPart, "<link") {
		t.Error("the email links an external stylesheet")
	}
	// Table layout, since Outlook renders through Word.
	if !strings.Contains(htmlPart, `role="presentation"`) {
		t.Error("layout is not table-based")
	}
}
