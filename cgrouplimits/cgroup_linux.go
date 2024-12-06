//go:build linux
// +build linux

package cgrouplimits

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vimeo/procstats"
	"github.com/vimeo/procstats/cgresolver"
	"github.com/vimeo/procstats/pparser"
)

const (
	cgroupCpuStatFile = "cpu.stat"
	cgroupMemStatFile = "memory.stat"

	// cgroups V1 files
	cgroupV1CFSQuotaFile  = "cpu.cfs_quota_us"
	cgroupV1CFSPeriodFile = "cpu.cfs_period_us"

	cgroupV1CpuUserUsageFile = "cpuacct.usage_user"
	cgroupV1CpuSysUsageFile  = "cpuacct.usage_sys"
	cgroupV1CpuAcctStatFile  = "cpuacct.stat"

	cgroupV1MemLimitFile = "memory.limit_in_bytes"
	cgroupV1MemUsageFile = "memory.usage_in_bytes"

	cgroupV1MemOOMControlFile = "memory.oom_control"

	// cgroups V2 files
	cgroupV2CFSQuotaPeriodFile = "cpu.max"
	cgroupV2MemLimitFile       = "memory.max"
	cgroupV2MemEventsFile      = "memory.events"
	cgroupV2MemCurrentFile     = "memory.current"
)

func getCGroupCPULimitSingle(cpuPath *cgresolver.CGroupPath) (float64, error) {
	switch cpuPath.Mode {
	case cgresolver.CGModeV1:
		f := os.DirFS(cpuPath.AbsPath)

		quotaµs, quotaReadErr := readIntValFile(f, cgroupV1CFSQuotaFile)
		if quotaReadErr != nil {
			return -1.0, fmt.Errorf("failed to read quota file %s", quotaReadErr)
		}
		periodµs, periodReadErr := readIntValFile(f, cgroupV1CFSPeriodFile)
		if periodReadErr != nil {
			return -1.0, fmt.Errorf("failed to read cfs period file: %s", periodReadErr)
		}
		if periodµs <= 0 {
			return 0.0, nil
		}
		if quotaµs <= 0 {
			return 0.0, nil
		}
		return float64(quotaµs) / float64(periodµs), nil
	case cgresolver.CGModeV2:
		maxPath := filepath.Join(cpuPath.AbsPath, cgroupV2CFSQuotaPeriodFile)
		quotaStr, quotaReadErr := os.ReadFile(maxPath)
		if quotaReadErr != nil {
			return -1.0, fmt.Errorf("failed to read max CPU file %q: %w", maxPath, quotaReadErr)
		}
		maxParts := strings.Fields(string(quotaStr))
		if len(maxParts) != 2 {
			return -1.0, fmt.Errorf("unable to parse %q; unexpected number of components: %d", maxPath, len(maxParts))
		}
		if maxParts[0] == "max" {
			// max == no limit :)
			return 0.0, nil
		}
		limitμs, parseLimitErr := strconv.Atoi(maxParts[0])
		if parseLimitErr != nil {
			return -1.0, fmt.Errorf("failed to parse limit component of %q as integer: %w",
				maxPath, parseLimitErr)
		}
		periodμs, parsePeriodErr := strconv.Atoi(maxParts[1])
		if parsePeriodErr != nil {
			return -1.0, fmt.Errorf("failed to parse period component of %q as integer: %w",
				maxPath, parsePeriodErr)
		}
		if limitμs <= 0 || periodμs <= 0 {
			return 0.0, nil
		}

		return float64(limitμs) / float64(periodμs), nil
	default:
		return -1.0, fmt.Errorf("unknown cgroup type: %d", cpuPath.Mode)
	}
}

// GetCgroupCPULimit fetches the Cgroup's CPU limit
func GetCgroupCPULimit() (float64, error) {
	cpuPath, cgroupFindErr := cgresolver.SelfSubsystemPath("cpu")
	if cgroupFindErr != nil {
		return -1.0, fmt.Errorf("unable to find cgroup directory: %s", cgroupFindErr)
	}

	minLimit := math.Inf(+1)
	allFailed := true
	leafCGReadErr := error(nil)

	for newDir := true; newDir; cpuPath, newDir = cpuPath.Parent() {
		cgLim, cgReadErr := getCGroupCPULimitSingle(&cpuPath)
		if cgReadErr != nil {
			if leafCGReadErr == nil && allFailed {
				leafCGReadErr = cgReadErr
			}
			continue
		}

		allFailed = false
		if (cgLim != -1 && cgLim != 0.0) && cgLim < minLimit {
			minLimit = cgLim
		}
	}
	if allFailed {
		return -1, leafCGReadErr
	}
	return minLimit, nil
}

