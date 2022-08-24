//go:build linux
// +build linux

package cgrouplimits

import "testing"

func TestParseMemInfo(t *testing.T) {
	mi, err := parseMemInfo([]byte(testProcMemInfoVal))
	if err != nil {
		t.Fatalf("failed to parse test value for meminfo: %s", err)
	}
	t.Logf("meminfo val: %+v", mi)

	if mi.DirectMap1G != 10240 {
		t.Errorf("unexpected value of DirectMap1G %d (expected 10kB)",
			mi.DirectMap1G)
	}
	if mi.MemFree != 7989592*1024 {
		t.Errorf("unexpected value of MemFree %d (expected 7989592 kB)",
			mi.MemFree)
	}
	if mi.MemAvailable != 10664856*1024 {
		t.Errorf("unexpected value of MemFree %d (expected 10664856 kB)",
			mi.MemAvailable)
	}
	if mi.ActiveAnon != 6349500*1024 {
		t.Errorf("unexpected value of Active(anon) %d (expected 6349500 kB)",
			mi.ActiveAnon)
	}

	if mi.UnknownFields["AlsoNotReal"] != 342 {
		t.Errorf("unexpected value for unknown field AlsoNotReal %d; expected 342",
			mi.UnknownFields["AlsoNotReal"])
	}
	if mi.UnknownFields["notarealfield"] != 42*1024 {
		t.Errorf("unexpected value for unknown field notarealfield %d; expected 42 kB",
			mi.UnknownFields["notarealfield"])
	}
}

const testProcMemInfoVal = `MemTotal:       20285380 kB
MemFree:         7989592 kB
MemAvailable:   10664856 kB
Buffers:          399020 kB
Cached:          4334480 kB
SwapCached:       772344 kB
Active:          7183992 kB
Inactive:        4296268 kB
Active(anon):    6349500 kB
Inactive(anon):  2345848 kB
Active(file):     834492 kB
Inactive(file):  1950420 kB
Unevictable:          48 kB
Mlocked:              48 kB
SwapTotal:      20713468 kB
SwapFree:       14719144 kB
Dirty:              1408 kB
Writeback:             0 kB
AnonPages:       6611376 kB
Mapped:           908248 kB
Shmem:           1948588 kB
KReclaimable:     260292 kB
Slab:             403956 kB
SReclaimable:     260292 kB
SUnreclaim:       143664 kB
KernelStack:       31408 kB
PageTables:       162168 kB
NFS_Unstable:          0 kB
Bounce:                0 kB
WritebackTmp:          0 kB
CommitLimit:    30856156 kB
Committed_AS:   46860432 kB
VmallocTotal:   34359738367 kB
VmallocUsed:           0 kB
VmallocChunk:          0 kB
Percpu:             1936 kB
HardwareCorrupted:     0 kB
AnonHugePages:     20480 kB
ShmemHugePages:        0 kB
ShmemPmdMapped:        0 kB
CmaTotal:              0 kB
CmaFree:               0 kB
HugePages_Total:       0
HugePages_Free:        0
HugePages_Rsvd:        0
HugePages_Surp:        0
Hugepagesize:       2048 kB
Hugetlb:               0 kB
notarealfield:        42 kB
DirectMap4k:      825432 kB
AlsoNotReal:         342
DirectMap2M:    19886080 kB
DirectMap1G:          10 kB`

func TestParseVMStat(t *testing.T) {
	vms, err := parseVMStat([]byte(testProcVMStatVal))
	if err != nil {
		t.Fatalf("failed to parse test value for vmstat: %s", err)
	}
	t.Logf("vmstat val: %+v", vms)

	if vms.OomKill != 18 {
		t.Errorf("unexpected value of oom_kill %d (expected 18)",
			vms.OomKill)
	}

	if vms.UnknownFields["AlsoNotReal"] != 342 {
		t.Errorf("unexpected value for unknown field AlsoNotReal %d; expected 342",
			vms.UnknownFields["AlsoNotReal"])
	}
}

