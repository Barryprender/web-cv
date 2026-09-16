package cvpdf

import (
	"regexp"
	"strconv"
	"testing"

	"barrypre.com/webcv/internal/data"
)

// maxPages is the length a recruiter will actually read. The editorial limits
// at the top of cvpdf.go exist to hold the document to it.
const maxPages = 2

var pageCountRe = regexp.MustCompile(`/Count (\d+)`)

// TestFitsTwoPages fails when added CV content pushes the PDF past two pages.
//
// The failure is the point: it is a prompt to cut copy or lower one of the
// editorial limits, not to raise maxPages. A CV that grows a page every year
// is the thing this guards against.
func TestFitsTwoPages(t *testing.T) {
	// Checked per language: Spanish runs longer than English for the same
	// meaning, so a translation is the edit most likely to push the document
	// onto a third page.
	for _, lang := range data.Langs {
		t.Run(string(lang), func(t *testing.T) {
			m := pageCountRe.FindSubmatch(Build(lang))
			if m == nil {
				t.Fatal("no /Count in the page tree; the document structure changed")
			}

			pages, err := strconv.Atoi(string(m[1]))
			if err != nil {
				t.Fatalf("unreadable page count %q: %v", m[1], err)
			}
			if pages > maxPages {
				t.Errorf("CV is %d pages, maximum is %d: cut copy, or lower detailedRoles/jobBullets/projectBullets", pages, maxPages)
			}
		})
	}
}
