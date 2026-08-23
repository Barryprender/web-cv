package site

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"barrypre.com/webcv/internal/data"
)

func TestCanonicalAndOpenGraphPerPage(t *testing.T) {
	// Built from the routes and the stored origin, so adding a page or moving
	// the site does not leave this list quietly asserting the old world.
	origin := strings.TrimSuffix(data.Me.Contact.Site, "/")
	var tests []struct {
		path      string
		canonical string
	}
	for _, route := range routes() {
		tests = append(tests, struct {
			path      string
			canonical string
		}{route, origin + route})
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			h := newTestHandler(t)
			body := get(t, h, tc.path).Body.String()

			for _, want := range []string{
				`<link rel="canonical" href="` + tc.canonical + `">`,
				`<meta property="og:url" content="` + tc.canonical + `">`,
				`<meta property="og:type" content="profile">`,
				`<meta name="twitter:card" content="summary">`,
			} {
				if !strings.Contains(body, want) {
					t.Errorf("page is missing %s", want)
				}
			}
		})
	}
}

// Every page had the same hardcoded description before, which tells a crawler
// the four pages are interchangeable.
func TestPageDescriptionsAreDistinct(t *testing.T) {
	seen := map[string]string{}
	for _, path := range []string{"/", "/experience", "/projects", "/skills", "/contact"} {
		h := newTestHandler(t)
		body := get(t, h, path).Body.String()

		_, rest, found := strings.Cut(body, `<meta name="description" content="`)
		if !found {
			t.Fatalf("%s has no meta description", path)
		}
		desc, _, _ := strings.Cut(rest, `"`)

		if desc == "" {
			t.Errorf("%s has an empty description", path)
		}
		if other, dup := seen[desc]; dup {
			t.Errorf("%s reuses the description from %s", path, other)
		}
		seen[desc] = path
	}
}

// The title is the strongest single SEO signal on a CV site, so the home page
// leads with the name and role rather than the word "Home".
func TestHomeTitleLeadsWithTheName(t *testing.T) {
	h := newTestHandler(t)
	body := get(t, h, "/").Body.String()

	if !strings.Contains(body, "<title>Barry Prendergast — Senior Full-Stack Engineer</title>") {
		t.Error("home title does not lead with the name and role")
	}
}

func TestRobotsTxt(t *testing.T) {
	h := newTestHandler(t)
	rec := get(t, h, "/robots.txt")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("Content-Type = %q, want text/plain", ct)
	}
	body := rec.Body.String()
	for _, want := range []string{"User-agent: *", "Allow: /", "Sitemap: " + strings.TrimSuffix(data.Me.Contact.Site, "/") + "/sitemap.xml"} {
		if !strings.Contains(body, want) {
			t.Errorf("robots.txt is missing %q", want)
		}
	}
}

// The sitemap has to stay in step with the route table, not a hand-kept list.
func TestSitemapCoversEveryRoute(t *testing.T) {
	h := newTestHandler(t)
	rec := get(t, h, "/sitemap.xml")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/xml") {
		t.Errorf("Content-Type = %q, want application/xml", ct)
	}

	body := rec.Body.String()
	for route := range pages {
		loc := "<loc>" + canonicalURL(route) + "</loc>"
		if !strings.Contains(body, loc) {
			t.Errorf("sitemap is missing %s", loc)
		}
	}
	if got := strings.Count(body, "<loc>"); got != len(pages) {
		t.Errorf("sitemap lists %d URLs, want %d", got, len(pages))
	}
}

func TestCanonicalURL(t *testing.T) {
	// Derived from the CV data rather than pinned to a literal. The origin has
	// moved once already, and a test that hardcodes it just has to be edited
	// alongside every other place that went stale — which is the failure this
	// whole single-source-of-truth arrangement exists to prevent.
	origin := strings.TrimSuffix(data.Me.Contact.Site, "/")
	for _, route := range routes() {
		want := origin + route
		if got := canonicalURL(route); got != want {
			t.Errorf("canonicalURL(%q) = %q, want %q", route, got, want)
		}
	}
}

