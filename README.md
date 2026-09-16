# barrypre.com

Barry Prendergast's CV site. Go standard-library backend, native Web Components
on the frontend, zero external runtime dependencies — `go.mod` requires nothing
but Go itself, and no JavaScript, CSS or font is fetched from a third party.

## Stack

- **Backend**: `net/http` (Go 1.22+ method-tagged routing), `html/template`,
  everything embedded into one binary via `embed.FS`. No framework.
- **Frontend**: native custom elements (`<cv-nav>`, `<cv-theme-toggle>`,
  `<cv-filter>`, `<cv-timeline>`, `<cv-contact-form>`) plus two plain modules
  (`cv-palette.js`, `cv-transitions.js`), loaded as ES modules — no bundler, no
  npm dependency. Content stays in the light DOM so it is crawlable and works
  with JavaScript disabled; the scripts only add behaviour (mobile menu, theme
  and palette persistence, filtering, scroll-reveal, view transitions, form
  submission).
- **Fonts**: Public Sans, Newsreader and JetBrains Mono, self-hosted as
  subsetted woff2 files under `internal/site/static/fonts`. Nothing is
  requested from a font CDN.
- **Content**: `internal/data/cv.go` is the single source of truth. Templates
  render from it — edit the Go values, not the HTML, to update the CV.
- **Email**: `internal/mail` sends contact form submissions through Resend or
  Postmark (or both, with failover) over plain `net/http` — no SDK, no
  dependency.
