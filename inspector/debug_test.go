package inspector

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jarvisfriends/snap/navigation"
	"github.com/jarvisfriends/snap/styles"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	testLogMsgDbg         = "dbg-msg"
	testLogMsgInfo        = "info-msg"
	testLogMsgWarn        = "warn-msg"
	testLogMsgErr         = "err-msg"
	testLogMsgIntercepted = "intercepted-msg"
)

func TestLogCapturesMessage(t *testing.T) {
	t.Parallel()
	m := New()
	if len(m.Logs) != 0 {
		t.Fatalf("expected empty logs initially; got %d entries", len(m.Logs))
	}

	// Send a navigation.SelectedMsg and verify it gets logged
	_, _ = m.Update(navigation.SelectedMsg{PageIndex: 1})
	if len(m.Logs) == 0 {
		t.Fatalf("expected logs after update")
	}
	last := m.Logs[len(m.Logs)-1]
	expectedType := fmt.Sprintf("%T", navigation.SelectedMsg{})
	if last.Type != expectedType {
		t.Fatalf("logged Type = %q; want %q", last.Type, expectedType)
	}
}

func TestStackingAndTrim(t *testing.T) {
	t.Parallel()
	m := New()

	// identical messages should stack (increase Count) rather than append
	msg := "repeat"
	_, _ = m.Update(msg)
	_, _ = m.Update(msg)
	if len(m.Logs) != 1 {
		t.Fatalf("expected 1 log entry after stacking; got %d", len(m.Logs))
	}
	if m.Logs[0].Count != 2 {
		t.Fatalf("expected stacked Count=2; got %d", m.Logs[0].Count)
	}

	// Add many unique messages to force trimming to 50 entries
	for i := range 55 {
		m.LogMessageForDebugging(fmt.Sprintf("u%d", i))
	}
	if len(m.Logs) != 50 {
		t.Fatalf("expected logs trimmed to 50; got %d", len(m.Logs))
	}
	// earliest remaining should be u5
	if m.Logs[0].Content != "u5" {
		t.Fatalf("expected first log Content 'u5'; got %q", m.Logs[0].Content)
	}
}

func TestLogLevelFilterWarnPlus(t *testing.T) {
	t.Parallel()
	m := New()
	_ = styles.SetCurrentTint("dracula")
	m.SetColors(styles.Active())

	m.Logs = []MsgLog{
		{Timestamp: time.Now(), Type: "DEBUG", Content: testLogMsgDbg, Count: 1},
		{Timestamp: time.Now(), Type: "INFO", Content: testLogMsgInfo, Count: 1},
		{Timestamp: time.Now(), Type: "WARN", Content: testLogMsgWarn, Count: 1},
		{Timestamp: time.Now(), Type: "ERROR", Content: testLogMsgErr, Count: 1},
		{Timestamp: time.Now(), Type: "tea.KeyPressMsg", Content: testLogMsgIntercepted, Count: 1},
	}
	c := m.Colors()

	// Unfiltered: every entry is rendered.
	all := m.renderLogContent(c)
	for _, want := range []string{testLogMsgDbg, testLogMsgInfo, testLogMsgWarn, testLogMsgErr, testLogMsgIntercepted} {
		if !strings.Contains(all, want) {
			t.Errorf("unfiltered log missing %q", want)
		}
	}

	// 'f' cycles: everything → INFO+ → WARN+ → everything.
	_, _ = m.Update(tea.KeyPressMsg{Text: "f"})
	if m.logLevelFloor != 1 {
		t.Fatalf("logLevelFloor = %d after first 'f'; want 1 (INFO+)", m.logLevelFloor)
	}
	infoPlus := m.renderLogContent(c)
	for _, want := range []string{testLogMsgInfo, testLogMsgWarn, testLogMsgErr} {
		if !strings.Contains(infoPlus, want) {
			t.Errorf("INFO+ log should keep %q", want)
		}
	}
	for _, gone := range []string{testLogMsgDbg, testLogMsgIntercepted} {
		if strings.Contains(infoPlus, gone) {
			t.Errorf("INFO+ log should drop %q", gone)
		}
	}

	_, _ = m.Update(tea.KeyPressMsg{Text: "f"})
	if m.logLevelFloor != logLevelRankWarn {
		t.Fatalf("logLevelFloor = %d after second 'f'; want WARN+", m.logLevelFloor)
	}
	filtered := m.renderLogContent(c)
	for _, want := range []string{testLogMsgWarn, testLogMsgErr} {
		if !strings.Contains(filtered, want) {
			t.Errorf("filtered log should keep %q", want)
		}
	}
	for _, gone := range []string{testLogMsgDbg, testLogMsgInfo, testLogMsgIntercepted} {
		if strings.Contains(filtered, gone) {
			t.Errorf("filtered log should drop %q", gone)
		}
	}

	// A third press wraps back to everything.
	_, _ = m.Update(tea.KeyPressMsg{Text: "f"})
	if m.logLevelFloor != 0 {
		t.Fatalf("logLevelFloor = %d after third 'f'; want 0 (everything)", m.logLevelFloor)
	}
}

