//go:build !linux
// +build !linux

package cgrouplimits

// GetCgroupCPULimit fetches the Cgroup's CPU limit
func GetCgroupCPULimit() (float64, error) {
	return 0.0, ErrCGroupsNotSupported
}

// GetCgroupCPUStats gets Cgroup CPU Stats
func GetCgroupCPUStats() (CPUStats, error) {
	return CPUStats{}, ErrCGroupsNotSupported
}

// GetCgroupMemoryLimit looks up the current process's memory cgroup, and
// returns the memory limit. (on unsupported systems it returns
// ErrCGroupsNotSupported)
func GetCgroupMemoryLimit() (int64, error) {
	return 0, ErrCGroupsNotSupported
}

// GetCgroupMemoryStats queries the current process's memory cgroup's memory
// usage/limits.
func GetCgroupMemoryStats() (MemoryStats, error) {
	return MemoryStats{}, ErrCGroupsNotSupported
}
