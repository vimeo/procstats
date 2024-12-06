//go:build linux
// +build linux

package cgrouplimits

import (
	"fmt"
	"os"

	"github.com/vimeo/procstats/pparser"
)

func getMemInfo() (hostMemInfo, error) {
	const procMemInfo = "/proc/meminfo"
	memInfoBytes, procReadErr := os.ReadFile(procMemInfo)
	if procReadErr != nil {
		return hostMemInfo{}, fmt.Errorf(
			"failed to read contents of %q: %s",
			procMemInfo, procReadErr)
	}

	mi, parseErr := parseMemInfo(memInfoBytes)
	if parseErr != nil {
		return hostMemInfo{}, fmt.Errorf(
			"failed to parse %q contents: %s",
			procMemInfo, parseErr)
	}
	return mi, nil
}

func getVMStat() (hostVMStat, error) {

	const procVMStat = "/proc/vmstat"
	vmStatBytes, procReadErr := os.ReadFile(procVMStat)
	if procReadErr != nil {
		return hostVMStat{}, fmt.Errorf(
			"failed to read contents of %q: %s",
			procVMStat, procReadErr)
	}

	vms, parseErr := parseVMStat(vmStatBytes)
	if parseErr != nil {
		return hostVMStat{}, fmt.Errorf(
			"failed to parse %q contents: %s",
			procVMStat, parseErr)
	}
	return vms, nil
}

// HostMemStats gets the current memory usage from /proc/meminfo and
// synthesizes it into a MemoryStats object.
func HostMemStats() (MemoryStats, error) {
	mi, err := getMemInfo()
	if err != nil {
		return MemoryStats{}, err
	}
	vms, vmsErr := getVMStat()
	if vmsErr != nil {
		return MemoryStats{}, vmsErr
	}
	return MemoryStats{
		Total:     mi.MemTotal + mi.SwapTotal,
		Free:      mi.MemFree + mi.SwapFree,
		Available: mi.MemAvailable,
		OOMKills:  vms.OomKill,
	}, nil
}

func parseMemInfo(contentBytes []byte) (hostMemInfo, error) {

	mi := hostMemInfo{UnknownFields: make(map[string]int64)}

	parseErr := hostMemInfoFieldIdx.Parse(contentBytes, &mi)
	if parseErr != nil {
		return mi, parseErr
	}
	return mi, nil

}

type hostMemInfo struct {
	MemTotal          int64
	MemFree           int64
	MemAvailable      int64
	Buffers           int64
	Cached            int64
	SwapCached        int64
	Active            int64
	Inactive          int64
	ActiveAnon        int64 `pparser:"Active(anon)"`
	InactiveAnon      int64 `pparser:"Inactive(anon)"`
	ActiveFile        int64 `pparser:"Active(file)"`
	InactiveFile      int64 `pparser:"Inactive(file)"`
	Unevictable       int64
	Mlocked           int64
	SwapTotal         int64
	SwapFree          int64
	Dirty             int64
	Writeback         int64
	AnonPages         int64
	Mapped            int64
	Shmem             int64
	KReclaimable      int64
	Slab              int64
	SReclaimable      int64
	SUnreclaim        int64
	KernelStack       int64
	PageTables        int64
	NFSUnstable       int64 `pparser:"NFS_Unstable"`
	Bounce            int64
	WritebackTmp      int64
	CommitLimit       int64
	CommittedAS       int64 `pparser:"Committed_AS"`
	VmallocTotal      int64
	VmallocUsed       int64
	VmallocChunk      int64
	Percpu            int64
	HardwareCorrupted int64
	AnonHugePages     int64
	ShmemHugePages    int64
	ShmemPmdMapped    int64
	CmaTotal          int64
	CmaFree           int64
	HugePagesTotal    int64 `pparser:"HugePages_Total"`
	HugePagesFree     int64 `pparser:"HugePages_Free"`
	HugePagesRsvd     int64 `pparser:"HugePages_Rsvd"`
	HugePagesSurp     int64 `pparser:"HugePages_Surp"`
	Hugepagesize      int64
	Hugetlb           int64
	DirectMap4k       int64
	DirectMap2M       int64
	DirectMap1G       int64
	UnknownFields     map[string]int64 `pparser:"skip,unknown"`
}

