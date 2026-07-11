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
