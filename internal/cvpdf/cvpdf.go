// Package cvpdf lays Barry Prendergast's CV out as a printable PDF.
//
// It reads internal/data directly, so the PDF cannot say anything the site
// does not: the same Go values feed the templates, the JSON-LD and this. The
// output is generated at build time by cmd/pdfgen and committed to
// internal/site/static, which keeps the running server free of any PDF work —
// it just serves bytes out of the embedded filesystem.
package cvpdf

import (
	"fmt"
	"strings"

	"barrypre.com/webcv/internal/data"
	"barrypre.com/webcv/internal/pdf"
)

// Page geometry. The metrics column on the left holds dates and locations; the
// body column holds everything else, mirroring the timeline layout on the site.
const (
	marginX = 48.0
	marginY = 42.0

	metaWidth = 88.0 // dates and location
	metaGap   = 16.0
	bodyX     = marginX + metaWidth + metaGap

	footerHeight = 22.0 // space reserved below the content for the running footer
)

// Editorial limits. The site is the full record and scrolls for free; the PDF
// is the two-page version a recruiter reads in half a minute, so it renders a
// subset of the same data rather than a different set of facts.
//
// Both numbers are tuned against page count. TestFitsTwoPages fails if a change
// to the CV pushes the document past two pages, which is the signal to cut
// copy rather than to raise these.
const (
	detailedRoles   = 2 // employers whose bullets and stack are printed
	jobBullets      = 3 // bullets printed per detailed role
	projectBullets  = 2 // bullets printed per project
	detailedProject = 3 // projects printed in full before the rest go to one line
)

// Type sizes and leadings, in points.
const (
	nameSize     = 21.0
	headlineSize = 10.5
	taglineSize  = 9.5
	contactSize  = 8.5

	sectionSize = 9.0
	companySize = 10.5
	roleSize    = 9.5
	metaSize    = 8.0
	stackSize   = 7.8
)

// copyMetrics are the sizes of the flowed body text. They are per-language
// because Spanish runs about a fifth longer than English for the same meaning,
// and the alternative to setting it slightly tighter is printing fewer of the
// achievements in Spanish than in English.
//
// Content parity is the thing worth protecting: a reader sees one document, and
// half a point of body size is invisible to them. A CV that silently says less
// in their language is not.
type copyMetrics struct {
	summary float64
	bullet  float64
	leading float64 // added to bullet size for the line advance
}

func metricsFor(l data.Lang) copyMetrics {
	if l == data.ES {
		return copyMetrics{summary: 8.3, bullet: 7.9, leading: 2.4}
	}
	return copyMetrics{summary: 9.0, bullet: 8.6, leading: 2.8}
}

// Section headings and the running footer. These are the document's own words
// rather than the CV's, so they live here and not in internal/data.
var (
	secExperience = data.T{EN: "Experience", ES: "Experiencia"}
	secProjects   = data.T{EN: "Selected projects", ES: "Proyectos destacados"}
	secSkills     = data.T{EN: "Skills", ES: "Competencias"}
	secEducation  = data.T{EN: "Education", ES: "Formación"}
	secLanguages  = data.T{EN: "Languages", ES: "Idiomas"}
	pageWord      = data.T{EN: "Page", ES: "Página"}
)

// The palette is a greyscale reading of the site's tokens, plus the one accent.
// Print has no dark mode and no theme toggle, so these are absolute rather than
// token lookups.
var (
	ink     = pdf.Gray(0.11)
	inkDim  = pdf.Gray(0.34)
	inkFine = pdf.Gray(0.48)
	rule    = pdf.Gray(0.80)
	accent  = pdf.Color{R: 0.11, G: 0.47, B: 0.31} // the site's muted phosphor green, flattened for paper
)

// Build renders the CV and returns the finished PDF file.
//
// The document carries no creation date, so the same CV data always produces
// byte-identical output. That is what lets a test assert the committed file is
// current instead of merely well-formed.
func Build(lang data.Lang) []byte {
	b := &builder{lang: lang, m: metricsFor(lang), doc: pdf.New(pdf.Options{
		Margin:   pdf.Margin{Top: marginY, Right: marginX, Bottom: marginY, Left: marginX},
		Title:    data.Me.Name + " — " + data.Me.Headline.In(lang),
		Author:   data.Me.Name,
		Subject:  data.Me.Tagline.In(lang),
		Keywords: keywords(lang),
	})}

	b.header()
	b.experience()
	b.projects()
	b.skills()
	b.education()
	b.footer()

	return b.doc.Bytes()
}

