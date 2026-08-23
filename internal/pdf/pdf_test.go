package pdf

import (
	"bytes"
	"compress/zlib"
	"io"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestWidthUsesFontMetrics(t *testing.T) {
	// "iii" and "mmm" are the extremes of Helvetica's proportional widths;
	// if they measure the same, the metrics table is not being consulted.
	narrow := Width("iii", Regular, 10)
	wide := Width("mmm", Regular, 10)
	if narrow >= wide {
		t.Fatalf("expected iii (%v) narrower than mmm (%v)", narrow, wide)
	}
	// 'i' is 222/1000 em in Helvetica.
	if got, want := Width("i", Regular, 1000), 222.0; got != want {
		t.Errorf("Width(i) = %v, want %v", got, want)
	}
	// Bold is wider than regular for the same string.
	if Width("Barry", Bold, 10) <= Width("Barry", Regular, 10) {
		t.Error("bold should measure wider than regular")
	}
	// Oblique shares Helvetica's metrics.
	if Width("Barry", Italic, 10) != Width("Barry", Regular, 10) {
		t.Error("oblique should be metrically identical to regular")
	}
}

func TestWidthScalesWithSize(t *testing.T) {
	if got, want := Width("Barry", Regular, 20), Width("Barry", Regular, 10)*2; got != want {
		t.Errorf("width at 20pt = %v, want %v", got, want)
	}
}

func TestWrapFitsWithinWidth(t *testing.T) {
	const (
		size = 10.0
		maxW = 120.0
	)
	text := "Architected a complete Angular application from the ground up with an admin dashboard"
	lines := Wrap(text, Regular, size, maxW)
	if len(lines) < 2 {
		t.Fatalf("expected the text to wrap, got %d line(s)", len(lines))
	}
	for _, line := range lines {
		// A line may only exceed the measure when it is a single unbreakable word.
		if w := Width(line, Regular, size); w > maxW && len(strings.Fields(line)) > 1 {
			t.Errorf("line %q measures %v, over the %v limit", line, w, maxW)
		}
	}
	if joined := strings.Join(lines, " "); joined != text {
		t.Errorf("wrapping changed the text:\n got %q\nwant %q", joined, text)
	}
}

func TestWrapKeepsLongWordsIntact(t *testing.T) {
	const url = "https://www.npmjs.com/package/secure-ui-components"
	lines := Wrap(url, Regular, 10, 20)
	if len(lines) != 1 || lines[0] != url {
		t.Errorf("a word wider than the measure should stay whole, got %q", lines)
	}
}

func TestWrapEmpty(t *testing.T) {
	if lines := Wrap("   ", Regular, 10, 100); lines != nil {
		t.Errorf("blank input should wrap to nothing, got %q", lines)
	}
}

func TestEncodeMapsTypographicCharacters(t *testing.T) {
	cases := map[string][]byte{
		"a—b":   {'a', 0x97, 'b'},           // em dash
		"v1–20": {'v', '1', 0x96, '2', '0'}, // en dash
		"a·b":   {'a', 0xB7, 'b'},           // middle dot, Latin-1 passthrough
		"ó":     {0xF3},                     // Maquetación
		"“q”":   {0x93, 'q', 0x94},          // curly quotes
	}
	for in, want := range cases {
		if got := encode(in); !bytes.Equal(got, want) {
			t.Errorf("encode(%q) = % x, want % x", in, got, want)
		}
	}
}

func TestEncodeFallsBackVisibly(t *testing.T) {
	// An arrow has no WinAnsi code but an obvious plain-text stand-in.
	if got, want := string(encode("a→b")), "a->b"; got != want {
		t.Errorf("encode arrow = %q, want %q", got, want)
	}
	// Anything with neither becomes a visible '?' rather than disappearing.
	if got, want := string(encode("日")), "?"; got != want {
		t.Errorf("encode CJK = %q, want %q", got, want)
	}
}

func TestEscapeStringEscapesStructuralBytes(t *testing.T) {
	got := string(escapeString([]byte(`a(b)c\d`)))
	want := `a\(b\)c\\d`
	if got != want {
		t.Errorf("escapeString = %q, want %q", got, want)
	}
	// A string needing no escape is returned untouched.
	plain := []byte("plain text")
	if got := escapeString(plain); &got[0] != &plain[0] {
		t.Error("escapeString should not copy when nothing needs escaping")
	}
}

// A literal string carrying an unescaped paren would end the string early and
// corrupt every object after it, so this is the one encoding bug that must not
// survive.
func TestTextWithParensStaysBalanced(t *testing.T) {
	d := New(Options{})
	d.Text(50, 50, "Angular (v1-20)", Regular, 10, Black)
	content := firstStream(t, d.Bytes())
	if !strings.Contains(content, `(Angular \(v1-20\)) Tj`) {
		t.Errorf("parens not escaped in content stream:\n%s", content)
	}
}

func TestBytesProducesValidStructure(t *testing.T) {
	d := New(Options{Title: "CV"})
	d.TextLine(50, "page one", Regular, 10, 14, Black)
	d.NewPage()
	d.TextLine(50, "page two", Regular, 10, 14, Black)

	out := d.Bytes()

	if !bytes.HasPrefix(out, []byte("%PDF-1.4\n")) {
		t.Error("missing PDF header")
	}
	if !bytes.HasSuffix(out, []byte("%%EOF\n")) {
		t.Error("missing EOF marker")
	}
	if !bytes.Contains(out, []byte("/Count 2")) {
		t.Error("page tree does not report two pages")
	}
	assertXrefResolves(t, out)
}

// assertXrefResolves checks that every cross-reference offset lands on the
// object header it claims to. A reader that cannot resolve the table treats
// the whole file as damaged, and nothing else in the document matters.
func assertXrefResolves(t *testing.T, doc []byte) {
	t.Helper()

	start := regexp.MustCompile(`startxref\s+(\d+)\s+%%EOF`).FindSubmatch(doc)
	if start == nil {
		t.Fatal("no startxref pointer")
	}
	offset, err := strconv.Atoi(string(start[1]))
	if err != nil || offset <= 0 || offset >= len(doc) {
		t.Fatalf("startxref points outside the file: %q", start[1])
	}

	header := regexp.MustCompile(`^xref\s+0 (\d+)\s+`).FindSubmatch(doc[offset:])
	if header == nil {
		t.Fatalf("startxref does not point at an xref table, found %q", doc[offset:min(offset+16, len(doc))])
	}
	count, _ := strconv.Atoi(string(header[1]))

	entries := doc[offset+len(header[0]):]
	for i := 1; i < count; i++ {
		entry := entries[i*20 : (i+1)*20]
		at, err := strconv.Atoi(strings.TrimSpace(string(entry[:10])))
		if err != nil {
			t.Fatalf("object %d has an unreadable offset %q", i, entry[:10])
		}
		want := []byte(strconv.Itoa(i) + " 0 obj")
		if !bytes.HasPrefix(doc[at:], want) {
			t.Errorf("object %d: offset %d points at %q, want %q", i, at, doc[at:min(at+16, len(doc))], want)
		}
	}
}

func TestLinkBecomesAnnotation(t *testing.T) {
	d := New(Options{})
	d.Link(50, 100, 80, 12, "https://example.com/a(b)")
	out := d.Bytes()

	if !bytes.Contains(out, []byte("/Subtype /Link")) {
		t.Error("no link annotation emitted")
	}
	if !bytes.Contains(out, []byte(`/URI (https://example.com/a\(b\))`)) {
		t.Errorf("link URI missing or unescaped:\n%s", out)
	}
	if !bytes.Contains(out, []byte("/Annots [")) {
		t.Error("page does not reference its annotations")
	}
}

func TestLinkIgnoresEmptyURI(t *testing.T) {
	d := New(Options{})
	d.Link(50, 100, 80, 12, "")
	if bytes.Contains(d.Bytes(), []byte("/Subtype /Link")) {
		t.Error("an empty URI should produce no annotation")
	}
}

func TestOutputIsDeterministic(t *testing.T) {
	build := func() []byte {
		d := New(Options{Title: "CV", Author: "Barry Prendergast"})
		d.TextLine(50, "same input", Regular, 10, 14, Black)
		return d.Bytes()
	}
	if !bytes.Equal(build(), build()) {
		t.Error("identical input produced different bytes; the generated file cannot be diffed")
	}
}

func TestCreatedStampsDates(t *testing.T) {
	d := New(Options{Created: time.Date(2026, 8, 22, 10, 30, 0, 0, time.UTC)})
	out := d.Bytes()
	if !bytes.Contains(out, []byte("/CreationDate (D:20260822103000Z)")) {
		t.Errorf("creation date missing:\n%s", out)
	}

	// Left zero, no date is written at all — that is what keeps the output
	// reproducible.
	if bytes.Contains(New(Options{}).Bytes(), []byte("/CreationDate")) {
		t.Error("zero Created should write no date")
	}
}

func TestNumAvoidsExponentNotation(t *testing.T) {
	// PDF accepts plain decimals only. strconv's 'g' would render this as
	// 1e-07 and every reader would reject the operand.
	if got := num(0.0000001); strings.ContainsAny(got, "eE") {
		t.Errorf("num produced exponent notation: %q", got)
	}
	if got, want := num(48.0), "48"; got != want {
		t.Errorf("num(48) = %q, want %q", got, want)
	}
	if got, want := num(10.50), "10.5"; got != want {
		t.Errorf("num(10.50) = %q, want %q", got, want)
	}
}

func TestFillColorEmittedOnce(t *testing.T) {
	d := New(Options{})
	for range 3 {
		d.TextLine(50, "same colour", Regular, 10, 14, Black)
	}
	if got := strings.Count(firstStream(t, d.Bytes()), " rg"); got != 1 {
		t.Errorf("colour operator emitted %d times for one colour, want 1", got)
	}
}

// firstStream decompresses the first content stream in a document.
func firstStream(t *testing.T, doc []byte) string {
	t.Helper()
	m := regexp.MustCompile(`/Length (\d+) /Filter /FlateDecode >>\nstream\n`).FindSubmatchIndex(doc)
	if m == nil {
		t.Fatal("no content stream found")
	}
	length, _ := strconv.Atoi(string(doc[m[2]:m[3]]))
	zr, err := zlib.NewReader(bytes.NewReader(doc[m[1] : m[1]+length]))
	if err != nil {
		t.Fatalf("content stream is not valid zlib: %v", err)
	}
	defer zr.Close()
	body, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("read content stream: %v", err)
	}
	return string(body)
}
