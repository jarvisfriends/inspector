// Copyright (c) 2026 Jarvis Friends contributors
// SPDX-License-Identifier: MIT

package inspector

import (
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/jarvisfriends/snap/gate"
	"github.com/jarvisfriends/snap/styles"
)

const testServerURL = "http://127.0.0.1:6060/debug/pprof/"

// enterOn positions the settings cursor on row and presses Enter, returning
// the resulting command (not executed — several rows return side-effecting
// tea.Cmds).
func enterOn(m *InspectorModel, row settingsRowIndex) tea.Cmd {
	m.settingsCursor = int(row)
	return m.handleSettingsKey(tea.KeyPressMsg{Code: tea.KeyEnter})
}

// TestHandleSettingsKeyToggleRows walks every toggle/cycle row on the Settings
// tab and asserts Enter mutates the expected field.
func TestHandleSettingsKeyToggleRows(t *testing.T) {
	t.Parallel()

	m := New()

	tests := []struct {
		name  string
		row   settingsRowIndex
		check func(t *testing.T)
	}{
		{"latest refresh increments", settingsRowLatestRefresh, func(t *testing.T) {
			t.Helper()
			if m.latestValueInterval <= defaultLatestValueRenderInterval {
				t.Errorf("latestValueInterval = %s; want > default", m.latestValueInterval)
			}
		}},
		{"stats refresh increments", settingsRowStatsRefresh, func(t *testing.T) {
			t.Helper()
			if m.statsRefreshInterval <= defaultStatsRefreshInterval {
				t.Errorf("statsRefreshInterval = %s; want > default", m.statsRefreshInterval)
			}
		}},
		{"status summary toggles", settingsRowStatusSummary, func(t *testing.T) {
			t.Helper()
			if !m.statusSummary.Enabled {
				t.Error("statusSummary.Enabled should flip to true")
			}
		}},
		{"show term toggles", settingsRowShowTerm, func(t *testing.T) {
			t.Helper()
			if m.statusSummary.ShowTerm {
				t.Error("ShowTerm should flip to false")
			}
		}},
		{"show heap toggles", settingsRowShowHeap, func(t *testing.T) {
			t.Helper()
			if m.statusSummary.ShowHeap {
				t.Error("ShowHeap should flip to false")
			}
		}},
		{"show gc toggles", settingsRowShowGC, func(t *testing.T) {
			t.Helper()
			if m.statusSummary.ShowGC {
				t.Error("ShowGC should flip to false")
			}
		}},
		{"show goroutines toggles", settingsRowShowGoroutines, func(t *testing.T) {
			t.Helper()
			if m.statusSummary.ShowGorout {
				t.Error("ShowGorout should flip to false")
			}
		}},
		{"show link toggles", settingsRowShowLink, func(t *testing.T) {
			t.Helper()
			if m.statusSummary.ShowLink {
				t.Error("ShowLink should flip to false")
			}
		}},
		{"pprof addr cycles to alt", settingsRowPprofAddr, func(t *testing.T) {
			t.Helper()
			if m.pprof.Addr != pprofAltAddr {
				t.Errorf("Addr = %q; want %q", m.pprof.Addr, pprofAltAddr)
			}
		}},
		{"tool ui addr cycles to alt", settingsRowPprofToolAddr, func(t *testing.T) {
			t.Helper()
			if m.pprof.ToolUIAddr != pprofAltToolUI {
				t.Errorf("ToolUIAddr = %q; want %q", m.pprof.ToolUIAddr, pprofAltToolUI)
			}
		}},
		{"cpu secs increments", settingsRowCPUSecs, func(t *testing.T) {
			t.Helper()
			if m.pprof.CPUCaptureSecs != pprofDefaultCaptureSecs+1 {
				t.Errorf("CPUCaptureSecs = %d; want %d", m.pprof.CPUCaptureSecs, pprofDefaultCaptureSecs+1)
			}
		}},
	}
	for _, tc := range tests {
		if cmd := enterOn(m, tc.row); cmd != nil {
			t.Errorf("%s: Enter returned a cmd; want nil", tc.name)
		}
		tc.check(t)
	}

	// The addr rows cycle back to the defaults on a second press.
	_ = enterOn(m, settingsRowPprofAddr)
	if m.pprof.Addr != pprofDefaultAddr {
		t.Errorf("Addr after second Enter = %q; want default", m.pprof.Addr)
	}
	_ = enterOn(m, settingsRowPprofToolAddr)
	if m.pprof.ToolUIAddr != pprofDefaultToolUI {
		t.Errorf("ToolUIAddr after second Enter = %q; want default", m.pprof.ToolUIAddr)
	}
}