// GetCgroupMemoryLimit looks up the current process's memory cgroup, and
// returns the memory limit.
func GetCgroupMemoryLimit() (int64, error) {
	memPath, cgroupFindErr := cgresolver.SelfSubsystemPath("memory")
	if cgroupFindErr != nil {
		return -1, fmt.Errorf("unable to find cgroup directory: %s", cgroupFindErr)
	}
	memLimitFilename := ""
	switch memPath.Mode {
	case cgresolver.CGModeV1:
		memLimitFilename = cgroupV1MemLimitFile
	case cgresolver.CGModeV2:
		memLimitFilename = cgroupV2MemLimitFile
	default:
		return -1, fmt.Errorf("unknown cgroup type: %d", memPath.Mode)
	}

	minLimit := int64(math.MaxInt64)

	allFailed := true
	leafCGReadErr := error(nil)

	for newDir := true; newDir; memPath, newDir = memPath.Parent() {
		f := os.DirFS(memPath.AbsPath)

		limitBytes, limitReadErr := readIntValFile(f, memLimitFilename)
		if limitReadErr != nil {
			if leafCGReadErr == nil && allFailed {
				leafCGReadErr = fmt.Errorf("failed to read cgroup memory limit file %s", limitReadErr)
			}
			continue
		}
		allFailed = false
		if limitBytes > 0 && limitBytes < minLimit {
			minLimit = limitBytes
		}
	}
	if allFailed {
		return -1, leafCGReadErr
	}
	return minLimit, nil
}

type cg1MemoryStatContents struct {
	Cache                      int64 `pparser:"cache"`
	RSS                        int64 `pparser:"rss"`
	RSSHuge                    int64 `pparser:"rss_huge"`
	Shmem                      int64 `pparser:"shmem"`
	MappedFile                 int64 `pparser:"mapped_file"`
	Dirty                      int64 `pparser:"dirty"`
	Writeback                  int64 `pparser:"writeback"`
	WorkingsetRefaultAnon      int64 `pparser:"workingset_refault_anon"`
	WorkingsetRefaultFile      int64 `pparser:"workingset_refault_file"`
	Swap                       int64 `pparser:"swap"`
	PgpgIn                     int64 `pparser:"pgpgin"`
	PgpgOut                    int64 `pparser:"pgpgout"`
	Pgfault                    int64 `pparser:"pgfault"`
	Pgmajfault                 int64 `pparser:"pgmajfault"`
	InactiveAnon               int64 `pparser:"inactive_anon"`
	ActiveAnon                 int64 `pparser:"active_anon"`
	InactiveFile               int64 `pparser:"inactive_file"`
	ActiveFile                 int64 `pparser:"active_file"`
	Unevictable                int64 `pparser:"unevictable"`
	HierarchicalMemoryLimit    int64 `pparser:"hierarchical_memory_limit"`
	HierarchicalMemswLimit     int64 `pparser:"hierarchical_memsw_limit"`
	TotalCache                 int64 `pparser:"total_cache"`
	TotalRSS                   int64 `pparser:"total_rss"`
	TotalRSSHuge               int64 `pparser:"total_rss_huge"`
	TotalShmem                 int64 `pparser:"total_shmem"`
	TotalMappedFile            int64 `pparser:"total_mapped_file"`
	TotalDirty                 int64 `pparser:"total_dirty"`
	TotalWriteback             int64 `pparser:"total_writeback"`
	TotalWorkingsetRefaultAnon int64 `pparser:"total_workingset_refault_anon"`
	TotalWorkingsetRefaultFile int64 `pparser:"total_workingset_refault_file"`
	TotalSwap                  int64 `pparser:"total_swap"`
	TotalPgpgIn                int64 `pparser:"total_pgpgin"`
	TotalPgpgOut               int64 `pparser:"total_pgpgout"`
	TotalPgFault               int64 `pparser:"total_pgfault"`
	TotalPgMajFault            int64 `pparser:"total_pgmajfault"`
	TotalInactiveAnon          int64 `pparser:"total_inactive_anon"`
	TotalActiveAnon            int64 `pparser:"total_active_anon"`
	TotalInactiveFile          int64 `pparser:"total_inactive_file"`
	TotalActiveFile            int64 `pparser:"total_active_file"`
	TotalUnevictable           int64 `pparser:"total_unevictable"`

	UnknownFields map[string]int64 `pparser:"skip,unknown"`
}

