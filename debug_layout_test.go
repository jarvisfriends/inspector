package inspector

import (
	"os"
	"testing"

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

// TestInspectorFillsViewport guards the standalone inspector's vertical layout:
// run on its own it is a full-screen bordered frame, so it must fill the
// viewport EXACTLY at every size. Regression test for the bordered frame
// rendering Height()+2 rows (the outer border's vertical frame was subtracted
// from the width but not the height), which clipped the bottom border and last
// row. CheckNoLineOverflowAtSizes only checks widths, so it missed this.
func TestInspectorFillsViewport(t *testing.T) {
	_ = styles.SetCurrentTint("dracula")
	m := New()
	m.SetColors(styles.Active())

	rendercheck.CheckFillsViewport(t, m)
}