// TestHandleSettingsKeyViewModeCycles asserts the pprof view mode row cycles
// builtin → go-tool → graphviz → builtin.
func TestHandleSettingsKeyViewModeCycles(t *testing.T) {
	t.Parallel()

	m := New()
	seen := make([]string, 0, 3)
	for range 3 {
		_ = enterOn(m, settingsRowPprofViewMode)
		seen = append(seen, m.pprof.ViewMode)
	}
	if !strings.HasPrefix(seen[0], "go-") || seen[1] != "graphviz" {
		t.Fatalf("cycle order = %v; want go-tool then graphviz", seen)
	}
	if seen[2] != pprofViewModeBuiltin {
		t.Fatalf("ViewMode after full cycle = %q; want %q", seen[2], pprofViewModeBuiltin)
	}
}

// TestHandleSettingsKeyPprofServerToggle asserts the enable row returns a
// start cmd, and disabling with no live server yields a benign stopped msg.
func TestHandleSettingsKeyPprofServerToggle(t *testing.T) {
	t.Parallel()

	m := New()
	cmd := enterOn(m, settingsRowPprofEnabled)
	if !m.pprof.Enabled {
		t.Fatal("Enter should enable the pprof server flag")
	}
	if cmd == nil {
		t.Fatal("enabling should return the start cmd")
	}

	// Disable again: no server was ever started, so the stop cmd resolves to a
	// plain stopped message immediately.
	cmd = enterOn(m, settingsRowPprofEnabled)
	if m.pprof.Enabled {
		t.Fatal("second Enter should disable the pprof server flag")
	}
	if cmd == nil {
		t.Fatal("disabling should return the stop cmd")
	}
	if _, ok := cmd().(pprofServerStoppedMsg); !ok {
		t.Fatal("stop cmd with no live server should yield pprofServerStoppedMsg")
	}
}

// TestHandleSettingsKeyBrowserRowsRequireServer asserts every browser-endpoint
// row refuses to act while the pprof server is down, and returns an open cmd
// once a server URL is present.
func TestHandleSettingsKeyBrowserRowsRequireServer(t *testing.T) {
	t.Parallel()

	rows := []settingsRowIndex{
		settingsRowPprofIndex,
		settingsRowHeapDebug1,
		settingsRowHeapDebug2,
		settingsRowGoroutineDebug1,
		settingsRowGoroutineDebug2,
		settingsRowAllocsDebug1,
		settingsRowBlockDebug1,
		settingsRowMutexDebug1,
		settingsRowCPUStream,
		settingsRowTraceStream,
	}

	m := New()
	for _, row := range rows {
		m.settingsMessage = ""
		if cmd := enterOn(m, row); cmd != nil {
			t.Errorf("row %d: cmd returned while server is down", row)
		}
		if !strings.Contains(m.settingsMessage, "not running") {
			t.Errorf("row %d: settingsMessage = %q; want server-down hint", row, m.settingsMessage)
		}
	}

	m.pprof.ServerURL = testServerURL
	for _, row := range rows {
		if cmd := enterOn(m, row); cmd == nil {
			t.Errorf("row %d: no open-browser cmd with server running", row)
		}
	}
}

// TestHandleSettingsKeyActionRows asserts the capture and go-tool rows return
// their action cmds without mutating unrelated state.
func TestHandleSettingsKeyActionRows(t *testing.T) {
	t.Parallel()

	m := New()
	for _, row := range []settingsRowIndex{
		settingsRowWriteHeap,
		settingsRowCaptureCPU,
		settingsRowGotoolLatest,
		settingsRowGotoolLiveHeap,
		settingsRowGotoolLiveCPU,
	} {
		if cmd := enterOn(m, row); cmd == nil {
			t.Errorf("row %d: action row returned nil cmd", row)
		}
	}
}

