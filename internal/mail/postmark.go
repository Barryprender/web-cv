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

// postmarkEndpoint is the send API, a constant for the same reason Resend's is.
const postmarkEndpoint = "https://api.postmarkapp.com/email"

// Postmark delivers through https://postmarkapp.com.
type Postmark struct {
	ServerToken string
	From        string // must be a confirmed Sender Signature
	To          string

	// Stream is the message stream to send on. Postmark defaults to
	// "outbound"; a contact form is transactional, so that is the right one.
	Stream string

	// Endpoint overrides the API URL in tests. Empty means the real one.
	Endpoint string

	client *http.Client
}

// NewPostmark builds a Postmark sender.
func NewPostmark(token, from, to string) *Postmark {
	return &Postmark{
		ServerToken: token,
		From:        from,
		To:          to,
		Stream:      "outbound",
		client:      newHTTPClient(10 * time.Second),
	}
}

func (*Postmark) Name() string { return "postmark" }

// Recipient reports where this provider delivers.
func (p *Postmark) Recipient() string { return p.To }

// Field names are capitalised because Postmark's API is case-sensitive and
// documents them that way.
type postmarkRequest struct {
	From          string `json:"From"`
	To            string `json:"To"`
	Subject       string `json:"Subject"`
	TextBody      string `json:"TextBody"`
	HTMLBody      string `json:"HtmlBody,omitempty"`
	ReplyTo       string `json:"ReplyTo,omitempty"`
	MessageStream string `json:"MessageStream,omitempty"`
}

// postmarkResponse is returned on both success and failure. ErrorCode is 0 on
// success and non-zero otherwise.
type postmarkResponse struct {
	ErrorCode int    `json:"ErrorCode"`
	Message   string `json:"Message"`
}

func (p *Postmark) Send(ctx context.Context, m Message) error {
	if err := m.Validate(); err != nil {
		return err
	}

	stream := p.Stream
	if stream == "" {
		stream = "outbound"
	}

	// Both parts when both were built: see the note in resend.go.
	body, err := json.Marshal(postmarkRequest{
		From:          p.From,
		To:            p.To,
		Subject:       oneLine(m.Subject),
		TextBody:      m.Text,
		HTMLBody:      m.HTML,
		ReplyTo:       m.ReplyTo,
		MessageStream: stream,
	})
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	endpoint := p.Endpoint
	if endpoint == "" {
		endpoint = postmarkEndpoint
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("X-Postmark-Server-Token", p.ServerToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := p.client
	if client == nil {
		client = newHTTPClient(10 * time.Second)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("post: %w", err)
	}
	defer resp.Body.Close()

	payload, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))

	var detail postmarkResponse
	decoded := json.Unmarshal(payload, &detail) == nil

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		if decoded && detail.Message != "" {
			return fmt.Errorf("status %d (code %d): %s",
				resp.StatusCode, detail.ErrorCode, oneLine(truncate(detail.Message, 300)))
		}
		return fmt.Errorf("status %d", resp.StatusCode)
	}

	// Postmark can answer 200 with a non-zero ErrorCode — an inactive
	// recipient, for one. Treating the status alone as success would report a
	// message delivered that never went anywhere.
	if decoded && detail.ErrorCode != 0 {
		return fmt.Errorf("error code %d: %s", detail.ErrorCode, oneLine(truncate(detail.Message, 300)))
	}
	return nil
}
