// Package cgrouplimits provides abstractions for getting resource usage on various
// platforms and environments. e.g. it supports running with and
// without cgroups (containers) as well as darwin.
package cgrouplimits

import (
	"runtime"
	"time"

	"github.com/vimeo/procstats"
)

// CPU gets any limit from the current cgroup (if on a supported system),
// and then chooses the limiting limit from runtime.NumCPU() and the
// cgroup-limit.
func CPU() float64 {
	runtimeLimit := float64(runtime.NumCPU())
	cgroupLimit, cgroupErr := GetCgroupCPULimit()
	if cgroupErr != nil {
		// if we got an error, fall back to using the runtime-derived
		// limit. (under linux this uses CPU affinity so it takes into
		// account how many cores we can actually run on)
		return runtimeLimit
	}
	if cgroupLimit <= 0 || runtimeLimit < cgroupLimit {
		return runtimeLimit
	}
	return cgroupLimit
}

// CPUStats encapuslates the CPU Limit, throttling, etc.
type CPUStats struct {
	Limit         float64
	Usage         procstats.CPUTime
	ThrottledTime time.Duration
}

// CPUStat queries the current system-state for CPU usage and limits.
// Limit is always filled in, other fields are only present if there's a
// non-nil error.
// Currently only works within cgroups with cpu-limits (CS-34)
func CPUStat() (CPUStats, error) {
	cgcpustats, err := GetCgroupCPUStats()
	// TODO(CS-34): implement a host-level fallback for the non-l-limit
	// fields that are a useful approximation of the cgroup
	// usage/throttle-time me fields.
	if err != nil {
		return CPUStats{Limit: CPU()}, err
	}
	cgcpustats.Limit = CPU()
	return cgcpustats, nil
}
