package inspector

import "testing"

// TestInspectorAccessors exercises the small getters/setters the router uses to
// wire the inspector into a host app. They are pure state accessors, so a bare
// New() model is enough — this pins them against silent regressions and keeps
// the trivial-but-real wiring surface covered.
func TestInspectorAccessors(t *testing.T) {
	m := New()

	// Visibility toggles, and providers start/stop with it.
	if m.IsVisible() {
		t.Fatal("new inspector should start hidden")
	}
	m.ToggleVisible()
	if !m.IsVisible() {
		t.Fatal("ToggleVisible should show the inspector")
	}
	m.ToggleVisible()
	if m.IsVisible() {
		t.Fatal("ToggleVisible should hide the inspector again")
	}

	// SetColorProfileEnvVar: empty falls back to the default name.
	m.SetColorProfileEnvVar("MY_APP_COLOR")
	if m.colorProfileEnvVar != "MY_APP_COLOR" {
		t.Errorf("env var = %q, want MY_APP_COLOR", m.colorProfileEnvVar)
	}
	m.SetColorProfileEnvVar("")
	if m.colorProfileEnvVar != defaultColorProfileEnvVar {
		t.Errorf("empty env var should fall back to default, got %q", m.colorProfileEnvVar)
	}

	// SetLogCapacity clamps: 0 stays 0, sub-10 floors to 10, else verbatim.
	for _, tc := range []struct{ in, want int }{{0, 0}, {5, 10}, {250, 250}} {
		m.SetLogCapacity(tc.in)
		if m.logCapacity != tc.want {
			t.Errorf("SetLogCapacity(%d) => %d, want %d", tc.in, m.logCapacity, tc.want)
		}
	}

	// Status-summary toggles and the link-rate injector.
	m.SetStatusSummaryEnabled(false)
	if m.StatusSummaryEnabled() {
		t.Error("StatusSummaryEnabled should be false after disabling")
	}
	m.SetStatusSummaryEnabled(true)
	if !m.StatusSummaryEnabled() {
		t.Error("StatusSummaryEnabled should be true after enabling")
	}
	// Default model has ShowLink=true, so once enabled the link summary is wanted.
	if !m.StatusSummaryLinkEnabled() {
		t.Error("StatusSummaryLinkEnabled should be true when enabled and ShowLink is set")
	}
	m.SetLinkRateSummary(func() string { return "tx 1 B/s" })
	if m.linkSummary == nil {
		t.Error("SetLinkRateSummary should install the summary function")
	}
}

// TestInspectorHelpBindings covers ShortHelp/FullHelp for every tab (the
// help.KeyMap surface shown in the status bar) plus the accessibility keymap's
// own help methods.
func TestInspectorHelpBindings(t *testing.T) {
	m := New()
	for _, tab := range []debugTab{
		debugTabRuntime, debugTabInput, debugTabDisks, debugTabTerminal,
		debugTabAccessibility, debugTabLog, debugTabSettings,
	} {
		m.activeTab = tab
		if len(m.ShortHelp()) == 0 {
			t.Errorf("ShortHelp empty for tab %v", tab)
		}
		if len(m.FullHelp()) == 0 {
			t.Errorf("FullHelp empty for tab %v", tab)
		}
	}

	km := DefaultAccessibilityKeyMap()
	if len(km.ShortHelp()) == 0 {
		t.Error("accessibility ShortHelp should not be empty")
	}
	if len(km.FullHelp()) == 0 {
		t.Error("accessibility FullHelp should not be empty")
	}
}