// TestHandleSettingsKeyReadOnlyRows asserts display rows are no-ops on
// Enter. (Headers toggle collapse and OutputDir opens the folder picker —
// both covered by their own tests.)
func TestHandleSettingsKeyReadOnlyRows(t *testing.T) {
	t.Parallel()

	m := New()
	if cmd := enterOn(m, settingsRowServerState); cmd != nil {
		t.Error("server-state display row returned a cmd")
	}
}

// TestHeaderEnterTogglesCollapse: Enter on a SectionOnly header flips its
// collapsed state, hiding and re-showing the section's body rows.
func TestHeaderEnterTogglesCollapse(t *testing.T) {
	t.Parallel()

	m := New()
	items := m.settingsRows()
	if !m.collapsedSections[settingsRowBuiltinHeader] {
		t.Fatal("browser-endpoint section should start collapsed")
	}
	if m.settingsRowShown(items, int(settingsRowHeapDebug1)) {
		t.Fatal("collapsed section body row should be hidden")
	}

	if cmd := enterOn(m, settingsRowBuiltinHeader); cmd != nil {
		t.Fatal("header toggle must not return a cmd")
	}
	if m.collapsedSections[settingsRowBuiltinHeader] {
		t.Fatal("Enter on the header should expand the section")
	}
	if !m.settingsRowShown(items, int(settingsRowHeapDebug1)) {
		t.Fatal("expanded section body row should be visible")
	}

	_ = enterOn(m, settingsRowBuiltinHeader)
	if !m.collapsedSections[settingsRowBuiltinHeader] {
		t.Fatal("second Enter should collapse the section again")
	}
}

// TestMoveSettingsCursorSkipsCollapsedRows: with the pprof sections
// collapsed, Down from the section header lands on the NEXT header, not on a
// hidden body row.
func TestMoveSettingsCursorSkipsCollapsedRows(t *testing.T) {
	t.Parallel()

	m := New()
	items := m.settingsRows()
	m.settingsCursor = int(settingsRowBuiltinHeader)
	m.moveSettingsCursor(items, 1)
	if m.settingsCursor != int(settingsRowGotoolHeader) {
		t.Fatalf("Down from a collapsed header landed on %d; want the next header %d",
			m.settingsCursor, int(settingsRowGotoolHeader))
	}
	m.moveSettingsCursor(items, -1)
	if m.settingsCursor != int(settingsRowBuiltinHeader) {
		t.Fatalf("Up landed on %d; want %d", m.settingsCursor, int(settingsRowBuiltinHeader))
	}
}

// TestHandleSettingsKeyCursorAndOtherKeys covers the Up/Down cursor moves,
// their clamping at both ends, and the ignore path for non-Enter keys.
func TestHandleSettingsKeyCursorAndOtherKeys(t *testing.T) {
	t.Parallel()

	m := New()
	m.settingsCursor = 0
	_ = m.handleSettingsKey(tea.KeyPressMsg{Code: tea.KeyUp})
	if m.settingsCursor != 0 {
		t.Fatalf("Up at first row moved cursor to %d", m.settingsCursor)
	}
	_ = m.handleSettingsKey(tea.KeyPressMsg{Code: tea.KeyDown})
	if m.settingsCursor != 1 {
		t.Fatalf("Down moved cursor to %d; want 1", m.settingsCursor)
	}
	_ = m.handleSettingsKey(tea.KeyPressMsg{Code: tea.KeyUp})
	if m.settingsCursor != 0 {
		t.Fatalf("Up moved cursor to %d; want 0", m.settingsCursor)
	}

	// Cursor movement walks VISIBLE rows: the last stop is the last visible
	// row (the pprof sections start collapsed), and Down there clamps.
	vis := m.visibleSettingsRows(m.settingsRows())
	last := vis[len(vis)-1]
	m.settingsCursor = last
	_ = m.handleSettingsKey(tea.KeyPressMsg{Code: tea.KeyDown})
	if m.settingsCursor != last {
		t.Fatalf("Down at last visible row moved cursor to %d; want %d", m.settingsCursor, last)
	}

	// Any other key marks the view dirty but performs no action.
	m.dirty = false
	if cmd := m.handleSettingsKey(tea.KeyPressMsg{Code: tea.KeySpace}); cmd != nil {
		t.Fatal("space should not produce a cmd")
	}
	if !m.dirty {
		t.Fatal("non-Enter key should still mark the view dirty")
	}
}