var cg1MemStatFieldIdx = pparser.NewLineKVFileParser(cg1MemoryStatContents{}, " ")

type cg2MemoryStatContents struct {
	Anon                   int64 `pparser:"anon"`
	File                   int64 `pparser:"file"`
	Kernel                 int64 `pparser:"kernel"`
	KernelStack            int64 `pparser:"kernel_stack"`
	Pagetables             int64 `pparser:"pagetables"`
	SecondaryPagetables    int64 `pparser:"sec_pagetables"`
	PerCPU                 int64 `pparser:"percpu"`
	Sock                   int64 `pparser:"sock"`
	VMAlloc                int64 `pparser:"vmalloc"`
	Shmem                  int64 `pparser:"shmem"`
	Zswap                  int64 `pparser:"zswap"`
	Zswapped               int64 `pparser:"zswapped"`
	FileMapped             int64 `pparser:"file_mapped"`
	FileDirty              int64 `pparser:"file_dirty"`
	FileWriteback          int64 `pparser:"file_writeback"`
	SwapCached             int64 `pparser:"swapcached"`
	AnonTHP                int64 `pparser:"anon_thp"`
	FileTHP                int64 `pparser:"file_thp"`
	ShmemTHP               int64 `pparser:"shmem_thp"`
	InactiveAnon           int64 `pparser:"inactive_anon"`
	ActiveAnon             int64 `pparser:"active_anon"`
	InactiveFile           int64 `pparser:"inactive_file"`
	ActiveFile             int64 `pparser:"active_file"`
	Unevictable            int64 `pparser:"unevictable"`
	SlabReclaimable        int64 `pparser:"slab_reclaimable"`
	SlabUnreclaimable      int64 `pparser:"slab_unreclaimable"`
	SlabTotal              int64 `pparser:"slab"`
	WorkingsetRefaultAnon  int64 `pparser:"workingset_refault_anon"`
	WorkingsetRefaultFile  int64 `pparser:"workingset_refault_file"`
	WorkingsetActivateAnon int64 `pparser:"workingset_activate_anon"`
	WorkingsetActivateFile int64 `pparser:"workingset_activate_file"`
	WorkingsetRestoreAnon  int64 `pparser:"workingset_restore_anon"`
	WorkingsetRestoreFile  int64 `pparser:"workingset_restore_file"`
	WorkingsetNodeReclaim  int64 `pparser:"workingset_nodereclaim"`
	PgScan                 int64 `pparser:"pgscan"`
	PgSteal                int64 `pparser:"pgsteal"`
	PgScanKswapd           int64 `pparser:"pgscan_kswapd"`
	PgscanDirect           int64 `pparser:"pgscan_direct"`
	PgstealKswapd          int64 `pparser:"pgsteal_kswapd"`
	PgstealDirect          int64 `pparser:"pgsteal_direct"`
	PgFault                int64 `pparser:"pgfault"`
	PgMajFault             int64 `pparser:"pgmajfault"`
	PgRefill               int64 `pparser:"pgrefill"`
	PgActivate             int64 `pparser:"pgactivate"`
	PgDeactivate           int64 `pparser:"pgdeactivate"`
	PgLazyFree             int64 `pparser:"pglazyfree"`
	PgLazyFreed            int64 `pparser:"pglazyfreed"`
	ZswpIn                 int64 `pparser:"zswpin"`
	ZswpOut                int64 `pparser:"zswpout"`
	ThpFaultAlloc          int64 `pparser:"thp_fault_alloc"`
	ThpCollapseAlloc       int64 `pparser:"thp_collapse_alloc"`

	UnknownFields map[string]int64 `pparser:"skip,unknown"`
}

var cg2MemStatFieldIdx = pparser.NewLineKVFileParser(cg2MemoryStatContents{}, " ")

