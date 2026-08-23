// Copyright (c) 2026 Jarvis Friends contributors
// SPDX-License-Identifier: MIT

package inspector

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	"github.com/jarvisfriends/snap/styles"
)

// TestRenderDisksSectionSeverityBands drives every severity branch of the
// Disks table: error rows, low/medium/plenty free space, and the used-percent
// warning bands, with injected disk stats for determinism.
func TestRenderDisksSectionSeverityBands(t *testing.T) {
	t.Parallel()

	const gib = uint64(1024 * 1024 * 1024)
	m := New()
	_, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m.stats.Disks = []diskStat{
		{Path: "/bad", Error: "permission denied"},
		{Path: "/full", Total: 100 * gib, Used: 95 * gib, Free: 50 * 1024 * 1024}, // <100MiB free, ≥90% used
		{Path: "/tight", Total: 100 * gib, Used: 80 * gib, Free: gib / 2},         // <1GiB free, ≥75% used
		{Path: "/roomy", Total: 100 * gib, Used: 10 * gib, Free: 90 * gib},        // green everywhere
		{Path: "/nil", Total: 0, Used: 0, Free: 2 * gib},                          // Total==0: pct stays 0
	}

	out := m.renderDisksSection(styles.Active(), m.baseTableStyles(styles.Active()))
	for _, want := range []string{"/bad", "permission denied", "/full", "/tight", "/roomy", "/nil"} {
		if !strings.Contains(out, want) {
			t.Errorf("disks section missing %q", want)
		}
	}
	if !strings.Contains(out, "95%") || !strings.Contains(out, "80%") {
		t.Errorf("disks section missing use%% values:\n%s", out)
	}
}

// TestDisksTabRendersViaView asserts the Disks tab renders through the full
// View pipeline with the machine's real (read-only) mount data.
func TestDisksTabRendersViaView(t *testing.T) {
	t.Parallel()

	m := New()
	_, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m.switchTab(debugTabDisks)
	v := m.View()
	// The section title lives in the pinned footer; the brand rides the tab
	// bar's right edge (the old centered "<Title> (Inspector)" line is gone).
	if !strings.Contains(v.Content, "Disks") {
		t.Fatalf("disks tab title missing from view")
	}
	if !strings.Contains(v.Content, "Inspector") {
		t.Fatalf("brand missing from the tab bar")
	}
}

// TestSectionForActiveTabAllBranches walks sectionForActiveTab through every
// tab kind, including the accessibility fallbacks and the provider default.
func TestSectionForActiveTabAllBranches(t *testing.T) {
	t.Parallel()

	m := New()
	m.SetGates(newGatesWithAccessibility(true))
	_, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	c := styles.Active()
	s := m.baseTableStyles(c)
	inputRows := m.buildInputRows(c)

	section := func(tab debugTab) (string, string) {
		m.activeTab = tab
		return m.sectionForActiveTab(c, 96, s, "runtime-section", inputRows, "log-content")
	}

	// Each built-in tab's section title embeds its tab-bar label.
	var runtimeTitle string
	for _, tab := range []debugTab{
		debugTabRuntime, debugTabInput, debugTabDisks,
		debugTabTerminal, debugTabLog, debugTabSettings,
	} {
		title, content := section(tab)
		if !strings.Contains(title, m.tabTitle(tab)) {
			t.Errorf("tab %v title = %q; want it to contain %q", tab, title, m.tabTitle(tab))
		}
		if content == "" {
			t.Errorf("tab %v rendered empty content", tab)
		}
		if tab == debugTabRuntime {
			runtimeTitle = title
		}
	}

	// Log tab title gains the filter suffix when a level floor is active.
	m.logLevelFloor = logLevelRankWarn
	if title, _ := section(debugTabLog); !strings.Contains(title, "[WARN+ only]") {
		t.Errorf("filtered log title = %q; want WARN+ suffix", title)
	}
	m.logLevelFloor = 1
	if title, _ := section(debugTabLog); !strings.Contains(title, "[INFO+ only]") {
		t.Errorf("filtered log title = %q; want INFO+ suffix", title)
	}
	m.logLevelFloor = 0

	// Accessibility with a live panel: panel content is embedded.
	m.switchTab(debugTabAccessibility) // toggles the panel visible
	if title, content := section(debugTabAccessibility); title != debugTabTitleAccessibility ||
		!strings.Contains(content, "COLOR ACCESSIBILITY BROWSER") {
		t.Errorf("accessibility section = %q / %.40q", title, content)
	}

	// Accessibility without a panel falls back to the placeholder.
	m.acPanel = nil
	if _, content := section(debugTabAccessibility); !strings.Contains(content, "not available") {
		t.Errorf("nil-panel accessibility content = %q", content)
	}

	// Provider tabs render through the provider; unknown tabs fall back to
	// the runtime section.
	p := &fakeProvider{name: "Ext", rows: []string{"row-a", "row-b"}}
	m.providers = append(m.providers, p)
	title, content := section(debugTab(len(debugTabTitles)))
	if title != "Ext" || !strings.Contains(content, "row-a") {
		t.Errorf("provider section = %q / %q", title, content)
	}
	title, content = section(debugTab(len(debugTabTitles) + 5))
	if title != runtimeTitle || content != "runtime-section" {
		t.Errorf("unknown tab fallback = %q / %q", title, content)
	}
}

