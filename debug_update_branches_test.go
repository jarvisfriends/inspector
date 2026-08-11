// Copyright (c) 2026 Jarvis Friends contributors
// SPDX-License-Identifier: MIT

package inspector

import (
	"math"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/jarvisfriends/snap/notifications"
)

// TestUpdateLatestValueFlush covers the coalesced telemetry flush: the flush
// msg clears the timer flag and marks the view dirty only when pending.
func TestUpdateLatestValueFlush(t *testing.T) {
	t.Parallel()

	m := New()
	m.latestValueFlushTimer = true
	m.latestValueDirty = true
	m.dirty = false
	_, _ = m.Update(latestValueFlushMsg{})
	if m.latestValueFlushTimer {
		t.Fatal("flush msg should clear the timer flag")
	}
	if !m.dirty {
		t.Fatal("pending telemetry should mark the view dirty")
	}

	// Without pending telemetry the dirty flag stays down.
	m.latestValueFlushTimer = true
	_, _ = m.Update(latestValueFlushMsg{})
	if m.latestValueFlushTimer {
		t.Fatal("flush msg should clear the timer flag even when nothing is pending")
	}
	if m.latestValueDirty {
		t.Fatal("flush must not raise the telemetry-dirty flag")
	}
}

// TestScheduleLatestValueFlushTick covers the tick closure: the scheduled cmd
// resolves to a latestValueFlushMsg, and re-arming is a no-op while pending.
func TestScheduleLatestValueFlushTick(t *testing.T) {
	t.Parallel()

	m := New()
	m.latestValueInterval = time.Millisecond
	cmd := m.scheduleLatestValueFlush()
	if cmd == nil {
		t.Fatal("expected a flush tick cmd")
	}
	if again := m.scheduleLatestValueFlush(); again != nil {
		t.Fatal("second schedule while armed should return nil")
	}
	if _, ok := cmd().(latestValueFlushMsg); !ok {
		t.Fatal("tick should deliver latestValueFlushMsg")
	}
}

// TestInitSchedulesStatsTick covers Init and the stats tick closure end to
// end: the tick delivers a fresh snapshot which Update folds in and
// reschedules.
func TestInitSchedulesStatsTick(t *testing.T) {
	t.Parallel()

	m := New()
	m.statsRefreshInterval = time.Millisecond
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init should schedule the stats tick")
	}
	msg := cmd()
	tick, ok := msg.(statsTickMsg)
	if !ok {
		t.Fatalf("stats tick delivered %T; want statsTickMsg", msg)
	}

	prev := m.stats
	_, next := m.Update(tick)
	if next == nil {
		t.Fatal("Update(statsTickMsg) should reschedule the tick")
	}
	if m.prevStats.CapturedAt != prev.CapturedAt {
		t.Fatal("Update(statsTickMsg) should shift stats into prevStats")
	}
	if !m.stats.CapturedAt.After(prev.CapturedAt) {
		t.Fatal("Update(statsTickMsg) should install the fresh snapshot")
	}
}

// TestUpdateTermDiagMsg asserts terminal diagnostics are recorded.
func TestUpdateTermDiagMsg(t *testing.T) {
	t.Parallel()

	m := New()
	_, _ = m.Update(TermDiagMsg{BgIsDark: true, Profile: colorprofile.TrueColor})
	if !m.termDiagSet || m.termDiag == nil || !m.termDiag.BgIsDark {
		t.Fatal("TermDiagMsg was not recorded")
	}
}

// TestUpdateNotifyAndUtilityKeys covers the h/i/w/e/x shortcut branches. A
// throwaway key press arms the telemetry flush timer first so each shortcut's
// cmd comes back unbatched.
func TestUpdateNotifyAndUtilityKeys(t *testing.T) {
	t.Parallel()

	m := New()
	_, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF1}) // arms the flush timer

	// Highlight toggle.
	_, _ = m.Update(tea.KeyPressMsg{Code: 'h', Text: "h"})
	if !m.ShowHighlight {
		t.Fatal("'h' should enable the highlight overlay")
	}

	// Test notifications carry escalating severities.
	wantSeverity := map[string]notifications.Severity{
		"i": notifications.SeverityInfo,
		"w": notifications.SeverityWarning,
		"e": notifications.SeverityError,
	}
	for text, want := range wantSeverity {
		_, cmd := m.Update(tea.KeyPressMsg{Code: rune(text[0]), Text: text})
		if cmd == nil {
			t.Fatalf("%q should return a notification cmd", text)
		}
		note, ok := cmd().(notifications.AddMsg)
		if !ok {
			t.Fatalf("%q produced %T; want notifications.AddMsg", text, note)
		}
		if note.Severity != want {
			t.Fatalf("%q severity = %v; want %v", text, note.Severity, want)
		}
	}

	// Export shortcut returns the export cmd (not executed here).
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	if cmd == nil {
		t.Fatal("'x' should return the export-log cmd")
	}
}

