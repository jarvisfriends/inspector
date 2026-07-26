// Copyright (c) 2026 Jarvis Friends contributors
// SPDX-License-Identifier: MIT

package inspector

import "testing"

func FuzzLogMessage(f *testing.F) {
	f.Add("seed")
	f.Fuzz(func(t *testing.T, s string) {
		m := New()
		// Pass arbitrary string messages to verify logging is robust.
		_, _ = m.Update(s)
	})
}