const testProcVMStatVal = `nr_free_pages 726301
nr_zone_inactive_anon 486592
nr_zone_active_anon 2957195
nr_zone_inactive_file 169324
nr_zone_active_file 553903
nr_zone_unevictable 12
nr_zone_write_pending 39
nr_mlock 12
nr_page_table_pages 31588
nr_kernel_stack 26592
nr_bounce 0
nr_zspages 0
nr_free_cma 0
numa_hit 837259281
numa_miss 0
numa_foreign 0
numa_interleave 23558
numa_local 837259281
numa_other 0
nr_inactive_anon 486592
nr_active_anon 2957195
nr_inactive_file 169324
nr_active_file 553903
nr_unevictable 12
nr_slab_reclaimable 56435
nr_slab_unreclaimable 36538
nr_isolated_anon 0
nr_isolated_file 0
workingset_nodes 14367
workingset_refault 2065336
workingset_activate 639207
workingset_restore 507290
workingset_nodereclaim 0
nr_anon_pages 3045316
nr_mapped 163427
nr_file_pages 1222408
nr_dirty 39
nr_writeback 0
nr_writeback_temp 0
nr_shmem 383494
nr_shmem_hugepages 0
nr_shmem_pmdmapped 0
nr_anon_transparent_hugepages 175
nr_unstable 0
nr_vmscan_write 3561701
nr_vmscan_immediate_reclaim 4757
nr_dirtied 8056296
nr_written 11471429
nr_kernel_misc_reclaimable 0
nr_dirty_threshold 560248
nr_dirty_background_threshold 139891
pgpgin 28131850
pgpgout 53338548
pswpin 1073770
pswpout 3561701
pgalloc_dma 0
pgalloc_dma32 69330264
pgalloc_normal 806562589
pgalloc_movable 0
allocstall_dma 0
allocstall_dma32 0
allocstall_normal 4696
allocstall_movable 6408
pgskip_dma 0
pgskip_dma32 0
pgskip_normal 0
pgskip_movable 0
pgfree 881147907
pgactivate 31894332
pgdeactivate 6133619
pglazyfree 108398
pgfault 934561471
pgmajfault 415370
pglazyfreed 66936
pgrefill 7312443
pgsteal_kswapd 6252203
pgsteal_direct 2805973
pgscan_kswapd 10732235
pgscan_direct 16322252
pgscan_direct_throttle 0
zone_reclaim_failed 0
pginodesteal 429520
slabs_scanned 117871786
kswapd_inodesteal 232918
kswapd_low_wmark_hit_quickly 247
kswapd_high_wmark_hit_quickly 134
pageoutrun 889
pgrotated 3042292
drop_pagecache 0
drop_slab 0
oom_kill 18
numa_pte_updates 0
numa_huge_pte_updates 0
numa_hint_faults 0
numa_hint_faults_local 0
numa_pages_migrated 0
pgmigrate_success 4432491
pgmigrate_fail 22512043
compact_migrate_scanned 31801849
compact_free_scanned 910007689
compact_isolated 31502771
compact_stall 4905
compact_fail 1989
compact_success 2916
compact_daemon_wake 24
compact_daemon_migrate_scanned 40063
compact_daemon_free_scanned 4663659
htlb_buddy_alloc_success 0
htlb_buddy_alloc_fail 0
unevictable_pgs_culled 12756
unevictable_pgs_scanned 0
unevictable_pgs_rescued 6657
unevictable_pgs_mlocked 11483
unevictable_pgs_munlocked 11455
unevictable_pgs_cleared 4
unevictable_pgs_stranded 5
AlsoNotReal 342
thp_fault_alloc 40662
thp_fault_fallback 4251
thp_collapse_alloc 1490
thp_collapse_alloc_failed 38
thp_file_alloc 0
thp_file_mapped 0
thp_split_page 623
thp_split_page_failed 8
thp_deferred_split_page 41669
thp_split_pmd 902
thp_split_pud 0
thp_zero_page_alloc 6
thp_zero_page_alloc_failed 0
thp_swpout 0
thp_swpout_fallback 0
swap_ra 700536
swap_ra_hit 659514`
