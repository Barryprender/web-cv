package site

import (
	"os"
	"testing"
	"time"
)

// TestRenderSample writes a rendered email to disk for eyeballing. Skipped
// unless EMAIL_SAMPLE names a path.
func TestRenderSample(t *testing.T) {
	out := os.Getenv("EMAIL_SAMPLE")
	if out == "" {
		t.Skip("set EMAIL_SAMPLE to a path to write a sample")
	}
	m := composeContactEmail(contactMessage{
		Name:  "Elena Marquez",
		Email: "elena.marquez@northwind.example",
		Message: "Hi Barry,\n\nI came across your CV while looking for a senior " +
			"engineer with deep Angular experience who is also comfortable in Go. " +
			"We're rebuilding a clinical scheduling platform and the Telefy work " +
			"looks very close to our problem.\n\nWould you be open to a call next week?\n\nElena",
	}, time.Date(2026, 8, 23, 14, 32, 0, 0, time.UTC))

	if err := os.WriteFile(out, []byte(m.HTML), 0o644); err != nil {
		t.Fatalf("write sample: %v", err)
	}
	t.Logf("wrote %d bytes to %s", len(m.HTML), out)
	t.Logf("--- text part ---\n%s", m.Text)
}
