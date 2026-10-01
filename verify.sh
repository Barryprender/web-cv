#!/bin/sh
# Every gate CI applies, run locally, in one command.
#
#   sh verify.sh          every gate
#   sh verify.sh --fast   the cheap tier only, for the edit loop
#
# CI runs this script rather than a copy of it (.github/workflows/ci.yml), so a
# new check is added here and nowhere else.
#
# `--fast` is a contract: the edit and Stop hooks in ~/.claude/hooks run it
# after every edit and at the end of every turn, so it has to finish in seconds.
# It keeps formatting, vet and the check that static/cv.pdf matches
# internal/data -- the mistake most likely to be made in an edit -- and drops
# the full suite, the vulnerability scan, the SBOM diff and the image build.
#
# Three outcomes, because two would be a lie:
#
#   0  every check ran and passed
#   1  a check failed
#   2  every check that ran passed, but one could not run
set -eu

FAST=""
case "${1:-}" in
"") ;;
--fast) FAST=1 ;;
*)
    sed -n '2,5p' "$0" | sed 's/^# \{0,1\}//'
    exit 2
    ;;
esac

cd "$(dirname "$0")"

# go.mod's `go` line is a minimum, so a newer local Go would build, scan and
# describe a standard library CI never sees. Pin the run to it; Go downloads
# that toolchain once and caches it.
GOTOOLCHAIN="go$(sed -n 's/^go //p' go.mod | tr -d '\r')"
export GOTOOLCHAIN

LOGS="$(mktemp -d)"
trap 'rm -rf "$LOGS"' EXIT

FAILED=""
UNRUN=""

# Run a check, keep its output, and show that output only when it fails.
check() {
    name="$1"
    shift
    printf '%-26s' "$name"
    if "$@" >"$LOGS/$name.log" 2>&1; then
        echo "ok"
    else
        echo "FAIL"
        sed 's/^/    /' "$LOGS/$name.log"
        FAILED="$FAILED $name"
    fi
}

# A check that cannot run is recorded as not having run, never as having passed.
unrun() {
    printf '%-26s%s\n' "$1" "NOT RUN -- $2"
    UNRUN="$UNRUN $1"
}

have() { command -v "$1" >/dev/null 2>&1; }

# --- the checks --------------------------------------------------------------

# gofmt reports on stdout and exits 0 either way.
fmt() {
    unformatted="$(gofmt -l .)"
    [ -z "$unformatted" ] && return 0
    echo "gofmt would change these files:"
    echo "$unformatted"
    return 1
}

vet() { go vet ./...; }

pdf_current() { go test ./internal/site/ -count=1 -run '^TestPDFIsCurrent$'; }

build() { go build ./...; }

# One run gives both the coverage and the verbose output the skip check reads.
# -coverpkg=./internal/... so internal/data reached through the site handlers
# counts; per-package coverage would report it at zero.
tests() {
    go test ./... -count=1 -v -coverpkg=./internal/... -coverprofile="$LOGS/coverage.out" >"$LOGS/verbose.txt" 2>&1 || {
        grep -E '^(--- FAIL|FAIL|panic)' "$LOGS/verbose.txt"
        return 1
    }
    go tool cover -func="$LOGS/coverage.out" | tail -1
}

# A skipped test reports as a pass. Exactly one may skip: TestRenderSample, a
# design aid that writes an email sample. Any other skip is a real test that
# stopped running.
no_unexpected_skips() {
    unexpected="$(grep -- "--- SKIP" "$LOGS/verbose.txt" | grep -v "TestRenderSample" || true)"
    [ -z "$unexpected" ] && return 0
    echo "tests skipped that are not the documented sample writer:"
    echo "$unexpected"
    return 1
}

# TestPDFIsCurrent compares against the committed bytes. This proves the
# generator itself still reproduces them.
pdf_regenerates() {
    before="$(git hash-object internal/site/static/*.pdf)"
    go generate ./internal/site
    [ "$before" = "$(git hash-object internal/site/static/*.pdf)" ] && return 0
    echo "go generate ./internal/site changed the committed PDFs; the working tree now holds the regenerated files"
    return 1
}

vulnerabilities() { govulncheck ./...; }

# The build constraints are the deployed container's: module selection depends
# on them, so an SBOM generated on Windows is not an SBOM of what ships. -std
# because the standard library is this project's entire dependency list. The
# sed drops the fields that change without the dependencies changing: the
# timestamp, the git-derived pseudo-version, and the generator's own hashes.
sbom() {
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
        cyclonedx-gomod app -json -std -noserial -main ./cmd/server -output - |
        sed -E '/^    "timestamp": /d; /^        "hashes": \[$/,/^        \],$/d; s/v0\.0\.0-[0-9]{14}-[0-9a-f]{12}/v0.0.0-devel/g' \
            >"$LOGS/sbom.json"
    # A Windows checkout gives sbom.json CRLF line endings; the content is what
    # is compared.
    tr -d '\r' <sbom.json >"$LOGS/committed.json"
    diff -u "$LOGS/committed.json" "$LOGS/sbom.json" && return 0
    echo "sbom.json is stale. Regenerate it with the command in SECURITY.md and commit the result."
    return 1
}

# The Dockerfile asserts a static binary and a distroless runtime with no
# shell. go build exercises neither.
image() { docker build -t barrypre-web:verify .; }

# --- the run -----------------------------------------------------------------

check formatting fmt
check vet vet
check pdf-current pdf_current

if [ -n "$FAST" ]; then
    echo
    if [ -n "$FAILED" ]; then
        echo "FAILED:$FAILED"
        exit 1
    fi
    echo "fast checks passed -- run without --fast before a push"
    exit 0
fi

check build build
check tests tests
sed -n 's/^total:/    total:/p' "$LOGS/tests.log"
check no-unexpected-skips no_unexpected_skips
check pdf-regenerates pdf_regenerates

if have govulncheck; then
    check vulnerabilities vulnerabilities
else
    unrun vulnerabilities "go install golang.org/x/vuln/cmd/govulncheck@v1.7.0"
fi

if have cyclonedx-gomod; then
    check sbom sbom
else
    unrun sbom "go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@v1.9.0"
fi

if ! have docker; then
    unrun image "Docker is not installed"
elif ! docker info >/dev/null 2>&1; then
    unrun image "Docker is not running"
else
    check image image
fi

echo
if [ -n "$FAILED" ]; then
    echo "FAILED:$FAILED"
    exit 1
fi
if [ -n "$UNRUN" ]; then
    echo "passed, but these could not run:$UNRUN"
    exit 2
fi
echo "all checks passed"
