// Package pdf writes simple, text-first PDF documents using nothing but the
// standard library.
//
// It exists because the site's one hard rule is zero external dependencies,
// and a CV is the case where a hand-written PDF writer is actually reasonable:
// the document is flowed text in three weights, with rules and hyperlinks and
// no images, transparency, or embedded fonts. Everything here is PDF 1.4
// (ISO 32000-1) — a handful of indirect objects, one Flate-compressed content
// stream per page, and a cross-reference table.
//
// What it deliberately does not do: embedded font programs, colour spaces
// beyond DeviceRGB, forms, tagging, or encryption. If any of those are ever
// needed, this package is the wrong tool and a real library is the answer.
package pdf

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Standard page sizes, in points (1/72 inch).
const (
	A4Width  = 595.28
	A4Height = 841.89
)

// Font selects one of the three base-14 faces this package uses.
type Font uint8

const (
	Regular Font = iota
	Bold
	Italic
)

// Color is a DeviceRGB colour with components in [0,1].
type Color struct{ R, G, B float64 }

// Gray returns a neutral colour at the given lightness.
func Gray(v float64) Color { return Color{v, v, v} }

// Black is the default text colour.
var Black = Gray(0)

// Margin is the printable inset on each side of the page.
type Margin struct{ Top, Right, Bottom, Left float64 }

// Options configures a document. The zero value yields an A4 page with
// half-inch margins.
type Options struct {
	Width, Height float64
	Margin        Margin

	// Document information, written to the Info dictionary.
	Title, Author, Subject, Keywords string

	// Created stamps CreationDate and ModDate. Leave it zero to omit both,
	// which is what makes the output byte-for-byte reproducible — the
	// property a "regenerate and diff" check depends on.
	Created time.Time
}

// Doc is a document under construction. Build it with New, draw into it, then
// call Bytes exactly once.
type Doc struct {
	w, h   float64
	margin Margin

	info Options

	// y is the drawing cursor, measured downward from the top of the page so
	// that layout code reads top to bottom. PDF's own origin is the bottom
	// left; the conversion happens at the point of drawing and nowhere else.
	y float64

	page  bytes.Buffer // content stream of the page being drawn
	pages []page       // pages already finished
	links []annot      // link annotations on the page being drawn

	// fill tracks the current fill colour so a run of same-coloured text does
	// not re-emit the operator on every line.
	fill    Color
	hasFill bool
}

type page struct {
	content []byte
	links   []annot
}

type annot struct {
	x, y, w, h float64 // in PDF coordinates, origin bottom left
	uri        string
}

// New starts a document.
func New(o Options) *Doc {
	if o.Width == 0 {
		o.Width = A4Width
	}
	if o.Height == 0 {
		o.Height = A4Height
	}
	if o.Margin == (Margin{}) {
		o.Margin = Margin{36, 36, 36, 36}
	}
	d := &Doc{w: o.Width, h: o.Height, margin: o.Margin, info: o}
	d.y = o.Margin.Top
	return d
}

// Geometry.

// Left returns the x of the left text margin.
func (d *Doc) Left() float64 { return d.margin.Left }

// Right returns the x just past the right text margin.
func (d *Doc) Right() float64 { return d.w - d.margin.Right }

// ContentWidth is the usable width between the margins.
func (d *Doc) ContentWidth() float64 { return d.w - d.margin.Left - d.margin.Right }

// Bottom is the y, measured from the top, at which content must stop.
func (d *Doc) Bottom() float64 { return d.h - d.margin.Bottom }

// Y returns the drawing cursor, measured down from the top of the page.
func (d *Doc) Y() float64 { return d.y }

// SetY moves the drawing cursor.
func (d *Doc) SetY(y float64) { d.y = y }

// Advance moves the cursor down the page.
func (d *Doc) Advance(dy float64) { d.y += dy }

// PageCount reports how many pages exist, counting the one in progress.
func (d *Doc) PageCount() int { return len(d.pages) + 1 }

// Text metrics.

// Width returns how wide s renders at the given font and size, in points.
func Width(s string, f Font, size float64) float64 {
	widths := widthsFor(f)
	var total float64
	for _, c := range encode(s) {
		total += float64(widths[c])
	}
	return total * size / 1000
}

