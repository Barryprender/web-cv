package site

import (
	"fmt"
	"html/template"
	"log"
	"strings"
	"time"

	_ "time/tzdata" // the runtime image is distroless and carries no zoneinfo

	"barrypre.com/webcv/internal/mail"
)

// emailTemplate renders the HTML part of a contact message.
//
// html/template, not text/template: it escapes per context, so a name
// containing a tag comes out as text in the body and as an encoded parameter
// inside an href. That contextual escaping is the entire reason sending HTML
// here is safe, and it is why nothing in this file builds markup by
// concatenation. Keep it that way.
var emailTemplate = template.Must(
	template.New("email.html").ParseFS(templatesFS, "templates/email.html"))

// localTZ is where the received timestamp is rendered — the reader's own
// timezone rather than the server's UTC.
var localTZ = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		return time.UTC // embedded tzdata makes this unreachable; UTC is still correct
	}
	return loc
}()

// Field caps. The endpoint already refuses a body over 64 KB; these bound what
// is handed to an email provider, which has no business receiving all of it.
const (
	maxNameBytes    = 200
	maxEmailBytes   = 320
	maxMessageBytes = 16 << 10
	previewBytes    = 140
)

// emailData is what templates/email.html renders from.
type emailData struct {
	Subject      string
	Preview      string
	Name         string
	FirstName    string
	Email        string
	Message      string
	ReplySubject string
	SiteURL      string
	SiteHost     string
	Received     string
}

// composeContactEmail turns a submission into the mail that gets sent.
//
// Both parts are built: the HTML for reading and the plain text as the
// fallback. Sending HTML alone would leave text-only clients with nothing and
// score worse with spam filters, so the text part is not optional.
func composeContactEmail(msg contactMessage, when time.Time) mail.Message {
	var (
		name = truncate(msg.Name, maxNameBytes)
		addr = truncate(msg.Email, maxEmailBytes)
		body = truncate(msg.Message, maxMessageBytes)
	)
	subject := siteHost + " — message from " + truncate(msg.Name, 100)

	d := emailData{
		Subject:      subject,
		Preview:      preview(body),
		Name:         name,
		FirstName:    firstName(name),
		Email:        addr,
		Message:      body,
		ReplySubject: "Re: your message via " + siteHost,
		SiteURL:      siteURL,
		SiteHost:     siteHost,
		Received:     when.In(localTZ).Format("2 January 2006 at 15:04 MST"),
	}

	out := mail.Message{
		Subject: subject,
		Text:    plainTextBody(d),
		ReplyTo: addr,
	}

	// A template that fails to execute must not take the message down with it.
	// The plain-text part is already built and carries everything the HTML
	// does, so a broken render costs the styling and nothing else.
	var html strings.Builder
	if err := emailTemplate.ExecuteTemplate(&html, "email", d); err != nil {
		log.Printf("contact: rendering the HTML email failed, sending text only: %v", err)
		return out
	}
	out.HTML = html.String()
	return out
}

// plainTextBody is the fallback part, and the whole message for any client that
// does not render HTML.
func plainTextBody(d emailData) string {
	var b strings.Builder
	b.WriteString("New message from the contact form at ")
	b.WriteString(d.SiteURL)
	b.WriteString("\n\n")
	fmt.Fprintf(&b, "Name:  %s\n", d.Name)
	fmt.Fprintf(&b, "Email: %s\n", d.Email)
	fmt.Fprintf(&b, "Sent:  %s\n", d.Received)
	b.WriteString("\n---\n\n")
	b.WriteString(d.Message)
	b.WriteString("\n")
	return b.String()
}

// preview is the line the inbox list shows beside the subject. Newlines are
// flattened so it reads as one sentence rather than breaking mid-list.
func preview(message string) string {
	flat := strings.Join(strings.Fields(message), " ")
	if len(flat) <= previewBytes {
		return flat
	}
	return truncate(flat, previewBytes) + "…"
}

// firstName is what the reply button is labelled with. An empty or unusable
// name falls back to something that still reads as a sentence.
func firstName(name string) string {
	if first, _, _ := strings.Cut(strings.TrimSpace(name), " "); first != "" {
		return truncate(first, 40)
	}
	return "sender"
}
