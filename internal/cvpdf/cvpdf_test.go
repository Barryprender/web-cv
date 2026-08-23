package cvpdf

import (
	"bytes"
	"compress/zlib"
	"io"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"barrypre.com/webcv/internal/data"
)

// text pulls every drawn string out of a document, in drawing order, so a test
// can assert on what the PDF actually says rather than on its structure.
func text(t *testing.T, doc []byte) string {
	t.Helper()

	stream := regexp.MustCompile(`/Length (\d+) /Filter /FlateDecode >>\nstream\n`)
	show := regexp.MustCompile(`\(((?:[^()\\]|\\.)*)\) Tj`)
	unescape := strings.NewReplacer(`\(`, "(", `\)`, ")", `\\`, `\`)

	var out strings.Builder
	for _, m := range stream.FindAllSubmatchIndex(doc, -1) {
		length, err := strconv.Atoi(string(doc[m[2]:m[3]]))
		if err != nil {
			t.Fatalf("unreadable stream length: %v", err)
		}
		zr, err := zlib.NewReader(bytes.NewReader(doc[m[1] : m[1]+length]))
		if err != nil {
			t.Fatalf("stream is not valid zlib: %v", err)
		}
		body, err := io.ReadAll(zr)
		zr.Close()
		if err != nil {
			t.Fatalf("read stream: %v", err)
		}
		for _, s := range show.FindAllSubmatch(body, -1) {
			out.WriteString(unescape.Replace(string(s[1])))
			out.WriteByte('\n')
		}
	}
	return out.String()
}

// flat is text with the line breaks taken out and runs of whitespace collapsed,
// so an assertion can match a phrase the layout happened to wrap. Wrapping only
// ever breaks at a space, so rejoining with one restores the original run.
func flat(t *testing.T, doc []byte) string {
	t.Helper()
	return strings.Join(strings.Fields(text(t, doc)), " ")
}

// winAnsi renders a string the way the PDF stores it, so expectations written
// in UTF-8 can be compared against extracted text.
func winAnsi(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 0x20 && r <= 0x7E, r >= 0xA0 && r <= 0xFF:
			b.WriteByte(byte(r))
		case r == '—':
			b.WriteByte(0x97)
		case r == '–':
			b.WriteByte(0x96)
		case r == '→':
			// No WinAnsi code exists, so internal/pdf substitutes ASCII.
			b.WriteString("->")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func TestBuildIncludesEveryRole(t *testing.T) {
	got := flat(t, Build())
	for _, job := range data.Me.Jobs {
		if !strings.Contains(got, winAnsi(job.Company)) {
			t.Errorf("PDF is missing the role at %q", job.Company)
		}
		for _, bullet := range job.Bullets {
			if !strings.Contains(got, winAnsi(bullet)) {
				t.Errorf("%s: bullet missing or garbled: %q", job.Company, bullet)
			}
		}
	}
}

func TestBuildIncludesEveryProject(t *testing.T) {
	got := flat(t, Build())
	for _, p := range data.Me.Projects {
		if !strings.Contains(got, winAnsi(p.Name)) {
			t.Errorf("PDF is missing the project %q", p.Name)
		}
		if !strings.Contains(got, winAnsi(p.Summary)) {
			t.Errorf("%s: summary missing or garbled", p.Name)
		}
	}
}

func TestBuildIncludesHeaderAndContact(t *testing.T) {
	got := flat(t, Build())
	for _, want := range []string{
		data.Me.Name,
		data.Me.Headline,
		data.Me.Contact.Email,
		data.Me.Contact.Phone,
		data.Me.Contact.Location,
	} {
		if !strings.Contains(got, winAnsi(want)) {
			t.Errorf("PDF is missing %q", want)
		}
	}
}

func TestBuildIncludesSkillsEducationAndLanguages(t *testing.T) {
	got := flat(t, Build())
	for _, group := range data.Me.Skills {
		for _, skill := range group.Skills {
			if !strings.Contains(got, winAnsi(skill)) {
				t.Errorf("PDF is missing the skill %q", skill)
			}
		}
	}
	for _, e := range data.Me.Education {
		if !strings.Contains(got, winAnsi(e.Institution)) {
			t.Errorf("PDF is missing the education entry %q", e.Institution)
		}
	}
	for _, l := range data.Me.Languages {
		if !strings.Contains(got, winAnsi(l.Name)) {
			t.Errorf("PDF is missing the language %q", l.Name)
		}
	}
}

// The career break is data with Kind "break", not a job. It belongs on the CV
// as a dated entry, but nothing may present it as an employer — the same rule
// the JSON-LD follows.
func TestBuildKeepsNonEmploymentEntries(t *testing.T) {
	got := flat(t, Build())
	for _, job := range data.Me.Jobs {
		if job.IsEmployment() {
			continue
		}
		if !strings.Contains(got, winAnsi(job.Company)) {
			t.Errorf("PDF drops the non-employment entry %q", job.Company)
		}
	}
}

func TestBuildLinksEveryDestination(t *testing.T) {
	doc := Build()
	want := []string{data.Me.Contact.LinkedIn, data.Me.Contact.GitHub, data.Me.Contact.Site}
	for _, p := range data.Me.Projects {
		for _, l := range p.Links {
			want = append(want, l.URL)
		}
	}
	for _, uri := range want {
		if !bytes.Contains(doc, []byte("/URI ("+uri+")")) {
			t.Errorf("no link annotation for %s", uri)
		}
	}
	if !bytes.Contains(doc, []byte("/URI (mailto:"+data.Me.Contact.Email+")")) {
		t.Error("the email address is not a mailto link")
	}
}

func TestBuildIsDeterministic(t *testing.T) {
	// cmd/pdfgen commits its output, and TestPDFIsCurrent in internal/site
	// diffs it. Both depend on the same data producing the same bytes.
	if !bytes.Equal(Build(), Build()) {
		t.Error("two builds of the same data differ")
	}
}

func TestBuildFitsAPlausiblePageCount(t *testing.T) {
	doc := Build()
	m := regexp.MustCompile(`/Count (\d+)`).FindSubmatch(doc)
	if m == nil {
		t.Fatal("no page count in the page tree")
	}
	pages, _ := strconv.Atoi(string(m[1]))
	// A guard against the layout silently collapsing (one page could not hold
	// this much) or running away (a wrapping bug that emits a line per word).
	if pages < 2 || pages > 8 {
		t.Errorf("PDF is %d pages; expected between 2 and 8", pages)
	}
}

// Every page must carry the running footer. Without it a page that gets
// separated from the others has nothing on it saying whose CV it is.
func TestEveryPageCarriesTheFooter(t *testing.T) {
	doc := Build()
	pages := regexp.MustCompile(`/Count (\d+)`).FindSubmatch(doc)
	count, _ := strconv.Atoi(string(pages[1]))

	got := flat(t, doc)
	for i := 1; i <= count; i++ {
		if !strings.Contains(got, "Page "+strconv.Itoa(i)) {
			t.Errorf("page %d has no footer", i)
		}
	}
}

func TestTrimScheme(t *testing.T) {
	cases := map[string]string{
		"https://www.barrypre.com":                "www.barrypre.com",
		"https://github.com/Barryprender/":        "github.com/Barryprender",
		"http://example.com":                      "example.com",
		"https://www.linkedin.com/in/barrypdrgst": "www.linkedin.com/in/barrypdrgst",
	}
	for in, want := range cases {
		if got := trimScheme(in); got != want {
			t.Errorf("trimScheme(%q) = %q, want %q", in, got, want)
		}
	}
}
