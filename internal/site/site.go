// Package site wires the embedded templates and static assets into an
// http.Handler for barrypre.com.
package site

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"mime"
	"net/http"
	netmail "net/mail"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"barrypre.com/webcv/internal/data"
	"barrypre.com/webcv/internal/mail"
)

//go:embed templates/*.html
var templatesFS embed.FS

// static/cv.pdf is generated from internal/data rather than written by hand.
// Regenerate it after editing the CV values; TestPDFIsCurrent fails if the
// committed file has fallen behind them.
//
//go:generate go run ../../cmd/pdfgen -o static/cv.pdf

//go:embed static
var staticFS embed.FS

// cvPDFAsset is where the generated CV lives inside the embedded filesystem,
// and cvPDFPath is the single public URL it is served on.
const (
	cvPDFAsset = "cv.pdf"
	cvPDFPath  = "/cv.pdf"
)

// cvPDFAssetFor and cvPDFPathFor name one language's copy of the download.
// English keeps the unsuffixed name and URL, which are already in circulation;
// every other language gets a suffix. Both derive from the same constants, so
// the embedded file and the public URL cannot disagree about which is which.
func cvPDFAssetFor(l data.Lang) string {
	if l == data.EN {
		return cvPDFAsset
	}
	return strings.TrimSuffix(cvPDFAsset, ".pdf") + "-" + string(l) + ".pdf"
}

func cvPDFPathFor(l data.Lang) string {
	if l == data.EN {
		return cvPDFPath
	}
	return l.Prefix() + cvPDFPath
}

// cvPDFFilenameFor is what the browser saves that language's download as.
func cvPDFFilenameFor(l data.Lang) string {
	if l == data.EN {
		return cvPDFFilename
	}
	return strings.TrimSuffix(cvPDFFilename, ".pdf") + "-" + strings.ToUpper(string(l)) + ".pdf"
}

// alternatesFor lists every language's URL for one route, plus x-default
// pointing at English. Each page names all of them, itself included, which is
// what hreflang requires to be treated as a reciprocal set.
func alternatesFor(route string) []alternate {
	out := make([]alternate, 0, len(data.Langs)+1)
	for _, l := range data.Langs {
		out = append(out, alternate{Tag: l.Tag(), Href: canonicalURL(l, route)})
	}
	return append(out, alternate{Tag: "x-default", Href: canonicalURL(data.EN, route)})
}

// cvPDFFilename is what a browser saves the download as. It is derived from the
// CV data so it cannot drift from the name on the document itself.
//
// The value goes into a quoted header parameter, so it is reduced to a
// conservative character set first. The name is compile-time data and not
// attacker-controlled, but a filename built by string concatenation and dropped
// into a response header is exactly the shape of a header-injection bug, and
// the cost of closing it is one filter.
var cvPDFFilename = safeFilename(data.Me.Name) + "-CV.pdf"

// safeFilename keeps ASCII letters, digits and hyphens, folding runs of
// anything else into a single hyphen.
func safeFilename(s string) string {
	var b strings.Builder
	lastHyphen := true // leading separators are dropped
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastHyphen = false
		case !lastHyphen:
			b.WriteByte('-')
			lastHyphen = true
		}
	}
	return strings.TrimSuffix(b.String(), "-")
}

// pageData is what every template renders from.
type pageData struct {
	Title       string
	Description string
	Canonical   string
	Active      string
	Year        int
	// Lang is the language this render is in, for the html lang attribute and
	// for building links that stay inside it.
	Lang data.Lang
	// Tag and OGLocale are Lang in the forms the markup needs.
	Tag      string
	OGLocale string
	// Alternates are every language's URL for this same page, including this
	// one, for the hreflang links.
	Alternates []alternate
	// Switch is the other language's URL for this page, for the nav switcher.
	Switch     string
	SwitchLang string
	// Nav holds this language's version of each route, so links in the page
	// do not drop the visitor back into English.
	Nav navLinks
	// UI is every interface string already resolved into this language.
	UI uiText
	// JSONLD is the schema.org/Person block, identical on every page.
	JSONLD template.JS
	// Status is the no-JS contact form result read back off the query string
	// after the POST redirect: "sent", "error", or "" for a normal visit.
	Status string
	CV     any
}

// alternate is one hreflang link.
type alternate struct {
	Tag  string
	Href string
}

// navLinks holds the language-local URL of every page the templates link to.
type navLinks struct {
	Home       string
	Experience string
	Projects   string
	Skills     string
	Contact    string
	CV         string
}

// navFor builds the link set for one language.
func navFor(l data.Lang) navLinks {
	return navLinks{
		Home:       localPath(l, "/"),
		Experience: localPath(l, "/experience"),
		Projects:   localPath(l, "/projects"),
		Skills:     localPath(l, "/skills"),
		Contact:    localPath(l, "/contact"),
		CV:         cvPDFPathFor(l),
	}
}

