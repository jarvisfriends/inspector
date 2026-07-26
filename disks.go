// Copyright (c) 2026 Jarvis Friends contributors
// SPDX-License-Identifier: MIT

package inspector

import (
	"bufio"
	"log"
	"os"
	"strings"
)

// DiskUsage reports space information for a single mounted drive or volume —
// the exported form of the per-drive data shown on the Inspector's Disks tab.
// An entry whose Error is non-empty could not be queried (Total/Free/Used are
// then meaningless).
type DiskUsage struct {
	Path  string // drive letter ("C:\\") or mount point ("/", "/home")
	Total uint64 // total capacity in bytes
	Free  uint64 // bytes available to the caller (respects per-user quotas)
	Used  uint64 // Total - Free
	Error string // non-empty if the drive could not be queried
}

// Disks enumerates the machine's mounted drives/volumes with their space usage.
// It is cross-platform: Windows logical drives (A:\ … Z:\) and Unix mount points
// (from /proc/mounts on Linux, "/" elsewhere). This is the same collection the
// Inspector's Disks tab renders, exported so applications can surface it too.
func Disks() []DiskUsage {
	stats := listDriveStats()
	out := make([]DiskUsage, len(stats))
	for i, d := range stats {
		out[i] = DiskUsage(d)
	}
	return out
}

// collectMountPoints reads /proc/mounts on Linux; falls back to ["/"] elsewhere.
func collectMountPoints() []string {
	f, err := os.Open("/proc/mounts")
	if err != nil {
		return []string{"/"}
	}
	defer func() {
		e := f.Close()
		if e != nil {
			// handle error if needed
			log.Default().Printf("warning: failed to close /proc/mounts: %v", e)
		}
	}()

	// Filesystem types that carry no real disk data.
	skipTypes := map[string]bool{
		"proc": true, "sysfs": true, "devtmpfs": true, "devpts": true,
		"tmpfs": true, "cgroup": true, "cgroup2": true, "debugfs": true,
		"tracefs": true, "securityfs": true, "pstore": true, "bpf": true,
		"fusectl": true, "hugetlbfs": true, "mqueue": true, "configfs": true,
		"efivarfs": true, "none": true, "overlay": true, "aufs": true,
		"rpc_pipefs": true, "nfsd": true,
	}

	seen := map[string]bool{}
	var mounts []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 3 {
			continue
		}
		mp, fsType := fields[1], fields[2]
		if skipTypes[fsType] || seen[mp] {
			continue
		}
		seen[mp] = true
		mounts = append(mounts, mp)
	}
	if len(mounts) == 0 {
		return []string{"/"}
	}
	return mounts
}