// Wrap breaks s into lines that each fit within maxW, splitting on spaces.
//
// A single word wider than maxW is placed on a line of its own and allowed to
// overhang rather than being hyphenated: the inputs here are CV copy, where
// the only such words are URLs, and a broken URL is worse than a long one.
func Wrap(s string, f Font, size, maxW float64) []string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return nil
	}
	var (
		lines []string
		line  strings.Builder
		width float64
	)
	spaceW := Width(" ", f, size)
	for _, word := range words {
		wordW := Width(word, f, size)
		if line.Len() > 0 && width+spaceW+wordW > maxW {
			lines = append(lines, line.String())
			line.Reset()
			width = 0
		}
		if line.Len() > 0 {
			line.WriteByte(' ')
			width += spaceW
		}
		line.WriteString(word)
		width += wordW
	}
	if line.Len() > 0 {
		lines = append(lines, line.String())
	}
	return lines
}

// Drawing.

// Text draws a single line with its baseline at y, measured down from the top
// of the page. It performs no wrapping and no page breaking.
func (d *Doc) Text(x, y float64, s string, f Font, size float64, c Color) {
	body := escapeString(encode(s))
	if len(body) == 0 {
		return
	}
	d.setFill(c)
	fmt.Fprintf(&d.page, "BT /F%d %s Tf 1 0 0 1 %s %s Tm (%s) Tj ET\n",
		f+1, num(size), num(x), num(d.h-y), body)
}

// TextLine draws s at the cursor and advances by leading. It is the common
// case: one line of a flowed block.
func (d *Doc) TextLine(x float64, s string, f Font, size, leading float64, c Color) {
	d.y += size // the cursor tracks the top of the line box; text sits on the baseline
	d.Text(x, d.y, s, f, size, c)
	d.y += leading - size
}

// Rule draws a filled horizontal hairline at the cursor.
func (d *Doc) Rule(x, width, thickness float64, c Color) {
	d.Rect(x, d.y, width, thickness, c)
}

// Rect fills a rectangle whose top edge is at y, measured down from the top.
func (d *Doc) Rect(x, y, w, h float64, c Color) {
	d.setFill(c)
	fmt.Fprintf(&d.page, "%s %s %s %s re f\n", num(x), num(d.h-y-h), num(w), num(h))
}

// Link registers a clickable region over a box whose top edge is at y.
//
// Nothing is drawn: the caller draws the visible text, and this only adds the
// annotation. Keeping the two separate means a link can cover several drawn
// lines, or none at all.
func (d *Doc) Link(x, y, w, h float64, uri string) {
	if uri == "" {
		return
	}
	d.links = append(d.links, annot{x: x, y: d.h - y - h, w: w, h: h, uri: uri})
}

// NewPage finishes the current page and starts a fresh one with the cursor at
// the top margin.
func (d *Doc) NewPage() {
	d.pages = append(d.pages, page{content: append([]byte(nil), d.page.Bytes()...), links: d.links})
	d.page.Reset()
	d.links = nil
	d.hasFill = false
	d.y = d.margin.Top
}

// setFill emits a colour operator only when the colour actually changes.
func (d *Doc) setFill(c Color) {
	if d.hasFill && d.fill == c {
		return
	}
	fmt.Fprintf(&d.page, "%s %s %s rg\n", num(c.R), num(c.G), num(c.B))
	d.fill = c
	d.hasFill = true
}

// num formats a coordinate compactly: PDF accepts plain decimals only, never
// exponent notation, which is what strconv's 'g' would produce for small
// values.
func num(v float64) string {
	s := strconv.FormatFloat(v, 'f', 2, 64)
	s = strings.TrimRight(s, "0")
	return strings.TrimSuffix(s, ".")
}

// Serialisation.

