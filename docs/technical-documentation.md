# Technical documentation — barrypre.com

Kept for the EU Cyber Resilience Act. It is updated on every material release;
"material" means a change to what the product does, what it processes, what it
depends on, or how it is deployed.

| | |
| --- | --- |
| Product | barrypre.com — personal CV website |
| Manufacturer | Barry Prendergast |
| Repository | https://github.com/Barryprender/web-cv |
| Deployment | Fly.io, app `barrypre-web`, region `ams` |
| This revision | 2026-09-09 |
| First release | 2026-08-23 |
| Support end date | 2031-08-23 (five years from first release) |

## Product description

A single stateless HTTP server that renders a CV as a website, serves a
generated PDF of the same CV, and accepts contact form submissions which it
forwards as email.

It is one Go binary. The templates, stylesheet, fonts, icons and the generated
`cv.pdf` are compiled into it with `embed.FS`, so the running artifact is one
file with no filesystem dependencies. There is no database, no cache, no
session, no user account, and nothing is written to disk at runtime.

The CV content lives in `internal/data/cv.go` and is the single source of truth;
the site and the PDF are both rendered from it.

## Intended use

Public, unauthenticated, read-mostly browsing by recruiters and hiring managers,
plus an occasional contact form submission. Expected load is a few hundred
requests a day.

**Reasonably foreseeable misuse** the design accounts for:

- Automated scraping and crawling. Everything except the form is a `GET` of
  embedded content, so this costs nothing beyond bandwidth.
- Contact form spam. Bounded by a per-IP token bucket.
- Attempts to make the server render something expensive. The PDF is generated
  at build time and committed, not rendered per request, so there is no path by
  which a visitor causes generation work.

**Out of intended use:** anything authenticated, anything storing personal data
belonging to a third party, and running on more than one instance without
replacing the in-process rate limiter with a shared store.

## Architecture and dependencies

Go 1.25.13 standard library only. There are no third-party modules; `go.mod`
has no `require` block. `sbom.json` records this in CycloneDX 1.6 form,
including the standard library as a component so a toolchain change shows up
as an SBOM change.

| Component | Role |
| --- | --- |
| `cmd/server` | Entrypoint, graceful shutdown |
| `cmd/pdfgen` | Build-time generator for `internal/site/static/cv.pdf` |
| `internal/data` | CV content, single source of truth |
| `internal/site` | Routes, templates, static assets, contact handler, rate limiter, security headers, SEO |
| `internal/mail` | Resend and Postmark senders with failover, environment configuration |
| `internal/pdf` | Minimal PDF writer on the standard library |
| `internal/cvpdf` | CV page layout, built on `internal/pdf` |

Runtime services: Fly.io for hosting and TLS termination; Resend and/or
Postmark for outbound email. No other outbound network calls are made.

## Risk assessment summary

The full reasoning, including the worst plausible defect, is in
[`../SECURITY.md`](../SECURITY.md). In summary:

| Risk | Severity | Control |
| --- | --- | --- |
| Email header injection through the `Reply-To` field | High | Parsed with `net/mail`, rejected unless a bare address; covered by `internal/site/email_test.go` |
| Cross-site scripting in a rendered page or email | High | All output through `html/template`; no `template.HTML` on visitor input |
| Provider credential disclosure | High | Keys held as Fly secrets, never in the repository; rotation is the first response |
| Contact endpoint abuse flooding the inbox | Medium | Per-IP token bucket, `internal/site/ratelimit.go`, capped at 4096 tracked IPs so the limiter is not itself an exhaustion vector |
| Silent loss of a real message | Medium | Fail-closed: no provider configured means submissions are refused, not accepted and dropped. A key without `CONTACT_FROM` refuses to start. Undelivered messages are logged in full |
| Transport downgrade | Medium | `force_https` at the Fly proxy; HSTS sent under the explicit `FORCE_HSTS` opt-in, because Fly forwards plain HTTP and a forwarded header cannot be trusted to infer it |
| Vulnerable standard library | Medium | Pinned Go floor of 1.25.13, enforced by `govulncheck` in CI |
| Compromise of the running container | Low | distroless/static image, no shell, no package manager, no libc; static binary; runs as uid 65532 with no write access needed |

No personal data belonging to third parties is stored. A contact form
submission exists only in flight and in the provider's own logs.

## Standards and obligations applied

- **EU Cyber Resilience Act (Regulation (EU) 2024/2847)** — this document, the
  SBOM, the vulnerability handling policy in `SECURITY.md`, the CI gates, and
  the support lifecycle below.
- **OWASP Top 10 (2021)** — applied as the runtime control set. Access control
  is trivial here (everything is public and read-only); the categories that
  carry weight are A03 Injection, A05 Security Misconfiguration and A09 Logging.
- **WCAG 2.2 AA** — the site is a CV, so it has to be readable by everyone
  reading CVs. Asserted by `internal/site/accessibility_test.go`.

## Verification

CI is `.github/workflows/ci.yml`. Every job fails the build; none warn. The
gates and what each catches are tabulated in `SECURITY.md`.

Test coverage is **95.0%** of statements across `./internal/...`. That number is
printed by the test job on every run. Coverage is not correctness — it says the
lines executed, not that the assertions were the right ones.

## Support lifecycle

| | |
| --- | --- |
| First release | 2026-08-23 |
| Support end date | 2031-08-23 |
| Support period | Five years |

Until that date, security fixes ship as a new deployment of `main`. There are no
release branches and no back-ports.

This is a personal CV site with one user, so five years is the CRA default
rather than a figure derived from a longer expected product life. If the site is
retired earlier, that is a decision to record here — with a date and a reason —
and not something to let lapse quietly.

## Revision history

| Date | Change |
| --- | --- |
| 2026-09-09 | First revision. Recorded alongside the CI, SBOM and `SECURITY.md` baseline, and the Go floor raised to 1.25.13. |
