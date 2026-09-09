# Security policy

## What this product is, and what it holds

barrypre.com is a personal CV website. It is one static Go binary with the
templates, stylesheet, fonts and generated PDF compiled into it. There is no
database, no user account, no session, no upload path and no admin surface.
Nothing is written to disk at runtime.

It processes exactly one piece of untrusted input: a contact form submission —
a name, an email address and a message — which is rendered into an email and
handed to Resend or Postmark over TLS. The message is not stored. It is written
to the log only when delivery failed, so that an undelivered message stays
recoverable.

Two credentials exist: the Resend and Postmark API keys. They are held as Fly
secrets and are never in this repository.

## The worst plausible defect

**Header injection through the contact form.** The visitor's address becomes the
`Reply-To` of an outgoing email. That is the one piece of visitor input that
reaches an email header, and a defect there turns this site into a relay for
someone else's mail. It is parsed with `net/mail` and rejected unless it is a
bare address; `internal/site/email_test.go` is the check that keeps it so.

After that, in order:

- **Cross-site scripting in the rendered CV or the email template.** All output
  goes through `html/template`, and a change that reaches for `template.HTML`
  is the change to look at hardest.
- **Credential disclosure.** A leaked provider key lets someone send mail as the
  verified domain. Rotation is immediate and is the first response, not a
  follow-up.
- **Abuse of the contact endpoint** to flood the inbox behind it. Bounded by a
  per-IP token bucket in `internal/site/ratelimit.go`. That state is per
  process, which is correct for a single small instance and would need a shared
  store if this ever ran on more than one.

A defect here does not expose anybody else's data, because there is none. The
realistic loss is reputational — mail sent in this domain's name.

## Reporting a vulnerability

Email **barryprendergast78@gmail.com** with `SECURITY` in the subject.

Please include what you did, what happened, and the URL or request. A proof of
concept is welcome and is never treated as an attack. Please do not open a
public GitHub issue for a suspected vulnerability.

There is no bounty. There is an acknowledgement in the release notes if you
want one, and a reply either way.

## Response windows

These are the windows this project commits to. They are what one person
maintaining one small site can actually meet, which is why they are not shorter.

| Stage | Window |
| --- | --- |
| Acknowledge your report | 3 working days |
| First assessment, with a severity and a plan | 10 working days |
| Fix or mitigation — critical or high | 14 days from assessment |
| Fix or mitigation — medium or low | Next release, at most 90 days |
| Public disclosure | Coordinated with you, by default 90 days after the report |

If a vulnerability in a released version is found to be **actively exploited**,
the EU early-warning obligation applies and the notification clock to ENISA
starts at **discovery, not at triage** — 24 hours for the early warning. That
clock is shorter than every window in the table above and overrides them.

## Supported versions

The deployed site at barrypre.com is the only supported version. There are no
release branches and no back-ports: a fix ships as a new deployment of `main`.

Intended support end date and the rest of the lifecycle commitment are recorded
in [`docs/technical-documentation.md`](docs/technical-documentation.md).

## How this repository defends itself

Every one of these fails the build rather than warning. See
`.github/workflows/ci.yml`.

| Gate | What it catches |
| --- | --- |
| `gofmt -l`, `go vet` | Formatting drift, and the vet checks including printf and struct tags |
| `go test -count=1` with `-coverpkg=./internal/...` | Regressions, uncached; coverage is 94.7% of internal statements |
| Skip detection | A test that quietly opted out on a missing environment variable |
| `go generate` reproducibility | A committed `cv.pdf` that no longer matches its generator |
| `govulncheck ./...` | Known vulnerabilities. This project has no third-party dependencies, so every finding is a standard-library one and the fix is raising the pinned Go version |
| SBOM freshness | A committed `sbom.json` that has fallen behind the build |
| `docker build` | A Dockerfile whose static-binary and distroless assertions have stopped holding |

The Go version is pinned in three places — `go.mod`, `Dockerfile`, and
`GO_VERSION` in the workflow. Raise all three together. The floor is **1.25.13**
because `govulncheck` reports 22 reachable standard-library vulnerabilities
below it.

What these gates do not do: they are not a threat model, and they do not make
untested code correct. They catch what they are pointed at.

## Regenerating the SBOM

`sbom.json` is CycloneDX 1.6, generated under the build constraints of the
deployed container rather than a developer's machine, because those constraints
decide module selection. Two volatile fields — the generation timestamp and the
main component's commit-derived pseudo-version — are normalised away so the
freshness check compares content and not noise.

```
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  cyclonedx-gomod app -json -std -noserial -main ./cmd/server -output - \
  | sed -E '/^    "timestamp": /d; s/v0\.0\.0-[0-9]{14}-[0-9a-f]{12}/v0.0.0-devel/g' \
  > sbom.json
```

Install the generator at the pinned version CI uses:

```
go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@v1.9.0
```
