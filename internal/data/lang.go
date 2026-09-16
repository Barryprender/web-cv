package data

import (
	"slices"
	"strconv"
)

// Lang is a published content language.
type Lang string

const (
	EN Lang = "en"
	ES Lang = "es"
)

// Langs lists every language the site publishes, in menu order. English is
// first because it is the language the copy is written in; Spanish is a
// translation of it, and a missing Spanish string falls back to English rather
// than rendering an empty element.
var Langs = []Lang{EN, ES}

// Valid reports whether l is a language this site publishes. Anything arriving
// from a URL is checked with it before being used.
func (l Lang) Valid() bool { return slices.Contains(Langs, l) }

// Prefix is the path segment a language's pages live under. English is served
// from the root, so it has none: the canonical URLs that already exist keep
// working, and adding a translation does not move the pages search engines and
// recruiters have already been given.
func (l Lang) Prefix() string {
	if l == EN {
		return ""
	}
	return "/" + string(l)
}

// Tag is the BCP 47 tag for lang attributes and hreflang.
func (l Lang) Tag() string {
	switch l {
	case ES:
		return "es-ES"
	default:
		return "en-GB"
	}
}

// OGLocale is the Open Graph form of the same thing, which uses an underscore.
func (l Lang) OGLocale() string {
	switch l {
	case ES:
		return "es_ES"
	default:
		return "en_GB"
	}
}

// T is one piece of copy in both languages.
//
// Both strings sit in the same literal, so a role cannot be added, or its
// wording changed, without the other language being visible in the same edit.
// That adjacency is the whole reason for the type: two parallel files drift the
// first time someone is in a hurry.
type T struct{ EN, ES string }

// In returns the copy for l, falling back to English when the translation is
// missing. A half-translated page reads oddly; a page with holes in it reads as
// broken.
func (t T) In(l Lang) string {
	if l == ES && t.ES != "" {
		return t.ES
	}
	return t.EN
}

// TS is a list of copy in both languages — the bullet lists.
type TS struct{ EN, ES []string }

// In returns the list for l, falling back to English when the translation is
// missing. The fallback is all-or-nothing: a partly translated list would
// otherwise interleave two languages in one set of bullets.
func (t TS) In(l Lang) []string {
	if l == ES && len(t.ES) > 0 {
		return t.ES
	}
	return t.EN
}

// Date is a point on the CV timeline, held as numbers so it can be written out
// in either language rather than translated twice.
//
// A zero Year means "now" — the open end of the current role. A zero Month
// means only the year is known, which is true of the oldest entries.
type Date struct {
	Year  int
	Month int // 1-12, or 0 when only the year is recorded
}

var monthNames = map[Lang][13]string{
	EN: {"", "January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December"},
	// Lowercase, and joined to the year with "de", because that is how Spanish
	// writes a month and year. Capitalising them is the usual tell of a CV run
	// through a translation tool.
	ES: {"", "enero", "febrero", "marzo", "abril", "mayo", "junio",
		"julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre"},
}

var present = T{EN: "Present", ES: "Actualidad"}

// In writes the date out in l.
func (d Date) In(l Lang) string {
	if d.Year == 0 {
		return present.In(l)
	}
	year := strconv.Itoa(d.Year)
	if d.Month < 1 || d.Month > 12 {
		return year
	}
	month := monthNames[EN][d.Month]
	if l == ES {
		return monthNames[ES][d.Month] + " de " + year
	}
	return month + " " + year
}

// Range writes a start and end as one span, for the places that show both.
func Range(from, to Date, l Lang) string {
	return from.In(l) + " — " + to.In(l)
}