// langOf reads the language prefix back off a request path, returning the
// empty prefix for English. It is used on the contact POST, which is the one
// request whose response has to know which language page it came from.
//
// The path is matched against the known prefixes rather than parsed, so an
// unrecognised first segment cannot become part of a redirect target.
func langOf(path string) string {
	for _, l := range data.Langs {
		p := l.Prefix()
		if p != "" && (path == p || strings.HasPrefix(path, p+"/")) {
			return p
		}
	}
	return ""
}

// formStatus whitelists the ?status= value that the contact POST redirects
// back with. Anything unrecognised is dropped rather than echoed into the page.
func formStatus(r *http.Request) string {
	switch s := r.URL.Query().Get("status"); s {
	case "sent", "error", "busy", "failed":
		return s
	default:
		return ""
	}
}

var pages = map[string]struct {
	path   string
	active string
}{
	"/":           {"home.html", "home"},
	"/experience": {"experience.html", "experience"},
	"/projects":   {"projects.html", "projects"},
	"/skills":     {"skills.html", "skills"},
	"/contact":    {"contact.html", "contact"},
}

// routes returns every page path, sorted, for the sitemap.
func routes() []string {
	out := make([]string, 0, len(pages))
	for route := range pages {
		out = append(out, route)
	}
	slices.Sort(out)
	return out
}

// NewHandler builds the site's routes.
//
// sender delivers contact form submissions. A nil sender is treated as an
// unconfigured one: submissions are refused with a delivery failure rather than
// accepted and dropped.
func NewHandler(year int, sender mail.Sender) http.Handler {
	if sender == nil {
		sender = mail.NewChain(log.Printf)
	}
	mux := http.NewServeMux()

	staticContent, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err) // embedded FS, cannot fail at runtime
	}
	static, err := newStaticHandler(staticContent)
	if err != nil {
		panic(err) // embedded FS, cannot fail at runtime
	}

	jsonLD := make(map[data.Lang]template.JS, len(data.Langs))
	for _, lang := range data.Langs {
		block, err := personJSONLD(lang)
		if err != nil {
			panic(err) // built from static CV values, cannot fail at runtime
		}
		jsonLD[lang] = block
	}

	// Templates resolve asset URLs through the static handler so every
	// reference carries the current content hash.
	funcs := template.FuncMap{"asset": static.assetURL, "Range": data.Range}

	for route, p := range pages {
		tmpl := template.Must(template.New(p.path).Funcs(funcs).
			ParseFS(templatesFS, "templates/layout.html", "templates/"+p.path))

		meta := pageMeta[route]

		for _, lang := range data.Langs {
			// Everything that does not depend on the request is resolved here,
			// once at startup, rather than per visit.
			base := pageData{
				Title:       meta.Title.In(lang),
				Description: meta.Description.In(lang),
				Canonical:   canonicalURL(lang, route),
				Active:      p.active,
				Year:        year,
				Lang:        lang,
				Tag:         lang.Tag(),
				OGLocale:    lang.OGLocale(),
				Alternates:  alternatesFor(route),
				Switch:      localPath(otherLang(lang), route),
				SwitchLang:  string(otherLang(lang)),
				Nav:         navFor(lang),
				UI:          textFor(lang),
				JSONLD:      jsonLD[lang],
				CV:          data.Me,
			}

			// A pattern ending in "/" is a subtree wildcard in this mux, so
			// the English root is pinned with {$}. Every other path, including
			// "/es", is already an exact match and must not gain a slash.
			pattern := localPath(lang, route)
			if route == "/" && lang == data.EN {
				pattern = "/{$}"
			}

			mux.HandleFunc("GET "+pattern, func(w http.ResponseWriter, r *http.Request) {
				d := base
				d.Status = formStatus(r)
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				if err := tmpl.ExecuteTemplate(w, "layout", d); err != nil {
					log.Printf("render %s %s: %v", lang, route, err)
					http.Error(w, "internal error", http.StatusInternalServerError)
				}
			})
		}
	}

	for _, lang := range data.Langs {
		if lang == data.EN {
			continue
		}
		canonical := lang.Prefix()
		mux.HandleFunc("GET "+canonical+"/{$}", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, canonical, http.StatusMovedPermanently)
		})
	}

	mux.Handle("GET "+staticPrefix, static)

	// Browsers probe /favicon.ico regardless of the link tags, so answer it
	// from the same bytes instead of letting it 404 on every visit.
	mux.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		r2 := r.Clone(r.Context())
		r2.URL.Path = staticPrefix + "favicon.ico"
		static.ServeHTTP(w, r2)
	})

	// The CV download gets a short, stable URL of its own — it is the one
	// asset that gets shared, bookmarked and pasted into an email, and
	// /static/cv.pdf?v=<hash> is not a link anyone should have to hand out.
	// The bytes still come through the static handler, so the download keeps
	// its ETag and answers a repeat visit with a 304.
	//
	// Disposition is "inline" rather than "attachment": it lets the PDF open
	// in the browser's viewer for someone who only wants a look, while the
	// download attribute on the link still saves it under the filename below
	// for someone who wants the file.
	for _, lang := range data.Langs {
		asset, filename := cvPDFAssetFor(lang), cvPDFFilenameFor(lang)
		mux.HandleFunc("GET "+cvPDFPathFor(lang), func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Disposition", `inline; filename="`+filename+`"`)
			r2 := r.Clone(r.Context())
			r2.URL.Path = staticPrefix + asset
			static.ServeHTTP(w, r2)
		})
	}

	// Liveness probe for the platform's health checks. Deliberately does no
	// work beyond answering: it says the process is up and serving, which is
	// all a restart decision should ever be based on. Checking the email
	// providers here would take the site down every time one of them had a bad
	// minute, and the site does not need them to serve a single page.
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = io.WriteString(w, "ok\n")
	})

	serveText := func(contentType, body string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", contentType)
			w.Header().Set("Cache-Control", "public, max-age=3600")
			_, _ = io.WriteString(w, body)
		}
	}
	mux.HandleFunc("GET /robots.txt", serveText("text/plain; charset=utf-8", robotsTXT()))
	mux.HandleFunc("GET /sitemap.xml", serveText("application/xml", sitemapXML(routes())))

	// The contact endpoint is the only thing here that accepts input, so it
	// carries its own budget: five in a burst, then one more every 30s.
	limiter := newIPRateLimiter(5, 30*time.Second)
	limiter.deny = func(w http.ResponseWriter, r *http.Request, retryAfter time.Duration) {
		respondContact(w, r, http.StatusTooManyRequests,
			fmt.Sprintf("Too many messages just now. Wait %d seconds and try again.", int(retryAfter.Seconds())))
	}
	for _, lang := range data.Langs {
		mux.Handle("POST "+localPath(lang, "/contact"), limiter.limit(contactHandler(sender)))
	}

	// Reject non-safe cross-origin browser requests before they reach any
	// handler. GET/HEAD/OPTIONS are left alone, which is sound here because no
	// safe method on this site changes state.
	csrf := http.NewCrossOriginProtection()

	return securityHeaders(csrf.Handler(mux))
}

