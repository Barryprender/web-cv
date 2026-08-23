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

// canonicalURL turns a route into its absolute form for rel=canonical and og:url.
func canonicalURL(route string) string {
	return siteURL + route
}

// personJSONLD renders the schema.org/Person block from the CV values, so the
// structured data cannot drift from what the pages actually say.
//
// json.Marshal escapes <, > and & to their \u form, so no CV value can close
// the surrounding <script> element; the result is safe as template.JS.
func personJSONLD() (template.JS, error) {
	locality, country, found := strings.Cut(data.Me.Contact.Location, ", ")
	address := map[string]any{"@type": "PostalAddress", "addressLocality": locality}
	if found {
		address["addressCountry"] = country
	}

	var knowsAbout []string
	for _, group := range data.Me.Skills {
		knowsAbout = append(knowsAbout, group.Skills...)
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
		knowsLanguage = append(knowsLanguage, l.Name)
	}

	person := map[string]any{
		"@context":      "https://schema.org",
		"@type":         "Person",
		"name":          data.Me.Name,
		"jobTitle":      data.Me.Headline,
		"description":   data.Me.Tagline,
		"url":           siteURL + "/",
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
		if job.End == "Present" && job.IsEmployment() {
			person["worksFor"] = map[string]any{"@type": "Organization", "name": job.Company}
			person["hasOccupation"] = map[string]any{
				"@type": "Occupation",
				"name":  job.Title,
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

// sitemapXML lists the site's four canonical pages.
func sitemapXML(routes []string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
	for _, route := range routes {
		fmt.Fprintf(&b, "  <url><loc>%s</loc></url>\n", canonicalURL(route))
	}
	b.WriteString("</urlset>\n")
	return b.String()
}
