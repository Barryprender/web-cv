package site

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"barrypre.com/webcv/internal/data"
)

// allPages is every route these structural checks run against.
var allPages = []string{"/", "/experience", "/projects", "/skills", "/contact"}

var (
	headingRE  = regexp.MustCompile(`(?i)<h([1-6])\b`)
	idRE       = regexp.MustCompile(`\bid="([^"]+)"`)
	idRefRE    = regexp.MustCompile(`\baria-(?:controls|labelledby|describedby)="([^"]+)"`)
	fragmentRE = regexp.MustCompile(`\bhref="#([^"]+)"`)
	inlineRE   = regexp.MustCompile(`\bstyle="`)
)

func pageBody(t *testing.T, path string) string {
	t.Helper()
	h := newTestHandler(t)
	rec := get(t, h, path)
	if rec.Code != http.StatusOK {
		t.Fatalf("%s returned %d", path, rec.Code)
	}
	return rec.Body.String()
}

func TestExactlyOneH1PerPage(t *testing.T) {
	for _, path := range allPages {
		t.Run(path, func(t *testing.T) {
			body := pageBody(t, path)
			var h1s int
			for _, m := range headingRE.FindAllStringSubmatch(body, -1) {
				if m[1] == "1" {
					h1s++
				}
			}
			if h1s != 1 {
				t.Errorf("found %d <h1> elements, want exactly 1", h1s)
			}
		})
	}
}

// A document outline that jumps h1 -> h3 leaves screen reader users unable to
// tell whether they have missed a level.
func TestHeadingLevelsNeverSkip(t *testing.T) {
	for _, path := range allPages {
		t.Run(path, func(t *testing.T) {
			body := pageBody(t, path)
			previous := 0
			for _, m := range headingRE.FindAllStringSubmatch(body, -1) {
				level, err := strconv.Atoi(m[1])
				if err != nil {
					t.Fatalf("unparseable heading level %q", m[1])
				}
				if previous != 0 && level > previous+1 {
					t.Errorf("heading jumps from h%d to h%d", previous, level)
				}
				previous = level
			}
			if previous == 0 {
				t.Error("page has no headings at all")
			}
		})
	}
}

func TestLandmarks(t *testing.T) {
	for _, path := range allPages {
		t.Run(path, func(t *testing.T) {
			body := pageBody(t, path)
			// <cv-nav> is a custom element with no implicit role, so the
			// landmark has to come from a real <nav> inside it.
			for _, want := range []string{
				`<nav class="nav-inner" aria-label="Main">`,
				`<main class="wrap" id="main">`,
				`<footer class="site-footer">`,
			} {
				if !strings.Contains(body, want) {
					t.Errorf("page is missing %s", want)
				}
			}
			if n := strings.Count(body, "<nav"); n != 1 {
				t.Errorf("found %d <nav> elements, want exactly 1", n)
			}
		})
	}
}

func TestSkipLinkIsFirstAndResolves(t *testing.T) {
	for _, path := range allPages {
		t.Run(path, func(t *testing.T) {
			body := pageBody(t, path)

			_, afterBody, found := strings.Cut(body, "<body>")
			if !found {
				t.Fatal("no <body> in the rendered page")
			}
			if !strings.HasPrefix(strings.TrimSpace(afterBody), `<a class="skip-link" href="#main">`) {
				t.Error("the skip link is not the first element in the body")
			}
			if !strings.Contains(body, `id="main"`) {
				t.Error("the skip link target #main does not exist")
			}
		})
	}
}

// Every ID a page points at — from aria-controls, aria-labelledby, or an
// in-page href — has to actually exist, or the reference is silently inert.
func TestIDReferencesResolve(t *testing.T) {
	for _, path := range allPages {
		t.Run(path, func(t *testing.T) {
			body := pageBody(t, path)

			ids := make(map[string]bool)
			for _, m := range idRE.FindAllStringSubmatch(body, -1) {
				ids[m[1]] = true
			}

			var refs []string
			for _, m := range idRefRE.FindAllStringSubmatch(body, -1) {
				refs = append(refs, strings.Fields(m[1])...) // these attributes take ID lists
			}
			for _, m := range fragmentRE.FindAllStringSubmatch(body, -1) {
				refs = append(refs, m[1])
			}
			if len(refs) == 0 {
				t.Fatal("no ID references found — the scan is not matching anything")
			}
			for _, ref := range refs {
				if !ids[ref] {
					t.Errorf("reference to #%s but no element has that id", ref)
				}
			}
		})
	}
}

func TestIDsAreUnique(t *testing.T) {
	for _, path := range allPages {
		t.Run(path, func(t *testing.T) {
			seen := make(map[string]int)
			for _, m := range idRE.FindAllStringSubmatch(pageBody(t, path), -1) {
				seen[m[1]]++
			}
			for id, n := range seen {
				if n > 1 {
					t.Errorf("id %q appears %d times, want 1", id, n)
				}
			}
		})
	}
}

// The disclosure buttons must report their state and name what they control.
func TestTimelineTogglesAreWiredUp(t *testing.T) {
	body := pageBody(t, "/experience")

	toggles := strings.Count(body, `class="timeline-toggle"`)
	if toggles == 0 {
		t.Fatal("no timeline toggles rendered")
	}
	if n := strings.Count(body, `aria-expanded="false"`); n < toggles {
		t.Errorf("%d toggles but only %d aria-expanded attributes", toggles, n)
	}

	// Only entries that have bullets get a disclosure, so the indices are not
	// contiguous: a career break has nothing to expand.
	var expected int
	for i, job := range data.Me.Jobs {
		want := fmt.Sprintf(`aria-controls="job-%d-details"`, i)
		if len(job.Bullets) == 0 {
			if strings.Contains(body, want) {
				t.Errorf("%s has no bullets but rendered a disclosure", job.Company)
			}
			continue
		}
		expected++
		if !strings.Contains(body, want) {
			t.Errorf("missing %s (%s)", want, job.Company)
		}
	}
	if toggles != expected {
		t.Errorf("rendered %d toggles, want %d", toggles, expected)
	}

	// The company name rides along in a visually-hidden span so the accessible
	// name stays distinct once JS rewrites the visible label.
	if n := strings.Count(body, `class="visually-hidden"`); n < toggles {
		t.Errorf("%d toggles but only %d visually-hidden context spans", toggles, n)
	}
}

// Inline styles would force 'unsafe-inline' back into the CSP, so they must
// never reappear in a template.
func TestNoInlineStyles(t *testing.T) {
	for _, path := range allPages {
		t.Run(path, func(t *testing.T) {
			if inlineRE.MatchString(pageBody(t, path)) {
				t.Error("page contains a style attribute")
			}
		})
	}
}

// Contact details belong in <address>, and label/value pairs are a description
// list. <address> also has to lose its UA italic, which the stylesheet does.
func TestContactUsesAddressAndDescriptionList(t *testing.T) {
	body := pageBody(t, "/contact")
	for _, want := range []string{
		`<address class="contact-details">`,
		`<dl class="contact-list">`,
		`<dt class="contact-label">email</dt>`,
		`<dd class="contact-value">`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("contact page is missing %s", want)
		}
	}
	// Derived, not hardcoded: adding a contact row should not fail this.
	rows := strings.Count(body, `class="contact-row"`)
	if rows == 0 {
		t.Fatal("no contact rows rendered")
	}
	if n := strings.Count(body, "<dt "); n != rows {
		t.Errorf("found %d <dt> elements for %d rows", n, rows)
	}
	if n := strings.Count(body, "<dd "); n != rows {
		t.Errorf("found %d <dd> elements for %d rows", n, rows)
	}
}
