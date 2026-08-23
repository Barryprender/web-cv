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

	"barrypre.com/webcv/internal/cvpdf"
)

func main() {
	out := flag.String("o", filepath.Join("static", "cv.pdf"), "path to write the PDF to")
	flag.Parse()

	if err := run(*out); err != nil {
		fmt.Fprintln(os.Stderr, "pdfgen:", err)
		os.Exit(1)
	}
}

func run(out string) error {
	// Written through a temporary file in the same directory and renamed, so
	// an interrupted run cannot leave a truncated PDF in place of a good one.
	dir := filepath.Dir(out)
	tmp, err := os.CreateTemp(dir, ".pdfgen-*")
	if err != nil {
		return fmt.Errorf("create temp file in %s: %w", dir, err)
	}
	defer os.Remove(tmp.Name()) // no-op once the rename below succeeds

	if _, err := tmp.Write(cvpdf.Build()); err != nil {
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
