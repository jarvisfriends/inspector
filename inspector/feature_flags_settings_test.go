package inspector

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/jarvisfriends/snap/gate"
)

// TestFeatureFlagsAbsentWithoutGates asserts the Settings tab adds no
// Feature Flags section when the app registers no gates (or SetGates is
// never called) — the section is opt-in, not a fixed row.
func TestFeatureFlagsAbsentWithoutGates(t *testing.T) {
	t.Parallel()

	m := New()
	for _, row := range m.settingsRows() {
		if row.Field == "Test Gate" {
			t.Fatal("settingsRows produced a gate row with no registry set")
		}
	}
}

// TestFeatureFlagsRowReflectsGateValue asserts the dynamic row appended after
// settingsRowFeatureFlagsHeader shows the gate's current value — this is the
// Inspector's own copy of the toggle that used to live on the app's main
// Settings page (moved here since it's a developer-only surface).
func TestFeatureFlagsRowReflectsGateValue(t *testing.T) {
	t.Parallel()

	const name = "Test Gate"
	g := gate.NewGateRegistry()
	g.Register(gate.FeatureGate{Name: name, Default: false, Description: "an unfinished capability"})
	m := New()
	m.SetGates(g)

	rows := m.settingsRows()
	idx := int(settingsRowFeatureFlagsHeader) + 1
	if idx >= len(rows) {
		t.Fatalf("settingsRows has %d rows; want at least %d for the gate row", len(rows), idx+1)
	}
	row := rows[idx]
	if row.Field != name {
		t.Fatalf("row.Field = %q; want %q", row.Field, name)
	}
	if row.Value != "Disabled" {
		t.Fatalf("row.Value = %q; want Disabled", row.Value)
	}
}

// TestFeatureFlagsRowTogglesAndBroadcasts asserts pressing Enter on a Feature
// Flags row flips the gate in the shared registry, re-derives the
// Inspector's own gate-dependent state (OnGatesChanged), and returns a cmd
// producing GatesChangedMsg so a host can re-broadcast the app-facing
// settings.GatesChangedMsg contract.
func TestFeatureFlagsRowTogglesAndBroadcasts(t *testing.T) {
	t.Parallel()

	const name = "Test Gate"
	g := gate.NewGateRegistry()
	g.Register(gate.FeatureGate{Name: name, Default: false})
	m := New()
	m.SetGates(g)
	m.settingsCursor = int(settingsRowFeatureFlagsHeader) + 1

	cmd := m.handleSettingsKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !g.Value(name) {
		t.Fatal("Enter on the gate row did not flip the registry value")
	}
	if cmd == nil {
		t.Fatal("Enter on the gate row returned a nil cmd; want a GatesChangedMsg cmd")
	}
	msg, ok := cmd().(GatesChangedMsg)
	if !ok {
		t.Fatalf("cmd produced %T; want GatesChangedMsg", msg)
	}
	if !msg.Values[name] {
		t.Fatalf("GatesChangedMsg.Values[%q] = false; want true", name)
	}
}
