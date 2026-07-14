package inspector

import (
	"os"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/jarvisfriends/snap/rendercheck"
	"github.com/jarvisfriends/snap/styles"
	tint "github.com/lrstanley/bubbletint/v2"
)

func TestMain(m *testing.M) {
	tint.NewDefaultRegistry()
	os.Exit(m.Run())
}

func TestInspectorLayoutOverflows(t *testing.T) {
	_ = styles.SetCurrentTint("dracula")
	m := New()
	m.SetColors(styles.Active())

	rendercheck.CheckNoLineOverflowAtSizes(t, m)
}

var ansiRE = regexp.MustCompile("\x1b\\[[0-9;]*m")

// TestInspectorFillsViewport guards the standalone inspector's vertical layout:
// run on its own it is a full-screen bordered frame, so it must fill the
// viewport EXACTLY at every size. Regression test for the bordered frame
// rendering Height()+2 rows (the outer border's vertical frame was subtracted
// from the width but not the height), which clipped the bottom border and last
// row. CheckNoLineOverflowAtSizes only checks widths, so it missed this.
//
// This asserts the invariant inline so the inspector keeps building against the
// released snap; snap/rendercheck.CheckFillsViewport is the reusable form
// (available from the next snap release).
func TestInspectorFillsViewport(t *testing.T) {
	_ = styles.SetCurrentTint("dracula")
	m := New()
	m.SetColors(styles.Active())

	sizes := []struct{ w, h int }{
		{80, 24}, {100, 30}, {120, 50}, {89, 78}, {60, 20}, {200, 50},
	}
	for _, sz := range sizes {
		var model tea.Model = m
		model, _ = model.Update(tea.WindowSizeMsg{Width: sz.w, Height: sz.h})
		// Full-screen frame doesn't end in a newline; tolerate one if present.
		lines := strings.Split(strings.TrimSuffix(model.View().Content, "\n"), "\n")
		if len(lines) != sz.h {
			t.Errorf("%dx%d: rendered %d rows, want exactly %d (frame over/underflows — bottom clipped or blank)",
				sz.w, sz.h, len(lines), sz.h)
			continue
		}
		if strings.TrimSpace(ansiRE.ReplaceAllString(lines[sz.h-1], "")) == "" {
			t.Errorf("%dx%d: bottom row blank — frame does not reach the last row (missing bottom border)", sz.w, sz.h)
		}
		for i, ln := range lines {
			if gw := lipgloss.Width(ln); gw > sz.w {
				t.Errorf("%dx%d: line %d width %d exceeds %d", sz.w, sz.h, i, gw, sz.w)
			}
		}
	}
}