// TestLogCompactAndExpandedLayouts: entries render one line each by default;
// 'v' switches to the verbose header+content layout (and back).
func TestLogCompactAndExpandedLayouts(t *testing.T) {
	t.Parallel()
	m := New()
	m.SetColors(styles.Active())
	m.Logs = []MsgLog{
		{Timestamp: time.Now(), Type: "INFO", Content: "line one\nline two", Count: 3},
		{Timestamp: time.Now(), Type: "WARN", Content: "warned", Count: 1},
	}
	c := m.Colors()

	compact := m.renderLogContent(c)
	if got := lipgloss.Height(compact); got != 2 {
		t.Fatalf("compact log = %d lines; want 2 (one per entry):\n%s", got, compact)
	}
	if !strings.Contains(compact, "×3") {
		t.Errorf("compact log missing the ×3 repeat badge:\n%s", compact)
	}

	_, _ = m.Update(tea.KeyPressMsg{Text: "v"})
	if !m.logExpanded {
		t.Fatal("expected logExpanded=true after 'v'")
	}
	expanded := m.renderLogContent(c)
	if lipgloss.Height(expanded) <= 2 {
		t.Fatalf("expanded log should span more lines than compact:\n%s", expanded)
	}

	_, _ = m.Update(tea.KeyPressMsg{Text: "v"})
	if m.logExpanded {
		t.Fatal("expected logExpanded=false after second 'v'")
	}
}

func TestWindowSizeIgnored(t *testing.T) {
	t.Parallel()
	m := New()
	_, _ = m.Update(navigation.SelectedMsg{PageIndex: 0})
	before := len(m.Logs)
	// WindowSizeMsg should not be logged
	_, _ = m.Update(tea.WindowSizeMsg{Width: 10, Height: 5})
	if len(m.Logs) != before {
		t.Fatalf(
			"expected no new logs after WindowSizeMsg; before=%d after=%d",
			before,
			len(m.Logs),
		)
	}
}

func TestViewShowsLogs(t *testing.T) {
	t.Parallel()
	m := New()
	m.LogMessageForDebugging("alpha")
	m.LogMessageForDebugging("beta")
	_, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 120})
	_ = m.View()
	logView := m.logViewport.View()
	if !strings.Contains(logView, "beta") {
		t.Fatalf("expected log viewport to include log messages; got %q", logView)
	}
}

