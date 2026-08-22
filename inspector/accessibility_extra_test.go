// Copyright (c) 2026 Jarvis Friends contributors
// SPDX-License-Identifier: MIT

package inspector

import (
	"image/color"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/jarvisfriends/snap/styles"
)

// TestSuggestFGRejectsInvalidColors covers the conversion guards: a fully
// transparent color cannot be converted, on either side of the pair.
func TestSuggestFGRejectsInvalidColors(t *testing.T) {
	t.Parallel()

	transparent := color.RGBA{}
	opaque := color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}

	if fg, hex := suggestFG(transparent, opaque, 4.5); fg != nil || hex != "" {
		t.Fatalf("transparent fg: got %v/%q; want nil/empty", fg, hex)
	}
	if fg, hex := suggestFG(opaque, transparent, 4.5); fg != nil || hex != "" {
		t.Fatalf("transparent bg: got %v/%q; want nil/empty", fg, hex)
	}

	// An achievable target returns a suggestion; an impossible one gives up.
	if fg, hex := suggestFG(
		color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff}, opaque, 4.5,
	); fg == nil || hex == "" {
		t.Fatal("achievable contrast should yield a suggestion")
	}
	if fg, hex := suggestFG(opaque, opaque, 25); fg != nil || hex != "" {
		t.Fatalf("impossible contrast: got %v/%q; want nil/empty", fg, hex)
	}
}

// TestAccessibilityPanelIgnoresNonKeyMsgWhileVisible covers the visible-panel
// guard for non-key messages.
func TestAccessibilityPanelIgnoresNonKeyMsgWhileVisible(t *testing.T) {
	t.Parallel()

	p := NewAccessibilityPanel()
	p.Toggle() // visible
	if _, cmd := p.Update(tea.WindowSizeMsg{Width: 80, Height: 24}); cmd != nil {
		t.Fatal("non-key msg should be ignored while visible")
	}
}

// TestAccessibilityPanelViewScrollFollowsCursor covers the list windowing:
// the offset tracks the cursor when it moves below and back above the
// visible rows, and unchecked filter boxes render empty.
func TestAccessibilityPanelViewScrollFollowsCursor(t *testing.T) {
	t.Parallel()

	_ = styles.SetCurrentTint("dracula")
	p := NewAccessibilityPanel()
	p.SetColors(styles.Active())
	p.SetSize(100, 7) // tiny list: 3 visible rows
	p.Toggle()
	if len(p.view) < 5 {
		t.Skipf("need at least 5 filtered themes; have %d", len(p.view))
	}

	// Default filters leave the CVD checkboxes unchecked.
	out := p.View().Content
	if !strings.Contains(out, "[ ]") {
		t.Fatal("expected unchecked filter boxes in the header")
	}

	// Cursor below the window: the offset must advance.
	p.cursor = len(p.view) - 1
	_ = p.View()
	if p.offset == 0 {
		t.Fatal("offset did not follow the cursor down")
	}

	// Cursor jumps back above the window: the offset must snap up.
	p.cursor = 0
	_ = p.View()
	if p.offset != 0 {
		t.Fatalf("offset = %d; want 0 after cursor moved to top", p.offset)
	}
}

// TestAccessibilityPanelSelectedRowShowsIssues renders a failing theme under
// the cursor and asserts its issue details (and any fix suggestion) appear.
func TestAccessibilityPanelSelectedRowShowsIssues(t *testing.T) {
	t.Parallel()

	_ = styles.SetCurrentTint("dracula")
	p := NewAccessibilityPanel()
	p.SetColors(styles.Active())
	p.SetSize(100, 30)
	p.Toggle()

	// Include failing themes by checking every CVD filter.
	p.cvd = [3]bool{true, true, true}
	p.refilter()

	target := -1
	for vi, idx := range p.view {
		if len(p.all[idx].issues) > 0 {
			target = vi
			break
		}
	}
	if target < 0 {
		t.Skip("no theme with normal-vision issues in the registry")
	}
	p.cursor = target
	p.offset = 0
	out := p.View().Content
	if !strings.Contains(out, "contrast") {
		t.Fatalf("selected failing theme should list contrast issues:\n%.400s", out)
	}
}

// TestAccessibilityPanelEnterWithEmptyView asserts Enter is a no-op when the
// filters leave nothing to select.
func TestAccessibilityPanelEnterWithEmptyView(t *testing.T) {
	t.Parallel()

	p := NewAccessibilityPanel()
	p.Toggle()
	p.showDark = false
	p.showLight = false
	p.refilter()
	if len(p.view) != 0 {
		t.Fatalf("filters should empty the view; have %d", len(p.view))
	}
	p.cursor = 0
	if _, cmd := p.Update(tea.KeyPressMsg{Code: tea.KeyEnter}); cmd != nil {
		t.Fatal("Enter with an empty view must not emit a cmd")
	}
}
