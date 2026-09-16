package site

import (
	"encoding/json"
	"fmt"
	"html/template"
	"strings"

	"barrypre.com/webcv/internal/data"
)

// siteURL is the canonical origin, taken from the CV data so there is one
// source of truth for it. Trailing slashes are never stored here.
var siteURL = strings.TrimSuffix(data.Me.Contact.Site, "/")

// siteHost is siteURL without its scheme — the bare hostname, for the places
// that name the site in prose rather than link to it.
//
// Derived rather than written out, because the one that was written out went
// stale the moment the origin changed: the contact email's subject line still
// said barrypre.com after the site had moved.
var siteHost = strings.TrimPrefix(strings.TrimPrefix(siteURL, "https://"), "http://")

// canonicalURL turns a route into its absolute form for rel=canonical and
// og:url, in the language it is served in.
//
// The English pages keep the bare paths they have always had; Spanish sits
// under /es. A route of "/" must not produce "/es/" with a trailing slash and
// "/es" without one, or the two would be separate URLs to a crawler.
func canonicalURL(lang data.Lang, route string) string {
	if lang == data.EN {
		return siteURL + route
	}
	if route == "/" {
		return siteURL + lang.Prefix()
	}
	return siteURL + lang.Prefix() + route
}

// localPath is canonicalURL without the origin — what a link in a page uses.
func localPath(lang data.Lang, route string) string {
	if lang == data.EN {
		return route
	}
	if route == "/" {
		return lang.Prefix()
	}
	return lang.Prefix() + route
}

// personJSONLD renders the schema.org/Person block from the CV values, so the
// structured data cannot drift from what the pages actually say.
//
// json.Marshal escapes <, > and & to their \u form, so no CV value can close
// the surrounding <script> element; the result is safe as template.JS.
func personJSONLD(lang data.Lang) (template.JS, error) {
	locality, country, found := strings.Cut(data.Me.Contact.Location.In(lang), ", ")
	address := map[string]any{"@type": "PostalAddress", "addressLocality": locality}
	if found {
		address["addressCountry"] = country
	}

	var knowsAbout []string
	for _, group := range data.Me.Skills {
		knowsAbout = append(knowsAbout, group.Skills.In(lang)...)
	}

	var alumniOf []any
	for _, e := range data.Me.Education {
		alumniOf = append(alumniOf, map[string]any{
			"@type": "EducationalOrganization",
			"name":  e.Institution,
		})
	}

	var knowsLanguage []string
	for _, l := range data.Me.Languages {
		knowsLanguage = append(knowsLanguage, l.Name.In(lang))
	}

	person := map[string]any{
		"@context":      "https://schema.org",
		"@type":         "Person",
		"name":          data.Me.Name,
		"jobTitle":      data.Me.Headline.In(lang),
		"description":   data.Me.Tagline.In(lang),
		"url":           canonicalURL(lang, "/"),
		"inLanguage":    lang.Tag(),
		"email":         "mailto:" + data.Me.Contact.Email,
		"telephone":     data.Me.Contact.Phone,
		"address":       address,
		"sameAs":        []string{data.Me.Contact.LinkedIn, data.Me.Contact.GitHub},
		"knowsAbout":    knowsAbout,
		"knowsLanguage": knowsLanguage,
		"alumniOf":      alumniOf,
	}

	// worksFor must name an employer or nothing. Self-directed work and career
	// breaks are current entries too, and claiming either as an Organization
	// would put a company that does not exist into the structured data.
	for _, job := range data.Me.Jobs {
		if job.To.Year == 0 && job.IsEmployment() {
			person["worksFor"] = map[string]any{"@type": "Organization", "name": job.Company.In(lang)}
			person["hasOccupation"] = map[string]any{
				"@type": "Occupation",
				"name":  job.Title.In(lang),
			}
			break
		}
	}

	encoded, err := json.Marshal(person)
	if err != nil {
		return "", fmt.Errorf("marshal person json-ld: %w", err)
	}
	return template.JS(encoded), nil
}

// robotsTXT allows everything and points crawlers at the sitemap. Built once
// at startup rather than served from a file so the origin stays in one place.
func robotsTXT() string {
	return "User-agent: *\nAllow: /\n\nSitemap: " + siteURL + "/sitemap.xml\n"
}

// sitemapXML lists every page in every language.
//
// Each entry carries xhtml:link alternates naming all of its translations,
// including itself, which is what the sitemap protocol asks for and what stops
// a crawler treating the Spanish pages as thin duplicates of the English ones.
func sitemapXML(routes []string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"` + "\n")
	b.WriteString(`        xmlns:xhtml="http://www.w3.org/1999/xhtml">` + "\n")
	for _, lang := range data.Langs {
		for _, route := range routes {
			fmt.Fprintf(&b, "  <url>\n    <loc>%s</loc>\n", canonicalURL(lang, route))
			for _, alt := range data.Langs {
				fmt.Fprintf(&b, `    <xhtml:link rel="alternate" hreflang="%s" href="%s"/>`+"\n",
					alt.Tag(), canonicalURL(alt, route))
			}
			fmt.Fprintf(&b, `    <xhtml:link rel="alternate" hreflang="x-default" href="%s"/>`+"\n",
				canonicalURL(data.EN, route))
			b.WriteString("  </url>\n")
		}
	}
	b.WriteString("</urlset>\n")
	return b.String()
}
