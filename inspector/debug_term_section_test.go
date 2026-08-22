// Copyright (c) 2026 Jarvis Friends contributors
// SPDX-License-Identifier: MIT

package inspector

import (
	"image/color"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/jarvisfriends/snap/styles"
)

// TestBuildTermSectionSSHColorHint covers the washed-out-colors remedy: an
// ANSI256 profile over SSH with no COLORTERM and no override must surface the
// hint and fix rows.
func TestBuildTermSectionSSHColorHint(t *testing.T) {
	t.Setenv("SSH_CLIENT", "192.0.2.1 12345 22")
	t.Setenv("SSH_TTY", "")
	t.Setenv("COLORTERM", "")
	t.Setenv(defaultColorProfileEnvVar, "")

	_ = styles.SetCurrentTint("dracula")
	m := New()
	m.SetColors(styles.Active())
	_, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	_, _ = m.Update(TermDiagMsg{
		DetectedBg: color.RGBA{R: 0x28, G: 0x2a, B: 0x36, A: 0xff},
		BgIsDark:   true,
		Profile:    colorprofile.ANSI256,
	})

	out := m.buildTermSection(m.Colors(), 116)
	for _, want := range []string{
		"YES — SSH_CLIENT=192.0.2.1",
		"Color hint",
		"quantized to ANSI256",
		"Fix",
		"COLORTERM=truecolor",
		"#282a36", // detected background hex from the swatch row
		"true",    // Dark Background
	} {
		if !strings.Contains(out, want) {
			t.Errorf("terminal section missing %q", want)
		}
	}
}

// TestBuildTermSectionProfileOverride covers the forced-profile annotation
// when the app's color-profile env var is set.
func TestBuildTermSectionProfileOverride(t *testing.T) {
	t.Setenv("SSH_CLIENT", "")
	t.Setenv("SSH_TTY", "")
	t.Setenv(defaultColorProfileEnvVar, "truecolor")

	m := New()
	m.SetColors(styles.Active())
	out := m.buildTermSection(m.Colors(), 116)
	if !strings.Contains(out, "forced: "+defaultColorProfileEnvVar+"=truecolor") {
		t.Fatalf("terminal section missing the forced-profile annotation:\n%s", out)
	}
	if !strings.Contains(out, "SSH") {
		t.Fatal("terminal section missing the SSH row")
	}
}

// TestBuildTermSectionActiveThemeSwatches asserts the active tint's key
// colors render with their hex values once a tint is current.
func TestBuildTermSectionActiveThemeSwatches(t *testing.T) {
	t.Parallel()

	_ = styles.SetCurrentTint("dracula")
	m := New()
	m.SetColors(styles.Active())
	out := m.buildTermSection(m.Colors(), 116)
	for _, want := range []string{"Tint", "dracula", "Background", "Foreground", "Accent", "Selection Bg"} {
		if !strings.Contains(out, want) {
			t.Errorf("terminal section missing theme row %q", want)
		}
	}
}