// hostMemInfoFieldIdx is an index of the name in /proc/meminfo to the field
// index in the hostMemInfo struct.
var (
	hostMemInfoFieldIdx = pparser.NewLineKVFileParser(hostMemInfo{}, ":")
	hostVMStatFieldIdx  = pparser.NewLineKVFileParser(hostVMStat{}, " ")
)

// fields from /proc/vmstat pulled from "mm/vmstat.c"
// generated with c&p of vmstat_text[] followed by some regexp mangling
type hostVMStat struct {
	NrFreePages                int64 `pparser:"nr_free_pages"`
	NrZoneInactiveAnon         int64 `pparser:"nr_zone_inactive_anon"`
	NrZoneActiveAnon           int64 `pparser:"nr_zone_active_anon"`
	NrZoneInactiveFile         int64 `pparser:"nr_zone_inactive_file"`
	NrZoneActiveFile           int64 `pparser:"nr_zone_active_file"`
	NrZoneUnevictable          int64 `pparser:"nr_zone_unevictable"`
	NrZoneWritePending         int64 `pparser:"nr_zone_write_pending"`
	NrMlock                    int64 `pparser:"nr_mlock"`
	NrPageTablePages           int64 `pparser:"nr_page_table_pages"`
	NrKernelStack              int64 `pparser:"nr_kernel_stack"`
	NrBounce                   int64 `pparser:"nr_bounce"`
	NrZspages                  int64 `pparser:"nr_zspages"`
	NrFreeCma                  int64 `pparser:"nr_free_cma"`
	NumaHit                    int64 `pparser:"numa_hit"`
	NumaMiss                   int64 `pparser:"numa_miss"`
	NumaForeign                int64 `pparser:"numa_foreign"`
	NumaInterleave             int64 `pparser:"numa_interleave"`
	NumaLocal                  int64 `pparser:"numa_local"`
	NumaOther                  int64 `pparser:"numa_other"`
	NrInactiveAnon             int64 `pparser:"nr_inactive_anon"`
	NrActiveAnon               int64 `pparser:"nr_active_anon"`
	NrInactiveFile             int64 `pparser:"nr_inactive_file"`
	NrActiveFile               int64 `pparser:"nr_active_file"`
	NrUnevictable              int64 `pparser:"nr_unevictable"`
	NrSlabReclaimable          int64 `pparser:"nr_slab_reclaimable"`
	NrSlabUnreclaimable        int64 `pparser:"nr_slab_unreclaimable"`
	NrIsolatedAnon             int64 `pparser:"nr_isolated_anon"`
	NrIsolatedFile             int64 `pparser:"nr_isolated_file"`
	WorkingsetNodes            int64 `pparser:"workingset_nodes"`
	WorkingsetRefault          int64 `pparser:"workingset_refault"`
	WorkingsetActivate         int64 `pparser:"workingset_activate"`
	WorkingsetRestore          int64 `pparser:"workingset_restore"`
	WorkingsetNodereclaim      int64 `pparser:"workingset_nodereclaim"`
	NrAnonPages                int64 `pparser:"nr_anon_pages"`
	NrMapped                   int64 `pparser:"nr_mapped"`
	NrFilePages                int64 `pparser:"nr_file_pages"`
	NrDirty                    int64 `pparser:"nr_dirty"`
	NrWriteback                int64 `pparser:"nr_writeback"`
	NrWritebackTemp            int64 `pparser:"nr_writeback_temp"`
	NrShmem                    int64 `pparser:"nr_shmem"`
	NrShmemHugepages           int64 `pparser:"nr_shmem_hugepages"`
	NrShmemPmdmapped           int64 `pparser:"nr_shmem_pmdmapped"`
	NrAnonTransparentHugepages int64 `pparser:"nr_anon_transparent_hugepages"`
	NrUnstable                 int64 `pparser:"nr_unstable"`
	NrVmscanWrite              int64 `pparser:"nr_vmscan_write"`
	NrVmscanImmediateReclaim   int64 `pparser:"nr_vmscan_immediate_reclaim"`
	NrDirtied                  int64 `pparser:"nr_dirtied"`
	NrWritten                  int64 `pparser:"nr_written"`
	NrKernelMiscReclaimable    int64 `pparser:"nr_kernel_misc_reclaimable"`

	NrDirtyThreshold           int64 `pparser:"nr_dirty_threshold"`
	NrDirtyBackgroundThreshold int64 `pparser:"nr_dirty_background_threshold"`

	Pgpgin  int64 `pparser:"pgpgin"`
	Pgpgout int64 `pparser:"pgpgout"`
	Pswpin  int64 `pparser:"pswpin"`
	Pswpout int64 `pparser:"pswpout"`

	PgallocDma     int64 `pparser:"pgalloc_dma"`
	PgallocDma32   int64 `pparser:"pgalloc_dma32"`
	PgallocNormal  int64 `pparser:"pgalloc_normal"`
	PgallocMovable int64 `pparser:"pgalloc_movable"`

	AllocstallDma     int64 `pparser:"allocstall_dma"`
	AllocstallDma32   int64 `pparser:"allocstall_dma32"`
	AllocstallNormal  int64 `pparser:"allocstall_normal"`
	AllocstallMovable int64 `pparser:"allocstall_movable"`

	PgskipDma     int64 `pparser:"pgskip_dma"`
	PgskipDma32   int64 `pparser:"pgskip_dma32"`
	PgskipNormal  int64 `pparser:"pgskip_normal"`
	PgskipMovable int64 `pparser:"pgskip_movable"`

	Pgfree                      int64            `pparser:"pgfree"`
	Pgactivate                  int64            `pparser:"pgactivate"`
	Pgdeactivate                int64            `pparser:"pgdeactivate"`
	Pglazyfree                  int64            `pparser:"pglazyfree"`
	Pgfault                     int64            `pparser:"pgfault"`
	Pgmajfault                  int64            `pparser:"pgmajfault"`
	Pglazyfreed                 int64            `pparser:"pglazyfreed"`
	Pgrefill                    int64            `pparser:"pgrefill"`
	PgstealKswapd               int64            `pparser:"pgsteal_kswapd"`
	PgstealDirect               int64            `pparser:"pgsteal_direct"`
	PgscanKswapd                int64            `pparser:"pgscan_kswapd"`
	PgscanDirect                int64            `pparser:"pgscan_direct"`
	PgscanDirectThrottle        int64            `pparser:"pgscan_direct_throttle"`
	ZoneReclaimFailed           int64            `pparser:"zone_reclaim_failed"`
	Pginodesteal                int64            `pparser:"pginodesteal"`
	SlabsScanned                int64            `pparser:"slabs_scanned"`
	KswapdInodesteal            int64            `pparser:"kswapd_inodesteal"`
	KswapdLowWmarkHitQuickly    int64            `pparser:"kswapd_low_wmark_hit_quickly"`
	KswapdHighWmarkHitQuickly   int64            `pparser:"kswapd_high_wmark_hit_quickly"`
	Pageoutrun                  int64            `pparser:"pageoutrun"`
	Pgrotated                   int64            `pparser:"pgrotated"`
	DropPagecache               int64            `pparser:"drop_pagecache"`
	DropSlab                    int64            `pparser:"drop_slab"`
	OomKill                     int64            `pparser:"oom_kill"`
	NumaPteUpdates              int64            `pparser:"numa_pte_updates"`
	NumaHugePteUpdates          int64            `pparser:"numa_huge_pte_updates"`
	NumaHintFaults              int64            `pparser:"numa_hint_faults"`
	NumaHintFaultsLocal         int64            `pparser:"numa_hint_faults_local"`
	NumaPagesMigrated           int64            `pparser:"numa_pages_migrated"`
	PgmigrateSuccess            int64            `pparser:"pgmigrate_success"`
	PgmigrateFail               int64            `pparser:"pgmigrate_fail"`
	CompactMigrateScanned       int64            `pparser:"compact_migrate_scanned"`
	CompactFreeScanned          int64            `pparser:"compact_free_scanned"`
	CompactIsolated             int64            `pparser:"compact_isolated"`
	CompactStall                int64            `pparser:"compact_stall"`
	CompactFail                 int64            `pparser:"compact_fail"`
	CompactSuccess              int64            `pparser:"compact_success"`
	CompactDaemonWake           int64            `pparser:"compact_daemon_wake"`
	CompactDaemonMigrateScanned int64            `pparser:"compact_daemon_migrate_scanned"`
	CompactDaemonFreeScanned    int64            `pparser:"compact_daemon_free_scanned"`
	HtlbBuddyAllocSuccess       int64            `pparser:"htlb_buddy_alloc_success"`
	HtlbBuddyAllocFail          int64            `pparser:"htlb_buddy_alloc_fail"`
	UnevictablePgsCulled        int64            `pparser:"unevictable_pgs_culled"`
	UnevictablePgsScanned       int64            `pparser:"unevictable_pgs_scanned"`
	UnevictablePgsRescued       int64            `pparser:"unevictable_pgs_rescued"`
	UnevictablePgsMlocked       int64            `pparser:"unevictable_pgs_mlocked"`
	UnevictablePgsMunlocked     int64            `pparser:"unevictable_pgs_munlocked"`
	UnevictablePgsCleared       int64            `pparser:"unevictable_pgs_cleared"`
	UnevictablePgsStranded      int64            `pparser:"unevictable_pgs_stranded"`
	ThpFaultAlloc               int64            `pparser:"thp_fault_alloc"`
	ThpFaultFallback            int64            `pparser:"thp_fault_fallback"`
	ThpCollapseAlloc            int64            `pparser:"thp_collapse_alloc"`
	ThpCollapseAllocFailed      int64            `pparser:"thp_collapse_alloc_failed"`
	ThpFileAlloc                int64            `pparser:"thp_file_alloc"`
	ThpFileMapped               int64            `pparser:"thp_file_mapped"`
	ThpSplitPage                int64            `pparser:"thp_split_page"`
	ThpSplitPageFailed          int64            `pparser:"thp_split_page_failed"`
	ThpDeferredSplitPage        int64            `pparser:"thp_deferred_split_page"`
	ThpSplitPmd                 int64            `pparser:"thp_split_pmd"`
	ThpSplitPud                 int64            `pparser:"thp_split_pud"`
	ThpZeroPageAlloc            int64            `pparser:"thp_zero_page_alloc"`
	ThpZeroPageAllocFailed      int64            `pparser:"thp_zero_page_alloc_failed"`
	ThpSwpout                   int64            `pparser:"thp_swpout"`
	ThpSwpoutFallback           int64            `pparser:"thp_swpout_fallback"`
	BalloonInflate              int64            `pparser:"balloon_inflate"`
	BalloonDeflate              int64            `pparser:"balloon_deflate"`
	BalloonMigrate              int64            `pparser:"balloon_migrate"`
	NrTlbRemoteFlush            int64            `pparser:"nr_tlb_remote_flush"`
	NrTlbRemoteFlushReceived    int64            `pparser:"nr_tlb_remote_flush_received"`
	NrTlbLocalFlushAll          int64            `pparser:"nr_tlb_local_flush_all"`
	NrTlbLocalFlushOne          int64            `pparser:"nr_tlb_local_flush_one"`
	VmacacheFindCalls           int64            `pparser:"vmacache_find_calls"`
	VmacacheFindHits            int64            `pparser:"vmacache_find_hits"`
	SwapRa                      int64            `pparser:"swap_ra"`
	SwapRaHit                   int64            `pparser:"swap_ra_hit"`
	UnknownFields               map[string]int64 `pparser:"skip,unknown"`
}

func parseVMStat(contentBytes []byte) (hostVMStat, error) {

	vms := hostVMStat{UnknownFields: make(map[string]int64)}

	parseErr := hostVMStatFieldIdx.Parse(contentBytes, &vms)
	if parseErr != nil {
		return vms, parseErr
	}
	return vms, nil

}