// Regression: some themes can produce SelectedItem fg/bg pairs that collide,
// which made table headers look blank (same foreground and background color).
// The shared styles.TableStyles (which baseTableStyles delegates to) must
// always produce a visible header pair.
func TestTableHeaderColorsFallbackAvoidsSameFgBg(t *testing.T) {
	t.Parallel()

	c := &styles.AppStyle{
		Bg:     lipgloss.Color("0"),
		Accent: lipgloss.Color("5"),
		Styles: &styles.Styles{
			SelectedItem: lipgloss.NewStyle().
				Background(lipgloss.Color("7")).
				Foreground(lipgloss.Color("7")),
			TextOnBg: lipgloss.NewStyle().Foreground(lipgloss.Color("15")),
		},
	}

	s := styles.TableStyles(c)
	headerBG := styles.ColorHex(s.Header.GetBackground())
	headerFG := styles.ColorHex(s.Header.GetForeground())
	if strings.EqualFold(headerBG, headerFG) {
		t.Fatalf("regression: header foreground/background are identical (%s)", headerBG)
	}
}

func TestStackingStructMessages(t *testing.T) {
	t.Parallel()
	m := New()

	// Send the same struct message twice; it should stack into one entry
	_, _ = m.Update(navigation.SelectedMsg{PageIndex: 2})
	_, _ = m.Update(navigation.SelectedMsg{PageIndex: 2})

	if len(m.Logs) != 1 {
		t.Fatalf("expected 1 log entry after stacking struct messages; got %d", len(m.Logs))
	}
	if m.Logs[0].Count != 2 {
		t.Fatalf("expected stacked Count=2; got %d", m.Logs[0].Count)
	}
	expectedType := fmt.Sprintf("%T", navigation.SelectedMsg{})
	if m.Logs[0].Type != expectedType {
		t.Fatalf("logged Type = %q; want %q", m.Logs[0].Type, expectedType)
	}
}

func TestInspectorWheelScrollMovesVisibleWindow(t *testing.T) {
	t.Parallel()

	m := New()
	_, _ = m.Update(tea.WindowSizeMsg{Width: 90, Height: 40})
	for i := range 12 {
		m.AddLog("INFO", time.Now(), fmt.Sprintf("log-%02d", i))
	}
	// Drain pendingLogs into m.Logs via a no-op Update.
	_, _ = m.Update(tea.WindowSizeMsg{Width: 90, Height: 40})

	if len(m.Logs) == 0 {
		t.Fatal("expected log entries")
	}
	// After AddLog, scrollToBottom should be set; newest log should be present.
	if got, want := m.Logs[len(m.Logs)-1].Content, "log-11"; got != want {
		t.Fatalf("newest log = %q; want %q", got, want)
	}

	// Wheel-down is forwarded to the viewport (returns early, does not add a log entry).
	// Just verify it does not panic.
	_, _ = m.Update(tea.MouseWheelMsg(tea.Mouse{Button: tea.MouseWheelDown}))
	_, _ = m.Update(tea.MouseWheelMsg(tea.Mouse{Button: tea.MouseWheelUp}))
}

func TestInspectorWheelScrollClampsAtBounds(t *testing.T) {
	t.Parallel()

	m := New()
	_, _ = m.Update(tea.WindowSizeMsg{Width: 90, Height: 40})
	for i := range 30 {
		m.AddLog("INFO", time.Now(), fmt.Sprintf("row-%02d", i))
	}
	// Drain pendingLogs into m.Logs via a no-op Update.
	_, _ = m.Update(tea.WindowSizeMsg{Width: 90, Height: 40})

	// Wheel-down many times -- should not panic and viewport stays valid.
	for range 200 {
		_, _ = m.Update(tea.MouseWheelMsg(tea.Mouse{Button: tea.MouseWheelDown}))
	}

	// Wheel-up many times -- viewport stays valid.
	for range 200 {
		_, _ = m.Update(tea.MouseWheelMsg(tea.Mouse{Button: tea.MouseWheelUp}))
	}

	// After all the scrolling, logs should still be consistent.
	if len(m.Logs) == 0 {
		t.Fatal("expected non-empty log after scrolling")
	}
}

