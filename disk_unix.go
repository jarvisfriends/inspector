// Copyright (c) 2026 Jarvis Friends contributors
// SPDX-License-Identifier: MIT

//go:build !windows

package inspector

import (
	"strings"
	"syscall"
)

// listDriveStats enumerates mount points from /proc/mounts (Linux) or falls
// back to "/" on other Unix systems, then returns space info via Statfs.
func listDriveStats() []diskStat {
	mounts := collectMountPoints()
	var stats []diskStat
	for _, mp := range mounts {
		if strings.Contains(mp, "snap") || strings.Contains(mp, "/export") {
			continue // skip snap mounts that cause Statfs to fail
		}
		var st syscall.Statfs_t
		if err := syscall.Statfs(mp, &st); err != nil {
			continue // skip inaccessible or virtual fs
		}
		if st.Blocks == 0 {
			continue // zero-size: tmpfs, cgroup, etc.
		}
		bsize := uint64(max(0, st.Bsize))
		total := st.Blocks * bsize
		free := st.Bavail * bsize
		stats = append(stats, diskStat{
			Path:  mp,
			Total: total,
			Free:  free,
			Used:  total - free,
		})
	}
	if len(stats) == 0 {
		return []diskStat{{Path: "/", Error: "no physical mounts found"}}
	}
	return stats
}
