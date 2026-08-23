package site

import (
	"regexp"
	"strings"
	"testing"

	"barrypre.com/webcv/internal/data"
)

// The enhancement layer is additive by contract. These tests pin the half of
// that contract that lives in the server output: what a visitor gets before,
// or without, any JavaScript running.

// Interactive affordances must not appear in the HTML, because the behaviour
// behind them only exists once the modules run. A tag rendered as a <button>
// with no script would be a control that does nothing.
func TestFilterAffordanceIsAddedByScriptOnly(t *testing.T) {
	for _, path := range []string{"/experience", "/projects"} {
		t.Run(path, func(t *testing.T) {
			body := pageBody(t, path)

			if !strings.Contains(body, `<span class="tag" data-stack=`) {
				t.Error("stack tags are not rendered as inert spans")
			}
			for _, forbidden := range []string{
				`<button class="tag`, // upgraded client-side
				`class="filter-status"`,
				`class="tag-filter"`,
				`aria-pressed`,
			} {
				if strings.Contains(body, forbidden) {
					t.Errorf("server output contains script-only markup %q", forbidden)
				}
			}
			if !strings.Contains(body, `<cv-filter data-item=`) {
				t.Error("the filter host element is missing")
			}
		})
	}
}

// The palette is a native popover: its trigger and every destination must work
// with scripting disabled. Only the search field is script-added.
func TestPaletteWorksWithoutScript(t *testing.T) {
	body := pageBody(t, "/")

	for _, want := range []string{
		`<div id="command-palette" popover class="palette"`,
		`popovertarget="command-palette"`,
		`aria-label="Quick jump"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("page is missing %s", want)
		}
	}
	if strings.Contains(body, "palette-search") {
		t.Error("the search field must be created by script, not rendered inert")
	}

	// Every page, job and project should be reachable from the panel.
	items := strings.Count(body, `class="palette-item"`)
	want := len(pages) + len(data.Me.Jobs) + len(data.Me.Projects)
	if items != want {
		t.Errorf("palette lists %d destinations, want %d", items, want)
	}
}

// Every href inside the palette must resolve to a real route, and every
// fragment must match an id on the page it points at.
func TestPaletteDestinationsResolve(t *testing.T) {
	body := pageBody(t, "/")
	hrefs := regexp.MustCompile(`class="palette-item" href="([^"]+)"`).FindAllStringSubmatch(body, -1)
	if len(hrefs) == 0 {
		t.Fatal("no palette destinations found")
	}

	for _, m := range hrefs {
		route, fragment, _ := strings.Cut(m[1], "#")
		if _, ok := pages[route]; !ok {
			t.Errorf("palette links to %q which is not a route", route)
			continue
		}
		if fragment == "" {
			continue
		}
		if target := pageBody(t, route); !strings.Contains(target, `id="`+fragment+`"`) {
			t.Errorf("palette links to %s but %s has no element with that id", m[1], route)
		}
	}
}

// The reveal wrapper is what aria-controls points at and what collapses, so it
// has to carry the id and wrap an overflow box.
func TestTimelineRevealStructure(t *testing.T) {
	body := pageBody(t, "/experience")

	if n := strings.Count(body, `class="timeline-reveal" id="job-`); n == 0 {
		t.Fatal("no reveal wrappers rendered")
	}
	if strings.Contains(body, `<ul class="timeline-bullets" id="job-`) {
		t.Error("the id is still on the list rather than the collapsing wrapper")
	}
	if n := strings.Count(body, `class="timeline-reveal-inner"`); n != strings.Count(body, `class="timeline-reveal" id="job-`) {
		t.Error("every reveal wrapper needs an inner overflow box")
	}
}

// @view-transition is only valid at the top level of a stylesheet or inside a
// conditional group rule. @layer is neither, and nesting it there parses but is
// not guaranteed to be honoured.
func TestViewTransitionRuleIsTopLevel(t *testing.T) {
	css := pageBody(t, "/static/css/style.css")

	index := strings.Index(css, "@view-transition")
	if index == -1 {
		t.Fatal("no @view-transition rule in the stylesheet")
	}

	// Walk the braces before the rule; a non-zero depth means it is nested.
	depth := 0
	for _, r := range css[:index] {
		switch r {
		case '{':
			depth++
		case '}':
			depth--
		}
	}
	if depth != 0 {
		t.Errorf("@view-transition is nested %d level(s) deep, want top level", depth)
	}
}

// Motion is opt-in: every animation the enhancement layer adds sits behind a
// no-preference query, and the reduce block neutralises what is left.
func TestMotionRespectsReducedMotion(t *testing.T) {
	css := pageBody(t, "/static/css/style.css")

	reduce := strings.Index(css, "@media (prefers-reduced-motion: reduce)")
	if reduce == -1 {
		t.Fatal("no reduced-motion block")
	}
	for _, want := range []string{
		"::view-transition-group(*)",
		"animation-duration: 0.01ms !important",
		"transition-duration: 0.01ms !important",
	} {
		if !strings.Contains(css[reduce:], want) {
			t.Errorf("reduced-motion block is missing %q", want)
		}
	}

	// The hero sequence and the scroll spine must never run under reduce.
	for _, gated := range []string{"hero-type", "spine-draw"} {
		at := strings.Index(css, "@keyframes "+gated)
		if at == -1 {
			t.Errorf("missing @keyframes %s", gated)
			continue
		}
		if !strings.Contains(css[:at], "prefers-reduced-motion: no-preference") {
			t.Errorf("%s is not gated behind a no-preference query", gated)
		}
	}
}