// TestRuntimeGridRendersAllMetrics: the Runtime tab's key-value grid carries
// every metric group and never renders a line wider than the section.
func TestRuntimeGridRendersAllMetrics(t *testing.T) {
	t.Parallel()

	m := New()
	_, _ = m.Update(tea.WindowSizeMsg{Width: 300, Height: 40})
	// Fix the timestamps so elapsed = 1 s (avoids / 0 fallback noise).
	base := time.Now()
	m.prevStats.CapturedAt = base.Add(-time.Second)
	m.stats.CapturedAt = base

	c := styles.Active()
	out := m.renderRuntimeSection(c, m.buildRuntimeRows(c), 120)
	for _, want := range []string{"Uptime", "Goroutines", "Heap Alloc", "GC Cycles", "Heap Objects"} {
		if !strings.Contains(out, want) {
			t.Errorf("runtime grid missing %q", want)
		}
	}
	for i, line := range strings.Split(out, "\n") {
		if lipgloss.Width(line) > 120 {
			t.Errorf("grid line %d overflows: width %d > 120", i, lipgloss.Width(line))
		}
	}
}

// TestKVGridReflowsToWidth: wide sections spread pairs across column pairs,
// narrow ones stack a single pair per line — and neither overflows.
func TestKVGridReflowsToWidth(t *testing.T) {
	t.Parallel()

	m := New()
	_, _ = m.Update(tea.WindowSizeMsg{Width: 300, Height: 40})
	c := styles.Active()
	pairs := flattenPairs(m.buildRuntimeRows(c))
	if len(pairs) < 8 {
		t.Fatalf("test premise: expected a rich pair set; got %d", len(pairs))
	}

	wide := renderKVGrid(c, pairs, 220)
	narrow := renderKVGrid(c, pairs, 24)
	if lipgloss.Height(narrow) <= lipgloss.Height(wide) {
		t.Errorf("narrow grid (%d lines) should stack more rows than wide (%d)",
			lipgloss.Height(narrow), lipgloss.Height(wide))
	}
	for i, line := range strings.Split(narrow, "\n") {
		if lipgloss.Width(line) > 24 {
			t.Errorf("narrow grid line %d overflows: width %d > 24", i, lipgloss.Width(line))
		}
	}
}

func TestStatusLineSummaryFollowsSettings(t *testing.T) {
	t.Parallel()

	m := New()
	m.stats = collectSnapshot(m.startTime)
	m.prevStats = m.stats

	if got := m.StatusLineSummary(); got != "" {
		t.Fatalf("expected empty summary when disabled, got %q", got)
	}

	m.statusSummary.Enabled = true
	got := m.StatusLineSummary()
	if !strings.Contains(got, "term") {
		t.Fatalf("expected summary to contain terminal info when enabled, got %q", got)
	}
}

func TestSettingsTabAdjustsRefreshInterval(t *testing.T) {
	t.Parallel()

	m := New()
	_, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m.activeTab = debugTabSettings
	m.settingsCursor = 0 // Latest-value refresh

	before := m.latestValueInterval
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // Enter increments by 100 ms
	if m.latestValueInterval <= before {
		t.Fatalf(
			"expected latest value interval to increase; before=%s after=%s",
			before,
			m.latestValueInterval,
		)
	}
}

func TestTabsAreClickableWithMouse(t *testing.T) {
	t.Parallel()

	m := New()
	_, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	v := m.View()
	if len(m.tabRanges) < 2 {
		t.Fatalf("expected tab hit ranges to be populated; got %d", len(m.tabRanges))
	}

	inputTab := m.tabRanges[1]
	// tabsOriginY is the actual Y row of the tab bar in the inner-content
	// coordinate space (after the router strips the top border char).
	tabsY := m.tabsOriginY
	if cmd := v.OnMouse(
		tea.MouseReleaseMsg(tea.Mouse{X: inputTab.StartX, Y: tabsY, Button: tea.MouseLeft}),
	); cmd != nil {
		_ = cmd()
	}

	if m.activeTab != debugTabInput {
		t.Fatalf("expected click to switch to Input tab; got %v", m.activeTab)
	}
}

