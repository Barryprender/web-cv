package mail

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// resendEndpoint is the send API. It is a constant, never anything derived from
// a request, so this cannot be steered at another host.
const resendEndpoint = "https://api.resend.com/emails"

// Resend delivers through https://resend.com.
type Resend struct {
	APIKey string
	From   string // must be an address on a domain verified with Resend
	To     string

	// Endpoint overrides the API URL in tests. Empty means the real one.
	Endpoint string

	client *http.Client
}

// NewResend builds a Resend sender.
func NewResend(apiKey, from, to string) *Resend {
	return &Resend{APIKey: apiKey, From: from, To: to, client: newHTTPClient(10 * time.Second)}
}

func (*Resend) Name() string { return "resend" }

// Recipient reports where this provider delivers.
func (r *Resend) Recipient() string { return r.To }

type resendRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Text    string   `json:"text"`
	HTML    string   `json:"html,omitempty"`
	ReplyTo string   `json:"reply_to,omitempty"`
}

// resendError is the shape of a failed response. Resend returns a JSON body
// with a human-readable message on every error it produces itself.
type resendError struct {
	Message string `json:"message"`
	Name    string `json:"name"`
}

func (r *Resend) Send(ctx context.Context, m Message) error {
	if err := m.Validate(); err != nil {
		return err
	}

	// Both parts go out when the caller built both, and the client picks. The
	// text part is never omitted: it is the fallback, and an HTML-only message
	// is treated as a spam signal.
	//
	// The HTML is safe here only because whatever produced it escaped per
	// context. Never assemble this field by concatenation — see mail.Message.
	body, err := json.Marshal(resendRequest{
		From:    r.From,
		To:      []string{r.To},
		Subject: oneLine(m.Subject),
		Text:    m.Text,
		HTML:    m.HTML,
		ReplyTo: m.ReplyTo,
	})
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	endpoint := r.Endpoint
	if endpoint == "" {
		endpoint = resendEndpoint
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+r.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := r.client
	if client == nil {
		client = newHTTPClient(10 * time.Second)
	}
	resp, err := client.Do(req)
	if err != nil {
		// The URL is in the error and the key is not; net/http redacts neither
		// on its own, so nothing that could carry the credential is added here.
		return fmt.Errorf("post: %w", err)
	}
	defer resp.Body.Close()

	// Read a bounded amount: an error body is small, and a provider having a
	// very bad day should not be able to stream an unbounded response into a
	// handler that is holding a visitor's request open.
	payload, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		var detail resendError
		if json.Unmarshal(payload, &detail) == nil && detail.Message != "" {
			return fmt.Errorf("status %d: %s", resp.StatusCode, oneLine(truncate(detail.Message, 300)))
		}
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}
