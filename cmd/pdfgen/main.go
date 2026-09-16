// Command pdfgen renders the CV data to the PDF the site serves.
//
// It runs at build time, not at request time: the generated file is committed
// and embedded, so the server never renders a PDF for a visitor and an
// anonymous GET cannot make it do work. Run it after editing internal/data:
//
//	go generate ./internal/site
//
// TestPDFIsCurrent in internal/site fails if the committed file is stale, so a
// forgotten regeneration is caught by the test suite rather than shipped.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"barrypre.com/webcv/internal/cvpdf"
	"barrypre.com/webcv/internal/data"
)

func main() {
	out := flag.String("o", filepath.Join("static", "cv.pdf"), "path to write the English PDF to")
	flag.Parse()

	// One invocation writes every language. The other files are named from the
	// English path, so the go:generate line stays a single command and no
	// language can be left behind by a run that only listed some of them.
	for _, lang := range data.Langs {
		if err := run(pathFor(*out, lang), lang); err != nil {
			fmt.Fprintln(os.Stderr, "pdfgen:", err)
			os.Exit(1)
		}
	}
}

// pathFor derives a language's output path from the English one: cv.pdf stays
// as it is, and every other language gets a suffix before the extension.
//
// English keeps the unsuffixed name because that URL is already in circulation.
func pathFor(out string, lang data.Lang) string {
	if lang == data.EN {
		return out
	}
	ext := filepath.Ext(out)
	return strings.TrimSuffix(out, ext) + "-" + string(lang) + ext
}

func run(out string, lang data.Lang) error {
	// Written through a temporary file in the same directory and renamed, so
	// an interrupted run cannot leave a truncated PDF in place of a good one.
	dir := filepath.Dir(out)
	tmp, err := os.CreateTemp(dir, ".pdfgen-*")
	if err != nil {
		return fmt.Errorf("create temp file in %s: %w", dir, err)
	}
	defer os.Remove(tmp.Name()) // no-op once the rename below succeeds

	if _, err := tmp.Write(cvpdf.Build(lang)); err != nil {
		tmp.Close()
		return fmt.Errorf("write %s: %w", tmp.Name(), err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close %s: %w", tmp.Name(), err)
	}
	if err := os.Rename(tmp.Name(), out); err != nil {
		return fmt.Errorf("rename onto %s: %w", out, err)
	}
	return nil
}