func TestSettingsRowsAreMouseSelectableAndActionable(t *testing.T) {
	t.Parallel()

	m := New()
	_, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 16})
	m.switchTab(debugTabSettings)
	v := m.View()

	row := 2 // "Status summary on close"
	// One rendered line per row: help and status live in the pinned footer,
	// so line == row and clicks always land on the row under the pointer.
	y := m.sectionOriginY + row - m.sectionViewport.YOffset()
	if cmd := v.OnMouse(
		tea.MouseReleaseMsg(tea.Mouse{X: 2, Y: y, Button: tea.MouseLeft}),
	); cmd != nil {
		_ = cmd()
	}

	if m.settingsCursor != row {
		t.Fatalf("expected settings cursor=%d after click; got %d", row, m.settingsCursor)
	}
	if !m.statusSummary.Enabled {
		t.Fatal("expected clicked settings toggle row to execute Enter action")
	}
}

// TestSettingsRowForLineIsOneToOne pins the rendered-line → row mapping: the
// settings list is exactly one line per row (help and status render in the
// pinned footer), so the mapping is identity inside the list and -1 outside.
func TestSettingsRowForLineIsOneToOne(t *testing.T) {
	t.Parallel()

	m := New()
	_, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m.switchTab(debugTabSettings)
	items := m.settingsRows()
	if len(items) < 4 {
		t.Fatalf("test premise: need at least 4 rows; got %d", len(items))
	}

	clear(m.collapsedSections) // expand everything: mapping is pure identity
	m.settingsCursor = 0       // the selected row must NOT shift lines below it
	for _, line := range []int{0, 1, 2, 3, len(items) - 1} {
		if got := m.settingsRowForLine(items, line); got != line {
			t.Errorf("settingsRowForLine(line=%d) = %d; want identity", line, got)
		}
	}
	if got := m.settingsRowForLine(items, -1); got != -1 {
		t.Errorf("negative line mapped to %d; want -1", got)
	}
	if got := m.settingsRowForLine(items, len(items)); got != -1 {
		t.Errorf("line past the last row mapped to %d; want -1", got)
	}

	// With a section collapsed, its body rows drop out of the mapping: the
	// line after the header maps to the NEXT header, not a hidden row.
	m.collapsedSections[settingsRowBuiltinHeader] = true
	if got := m.settingsRowForLine(items, int(settingsRowBuiltinHeader)+1); got != int(settingsRowGotoolHeader) {
		t.Errorf("line after a collapsed header mapped to %d; want %d",
			got, int(settingsRowGotoolHeader))
	}
}

func TestPerTabScrollPreservedAcrossSwitches(t *testing.T) {
	t.Parallel()

	m := New()
	// Short terminal so the runtime grid is taller than the section and the
	// viewport actually scrolls.
	_, _ = m.Update(tea.WindowSizeMsg{Width: 60, Height: 10})
	_ = m.View()

	// Runtime renders a key-value grid in the section viewport: KeyDown
	// scrolls it like any non-table tab and records per-tab scroll state.
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if got := m.tabScrollY[debugTabRuntime]; got <= 0 {
		t.Fatalf("expected the runtime grid to scroll with KeyDown; offset=%d", got)
	}

	// Viewport tabs (Terminal) keep per-tab scroll state across switches.
	_, _ = m.Update(tea.KeyPressMsg{Text: "4"}) // Terminal tab
	_ = m.View()
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	before := m.tabScrollY[debugTabTerminal]
	if before <= 0 {
		t.Fatalf("expected terminal tab to scroll with KeyDown; got offset=%d", before)
	}

	_, _ = m.Update(tea.KeyPressMsg{Text: "6"}) // Log tab
	_ = m.View()
	_, _ = m.Update(tea.KeyPressMsg{Text: "4"}) // back to Terminal
	_ = m.View()

	if got := m.sectionViewport.YOffset(); got != before {
		t.Fatalf("expected terminal tab scroll offset to be restored; got %d want %d", got, before)
	}
}