// TestSettingsRowsReflectPprofState asserts the rows surface the live pprof
// server state: enabled flag and the running URL.
func TestSettingsRowsReflectPprofState(t *testing.T) {
	t.Parallel()

	m := New()
	m.pprof.Enabled = true
	m.pprof.ServerURL = testServerURL

	var pprofValue, serverValue string
	for _, row := range m.settingsRows() {
		switch row.Field {
		case "Enable profiler HTTP server":
			pprofValue = row.Value
		case "Server":
			serverValue = row.Value
		}
	}
	if pprofValue != "on" {
		t.Errorf("profiler row value = %q; want on", pprofValue)
	}
	if serverValue != testServerURL {
		t.Errorf("server row value = %q; want %q", serverValue, testServerURL)
	}
}

// TestSettingsRowsEmptyGateRegistry asserts a registry with no gates adds
// neither the Feature Flags header nor any gate rows.
func TestSettingsRowsEmptyGateRegistry(t *testing.T) {
	t.Parallel()

	m := New()
	m.SetGates(gate.NewGateRegistry())
	for _, row := range m.settingsRows() {
		if strings.Contains(row.Field, "Feature Flags") {
			t.Fatal("empty gate registry must not add the Feature Flags header")
		}
	}
}

// TestSettingsRowsShowEnabledGate asserts an enabled gate renders as Enabled.
func TestSettingsRowsShowEnabledGate(t *testing.T) {
	t.Parallel()

	const name = "Ready Gate"
	g := gate.NewGateRegistry()
	g.Register(gate.FeatureGate{Name: name, Default: true})
	m := New()
	m.SetGates(g)

	rows := m.settingsRows()
	row := rows[int(settingsRowFeatureFlagsHeader)+1]
	if row.Field != name || row.Value != "Enabled" {
		t.Fatalf("gate row = %q/%q; want %q/Enabled", row.Field, row.Value, name)
	}
}

// TestRenderSettingsSectionShowsMessageAndActionPrefix asserts the ↵
// indicator marks a selected action-only row, and that the transient
// settings message renders in the pinned FOOTER — not inside the list, where
// it used to sit below the last row and scroll off-screen.
func TestRenderSettingsSectionShowsMessageAndActionPrefix(t *testing.T) {
	t.Parallel()

	m := New()
	_, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m.switchTab(debugTabSettings)
	m.settingsMessage = "snapshot saved somewhere"
	m.settingsCursor = int(settingsRowWriteHeap) // ActionOnly row

	out := m.renderSettingsSection(styles.Active())
	if strings.Contains(out, "snapshot saved somewhere") {
		t.Error("settings message must live in the footer, not the list")
	}
	if !strings.Contains(out, "↵") {
		t.Error("selected action row should use the ↵ indicator")
	}
	footer := m.buildFooterLine(styles.Active(), "Debug Settings", 100)
	if !strings.Contains(footer, "snapshot saved somewhere") {
		t.Error("footer missing the settings message")
	}

	// Without a message the footer surfaces the selected row's help instead.
	m.settingsMessage = ""
	m.settingsCursor = 0
	rows := m.settingsRows()
	if rows[0].Help == "" {
		t.Fatal("test premise: row 0 carries help")
	}
	footer = m.buildFooterLine(styles.Active(), "Debug Settings", 200)
	if !strings.Contains(footer, rows[0].Help) {
		t.Errorf("footer = %q; want row 0 help", footer)
	}
}

// TestSettingsRowsCaptureSecsShown asserts the CPU capture row reflects the
// configured duration in both value and stream rows.
func TestSettingsRowsCaptureSecsShown(t *testing.T) {
	t.Parallel()

	m := New()
	m.pprof.CPUCaptureSecs = 7
	secs := strconv.Itoa(7)
	var captureVal, streamVal string
	for _, row := range m.settingsRows() {
		switch row.Field {
		case "Capture CPU profile":
			captureVal = row.Value
		case "CPU profile stream":
			streamVal = row.Value
		}
	}
	if !strings.Contains(captureVal, secs) {
		t.Errorf("capture row value = %q; want to contain %q", captureVal, secs)
	}
	if !strings.Contains(streamVal, secs) {
		t.Errorf("stream row value = %q; want to contain %q", streamVal, secs)
	}
}