// TestUpdateSettingsTabKeyRouting asserts Up/Down are routed to the settings
// cursor while the Settings tab is active.
func TestUpdateSettingsTabKeyRouting(t *testing.T) {
	t.Parallel()

	m := New()
	_, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m.switchTab(debugTabSettings)

	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if m.settingsCursor != 1 {
		t.Fatalf("settingsCursor = %d after Down; want 1", m.settingsCursor)
	}
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if m.settingsCursor != 0 {
		t.Fatalf("settingsCursor = %d after Up; want 0", m.settingsCursor)
	}
}

// TestUpdateTableTabNavigationKeys covers the table-cursor routing for the
// full navigation key set on a table tab.
func TestUpdateTableTabNavigationKeys(t *testing.T) {
	t.Parallel()

	m := New()
	_, _ = m.Update(tea.WindowSizeMsg{Width: 300, Height: 40})
	_ = m.View() // marks the runtime table active

	for _, code := range []rune{tea.KeyDown, tea.KeyUp, tea.KeyPgDown, tea.KeyPgUp, tea.KeyEnd, tea.KeyHome} {
		_, _ = m.Update(tea.KeyPressMsg{Code: code})
	}
	if got := m.runtimeTbl.Cursor(); got != 0 {
		t.Fatalf("runtime cursor = %d after Home; want 0", got)
	}
}

// TestUpdateViewportTabScrollKeys covers the viewport scrolling branch
// (non-table tabs) for arrows and page keys.
func TestUpdateViewportTabScrollKeys(t *testing.T) {
	t.Parallel()

	m := New()
	_, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 12})
	m.switchTab(debugTabTerminal)
	_ = m.View()

	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyPgDown})
	down := m.sectionViewport.YOffset()
	if down <= 0 {
		t.Fatalf("PgDown did not scroll the terminal tab; offset=%d", down)
	}
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyPgUp})
	if got := m.sectionViewport.YOffset(); got != 0 {
		t.Fatalf("PgUp should return to the top; offset=%d", got)
	}
}

// TestUpdateAccessibilityTabRoutesKeysToPanel asserts non-nav keys reach the
// panel while the Accessibility tab is active (including key releases), and
// nav keys still escape to tab switching.
func TestUpdateAccessibilityTabRoutesKeysToPanel(t *testing.T) {
	t.Parallel()

	m := New()
	m.SetGates(newGatesWithAccessibility(true))
	_, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m.switchTab(debugTabAccessibility)

	before := m.acPanel.cursor
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if m.acPanel.cursor != before+1 {
		t.Fatalf("panel cursor = %d; want %d (Down routed to panel)", m.acPanel.cursor, before+1)
	}

	// Key releases are not tab-switching presses; they go to the panel too.
	_, _ = m.Update(tea.KeyReleaseMsg{Code: 'a', Text: "a"})
	if m.activeTab != debugTabAccessibility {
		t.Fatal("key release must not switch tabs")
	}

	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if m.activeTab == debugTabAccessibility {
		t.Fatal("Left must escape the accessibility tab")
	}
}

// TestUpdateDigitBeyondVisibleTabsIgnored asserts a digit past the visible tab
// count leaves the active tab unchanged.
func TestUpdateDigitBeyondVisibleTabsIgnored(t *testing.T) {
	t.Parallel()

	m := New()
	_, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	_, _ = m.Update(tea.KeyPressMsg{Code: '9', Text: "9"})
	if m.activeTab != debugTabRuntime {
		t.Fatalf("digit 9 switched to %v; want Runtime unchanged", m.activeTab)
	}
}