type contactMessage struct {
	Name    string
	Email   string
	Message string
}

// maxContactBody caps how much of a request the contact endpoint will read.
// A name, an email address and a message need nowhere near this.
const maxContactBody = 64 << 10 // 64 KiB

// sendTimeout bounds the whole delivery attempt, across every provider.
//
// It has to clear the server's 10s WriteTimeout with room to spare: a send that
// outlived the write deadline would have the connection closed underneath it,
// and the visitor would see a dropped request rather than the failure message
// this is all here to produce.
const sendTimeout = 6 * time.Second

// contactHandler handles a submission and delivers it through sender.
func contactHandler(sender mail.Sender) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxContactBody)

		if err := parseContactForm(r); err != nil {
			var tooBig *http.MaxBytesError
			if errors.As(err, &tooBig) {
				respondContact(w, r, http.StatusRequestEntityTooLarge, "That message is too large. Keep it under 64 KB.")
				return
			}
			respondContact(w, r, http.StatusBadRequest, "Could not read that submission. Try again.")
			return
		}
		if r.MultipartForm != nil {
			defer r.MultipartForm.RemoveAll()
		}

		msg := contactMessage{
			Name:    strings.TrimSpace(r.FormValue("name")),
			Email:   strings.TrimSpace(r.FormValue("email")),
			Message: strings.TrimSpace(r.FormValue("message")),
		}
		if missing := missingFields(msg); len(missing) > 0 {
			respondContact(w, r, http.StatusBadRequest, "Please enter "+prose(missing)+".")
			return
		}
		// The address becomes the Reply-To on the delivered mail, so it has to
		// be a real one. Checked here rather than left to the provider: a
		// typo should come back as "check your address", not as a delivery
		// failure that reads like the site is broken.
		if !validEmail(msg.Email) {
			respondContact(w, r, http.StatusBadRequest, "That email address does not look right. Check it and try again.")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), sendTimeout)
		defer cancel()

		if err := sender.Send(ctx, composeContactEmail(msg, time.Now())); err != nil {
			// The whole submission goes to the log, but only on the path where
			// it was not delivered. Someone took the trouble to write it and
			// is now being told it failed; this is what makes it recoverable
			// rather than lost. A delivered message is not logged, so the
			// ordinary case leaves no copy of a stranger's words on disk.
			//
			// %q escapes newlines and control characters, so a submitted value
			// cannot forge extra log lines or smuggle terminal escapes into
			// whatever reads them; truncating keeps one submission from
			// flooding the log.
			//
			// The provider's own words stay here too. They can name the
			// account, the domain, or why a key was rejected, none of which is
			// a visitor's business — the response below carries the one thing
			// they can act on.
			log.Printf("contact: delivery failed: %v — undelivered message name=%q email=%q message=%q",
				err, truncate(msg.Name, 100), truncate(msg.Email, 200), truncate(msg.Message, 1000))
			respondContact(w, r, http.StatusBadGateway,
				"That did not send — something is wrong on my end, not yours. "+
					"Please email me directly at "+data.Me.Contact.Email+".")
			return
		}

		// Delivered: record that it happened and who to, and nothing else. The
		// message itself is in the mailbox now and does not belong in a log.
		log.Printf("contact: delivered message from %q", truncate(msg.Email, 200))
		respondContact(w, r, http.StatusOK, "")
	}
}