// TestEnsureSettingsCursorVisibleScrollsBothWays pins the viewport-follow
// behavior: the cursor row scrolls into view whether it is above or below the
// visible window.
func TestEnsureSettingsCursorVisibleScrollsBothWays(t *testing.T) {
	t.Parallel()

	m := New()
	_, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 12})
	m.switchTab(debugTabSettings)
	_ = m.View()

	items := len(m.settingsRows())
	// Cursor below the window: viewport must scroll down.
	m.settingsCursor = items - 1
	m.ensureSettingsCursorVisible(items)
	if m.sectionViewport.YOffset() == 0 {
		t.Fatal("viewport did not scroll down to the last row")
	}

	// Cursor above the window: viewport must snap back up.
	m.settingsCursor = 0
	m.ensureSettingsCursorVisible(items)
	if got := m.sectionViewport.YOffset(); got != 0 {
		t.Fatalf("viewport YOffset = %d; want 0 after moving cursor to top", got)
	}

	// Degenerate item counts return without touching the viewport.
	m.sectionViewport.SetYOffset(3)
	m.ensureSettingsCursorVisible(0)
	if got := m.sectionViewport.YOffset(); got != 3 {
		t.Fatalf("YOffset = %d; want 3 (untouched for zero items)", got)
	}
}

// TestActivateSettingsRowByClickBounds covers the click hit-testing edges:
// clicks above the section and past the last row do nothing.
func TestActivateSettingsRowByClickBounds(t *testing.T) {
	t.Parallel()

	m := New()
	_, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m.switchTab(debugTabSettings)
	_ = m.View()

	if cmd := m.activateSettingsRowByClick(m.sectionOriginY - 1); cmd != nil {
		t.Fatal("click above the section must be ignored")
	}
	before := m.settingsCursor
	_ = m.activateSettingsRowByClick(m.sectionOriginY + len(m.settingsRows()) + 5)
	if m.settingsCursor != before {
		t.Fatal("click past the last row must not move the cursor")
	}
}

// TestOutputDirRowOpensFolderPicker: Enter on the "Output dir" row opens the
// snap/pickers folder picker, Esc closes it without touching the config, and
// Ctrl+S commits the browsed directory into pprof.OutputDir.
func TestOutputDirRowOpensFolderPicker(t *testing.T) {
	t.Parallel()

	m := New()
	_, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m.switchTab(debugTabSettings)
	m.settingsCursor = int(settingsRowOutputDir)
	orig := m.pprof.OutputDir

	if cmd := m.handleSettingsKey(tea.KeyPressMsg{Code: tea.KeyEnter}); cmd == nil {
		t.Fatal("opening the folder picker must return its Init command")
	}
	if m.dirPicker == nil {
		t.Fatal("Enter on the Output dir row must open the folder picker")
	}

	// Esc aborts: the picker closes, the configured directory is unchanged.
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.dirPicker != nil {
		t.Fatal("Esc must close the folder picker")
	}
	if m.pprof.OutputDir != orig {
		t.Fatalf("canceled picker changed OutputDir to %q", m.pprof.OutputDir)
	}

	// Reopen and commit the browsed directory with Ctrl+S.
	_ = m.handleSettingsKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.dirPicker == nil {
		t.Fatal("picker must reopen")
	}
	_, _ = m.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	if m.dirPicker != nil {
		t.Fatal("Ctrl+S must select the browsed directory and close the picker")
	}
	if m.pprof.OutputDir == "" {
		t.Fatal("committed picker left OutputDir empty")
	}
}

// TestSelectTabByXMissReturnsFalse asserts a click outside every tab range is
// reported as a miss.
func TestSelectTabByXMissReturnsFalse(t *testing.T) {
	t.Parallel()

	m := New()
	_, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	_ = m.View()
	if m.selectTabByX(9999) {
		t.Fatal("selectTabByX far past the tab bar should miss")
	}
}
