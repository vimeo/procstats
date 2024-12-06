//go:build linux
// +build linux

package cgrouplimits

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/opencontainers/runc/libcontainer/cgroups"
	"github.com/opencontainers/runc/libcontainer/cgroups/fs"

	"github.com/vimeo/procstats"
	"github.com/vimeo/procstats/pparser"
)

const cgroupCFSQuotaFile = "cpu.cfs_quota_us"
const cgroupCFSPeriodFile = "cpu.cfs_period_us"

const cgroupMemLimitFile = "memory.limit_in_bytes"

const cgroupMemOOMControlFile = "memory.oom_control"

// GetCgroupCPULimit fetches the Cgroup's CPU limit
func GetCgroupCPULimit() (float64, error) {
	cpuPath, cgroupFindErr := cgroups.GetOwnCgroupPath("cpu")
	if cgroupFindErr != nil {
		return -1.0, fmt.Errorf("unable to find cgroup directory: %s", cgroupFindErr)
	}

	quotaFilePath := filepath.Join(cpuPath, cgroupCFSQuotaFile)
	quotaStr, quotaReadErr := os.ReadFile(quotaFilePath)
	if quotaReadErr != nil {
		return -1.0, fmt.Errorf("failed to read quota file %q: %s", quotaFilePath, quotaReadErr)
	}
	enforcePeriodFilePath := filepath.Join(cpuPath, cgroupCFSPeriodFile)
	enforcePeriodStr, periodReadErr := os.ReadFile(enforcePeriodFilePath)
	if periodReadErr != nil {
		return -1.0, fmt.Errorf("failed to read cfs period file %q: %s",
			enforcePeriodFilePath, periodReadErr)
	}
	quotaµs, parseQuotaErr := strconv.Atoi(strings.TrimSpace(string(quotaStr)))
	if parseQuotaErr != nil {
		return -1.0, fmt.Errorf("failed to parse contents of %q as integer: %s",
			quotaFilePath, parseQuotaErr)
	}
	periodµs, parsePeriodErr := strconv.Atoi(strings.TrimSpace(string(enforcePeriodStr)))
	if parsePeriodErr != nil {
		return -1.0, fmt.Errorf("failed to parse contents of %q as integer: %s",
			enforcePeriodFilePath, parsePeriodErr)
	}

	if periodµs <= 0 {
		return 0.0, nil
	}
	if quotaµs <= 0 {
		return 0.0, nil
	}
	return float64(quotaµs) / float64(periodµs), nil
}

// GetCgroupMemoryLimit looks up the current process's memory cgroup, and
// returns the memory limit.
func GetCgroupMemoryLimit() (int64, error) {
	memPath, cgroupFindErr := cgroups.GetOwnCgroupPath("memory")
	if cgroupFindErr != nil {
		return -1, fmt.Errorf("unable to find cgroup directory: %s", cgroupFindErr)
	}
	limitFilePath := filepath.Join(memPath, cgroupMemLimitFile)
	limitFileContents, limitReadErr := os.ReadFile(limitFilePath)
	if limitReadErr != nil {
		return -1, fmt.Errorf("failed to read cgroup memory limit file %q: %s",
			limitFilePath, limitReadErr)
	}
	limitBytes, parseLimitErr := strconv.ParseInt(strings.TrimSpace(string(limitFileContents)), 10, 64)
	if parseLimitErr != nil {
		return -1, fmt.Errorf("failed to parse contents of %q as integer: %s",
			limitFilePath, parseLimitErr)
	}
	return limitBytes, nil
}

// GetCgroupMemoryStats queries the current process's memory cgroup's memory
// usage/limits.
func GetCgroupMemoryStats() (MemoryStats, error) {
	memPath, cgroupFindErr := cgroups.GetOwnCgroupPath("memory")
	if cgroupFindErr != nil {
		return MemoryStats{}, fmt.Errorf("unable to find cgroup directory: %s", cgroupFindErr)
	}
	mg := fs.MemoryGroup{}
	st := cgroups.NewStats()
	if err := mg.GetStats(memPath, st); err != nil {
		return MemoryStats{}, fmt.Errorf("failed to query memory stats: %s", err)
	}
	msUsage := st.MemoryStats.Usage

	ooms, oomErr := getCgroupOOMs(memPath)
	if oomErr != nil {
		return MemoryStats{}, fmt.Errorf("failed to look up OOMKills: %s",
			oomErr)
	}

	ms := MemoryStats{
		Total: int64(msUsage.Limit),
		Free:  int64(msUsage.Limit) - int64(msUsage.Usage),
		Available: int64(msUsage.Limit) - int64(msUsage.Usage) +
			int64(st.MemoryStats.Cache),
		OOMKills: int64(ooms),
	}
	return ms, nil
}