// validEmail reports whether addr is a single bare address that can be used as
// a Reply-To header.
//
// net/mail is the parser the standard library already trusts for this, so it is
// used rather than a regex. It accepts a display name ("Name <a@b>"), which is
// not what a form field should hold, so anything decorated is rejected.
func validEmail(addr string) bool {
	if addr == "" || len(addr) > 320 || strings.ContainsAny(addr, "\r\n") {
		return false
	}
	parsed, err := netmail.ParseAddress(addr)
	return err == nil && parsed.Name == "" && parsed.Address == addr
}

// parseContactForm decodes the body whichever way it was encoded, and reports
// why if it could not.
//
// The encoding is dispatched on explicitly rather than letting
// ParseMultipartForm handle both. It does fall back to ParseForm internally for
// a urlencoded body, but it then discards that error when the body turns out
// not to be multipart — so an oversized urlencoded body surfaced as
// ErrNotMultipart with empty fields, and the caller could not tell a too-large
// request apart from an empty one.
//
// Note that ParseForm must not be called ahead of ParseMultipartForm: it leaves
// r.Form non-nil and empty for a multipart body, which silently blanks every
// field the fetch path submits.
func parseContactForm(r *http.Request) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		// Missing or unparseable: ParseForm treats it as octet-stream and
		// leaves the body alone, which reads back as an empty submission.
		return r.ParseForm()
	}
	if mediaType == "multipart/form-data" {
		return r.ParseMultipartForm(maxContactBody)
	}
	return r.ParseForm()
}

// truncate shortens s to at most n bytes without splitting a rune.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}

// contactStatus maps a response code to the token the no-JS redirect carries,
// so the page can say something accurate rather than one generic failure line.
//
// "failed" is kept distinct from "error" because the two ask the visitor for
// different things: "error" means fix the submission, "failed" means the
// submission was fine and the site could not deliver it, so use the address on
// the page instead.
func contactStatus(code int) string {
	switch code {
	case http.StatusOK:
		return "sent"
	case http.StatusTooManyRequests:
		return "busy"
	case http.StatusBadGateway:
		return "failed"
	default:
		return "error"
	}
}

// missingFields names the empty fields so the response can say which one to
// fill in.
func missingFields(msg contactMessage) []string {
	var missing []string
	if msg.Name == "" {
		missing = append(missing, "your name")
	}
	if msg.Email == "" {
		missing = append(missing, "your email address")
	}
	if msg.Message == "" {
		missing = append(missing, "your message")
	}
	return missing
}

// prose renders a list the way a sentence would: "a", "a and b", "a, b and c".
func prose(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return items[0] + " and " + items[1]
	default:
		return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
	}
}

func respondContact(w http.ResponseWriter, r *http.Request, code int, errMsg string) {
	ok := code == http.StatusOK
	if strings.Contains(r.Header.Get("Accept"), "application/json") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(map[string]any{"ok": ok, "error": errMsg})
		return
	}
	// no-JS fallback: POST-redirect-GET back to the contact page. The fragment
	// points at the rendered status message so the browser scrolls to it —
	// without it the result was invisible to anyone not running JS.
	// Back to the contact page in the language the form was posted from, not
	// always the English one: a Spanish visitor who submits the form must not
	// be dropped into English to read the result.
	http.Redirect(w, r, langOf(r.URL.Path)+"/contact?status="+contactStatus(code)+"#form-status",
		http.StatusSeeOther)
}