type cg2MemEvents struct {
	Low          int64 `pparser:"low"`
	High         int64 `pparser:"high"`
	Max          int64 `pparser:"max"`
	OOMs         int64 `pparser:"oom"`
	OOMKills     int64 `pparser:"oom_kill"`
	OOMGroupKill int64 `pparser:"oom_group_kill"`

	UnknownFields map[string]int64 `pparser:"skip,unknown"`
}

var cg2MemEventsFieldIdx = pparser.NewLineKVFileParser(cg2MemEvents{}, " ")

// second return value is the memory limit for this CGroup (-1 is none)
func getCGroupMemoryStatsSingle(memPath *cgresolver.CGroupPath) (MemoryStats, int64, error) {
	switch memPath.Mode {
	case cgresolver.CGModeV1:
		f := os.DirFS(memPath.AbsPath)
		ooms, oomErr := getV1CgroupOOMs()
		if oomErr != nil {
			return MemoryStats{}, -1, fmt.Errorf("failed to look up OOMKills: %s",
				oomErr)
		}

		limitBytes, limitErr := readIntValFile(f, cgroupV1MemLimitFile)
		if limitErr != nil {
			return MemoryStats{}, -1, fmt.Errorf("failed to read limit: %w", limitErr)
		}

		usageBytes, usageErr := readIntValFile(f, cgroupV1MemUsageFile)
		if usageErr != nil {
			return MemoryStats{}, -1, fmt.Errorf("failed to read memory usage: %w", usageErr)
		}

		mstContents, readErr := os.ReadFile(filepath.Join(memPath.AbsPath, cgroupMemStatFile))
		if readErr != nil {
			return MemoryStats{}, -1, fmt.Errorf("failed to read memory.stat file for cgroup (%q): %w",
				filepath.Join(memPath.AbsPath, cgroupMemStatFile), readErr)
		}
		cg1Stats := cg1MemoryStatContents{}
		if parseErr := cg1MemStatFieldIdx.Parse(mstContents, &cg1Stats); parseErr != nil {
			return MemoryStats{}, -1, fmt.Errorf("failed to parse memory.stat file for cgroup (%q): %w",
				filepath.Join(memPath.AbsPath, cgroupCpuStatFile), parseErr)
		}

		ms := MemoryStats{
			Total:     limitBytes,
			Free:      limitBytes - usageBytes,
			Available: limitBytes - usageBytes + cg1Stats.TotalCache,
			OOMKills:  int64(ooms),
		}
		return ms, limitBytes, nil
	case cgresolver.CGModeV2:
		f := os.DirFS(memPath.AbsPath)
		mstContents, memStatErr := fs.ReadFile(f, cgroupMemStatFile)
		if memStatErr != nil {
			return MemoryStats{}, -1, fmt.Errorf("failed to read memory.stat: %w", memStatErr)
		}
		cg2Stats := cg2MemoryStatContents{}
		if parseErr := cg2MemStatFieldIdx.Parse(mstContents, &cg2Stats); parseErr != nil {
			return MemoryStats{}, -1, fmt.Errorf("failed to parse memory.stat file for cgroup (%q): %w",
				filepath.Join(memPath.AbsPath, cgroupMemStatFile), parseErr)
		}
		mevContents, memEventsErr := fs.ReadFile(f, cgroupV2MemEventsFile)
		if memEventsErr != nil {
			return MemoryStats{}, -1, fmt.Errorf("failed to read memory.events: %w", memEventsErr)
		}
		cg2Events := cg2MemEvents{}
		if parseErr := cg2MemEventsFieldIdx.Parse(mevContents, &cg2Events); parseErr != nil {
			return MemoryStats{}, -1, fmt.Errorf("failed to parse memory.events file for cgroup (%q): %w",
				filepath.Join(memPath.AbsPath, cgroupV2MemEventsFile), parseErr)
		}

		usageBytes, usageErr := readIntValFile(f, cgroupV2MemCurrentFile)
		if usageErr != nil {
			return MemoryStats{}, -1, fmt.Errorf("failed to parse memory.current file for cgroup : %w", usageErr)
		}
		limitBytes, limitReadErr := readIntValFile(f, cgroupV2MemLimitFile)
		if limitReadErr != nil {
			if !errors.Is(limitReadErr, fs.ErrNotExist) {
				return MemoryStats{}, -1, fmt.Errorf("failed to read cgroup memory limit file  %s",
					limitReadErr)
			}
			limitBytes = -1
		}

		return MemoryStats{
			Total: limitBytes,
			Free:  limitBytes - usageBytes,
			// TODO: verify that nothing here is getting double-counted
			// subtract total usage from the limit, and add back some memory-categories that can be evicted.
			// Notably, cached swap can be evicted immediately, as can any File memory that's not dirty or getting written back.
			// SlabReclaimable is kernel memory that can be freed under memory pressure.
			Available: limitBytes - usageBytes + cg2Stats.SwapCached + (cg2Stats.File - cg2Stats.FileDirty - cg2Stats.FileWriteback) + cg2Stats.SlabReclaimable,
			OOMKills:  cg2Events.OOMGroupKill,
		}, limitBytes, nil
	default:
		return MemoryStats{}, -1, fmt.Errorf("unknown cgroup type: %d", memPath.Mode)
	}
}

