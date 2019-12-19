package cgrouplimits

// MemoryStats encapsulates memory limits, usage and available.
type MemoryStats struct {
	Total int64
	// Free treats data in the kernel-page-cache for the cgroup/system as
	// "used"
	Free int64
	// Available treats data in the kernel-page-cache as "available", also
	// ignores unused swap.
	Available int64

	// Number of OOM-kills either within the memory cgroup or on the host
	// (if available)
	OOMKills int64
}

// MemStats queries the system for the current cgroup (if available) and total
// memory usage, available, etc., returning a MemoryStats struct with the best
// available data.
// Note: swap memory/limits are not handled properly in cgroups. It is expected
// that swap will not be enabled in production.
// Note: hierarchical cgroups do not recurse, so limits for child cgroups may
// be incorrect if limits were applied a on a parent. (for now, this should be
// irrelevant for production under k8s/docker, as they set the cgroup limits
// explicitly.
func MemStats() (MemoryStats, error) {
	cgMI, cgErr := GetCgroupMemoryStats()
	if cgErr == ErrCGroupsNotSupported {
		return MemoryStats{}, ErrCGroupsNotSupported
	}
	if cgErr != nil {
		return MemoryStats{}, cgErr
	}
	ms, miErr := HostMemStats()
	if miErr != nil {
		return MemoryStats{}, miErr
	}

	if cgMI.Total > 0 && cgMI.Total < ms.Total {
		return cgMI, nil
	}

	return ms, nil
}
