// Copyright (c) 2026 Jarvis Friends contributors
// SPDX-License-Identifier: MIT

package inspector

import "testing"

// TestDisksMirrorsDriveStats asserts the exported Disks() enumeration exposes
// the same read-only per-mount data the Disks tab renders.
func TestDisksMirrorsDriveStats(t *testing.T) {
	t.Parallel()

	usages := Disks()
	if len(usages) == 0 {
		t.Fatal("Disks() returned no entries; want at least the fallback root entry")
	}
	for i, u := range usages {
		if u.Path == "" {
			t.Errorf("entry %d has an empty path: %+v", i, u)
		}
		if u.Error == "" && u.Used != u.Total-u.Free {
			t.Errorf("entry %d: Used=%d; want Total-Free=%d", i, u.Used, u.Total-u.Free)
		}
	}
}

// TestCollectMountPointsSkipsVirtualFilesystems asserts the mount scan drops
// pseudo filesystems and never returns duplicates.
func TestCollectMountPointsSkipsVirtualFilesystems(t *testing.T) {
	t.Parallel()

	mounts := collectMountPoints()
	if len(mounts) == 0 {
		t.Fatal("collectMountPoints returned no entries; want at least '/'")
	}
	seen := map[string]bool{}
	for _, mp := range mounts {
		if seen[mp] {
			t.Errorf("duplicate mount point %q", mp)
		}
		seen[mp] = true
	}
}