// GetCgroupMemoryStats queries the current process's memory cgroup's memory
// usage/limits.
func GetCgroupMemoryStats() (MemoryStats, error) {
	memPath, cgroupFindErr := cgresolver.SelfSubsystemPath("memory")
	if cgroupFindErr != nil {
		return MemoryStats{}, fmt.Errorf("unable to find cgroup directory: %s", cgroupFindErr)
	}

	minLimit := uint64(math.MaxUint64)
	minLimCGMemStats := MemoryStats{}
	leafCGReadErr := error(nil)

	allFailed := true

	for newDir := true; newDir; memPath, newDir = memPath.Parent() {
		cgMemStats, cgLim, cgReadErr := getCGroupMemoryStatsSingle(&memPath)
		if cgReadErr != nil {
			if leafCGReadErr == nil && allFailed {
				leafCGReadErr = cgReadErr
			}
			continue
		}

		allFailed = false
		if cgLim != -1 && uint64(cgLim) < minLimit {
			minLimit = uint64(cgLim)
			minLimCGMemStats = cgMemStats
		}
	}
	if allFailed {
		return MemoryStats{}, leafCGReadErr
	}
	return minLimCGMemStats, nil
}

type memCgroupOOMControl struct {
	OomKillDisable int64            `pparser:"oom_kill_disable"`
	UnderOom       int64            `pparser:"under_oom"`
	OomKill        int64            `pparser:"oom_kill"`
	UnknownFields  map[string]int64 `pparser:"skip,unknown"`
}

var memCgroupOOMControlFieldIdx = pparser.NewLineKVFileParser(memCgroupOOMControl{}, " ")

// getV1CgroupOOMs looks up the current number of oom kills for the current cgroup.
func getV1CgroupOOMs() (int32, error) {
	memPath, cgroupFindErr := cgresolver.SelfSubsystemPath("memory")
	if cgroupFindErr != nil {
		return -1, fmt.Errorf("unable to find cgroup directory: %s", cgroupFindErr)
	}
	oomControlPath := filepath.Join(memPath.AbsPath, cgroupV1MemOOMControlFile)
	oomControlBytes, oomControlReadErr := os.ReadFile(oomControlPath)
	if oomControlReadErr != nil {
		return 0, fmt.Errorf(
			"failed to read contents of %q: %s",
			oomControlPath, oomControlReadErr)
	}
	oomc := memCgroupOOMControl{}
	parseErr := memCgroupOOMControlFieldIdx.Parse(oomControlBytes, &oomc)
	if parseErr != nil {
		return 0, parseErr
	}

	// The oom_kill line was only added to the oom_control file in linux
	// 4.13, so some systems (docker for Mac) don't have it.
	return int32(oomc.OomKill), nil
}

type cg2CPUStatContents struct {
	Usageμs          int64            `pparser:"usage_usec"`
	Userμs           int64            `pparser:"user_usec"`
	Sysμs            int64            `pparser:"system_usec"`
	TotalPeriods     int64            `pparser:"nr_periods"`
	ThrottledPeriods int64            `pparser:"nr_throttled"`
	Throttledμs      int64            `pparser:"throttled_usec"`
	BurstCount       int64            `pparser:"nr_bursts"`
	Burstμs          int64            `pparser:"burst_usec"`
	UnknownFields    map[string]int64 `pparser:"skip,unknown"`
}

var cg2CPUStatContentsFieldIdx = pparser.NewLineKVFileParser(cg2CPUStatContents{}, " ")

