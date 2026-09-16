# ADR-0001: Write the CV PDF with the standard library

## Status
**Accepted** — 16 September 2026.

The site carries roughly 600 lines of hand-written PDF machinery in
`internal/pdf` and `internal/cvpdf`, where a well-known module would have done
the same job. That is the kind of choice a reader assumes is an oversight, and
the reasoning for it currently survives only in package comments. It is
recorded here because the decision rests on a narrow argument that is easy to
overstate, and an overstated version of it would justify hand-writing things
that should never be hand-written.

## Context

`GET /cv.pdf` is the one artifact a stranger downloads from this site and opens
in an application on their own machine. Everything else the product emits is
HTML rendered in a browser sandbox. That makes the PDF the highest-consequence
output in the project, and the code that produces it the piece most worth being
deliberate about.

Producing it from a library means adding the project's first and only
third-party module. Under the CRA posture this repository already keeps, that
is not a free action: the module enters `sbom.json`, becomes something
`govulncheck` must be able to reason about, and becomes a dependency whose
transitive graph has to be reviewed on every upgrade, permanently, for a file
that changes when the CV changes — a few times a year.

The general-purpose PDF modules are general-purpose. They carry font parsing
and embedding, image decoding, and in several cases a reader as well as a
writer. None of that is reachable from this product, but all of it is
installed, and all of it is surface that an advisory can land on. The document
actually required is flowed text in three weights with horizontal rules and
hyperlinks. No images, no transparency, no embedded font programs.

The risk being managed is therefore not "a library might be badly written". It
is that the project would take on a permanent maintenance and disclosure
obligation, proportional to the whole of a dependency, to produce a document
that needs a small and well-specified fraction of it.

## Decision

Write the PDF with the standard library. `internal/pdf` emits PDF 1.4
(ISO 32000-1): indirect objects, one Flate-compressed content stream per page,
a cross-reference table, WinAnsi-encoded text in the base-14 fonts, and link
annotations. `internal/cvpdf` lays the CV out with it, reading
`internal/data` directly so the PDF cannot state anything the site does not.

Generation runs at build time. `cmd/pdfgen` writes
`internal/site/static/cv.pdf`, the file is committed, and the running server
only serves bytes out of its embedded filesystem. No PDF code executes in the
request path, and `TestPDFIsCurrent` fails if the committed file has fallen
behind the data.

The security argument for this is specific, and it is the only one that should
be made: the package writes and never reads. It parses no PDF, accepts no
file, and takes no input that did not come from a Go source file in this
repository. The vulnerability class that makes PDF handling dangerous belongs
to parsers consuming documents from strangers, and this product contains no
parser and consumes no document.

`internal/pdf` is explicitly not a general PDF library, and its package
documentation says so: no embedded font programs, no colour space beyond
DeviceRGB, no forms, no tagging, no encryption. If any of those become
necessary, this package is the wrong tool and a real library is the answer.

## Alternatives considered

**A third-party PDF module.** Correct for any project that must embed a
typeface, place images, or read an existing document. It was rejected because
this project needs none of those, and the cost of a dependency is not paid once
at import — it is paid on every advisory, every upgrade, and every SBOM review
for the life of the product. The ratio of installed capability to used
capability was the deciding factor, not the quality of any particular module.

**Rendering HTML to PDF with a headless browser.** Produces the best-looking
result and reuses the stylesheet that already exists. Rejected because it
replaces one Go module with a browser: a dependency several orders of magnitude
larger than the entire product, in the build pipeline of an artifact whose
appeal is that it is a single static binary with no filesystem dependencies.

**Serving a PDF exported by hand from a word processor.** The lowest-effort
option, and the one in use before this decision. Rejected because it removes
`internal/data` as the single source of truth. A hand-exported file drifts
silently from the site the first time the CV is edited and the export is
forgotten, and a CV that contradicts itself between two pages of the same
domain is a worse defect than any considered here.

**Embedding the site's own typefaces rather than using base-14.** Rejected for
now because it requires Brotli-decompressing woff2 and subsetting TrueType,
neither of which the standard library provides, and it would roughly double the
size of the package to change which typeface the document is set in. The text
is real selectable text under base-14, which is what applicant tracking systems
parse, so the functional requirement is already met.

## Consequences

**Negative.** There is no upstream. Every defect in `internal/pdf` is this
project's to find and fix, with no security team watching the format on its
behalf, and the CRA response windows in `SECURITY.md` apply to it in full.

The document is set in Helvetica rather than the site's Public Sans, so the PDF
and the website do not look like the same artifact. This is the most visible
cost of the decision and it is paid on every download.

Characters outside WinAnsi cannot be represented. `internal/pdf/encoding.go`
maps what it can, substitutes plain-text stand-ins where an unambiguous one
exists, and renders anything remaining as a visible `?` so it surfaces in
review rather than vanishing. A CV in a language needing a different script
would not be servable by this code at all.

The layout engine is a cursor and a wrap function. There is no float, no table,
and no widow or orphan control, so `internal/cvpdf` is sensitive to content
length in ways a real layout engine would absorb.

**Positive.** `go.mod` has no `require` block, and `sbom.json` records a
product whose only component is the Go standard library. A dependency advisory
cannot affect this product without being a standard-library advisory, which is
already handled by the pinned `GO_VERSION` and the `govulncheck` gate in CI.

No PDF code is reachable from a request. The generation path is a build-time
command, so no visitor input reaches it and no visitor can cause generation
work.

Output is byte-for-byte deterministic, verified by `TestOutputIsDeterministic`,
so the committed file diffs meaningfully and CI can prove the artifact matches
the data with `go generate` rather than trusting that someone remembered.

## Follow-up

1. Reverse this decision if the CV must carry an image, a non-Latin script, or
   the site's own typeface. Each of those needs code the standard library does
   not provide, and the balance of the argument in *Context* moves to the
   library at that point.
2. Reverse it if `internal/pdf` grows past roughly a thousand lines or acquires
   a reader. Either indicates it is no longer the narrow writer that the
   security argument in *Decision* depends on.