// TestLogMessageForDebuggingEnvMsg covers the env-var pretty printer,
// including the malformed (no '=') entry fallback.
func TestLogMessageForDebuggingEnvMsg(t *testing.T) {
	t.Parallel()

	m := New()
	_ = m.LogMessageForDebugging(tea.EnvMsg{"HOME=/root", "MALFORMED"})
	last := m.Logs[len(m.Logs)-1]
	if !strings.Contains(last.Content, "Key: HOME") || !strings.Contains(last.Content, "Value: /root") {
		t.Errorf("env content missing parsed pair: %q", last.Content)
	}
	if !strings.Contains(last.Content, "Env: MALFORMED") {
		t.Errorf("env content missing malformed fallback: %q", last.Content)
	}
}

// TestLogMessageForDebuggingMouseTelemetry covers each mouse msg branch: the
// telemetry kinds coalesce (no log entry) while motion only invalidates the
// view when the highlight overlay is on.
func TestLogMessageForDebuggingMouseTelemetry(t *testing.T) {
	t.Parallel()

	m := New()
	mouse := tea.Mouse{X: 3, Y: 4, Button: tea.MouseLeft}

	if cmd := m.LogMessageForDebugging(tea.MouseClickMsg(mouse)); cmd == nil {
		t.Fatal("click should schedule a telemetry flush")
	}
	if m.LastMouseClick.X != 3 {
		t.Fatal("click position not recorded")
	}
	// Timer already armed: subsequent telemetry returns nil but records state.
	if cmd := m.LogMessageForDebugging(tea.MouseReleaseMsg(mouse)); cmd != nil {
		t.Fatal("second telemetry event should not re-arm the flush timer")
	}
	if m.LastMouseRelease.Y != 4 {
		t.Fatal("release position not recorded")
	}
	_ = m.LogMessageForDebugging(tea.MouseWheelMsg(tea.Mouse{Button: tea.MouseWheelUp}))
	if m.LastMouseWheel.Button != tea.MouseWheelUp {
		t.Fatal("wheel not recorded")
	}

	// Motion without highlight records silently.
	m.latestValueDirty = false
	_ = m.LogMessageForDebugging(tea.MouseMotionMsg(tea.Mouse{X: 9, Y: 9}))
	if m.latestValueDirty {
		t.Fatal("motion without highlight must not invalidate the view")
	}
	if m.LastMouseMotion.X != 9 {
		t.Fatal("motion position not recorded")
	}

	// Motion with highlight enabled invalidates the cached view.
	m.ShowHighlight = true
	_ = m.LogMessageForDebugging(tea.MouseMotionMsg(tea.Mouse{X: 1, Y: 1}))
	if !m.latestValueDirty {
		t.Fatal("motion with highlight should invalidate the view")
	}

	// None of the telemetry kinds may create log entries.
	if len(m.Logs) != 0 {
		t.Fatalf("telemetry created %d log entries; want 0", len(m.Logs))
	}
}

// TestLogMessageForDebuggingKeyRelease covers the key-release telemetry
// branch.
func TestLogMessageForDebuggingKeyRelease(t *testing.T) {
	t.Parallel()

	m := New()
	_ = m.LogMessageForDebugging(tea.KeyReleaseMsg{Code: 'q', Text: "q"})
	if m.LastKeyRel.Text != "q" {
		t.Fatalf("LastKeyRel = %+v; want release recorded", m.LastKeyRel)
	}
	if len(m.Logs) != 0 {
		t.Fatal("key release must not be logged")
	}
}

// TestAppendExternalLogDedupes asserts external entries with identical
// type+content stack instead of appending.
func TestAppendExternalLogDedupes(t *testing.T) {
	t.Parallel()

	m := New()
	ts := time.Now()
	m.AddLog("INFO", ts, "same line")
	m.AddLog("INFO", ts.Add(time.Second), "same line")
	_, _ = m.Update(TermDiagMsg{}) // drains pendingLogs first

	if len(m.Logs) == 0 {
		t.Fatal("expected drained log entries")
	}
	first := m.Logs[0]
	if first.Content != "same line" || first.Count != 2 {
		t.Fatalf("external log entry = %+v; want deduped Count=2", first)
	}
}