type cg1CPUStatContents struct {
	TotalPeriods     int64            `pparser:"nr_periods"`
	ThrottledPeriods int64            `pparser:"nr_throttled"`
	Throttledns      int64            `pparser:"throttled_time"`
	BurstCount       int64            `pparser:"nr_bursts"`
	Burstns          int64            `pparser:"burst_time"`
	Waitns           int64            `pparser:"wait_sum"`
	UnknownFields    map[string]int64 `pparser:"skip,unknown"`
}

var cg1CPUStatContentsFieldIdx = pparser.NewLineKVFileParser(cg1CPUStatContents{}, " ")

type cg1CPUAcctStatContents struct {
	UserTicks     int64            `pparser:"user"`
	SysTicks      int64            `pparser:"system"`
	UnknownFields map[string]int64 `pparser:"skip,unknown"`
}

var cg1CPUAcctStatContentsFieldIdx = pparser.NewLineKVFileParser(cg1CPUAcctStatContents{}, " ")

func readIntValFile(f fs.FS, path string) (int64, error) {
	conts, readErr := fs.ReadFile(f, path)
	if readErr != nil {
		return -1, fmt.Errorf("failed to read %q: %w", path, readErr)
	}
	trimmedConts := bytes.TrimSpace(conts)
	if bytes.Equal(trimmedConts, []byte("max")) {
		return math.MaxInt64, nil
	}
	v, parseErr := strconv.ParseInt(string(trimmedConts), 10, 64)
	if parseErr != nil {
		return -1, fmt.Errorf("failed to parse %q (%q) as integer: %w", path, trimmedConts, parseErr)
	}
	return v, nil
}

func cgroupV1ReadCPUAcctStats(f fs.FS) (procstats.CPUTime, error) {
	cStatsBytes, readErr := fs.ReadFile(f, cgroupV1CpuAcctStatFile)
	if readErr != nil {
		return procstats.CPUTime{}, fmt.Errorf("failed to read cpuacct.stat file: %w", readErr)
	}
	cStats := cg1CPUAcctStatContents{}
	if parseErr := cg1CPUAcctStatContentsFieldIdx.Parse(cStatsBytes, &cStats); parseErr != nil {
		return procstats.CPUTime{}, fmt.Errorf("failed to parse cpuacct.stat: %w", parseErr)
	}
	return procstats.CPUTime{
		Utime: time.Duration(cStats.UserTicks) * 10 * time.Millisecond,
		Stime: time.Duration(cStats.SysTicks) * 10 * time.Millisecond,
	}, nil

}

// CGroupV1CPUUsage reads the CPU usage for a specific V1 cpuacct CGroup (and descendants)
// The fs.FS arg will usually be from os.DirFS, but may be any other fs.FS implementation.
func CGroupV1CPUUsage(f fs.FS) (procstats.CPUTime, error) {
	userCPUNS, userReadErr := readIntValFile(f, cgroupV1CpuUserUsageFile)
	if userReadErr != nil {
		if errors.Is(userReadErr, fs.ErrNotExist) {
			// fall back to reading just the cpuacct.stat file
			return cgroupV1ReadCPUAcctStats(f)
		}

		return procstats.CPUTime{}, fmt.Errorf("failed to read userspace CPU-time: %w", userReadErr)
	}
	sysCPUNS, sysReadErr := readIntValFile(f, cgroupV1CpuSysUsageFile)
	if sysReadErr != nil {
		return procstats.CPUTime{}, fmt.Errorf("failed to read kernelspace CPU-time: %w", sysReadErr)
	}

	return procstats.CPUTime{
		Utime: time.Duration(userCPUNS) * time.Nanosecond,
		Stime: time.Duration(sysCPUNS) * time.Nanosecond,
	}, nil
}

// CGroupV2CPUUsage reads the CPU usage for a specific V2 cpu CGroup (and descendants)
// The fs.FS arg will usually be from os.DirFS, but may be any other fs.FS implementation.
func CGroupV2CPUUsage(f fs.FS) (CPUStats, error) {
	cstContents, readErr := fs.ReadFile(f, cgroupCpuStatFile)
	if readErr != nil {
		return CPUStats{}, fmt.Errorf("failed to read cpu.stat file for cgroup: %w",
			readErr)
	}
	cg2Stats := cg2CPUStatContents{}
	if parseErr := cg2CPUStatContentsFieldIdx.Parse(cstContents, &cg2Stats); parseErr != nil {
		return CPUStats{}, fmt.Errorf("failed to parse cpu.stat file for cgroup: %w",
			readErr)
	}
	return CPUStats{
		Usage: procstats.CPUTime{
			Utime: time.Duration(cg2Stats.Userμs) * time.Microsecond,
			Stime: time.Duration(cg2Stats.Sysμs) * time.Microsecond,
		},
		ThrottledTime: time.Duration(cg2Stats.Throttledμs) * time.Microsecond,
	}, nil
}