// Whatever the origin is, it has to be a usable absolute https URL: it goes
// into rel=canonical, og:url and the sitemap, and a relative or unparseable
// value there is invisible until a crawler chokes on it.
func TestCanonicalOriginIsAbsoluteHTTPS(t *testing.T) {
	u, err := url.Parse(canonicalURL("/"))
	if err != nil {
		t.Fatalf("canonical URL does not parse: %v", err)
	}
	if u.Scheme != "https" {
		t.Errorf("scheme = %q, want https", u.Scheme)
	}
	if u.Host == "" {
		t.Error("canonical URL has no host")
	}
	if strings.HasSuffix(data.Me.Contact.Site, "/") {
		t.Error("the stored origin has a trailing slash; canonical URLs would double up")
	}
}

// siteHost is what names the site in prose — the contact email's subject line,
// for one. It must track the origin rather than being written out separately.
func TestSiteHostTracksTheOrigin(t *testing.T) {
	if strings.Contains(siteHost, "://") {
		t.Errorf("siteHost = %q, want no scheme", siteHost)
	}
	if !strings.Contains(siteURL, siteHost) {
		t.Errorf("siteHost %q is not the host of siteURL %q", siteHost, siteURL)
	}
}

// The projects page is where the CV carries outbound links, so the entries and
// their URLs have to actually render.
func TestProjectsPageRendersEveryProject(t *testing.T) {
	body := pageBody(t, "/projects")

	for _, want := range []string{
		"Secure UI Components",
		"SAUI — Server-Authoritative UI",
		"Archway Orthotics Portal",
		"Archway Orthotics site rebuild",
		`href="https://saui.fly.dev"`,
		"Client engagement",
		"Own project",
		"Product demo",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("projects page is missing %q", want)
		}
	}

	if n := strings.Count(body, `class="project "`) + strings.Count(body, `class="project"`); n != len(data.Me.Projects) {
		t.Errorf("rendered %d project entries, want %d", n, len(data.Me.Projects))
	}
}

// Material restored from the previous CV: the work has to be reachable, and the
// languages and security training have to actually render.
func TestOutboundLinksArePresent(t *testing.T) {
	contact := pageBody(t, "/contact")
	for _, want := range []string{data.Me.Contact.GitHub, data.Me.Contact.LinkedIn, data.Me.Contact.Email} {
		if !strings.Contains(contact, want) {
			t.Errorf("contact page is missing %q", want)
		}
	}

	// Every link declared anywhere must render on the page that owns it.
	pageFor := map[string]string{
		"/experience": pageBody(t, "/experience"),
		"/projects":   pageBody(t, "/projects"),
	}
	var declared int
	check := func(route string, links []data.Link, owner string) {
		for _, link := range links {
			declared++
			if !strings.Contains(pageFor[route], `href="`+link.URL+`"`) {
				t.Errorf("%s is missing %s (from %s)", route, link.URL, owner)
			}
		}
	}
	for _, job := range data.Me.Jobs {
		check("/experience", job.Links, job.Company)
	}
	for _, project := range data.Me.Projects {
		check("/projects", project.Links, project.Name)
	}
	if declared == 0 {
		t.Error("nothing declares a link; the published work is unreachable from the CV")
	}
}

func TestLanguagesAndEducationDetailRender(t *testing.T) {
	body := pageBody(t, "/skills")

	if len(data.Me.Languages) == 0 {
		t.Fatal("no languages in the CV data")
	}
	for _, lang := range data.Me.Languages {
		if !strings.Contains(body, lang.Name) || !strings.Contains(body, lang.Level) {
			t.Errorf("skills page is missing language %q", lang.Name)
		}
	}
	if n := strings.Count(body, `class="language-row"`); n != len(data.Me.Languages) {
		t.Errorf("rendered %d language rows, want %d", n, len(data.Me.Languages))
	}

	for _, entry := range data.Me.Education {
		for _, detail := range entry.Detail {
			if !strings.Contains(body, detail) {
				t.Errorf("skills page is missing education detail %q", detail[:40])
			}
		}
	}
}

func TestSecurityToolingIsListed(t *testing.T) {
	body := pageBody(t, "/skills")
	for _, tool := range []string{"Kali Linux", "Burp Suite", "SQLMap", "OWASP ZAP", "Metasploit", "Penetration testing"} {
		if !strings.Contains(body, tool) {
			t.Errorf("skills page does not list %q", tool)
		}
	}
}
