# barrypre.com

Barry Prendergast's CV site. Go stdlib backend, native Web Components on the
frontend, zero external JS/CSS dependencies.

## Stack

- **Backend**: `net/http` (Go 1.22+ method-tagged routing), `html/template`,
  everything embedded into one binary via `embed.FS`. No framework.
- **Frontend**: native custom elements (`<cv-nav>`, `<cv-theme-toggle>`,
  `<cv-timeline>`, `<cv-contact-form>`), loaded as a plain ES module — no
  bundler, no npm dependency. Content stays in the light DOM so it's
  crawlable and works with JS disabled; the elements only add behavior
  (mobile menu, theme persistence, scroll-reveal, form submission).
- **Content**: `internal/data/cv.go` is the single source of truth. Templates
  render from it — edit the Go values, not the HTML, to update the CV.
- **Email**: `internal/mail` sends contact form submissions through Resend or
  Postmark (or both, with failover) over plain `net/http` — no SDK, no
  dependency.
- **PDF**: `internal/pdf` is a small PDF writer built on the standard library
  (base-14 fonts, one Flate content stream per page, real link annotations).
  `internal/cvpdf` lays the CV out with it. The file is generated at build
  time, not per request, so a visitor cannot make the server render anything.

Originally planned to use `templ` (matching the stack on your other Fly.io
project) but this sandbox's network allowlist blocked `proxy.golang.org`, so
it fell back to `html/template` — arguably the better call anyway per your
own rule to avoid frameworks unless needed. Swapping to `templ` later is a
templates-only change if you want the extra type safety.

## Run locally

```
go run ./cmd/server
```

Serves on `:8080` (override with `PORT`).

The contact form needs an email provider. For local work, log the messages
instead of sending them:

```
CONTACT_TRANSPORT=log go run ./cmd/server
```

Without that and without credentials, the form is fail-closed: it reports a
delivery failure to the visitor rather than accepting a message it cannot
deliver.

## Updating the CV

Edit `internal/data/cv.go`, then regenerate the PDF:

```
go generate ./internal/site
```

`TestPDFIsCurrent` fails if the committed `internal/site/static/cv.pdf` has
fallen behind the data, so a forgotten regeneration shows up in `go test ./...`
rather than shipping a PDF that disagrees with the site.

## Contact form delivery

Configured entirely through the environment:

| Variable | Purpose |
|---|---|
| `RESEND_API_KEY` | Enables Resend. Tried first. |
| `POSTMARK_SERVER_TOKEN` | Enables Postmark. Tried second. |
| `CONTACT_FROM` | Sending address. Required when either key is set; must be a verified sender on that provider. |
| `CONTACT_TO` | Where messages go. Defaults to the address in `internal/data`. |
| `CONTACT_TRANSPORT` | Set to `log` for local development. Nothing else selects the log transport. |

Set both keys and a message only fails once both providers have refused it —
that is the whole reason for carrying two. Set neither and the server starts
with a warning and refuses submissions; set a key without `CONTACT_FROM` and it
refuses to start at all, because that configuration would lose its first real
message.

Delivery failures are shown to the visitor rather than swallowed. The response
says the fault is on the server side and offers the direct address, and the
no-JS path carries it back as `?status=failed`, distinct from the `error` a bad
submission gets. A delivered message is not written to the log; an undelivered
one is, in full, so it stays recoverable.

The visitor's address becomes the `Reply-To`, so replying from the mailbox
reaches them. It is parsed with `net/mail` and rejected unless it is a bare
address — that field is the one piece of visitor input that ends up in an email
header.

## Structure

```
cmd/server/main.go         entrypoint, graceful shutdown
cmd/pdfgen/main.go         writes internal/site/static/cv.pdf from the CV data
internal/data/cv.go        CV content (edit this to update the site)
internal/mail/             Resend + Postmark senders, failover chain, env config
internal/pdf/              minimal stdlib PDF writer (no dependencies)
internal/cvpdf/            CV page layout, built on internal/pdf
internal/site/site.go      routes, embed directives, contact handler
internal/site/templates/   html/template files (layout + one per page)
internal/site/static/      css + js + cv.pdf, embedded into the binary
```

## The CV download

`GET /cv.pdf` serves the generated file through the same static handler as
everything else, so it carries an ETag and answers a repeat visit with a 304.
It is sent `inline` with a `Content-Disposition` filename, so the browser's
viewer opens it while the `download` attribute on the link still saves it as
`Barry-Prendergast-CV.pdf`.

Set in Helvetica rather than the site's Public Sans: embedding a real typeface
would mean parsing woff2 (Brotli) and subsetting TrueType, neither of which is
in the standard library. The text is real selectable text either way, which is
what applicant tracking systems parse.

Printing a page from the browser is handled separately by the `print` cascade
layer in `style.css` — it drops the nav, palette, filters and contact form,
expands every collapsed timeline entry, and forces a light palette.

## Still open

- Set `CONTACT_FROM` and at least one provider key in the Fly secrets before
  this goes live, and verify the sending domain with that provider — the form
  is fail-closed until you do.
- There's a 14-month gap in the experience data between Arcmedia AG (ended
  July 2019) and Quality Compusoft (started September 2020) — carried over
  from the LinkedIn export as-is; fill in or explain if you want it covered.
- Deploy: a `Dockerfile` + `fly.toml` aren't in here yet — say the word and
  I'll add them to match your other Fly.io project.