func getCGroupCPUStatsSingle(cpuPath *cgresolver.CGroupPath) (CPUStats, float64, error) {
	lim, limErr := getCGroupCPULimitSingle(cpuPath)
	if limErr != nil {
		if !errors.Is(limErr, fs.ErrNotExist) {
			return CPUStats{}, -1, fmt.Errorf("failed to read CPU limit: %w", limErr)
		}
		lim = -1.0
	}
	switch cpuPath.Mode {
	case cgresolver.CGModeV1:
		cstContents, readErr := os.ReadFile(filepath.Join(cpuPath.AbsPath, cgroupCpuStatFile))
		if readErr != nil {
			return CPUStats{}, -1, fmt.Errorf("failed to read cpu.stat file for cgroup (%q): %w",
				filepath.Join(cpuPath.AbsPath, cgroupCpuStatFile), readErr)
		}
		cg1Stats := cg1CPUStatContents{}
		if parseErr := cg1CPUStatContentsFieldIdx.Parse(cstContents, &cg1Stats); parseErr != nil {
			return CPUStats{}, -1, fmt.Errorf("failed to parse cpu.stat file for cgroup (%q): %w",
				filepath.Join(cpuPath.AbsPath, cgroupCpuStatFile), readErr)
		}
		cpuAcctPath, cgroupFindErr := cgresolver.SelfSubsystemPath("cpuacct")
		if cgroupFindErr != nil {
			return CPUStats{}, -1, fmt.Errorf("unable to find cgroup directory: %s",
				cgroupFindErr)
		}
		f := os.DirFS(cpuAcctPath.AbsPath)
		usage, usageErr := CGroupV1CPUUsage(f)
		if usageErr != nil {
			return CPUStats{}, -1, fmt.Errorf("failed to query usage: %w", usageErr)
		}
		return CPUStats{
			Usage:         usage,
			ThrottledTime: time.Duration(cg1Stats.Throttledns) * time.Nanosecond,
		}, lim, nil

	case cgresolver.CGModeV2:
		f := os.DirFS(cpuPath.AbsPath)
		cpuStat, usageErr := CGroupV2CPUUsage(f)
		return cpuStat, lim, usageErr
	default:
		return CPUStats{}, -1, fmt.Errorf("unknown cgroup type: %d", cpuPath.Mode)
	}
}

// GetCgroupCPUStats queries the current process's memory cgroup's CPU
// usage/limits.
func GetCgroupCPUStats() (CPUStats, error) {
	cpuPath, cgroupFindErr := cgresolver.SelfSubsystemPath("cpu")
	if cgroupFindErr != nil {
		return CPUStats{}, fmt.Errorf("unable to find cgroup directory: %s",
			cgroupFindErr)
	}
	minLimit := math.Inf(+1)
	minCPUStats := CPUStats{}
	allFailed := true
	leafCGReadErr := error(nil)

	cpuStatsPopulated := false
	leafCPUStats := CPUStats{}

	for newDir := true; newDir; cpuPath, newDir = cpuPath.Parent() {
		cgCPUStats, cgLim, cgReadErr := getCGroupCPUStatsSingle(&cpuPath)
		if cgReadErr != nil {
			if leafCGReadErr == nil && allFailed {
				leafCGReadErr = cgReadErr
			}
			continue
		}
		if !cpuStatsPopulated {
			leafCPUStats = cgCPUStats
			cpuStatsPopulated = true
		}

		allFailed = false
		if (cgLim != -1 && cgLim != 0.0) && cgLim < minLimit {
			minLimit = cgLim
			minCPUStats = cgCPUStats
		}
	}
	if allFailed {
		return CPUStats{}, leafCGReadErr
	}
	if math.IsInf(minLimit, +1) {
		// if the limit is still infinite, return the first successfully read stats (the farthest out the leaf)
		return leafCPUStats, nil
	}
	return minCPUStats, nil
}