// TestSaturatingDurationClamps pins the overflow clamp.
func TestSaturatingDurationClamps(t *testing.T) {
	t.Parallel()

	if got := saturatingDuration(math.MaxUint64); got != math.MaxInt64 {
		t.Fatalf("saturatingDuration(MaxUint64) = %d; want MaxInt64", got)
	}
	if got := saturatingDuration(42); got != 42*time.Nanosecond {
		t.Fatalf("saturatingDuration(42) = %s; want 42ns", got)
	}
}

// TestGcSummaryFormats is a table-driven pin of every gcSummary output form.
func TestGcSummaryFormats(t *testing.T) {
	t.Parallel()

	base := time.Now()
	mk := func(numGC uint32, at time.Time) runtimeStatsSnapshot {
		return runtimeStatsSnapshot{NumGC: numGC, CapturedAt: at}
	}
	gcIdle := "gc idle"
	tests := []struct {
		name string
		st   runtimeStatsSnapshot
		pr   runtimeStatsSnapshot
		want string
	}{
		{"idle", mk(5, base.Add(time.Second)), mk(5, base), gcIdle},
		{"per second", mk(10, base.Add(time.Second)), mk(5, base), "gc 5.0/s"},
		{"slow cadence", mk(1, base.Add(20*time.Second)), mk(0, base), "gc 20.0s"},
		{"counter reset", mk(1, base.Add(time.Second)), mk(9, base), gcIdle},
		{"zero elapsed", mk(2, base), mk(1, base), "gc 1.0/s"},
	}
	for _, tc := range tests {
		if got := gcSummary(tc.st, tc.pr); got != tc.want {
			t.Errorf("%s: gcSummary = %q; want %q", tc.name, got, tc.want)
		}
	}
}

// TestStatusLineSummaryIncludesLinkRate covers the link-rate part: injected
// text is appended, and empty text is dropped.
func TestStatusLineSummaryIncludesLinkRate(t *testing.T) {
	t.Parallel()

	m := New()
	m.statusSummary.Enabled = true
	m.SetLinkRateSummary(func() string { return "tx 1 B/s rx 2 B/s" })
	if got := m.StatusLineSummary(); !strings.Contains(got, "tx 1 B/s") {
		t.Fatalf("summary %q missing link rate", got)
	}
	m.SetLinkRateSummary(func() string { return "" })
	if got := m.StatusLineSummary(); strings.Contains(got, "tx") {
		t.Fatalf("summary %q should drop empty link text", got)
	}
}

// TestShortHelpFallbackForProviderTabs asserts tabs outside the built-in
// range (provider tabs) fall back to the generic binding set.
func TestShortHelpFallbackForProviderTabs(t *testing.T) {
	t.Parallel()

	m := New()
	m.activeTab = debugTab(len(debugTabTitles)) // first provider slot
	if got := len(m.ShortHelp()); got == 0 {
		t.Fatal("provider-tab ShortHelp should not be empty")
	}
}

// TestActiveDataTablePerTab asserts each table tab exposes its own table and
// non-table tabs expose none.
func TestActiveDataTablePerTab(t *testing.T) {
	t.Parallel()

	m := New()
	m.setTableActive(debugTabRuntime, true)
	m.setTableActive(debugTabInput, true)
	m.setTableActive(debugTabDisks, true)
	m.setTableActive(debugTabTerminal, true) // not a table tab: still nil

	m.activeTab = debugTabRuntime
	if m.activeDataTable() != &m.runtimeTbl {
		t.Error("runtime tab should expose the runtime table")
	}
	m.activeTab = debugTabInput
	if m.activeDataTable() != &m.inputDbgTbl {
		t.Error("input tab should expose the input table")
	}
	m.activeTab = debugTabDisks
	if m.activeDataTable() != &m.diskTbl {
		t.Error("disks tab should expose the disk table")
	}
	m.activeTab = debugTabTerminal
	if m.activeDataTable() != nil {
		t.Error("terminal tab must not expose a table")
	}
}