// Bytes finishes the document and returns the complete file.
func (d *Doc) Bytes() []byte {
	pages := append(append([]page(nil), d.pages...), page{content: d.page.Bytes(), links: d.links})

	// Object numbers are assigned up front so every reference can be written
	// in a single pass. 1 catalog, 2 page tree, 3 info, 4-6 fonts.
	const (
		catalogObj = 1
		pagesObj   = 2
		infoObj    = 3
		firstFont  = 4
		fixedObjs  = 6
	)

	// Each page needs its page dictionary and its content stream, plus one
	// object per link annotation.
	pageObj := make([]int, len(pages))
	contentObj := make([]int, len(pages))
	annotObj := make([][]int, len(pages))
	next := fixedObjs + 1
	for i, p := range pages {
		pageObj[i] = next
		contentObj[i] = next + 1
		next += 2
		for range p.links {
			annotObj[i] = append(annotObj[i], next)
			next++
		}
	}

	objects := make([][]byte, next-1)
	put := func(n int, format string, args ...any) {
		objects[n-1] = fmt.Appendf(nil, format, args...)
	}

	var kids strings.Builder
	for _, n := range pageObj {
		fmt.Fprintf(&kids, "%d 0 R ", n)
	}

	put(catalogObj, "<< /Type /Catalog /Pages %d 0 R >>", pagesObj)
	put(pagesObj, "<< /Type /Pages /Count %d /Kids [%s] >>", len(pages), strings.TrimSpace(kids.String()))
	objects[infoObj-1] = d.infoDict()

	for i := range 3 {
		put(firstFont+i,
			"<< /Type /Font /Subtype /Type1 /BaseFont /%s /Encoding /WinAnsiEncoding >>",
			baseFontName(Font(i)))
	}

	resources := fmt.Sprintf(
		"<< /Font << /F1 %d 0 R /F2 %d 0 R /F3 %d 0 R >> /ProcSet [/PDF /Text] >>",
		firstFont, firstFont+1, firstFont+2)

	for i, p := range pages {
		var annots strings.Builder
		for j, a := range p.links {
			n := annotObj[i][j]
			fmt.Fprintf(&annots, "%d 0 R ", n)
			put(n, "<< /Type /Annot /Subtype /Link /Rect [%s %s %s %s] "+
				"/Border [0 0 0] /F 4 /A << /S /URI /URI (%s) >> >>",
				num(a.x), num(a.y), num(a.x+a.w), num(a.y+a.h),
				escapeString(encode(a.uri)))
		}
		annotEntry := ""
		if annots.Len() > 0 {
			annotEntry = " /Annots [" + strings.TrimSpace(annots.String()) + "]"
		}
		put(pageObj[i],
			"<< /Type /Page /Parent %d 0 R /MediaBox [0 0 %s %s] /Resources %s /Contents %d 0 R%s >>",
			pagesObj, num(d.w), num(d.h), resources, contentObj[i], annotEntry)
		objects[contentObj[i]-1] = streamObject(p.content)
	}

	return assemble(objects, catalogObj, infoObj)
}

// infoDict builds the document information dictionary.
func (d *Doc) infoDict() []byte {
	var b strings.Builder
	b.WriteString("<< /Producer (barrypre.com internal/pdf)")
	entry := func(key, value string) {
		if value == "" {
			return
		}
		fmt.Fprintf(&b, " /%s (%s)", key, escapeString(encode(value)))
	}
	entry("Title", d.info.Title)
	entry("Author", d.info.Author)
	entry("Subject", d.info.Subject)
	entry("Keywords", d.info.Keywords)
	if !d.info.Created.IsZero() {
		stamp := d.info.Created.UTC().Format("20060102150405") + "Z"
		fmt.Fprintf(&b, " /CreationDate (D:%s) /ModDate (D:%s)", stamp, stamp)
	}
	b.WriteString(" >>")
	return []byte(b.String())
}

// streamObject Flate-compresses a content stream and wraps it in a stream
// object. compress/zlib is standard library, so this costs no dependency and
// roughly quarters the file.
func streamObject(content []byte) []byte {
	var compressed bytes.Buffer
	zw := zlib.NewWriter(&compressed)
	// Writing to a bytes.Buffer cannot fail, and neither can closing the
	// zlib writer over one, so there is no error path to handle here.
	_, _ = zw.Write(content)
	_ = zw.Close()

	var b bytes.Buffer
	fmt.Fprintf(&b, "<< /Length %d /Filter /FlateDecode >>\nstream\n", compressed.Len())
	b.Write(compressed.Bytes())
	b.WriteString("\nendstream")
	return b.Bytes()
}

// assemble writes the header, every numbered object, the cross-reference table
// and the trailer.
func assemble(objects [][]byte, rootObj, infoObj int) []byte {
	var out bytes.Buffer
	// The binary comment on line two marks the file as binary for tools that
	// would otherwise treat it as text and mangle line endings in transit.
	out.WriteString("%PDF-1.4\n%\xE2\xE3\xCF\xD3\n")

	offsets := make([]int, len(objects)+1)
	for i, body := range objects {
		offsets[i+1] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n", i+1)
		out.Write(body)
		out.WriteString("\nendobj\n")
	}

	xref := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n", len(objects)+1)
	out.WriteString("0000000000 65535 f \n")
	for i := 1; i <= len(objects); i++ {
		fmt.Fprintf(&out, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root %d 0 R /Info %d 0 R >>\nstartxref\n%d\n%%%%EOF\n",
		len(objects)+1, rootObj, infoObj, xref)

	return out.Bytes()
}
