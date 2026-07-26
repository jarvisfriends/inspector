// Copyright (c) 2026 Jarvis Friends contributors
// SPDX-License-Identifier: MIT

package inspector_test

import (
	"testing"

	"github.com/jarvisfriends/snap/rendercheck"
)

// TestCodeStandards runs snap's AST code-standard checks over the whole
// module.
func TestCodeStandards(t *testing.T) {
	rendercheck.CheckCodeStandards(t, "github.com/jarvisfriends/inspector/...")
	rendercheck.CheckDescriptiveStructNames(t, "github.com/jarvisfriends/inspector/...")
}