- **PDF**: `internal/pdf` is a PDF writer built on the standard library, and
  `internal/cvpdf` lays the CV out with it. See
  [The PDF writer](#the-pdf-writer) below for why it is written rather than
  imported.

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

`POST /contact` sits behind a per-IP token bucket (`internal/site/ratelimit.go`)
with a capped tracking map, so neither the endpoint nor the inbox behind it can
be flooded by a single client.

## Structure

```
cmd/server/main.go         entrypoint, graceful shutdown
cmd/pdfgen/main.go         writes internal/site/static/cv.pdf from the CV data
internal/data/cv.go        CV content (edit this to update the site)
internal/mail/             Resend + Postmark senders, failover chain, env config
internal/pdf/              minimal stdlib PDF writer (no dependencies)
internal/cvpdf/            CV page layout, built on internal/pdf
internal/site/site.go      routes, embed directives, contact handler
internal/site/security.go  CSP and the rest of the security headers
internal/site/static.go    embedded asset handler: ETags, content types, 304s
internal/site/ratelimit.go per-IP token bucket for POST /contact
internal/site/seo.go       canonical URLs, JSON-LD, robots.txt, sitemap.xml
internal/site/email.go     HTML + text rendering of a contact message
internal/site/templates/   html/template files (layout + one per page)
internal/site/static/      css + js + fonts + icons + cv.pdf, all embedded
```

Beyond the content pages, the server answers `GET /cv.pdf`, `GET /robots.txt`,
`GET /sitemap.xml`, `GET /healthz`, `GET /favicon.ico`, `GET /static/…` and
`POST /contact`.

## Security headers

`internal/site/security.go` sets a Content-Security-Policy that denies
everything by default and carries no `unsafe-` token: every script ships as an
external module under `/static`, and no template sets a `style="…"` attribute.
Adding one would mean loosening `style-src` for the whole site, so do not add
one.

## The PDF writer

`internal/pdf` is about 600 lines of PDF 1.4 (ISO 32000-1) on the standard
library: indirect objects, one Flate-compressed content stream per page, a
cross-reference table, WinAnsi text in the base-14 fonts, and real link
annotations. `internal/cvpdf` lays the CV out with it, reading `internal/data`
directly so the PDF cannot say anything the site does not.

It is written rather than imported because this project has no other
third-party module, and a PDF library would have been the first. Adding one is
not a cost paid once at import — it is paid on every advisory, every upgrade
and every SBOM review, for the life of the product, to produce a document that
needs a small fraction of what a general-purpose library installs.

The security claim worth making is narrow: **this package writes and never
reads.** It parses no PDF, accepts no file, and takes no input that did not
come from a Go source file in this repository. The vulnerability class that
makes PDF handling dangerous belongs to parsers consuming documents from
strangers. There is no parser here. Generation also runs at build time, so no
PDF code is reachable from a request at all.

"We wrote it ourselves, so it is safer" is *not* the claim. Hand-written code
has its own defects and no upstream security team, and `SECURITY.md` response
windows apply to this package in full.

It is not a general PDF library and does not try to be. No embedded font
programs, no colour space beyond DeviceRGB, no forms, no tagging, no
encryption. If any of those are needed, this package is the wrong tool and a
real library is the answer.

Output is byte-for-byte deterministic (`TestOutputIsDeterministic`), so the
committed file diffs meaningfully and CI can prove it matches the data.

Full reasoning, the alternatives weighed, and the conditions that would reverse
the decision: [ADR-0001](docs/adr/0001-write-the-cv-pdf-with-the-standard-library.md).

## The CV download

`GET /cv.pdf` serves the generated file through the same static handler as
everything else, so it carries an ETag and answers a repeat visit with a 304.
It is sent `inline` with a `Content-Disposition` filename, so the browser's
viewer opens it while the `download` attribute on the link still saves it as
`Barry-Prendergast-CV.pdf`.

The PDF is an abridged version of the site, not a copy of it. The site scrolls
for free; the PDF is what a recruiter reads in half a minute, so it holds two
pages. Every employer still appears — only the detail is cut, so nothing on the
site is silently missing from the PDF. The editorial limits are the constants
at the top of `internal/cvpdf/cvpdf.go`, and `TestFitsTwoPages` fails if added
copy pushes the document to a third page.

Set in Helvetica rather than the site's Public Sans: embedding a real typeface
would mean parsing woff2 (Brotli) and subsetting TrueType, neither of which is
in the standard library. The text is real selectable text either way, which is
what applicant tracking systems parse.

Printing a page from the browser is handled separately by the `print` cascade
layer in `style.css` — it drops the nav, palette, filters and contact form,
expands every collapsed timeline entry, and forces a light palette.

## Deployment

`Dockerfile` and `fly.toml` are in the repository. `fly deploy` builds the image
and ships it to the `barrypre-web` app in `ams`.

The image is two stages: a pinned `golang:1.25.13-alpine` builder that produces
a static binary, and `distroless/static-debian12:nonroot` to run it — no shell,
no package manager, no libc, running as uid 65532.

`fly.toml` sets `FORCE_HSTS=1`. That is only correct because `force_https`
guarantees the browser's leg is HTTPS; Fly terminates TLS at its proxy, so
`r.TLS` is nil inside the app and `securityHeaders` refuses to infer HTTPS from
a forwarded header any client could set. Do not set it anywhere without
`force_https`.

The contact form stays fail-closed in production until `CONTACT_FROM` and at
least one provider key are set as Fly secrets and the sending domain is
verified with that provider.

## Verification and compliance

- `.github/workflows/ci.yml` — format, vet, build, tests with coverage, skip
  detection, `go generate` reproducibility, `govulncheck`, SBOM freshness, and a
  container image build. Every job fails the build. Nothing warns.
- `SECURITY.md` — what the product holds, the worst plausible defect, reporting
  address and response windows.
- `docs/technical-documentation.md` — CRA technical documentation: intended use,
  risk assessment, standards applied, support lifecycle.
- `sbom.json` — CycloneDX 1.6, regenerated by CI and diffed against the
  committed copy. The regeneration command is in `SECURITY.md`.
- `docs/adr/` — decision records, for the choices that would otherwise have to
  be reconstructed from source comments.

The Go version is pinned in three places — `go.mod`, `Dockerfile` and
`GO_VERSION` in the workflow. **Raise all three together.** The floor is 1.25.13
because `govulncheck` reports 22 reachable standard-library vulnerabilities
below it.

Coverage is 94.7% of statements across `./internal/...`, measured the way CI
measures it:

```
go test ./... -count=1 -coverpkg=./internal/... -coverprofile=coverage.out
go tool cover -func=coverage.out | tail -1
```

`-coverpkg` matters here: per-package coverage reports `internal/data` at 0%,
because it holds no logic of its own and is reached through the site handlers.

## Reporting a security issue

See `SECURITY.md`. Please do not open a public issue for a vulnerability.

## Licence

The code is MIT licensed — see `LICENSE`.

The CV content is not. `internal/data/cv.go`, the generated `cv.pdf`, the
fixtures in the tests, and the personal details throughout are Barry
Prendergast's own, and the MIT grant does not extend to them. Reuse the site,
not the CV.

The bundled fonts carry their own licence: Public Sans, Newsreader and
JetBrains Mono are all SIL Open Font License 1.1. The text and the copyright
notices ship beside them in `internal/site/static/fonts/OFL.txt`, which is
embedded with the fonts and served at `/static/fonts/OFL.txt`.