type builder struct {
	doc  *pdf.Doc
	lang data.Lang
	m    copyMetrics
}

// t resolves one piece of bilingual copy in the language being built.
func (b *builder) t(v data.T) string { return v.In(b.lang) }

// ts resolves a bilingual list in the language being built.
func (b *builder) ts(v data.TS) []string { return v.In(b.lang) }

// bodyWidth is the width of the main text column.
func (b *builder) bodyWidth() float64 { return b.doc.Right() - bodyX }

// contentBottom is the y at which a page's content must stop, leaving room for
// the running footer.
func (b *builder) contentBottom() float64 { return b.doc.Bottom() - footerHeight }

// ensure starts a new page when h more points of content would not fit.
func (b *builder) ensure(h float64) {
	if b.doc.Y()+h > b.contentBottom() {
		b.footer()
		b.doc.NewPage()
	}
}

// footer stamps the running line at the foot of the current page.
//
// It is drawn as each page is closed rather than in one pass at the end,
// because a finished page cannot be reopened. That rules out an "N of M"
// count — the total is unknown while page one is still being written — so the
// footer names the page only.
func (b *builder) footer() {
	y := b.doc.Bottom() - 10
	b.doc.Text(marginX, y, data.Me.Name+" — "+b.t(data.Me.Headline), pdf.Regular, 7.5, inkFine)

	page := fmt.Sprintf("%s %d", b.t(pageWord), b.doc.PageCount())
	b.doc.Text(b.doc.Right()-pdf.Width(page, pdf.Regular, 7.5), y, page, pdf.Regular, 7.5, inkFine)
}

// header draws the name block and contact details.
func (b *builder) header() {
	d := b.doc
	d.TextLine(marginX, data.Me.Name, pdf.Bold, nameSize, nameSize+6, ink)
	d.TextLine(marginX, b.t(data.Me.Headline), pdf.Regular, headlineSize, headlineSize+7, accent)

	for _, line := range pdf.Wrap(b.t(data.Me.Tagline), pdf.Italic, taglineSize, d.ContentWidth()) {
		d.TextLine(marginX, line, pdf.Italic, taglineSize, taglineSize+4, inkDim)
	}

	d.Advance(12)
	b.contactLine([]item{
		{text: data.Me.Contact.Email, uri: "mailto:" + data.Me.Contact.Email},
		{text: data.Me.Contact.Phone, uri: "tel:" + strings.ReplaceAll(data.Me.Contact.Phone, " ", "")},
		{text: b.t(data.Me.Contact.Location)},
	})
	b.contactLine([]item{
		{text: trimScheme(data.Me.Contact.Site), uri: data.Me.Contact.Site},
		{text: trimScheme(data.Me.Contact.LinkedIn), uri: data.Me.Contact.LinkedIn},
		{text: trimScheme(data.Me.Contact.GitHub), uri: data.Me.Contact.GitHub},
	})
}

// item is one linkable fragment on a contact line.
type item struct {
	text string
	uri  string
}

// contactLine draws separated fragments on one line, making each one clickable
// where it has a destination.
func (b *builder) contactLine(items []item) {
	d := b.doc
	const sep = "  ·  "
	sepW := pdf.Width(sep, pdf.Regular, contactSize)

	x := marginX
	top := d.Y()
	baseline := top + contactSize
	for i, it := range items {
		if it.text == "" {
			continue
		}
		if i > 0 {
			d.Text(x, baseline, sep, pdf.Regular, contactSize, rule)
			x += sepW
		}
		w := pdf.Width(it.text, pdf.Regular, contactSize)
		colour := inkDim
		if it.uri != "" {
			colour = accent
			// The rect is padded a little above and below the glyphs so the
			// clickable area matches the line, not just the x-height.
			d.Link(x, top-1, w, contactSize+3, it.uri)
		}
		d.Text(x, baseline, it.text, pdf.Regular, contactSize, colour)
		x += w
	}
	d.Advance(contactSize + 5)
}