// MemCgroupOOMControl contains the parsed contents of the cgroup's
// memory.oom_control file.
// Note that this struct is a linux-specific data-structure that should not be
// used in portable applications.
type MemCgroupOOMControl struct {
	OomKillDisable int64            `pparser:"oom_kill_disable"`
	UnderOom       int64            `pparser:"under_oom"`
	OomKill        int64            `pparser:"oom_kill"`
	UnknownFields  map[string]int64 `pparser:"skip,unknown"`
}

// ReadCGroupOOMControl reads the oom_control file for the cgroup directory
// passed as an argument. Parsing the contents into a MemCgroupOOMControl
// struct.
// Note that this is a non-portable linux-specific function that should not be
// used in portable applications.
func ReadCGroupOOMControl(memCgroupPath string) (MemCgroupOOMControl, error) {
	oomControlPath := filepath.Join(memCgroupPath, cgroupMemOOMControlFile)
	oomControlBytes, oomControlReadErr := os.ReadFile(oomControlPath)
	if oomControlReadErr != nil {
		return MemCgroupOOMControl{}, fmt.Errorf(
			"failed to read contents of %q: %s",
			oomControlPath, oomControlReadErr)
	}
	oomc := MemCgroupOOMControl{}
	parseErr := memCgroupOOMControlFieldIdx.Parse(oomControlBytes, &oomc)
	if parseErr != nil {
		return MemCgroupOOMControl{}, parseErr
	}
	return oomc, nil
}

var memCgroupOOMControlFieldIdx = pparser.NewLineKVFileParser(MemCgroupOOMControl{}, " ")

// getCgroupOOMs looks up the current number of oom kills for the cgroup
// specified by the path in its argument.
func getCgroupOOMs(memCgroupPath string) (int32, error) {
	oomc, readErr := ReadCGroupOOMControl(memCgroupPath)
	if readErr != nil {
		return 0, readErr
	}

	// The oom_kill line was only added to the oom_control file in linux
	// 4.13, so some systems (docker for Mac) don't have it.
	return int32(oomc.OomKill), nil
}

// GetCgroupCPUStats queries the current process's memory cgroup's CPU
// usage/limits.
func GetCgroupCPUStats() (CPUStats, error) {
	cpuPath, cgroupFindErr := cgroups.GetOwnCgroupPath("cpu")
	if cgroupFindErr != nil {
		return CPUStats{}, fmt.Errorf("unable to find cgroup directory: %s",
			cgroupFindErr)
	}
	cpuAcctPath, cgroupFindErr := cgroups.GetOwnCgroupPath("cpuacct")
	if cgroupFindErr != nil {
		return CPUStats{}, fmt.Errorf("unable to find cgroup directory: %s",
			cgroupFindErr)
	}
	cg := fs.CpuGroup{}
	st := cgroups.NewStats()
	if err := cg.GetStats(cpuPath, st); err != nil {
		return CPUStats{}, fmt.Errorf("failed to query CPU throttle stats: %s", err)
	}
	cag := fs.CpuacctGroup{}
	if err := cag.GetStats(cpuAcctPath, st); err != nil {
		return CPUStats{}, fmt.Errorf("failed to query CPU acct stats: %s", err)
	}

	cs := CPUStats{
		Usage: procstats.CPUTime{
			Utime: time.Duration(st.CpuStats.CpuUsage.UsageInUsermode) *
				time.Nanosecond,
			Stime: time.Duration(st.CpuStats.CpuUsage.UsageInKernelmode) *
				time.Nanosecond,
		},
		ThrottledTime: time.Duration(st.CpuStats.ThrottlingData.ThrottledTime) *
			time.Nanosecond,
	}

	return cs, nil
}
