package inspector

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