// section draws a heading with a rule under it.
func (b *builder) section(title string) {
	d := b.doc
	b.ensure(52)
	d.Advance(12)
	d.TextLine(marginX, strings.ToUpper(title), pdf.Bold, sectionSize, sectionSize+5, ink)
	d.Rule(marginX, d.ContentWidth(), 0.6, rule)
	d.Advance(8)
}

// entry draws one two-column record: metrics on the left, content on the right.
//
// meta is drawn at the y the entry starts on, so an entry that spills onto the
// next page leaves its dates behind with the heading they belong to.
func (b *builder) entry(meta []string, body func()) {
	d := b.doc
	start := d.Y()
	y := start + metaSize
	for _, line := range meta {
		for _, wrapped := range pdf.Wrap(line, pdf.Regular, metaSize, metaWidth) {
			d.Text(marginX, y, wrapped, pdf.Regular, metaSize, inkFine)
			y += metaSize + 2.5
		}
	}
	d.SetY(start)
	body()
	d.Advance(8)
}

// heading draws an entry's title pair: the name in bold, the role under it.
func (b *builder) heading(name, role string) {
	d := b.doc
	d.TextLine(bodyX, name, pdf.Bold, companySize, companySize+3.5, ink)
	if role != "" {
		for _, line := range pdf.Wrap(role, pdf.Italic, roleSize, b.bodyWidth()) {
			d.TextLine(bodyX, line, pdf.Italic, roleSize, roleSize+3.5, inkDim)
		}
	}
}

// paragraph draws wrapped body copy.
func (b *builder) paragraph(text string, size float64, colour pdf.Color) {
	if text == "" {
		return
	}
	d := b.doc
	d.Advance(3)
	for _, line := range pdf.Wrap(text, pdf.Regular, size, b.bodyWidth()) {
		b.ensure(size + 3.5)
		d.TextLine(bodyX, line, pdf.Regular, size, size+3.5, colour)
	}
}

// first returns at most n items, for the places where the PDF prints a subset
// of what the site shows.
func first(items []string, n int) []string {
	if len(items) <= n {
		return items
	}
	return items[:n]
}

// bullets draws a hanging-indent list.
func (b *builder) bullets(items []string) {
	if len(items) == 0 {
		return
	}
	d := b.doc
	const indent = 11.0
	bulletSize := b.m.bullet
	leading := bulletSize + b.m.leading
	d.Advance(4)
	for _, item := range items {
		lines := pdf.Wrap(item, pdf.Regular, bulletSize, b.bodyWidth()-indent)
		for i, line := range lines {
			// Keep at least the marker and its first line together; a bullet
			// whose text starts on the next page reads as a stray dash.
			if i == 0 {
				b.ensure(leading * 2)
				d.Text(bodyX, d.Y()+bulletSize, "•", pdf.Regular, bulletSize, inkFine)
			} else {
				b.ensure(leading)
			}
			d.TextLine(bodyX+indent, line, pdf.Regular, bulletSize, leading, ink)
		}
		d.Advance(2.5)
	}
}

// links draws an entry's public destinations as one clickable run.
func (b *builder) links(items []data.Link) {
	if len(items) == 0 {
		return
	}
	d := b.doc
	const sep = "  ·  "
	sepW := pdf.Width(sep, pdf.Regular, stackSize+0.4)
	size := stackSize + 0.4

	b.ensure(size + 6)
	d.Advance(4)
	x := bodyX
	top := d.Y()
	baseline := top + size
	for i, l := range items {
		w := pdf.Width(l.Label, pdf.Regular, size)
		// Wrap to a new line rather than running past the right margin.
		if x > bodyX && x+sepW+w > d.Right() {
			d.Advance(size + 3)
			x = bodyX
			top = d.Y()
			baseline = top + size
		} else if i > 0 {
			d.Text(x, baseline, sep, pdf.Regular, size, rule)
			x += sepW
		}
		d.Link(x, top-1, w, size+3, l.URL)
		d.Text(x, baseline, l.Label, pdf.Regular, size, accent)
		x += w
	}
	d.Advance(size + 2)
}

// oneLine draws an entry reduced to a single line: the employer in bold and the
// title beside it. Used for the older roles, where a reader wants the shape of
// the career rather than what each job involved.
func (b *builder) oneLine(name, title string) {
	d := b.doc
	w := pdf.Width(name, pdf.Bold, companySize)
	d.Text(bodyX, d.Y()+companySize, name, pdf.Bold, companySize, ink)
	if title != "" {
		const sep = " — "
		d.Text(bodyX+w, d.Y()+companySize, sep+title, pdf.Italic, roleSize, inkDim)
	}
	d.Advance(companySize + 3.5)
}