// TestRenderInputSectionFlatFallback asserts the input grid reflows at
// narrow widths without overflowing and never claims a table cursor.
func TestRenderInputSectionFlatFallback(t *testing.T) {
	t.Parallel()

	m := New()
	_, _ = m.Update(tea.WindowSizeMsg{Width: 40, Height: 20})
	c := styles.Active()
	rows := m.buildInputRows(c)

	out := m.renderInputSection(c, rows, 30)
	if m.tableActive[debugTabInput] {
		t.Fatal("input renders as a grid; it must never mark a table active")
	}
	for i, line := range strings.Split(out, "\n") {
		if lipgloss.Width(line) > 30 {
			t.Errorf("input grid line %d overflows: width %d > 30", i, lipgloss.Width(line))
		}
	}
	// The narrow key column truncates long metric names, so probe short ones.
	for _, want := range []string{"Button", "Mod"} {
		if !strings.Contains(out, want) {
			t.Fatalf("input grid missing %q:\n%s", want, out)
		}
	}
}

// TestRenderLogContentEmptyStates pins the two placeholder messages.
func TestRenderLogContentEmptyStates(t *testing.T) {
	t.Parallel()

	m := New()
	c := styles.Active()
	if got := m.renderLogContent(c); !strings.Contains(got, "No messages intercepted yet") {
		t.Errorf("empty log = %q; want placeholder", got)
	}
	m.logLevelFloor = logLevelRankWarn
	if got := m.renderLogContent(c); !strings.Contains(got, "No WARN+ messages") {
		t.Errorf("empty filtered log = %q; want WARN+ placeholder", got)
	}
}

// TestColorStatSeverityThresholds is a table-driven pin of the three
// severity styles chosen by colorStat.
func TestColorStatSeverityThresholds(t *testing.T) {
	t.Parallel()

	_ = styles.SetCurrentTint("dracula")
	m := New()
	c := styles.Active()

	tests := []struct {
		name string
		val  float64
		want string
	}{
		{"ok", 5, c.Styles.Success.Render("v")},
		{"warn", 50, c.Styles.Warning.Render("v")},
		{"crit", 500, c.Styles.Error.Bold(true).Render("v")},
	}
	for _, tc := range tests {
		if got := m.colorStat(c, tc.val, 10, 100, "v"); got != tc.want {
			t.Errorf("%s: colorStat = %q; want %q", tc.name, got, tc.want)
		}
	}
}

// TestRenderRuntimeFlatSkipsEmptyKeys asserts pairs with empty metric names
// are dropped when the legacy row shape is flattened for the grid.
func TestRenderRuntimeFlatSkipsEmptyKeys(t *testing.T) {
	t.Parallel()

	c := styles.Active()
	out := renderKVGrid(
		c,
		flattenPairs([]table.Row{
			{"Metric A", "1", "", "hidden", "Metric B", "2"},
		}),
		60,
	)
	if strings.Contains(out, "hidden") {
		t.Fatalf("grid kept a pair with an empty key:\n%s", out)
	}
	for _, want := range []string{"Metric A", "Metric B"} {
		if !strings.Contains(out, want) {
			t.Errorf("grid missing %q", want)
		}
	}
}

// TestLogLevelRank pins the level ordering used by the WARN+ filter.
func TestLogLevelRank(t *testing.T) {
	t.Parallel()

	// logLevelRank upper-cases internally, so lowercase inputs rank the same.
	tests := []struct {
		in   string
		want int
	}{
		{"debug", 0},
		{"info", 1},
		{"warn", 2},
		{"warning", 2},
		{"error", 3},
		{"inspector.statsTickMsg", -1},
	}
	for _, tc := range tests {
		if got := logLevelRank(tc.in); got != tc.want {
			t.Errorf("logLevelRank(%q) = %d; want %d", tc.in, got, tc.want)
		}
	}
}
