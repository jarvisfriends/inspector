// Copyright (c) 2026 Jarvis Friends contributors
// SPDX-License-Identifier: MIT

package inspector

import (
	"testing"
	"time"
)

// TestRegisterTabNilIsNoOp asserts a nil provider is rejected silently.
func TestRegisterTabNilIsNoOp(t *testing.T) {
	t.Parallel()

	m := New()
	m.RegisterTab(nil)
	if len(m.providers) != 0 {
		t.Fatalf("RegisterTab(nil) added a provider; providers=%d", len(m.providers))
	}
}

// TestRegisterTabStartsWhenVisible asserts a brand-new provider registered on
// an open inspector starts immediately.
func TestRegisterTabStartsWhenVisible(t *testing.T) {
	t.Parallel()

	m := New()
	m.ToggleVisible()
	p := &fakeProvider{name: "Live"}
	m.RegisterTab(p)
	if p.starts != 1 {
		t.Fatalf("starts=%d; want 1 (registered while visible)", p.starts)
	}

	// A second, differently named provider appends rather than replaces.
	p2 := &fakeProvider{name: "Other"}
	m.RegisterTab(p2)
	if len(m.providers) != 2 || p2.starts != 1 || p.stops != 0 {
		t.Fatalf("distinct registration went wrong: providers=%d p2.starts=%d p.stops=%d",
			len(m.providers), p2.starts, p.stops)
	}
}

// TestRemoveTabUnknownNameIsNoOp asserts removing an unregistered name leaves
// existing providers untouched.
func TestRemoveTabUnknownNameIsNoOp(t *testing.T) {
	t.Parallel()

	m := New()
	p := &fakeProvider{name: "Keep"}
	m.RegisterTab(p)
	m.RemoveTab("Missing")
	if len(m.providers) != 1 || p.stops != 0 {
		t.Fatalf("RemoveTab(unknown) disturbed providers: len=%d stops=%d", len(m.providers), p.stops)
	}
}

// TestTabTitleAndProviderLookupBounds covers the out-of-range lookups: no
// provider backs a tab index past the registered range, and its title is
// empty.
func TestTabTitleAndProviderLookupBounds(t *testing.T) {
	t.Parallel()

	m := New()
	if got := m.tabTitle(debugTab(len(debugTabTitles) + 3)); got != "" {
		t.Fatalf("tabTitle out of range = %q; want empty", got)
	}
	if m.providerForTab(debugTabRuntime) != nil {
		t.Fatal("built-in tabs must not resolve to a provider")
	}
	if m.providerForTab(debugTab(len(debugTabTitles))) != nil {
		t.Fatal("empty provider list must resolve to nil")
	}
}

// TestTickProviderRefreshGuards covers the early returns: built-in active tab
// (no provider) and a provider with no forced refresh interval.
func TestTickProviderRefreshGuards(t *testing.T) {
	t.Parallel()

	m := New()
	m.dirty = false
	m.tickProviderRefresh() // active tab is Runtime: no provider
	if m.dirty {
		t.Fatal("tick with no provider tab must not dirty the view")
	}

	p := &fakeProvider{name: "Static", refreshEvery: 0}
	m.RegisterTab(p)
	m.activeTab = debugTab(len(debugTabTitles))
	m.dirty = false
	m.tickProviderRefresh()
	if m.dirty {
		t.Fatal("provider without RefreshInterval must not force redraws")
	}

	// An elapsed interval with no recorded render time forces a redraw.
	fast := &fakeProvider{name: "Static", refreshEvery: time.Nanosecond}
	m.RegisterTab(fast)
	m.dirty = false
	m.tickProviderRefresh()
	if !m.dirty {
		t.Fatal("elapsed interval should dirty the view")
	}
}