func (b *builder) experience() {
	b.section(b.t(secExperience))

	// Roles are listed newest first, so this counter is a recency cutoff: the
	// most recent detailedRoles employers get their bullets, everything older
	// is reduced to the line a reader actually uses — who, what, when. Career
	// breaks do not count against the budget, because they carry a single line
	// already and spending a detail slot on one would push a real role out.
	detailed := 0

	for _, job := range data.Me.Jobs {
		meta := []string{data.Range(job.From, job.To, b.lang)}
		if loc := b.t(job.Location); loc != "" {
			meta = append(meta, loc)
		}

		brief := false
		if job.IsEmployment() {
			brief = detailed >= detailedRoles
			detailed++
		}

		b.ensure(58)
		b.entry(meta, func() {
			if brief {
				b.oneLine(b.t(job.Company), b.t(job.Title))
				return
			}
			b.heading(b.t(job.Company), b.t(job.Title))
			b.paragraph(b.t(job.Summary), b.m.summary, inkDim)
			b.bullets(first(b.ts(job.Bullets), jobBullets))
			b.links(job.Links)
		})
	}
}

func (b *builder) projects() {
	b.section(b.t(secProjects))
	for i, p := range data.Me.Projects {
		meta := []string{b.t(p.Role)}
		if period := p.Period(b.lang); period != "" {
			meta = append(meta, period)
		}
		brief := i >= detailedProject
		b.ensure(58)
		b.entry(meta, func() {
			b.heading(p.Name, "")
			b.paragraph(b.t(p.Summary), b.m.summary, inkDim)
			if brief {
				b.links(p.Links)
				return
			}
			b.bullets(first(b.ts(p.Bullets), projectBullets))
			b.links(p.Links)
		})
	}
}

func (b *builder) skills() {
	b.section(b.t(secSkills))
	for _, group := range data.Me.Skills {
		b.ensure(34)
		b.entry([]string{b.t(group.Category)}, func() {
			for _, line := range pdf.Wrap(strings.Join(b.ts(group.Skills), "  ·  "), pdf.Regular, b.m.summary, b.bodyWidth()) {
				b.ensure(b.m.summary + 4)
				b.doc.TextLine(bodyX, line, pdf.Regular, b.m.summary, b.m.summary+4, ink)
			}
		})
	}
	b.languages()
}

func (b *builder) education() {
	b.section(b.t(secEducation))
	for _, e := range data.Me.Education {
		// One line plus the gap after it: the reservation matches what oneLine
		// actually draws, so a row is not pushed to a new page over space it
		// was never going to use.
		b.ensure(companySize + 12)
		b.entry([]string{data.Range(e.From, e.To, b.lang)}, func() {
			b.oneLine(e.Institution, b.t(e.Program))
		})
	}
}

// languages draws the spoken languages as a final row of the skills table. It
// is one line, and a section heading of its own cost more vertical space than
// the content under it. The row keeps the word "Languages" in the left column,
// so a CV parser still sees the label it looks for.
func (b *builder) languages() {
	parts := make([]string, 0, len(data.Me.Languages))
	for _, l := range data.Me.Languages {
		parts = append(parts, b.t(l.Name)+" — "+b.t(l.Level))
	}
	b.ensure(24)
	b.entry([]string{b.t(secLanguages)}, func() {
		for _, line := range pdf.Wrap(strings.Join(parts, "  ·  "), pdf.Regular, b.m.summary, b.bodyWidth()) {
			b.doc.TextLine(bodyX, line, pdf.Regular, b.m.summary, b.m.summary+4, ink)
		}
	})
}

// keywords fills the Info dictionary's Keywords entry from the skills, which is
// what a document-indexing search actually reads.
func keywords(lang data.Lang) string {
	var all []string
	for _, group := range data.Me.Skills {
		all = append(all, group.Skills.In(lang)...)
	}
	return strings.Join(all, ", ")
}

// trimScheme shortens a URL for display without changing where it points.
func trimScheme(u string) string {
	u = strings.TrimPrefix(u, "https://")
	u = strings.TrimPrefix(u, "http://")
	return strings.TrimSuffix(u, "/")
}