// TestScrollActiveSectionBranches covers the zero-line no-op and the log-tab
// scrolling branch (both directions), plus the log-tab scroll save.
func TestScrollActiveSectionBranches(t *testing.T) {
	t.Parallel()

	m := New()
	_, _ = m.Update(tea.WindowSizeMsg{Width: 90, Height: 12})
	for i := range 40 {
		m.AddLog("INFO", time.Now(), strings.Repeat("x", 5)+"-"+strings.Repeat("y", i%3))
	}
	_, _ = m.Update(TermDiagMsg{}) // drain logs
	m.switchTab(debugTabLog)
	_ = m.View()

	m.dirty = false
	m.scrollActiveSection(0)
	if m.dirty {
		t.Fatal("zero-line scroll must be a no-op")
	}

	m.logViewport.GotoBottom()
	bottom := m.logViewport.YOffset()
	m.scrollActiveSection(-2)
	if got := m.logViewport.YOffset(); got != max(0, bottom-2) {
		t.Fatalf("log scroll up: offset=%d; want %d", got, max(0, bottom-2))
	}
	m.scrollActiveSection(2)
	if got := m.logViewport.YOffset(); got != bottom {
		t.Fatalf("log scroll down: offset=%d; want %d", got, bottom)
	}
	if got := m.tabScrollY[debugTabLog]; got != bottom {
		t.Fatalf("log tab scroll not saved: %d; want %d", got, bottom)
	}

	// Settings-tab branch moves the cursor rather than the viewport.
	m.switchTab(debugTabSettings)
	_ = m.View()
	m.scrollActiveSection(3)
	if m.settingsCursor != 3 {
		t.Fatalf("settings scroll moved cursor to %d; want 3", m.settingsCursor)
	}
	m.scrollActiveSection(-99)
	if m.settingsCursor != 0 {
		t.Fatalf("settings scroll clamped cursor to %d; want 0", m.settingsCursor)
	}
}

// TestHandleWheelOnViewportTab covers the plain vertical wheel branch on a
// non-table tab (section viewport scrolling both directions) and wheel-up on
// a table tab.
func TestHandleWheelOnViewportTab(t *testing.T) {
	t.Parallel()

	m := New()
	_, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 12})
	m.switchTab(debugTabTerminal)
	_ = m.View()

	m.handleWheel(tea.Mouse{Button: tea.MouseWheelDown})
	if got := m.sectionViewport.YOffset(); got != 3 {
		t.Fatalf("wheel down scrolled to %d; want 3", got)
	}
	m.handleWheel(tea.Mouse{Button: tea.MouseWheelUp})
	if got := m.sectionViewport.YOffset(); got != 0 {
		t.Fatalf("wheel up scrolled to %d; want 0", got)
	}

	// Table tab: wheel-up moves the row cursor.
	m.switchTab(debugTabRuntime)
	_, _ = m.Update(tea.WindowSizeMsg{Width: 300, Height: 40})
	_ = m.View()
	m.runtimeTbl.MoveDown(2)
	before := m.runtimeTbl.Cursor()
	m.handleWheel(tea.Mouse{Button: tea.MouseWheelUp})
	if got := m.runtimeTbl.Cursor(); got != before-1 {
		t.Fatalf("wheel up on table: cursor=%d; want %d", got, before-1)
	}
}

// TestViewLogTabAndMouseFallthrough covers the Log-tab render path in View
// and the OnMouse nil fallthrough for clicks outside interactive regions.
func TestViewLogTabAndMouseFallthrough(t *testing.T) {
	t.Parallel()

	m := New()
	_, _ = m.Update(tea.WindowSizeMsg{Width: 90, Height: 20})
	m.LogMessageForDebugging("visible-entry")
	m.switchTab(debugTabLog)
	v := m.View()
	if !strings.Contains(v.Content, "visible-entry") {
		t.Fatal("log tab view missing the log entry")
	}

	// A left release below the tab bar on a non-settings tab does nothing.
	m.switchTab(debugTabRuntime)
	v = m.View()
	if cmd := v.OnMouse(tea.MouseReleaseMsg(tea.Mouse{X: 1, Y: m.sectionOriginY + 1, Button: tea.MouseLeft})); cmd != nil {
		t.Fatal("click in a non-interactive region should return nil")
	}
	// Non-left releases are ignored outright.
	if cmd := v.OnMouse(tea.MouseReleaseMsg(tea.Mouse{X: 1, Y: 1, Button: tea.MouseRight})); cmd != nil {
		t.Fatal("right-click should be ignored")
	}
}
