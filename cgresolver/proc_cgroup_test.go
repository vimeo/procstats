package cgresolver

import (
	"errors"
	"slices"
	"testing"
)

func TestCGPath(t *testing.T) {
	for _, itbl := range []struct {
		name    string
		hier    CGProcHierarchy
		mounts  []Mount
		expPath CGroupPath
		expErr  error
	}{
		{
			name: "cg1_root_mount",
			hier: CGProcHierarchy{
				HierarchyID:   10,
				SubsystemsCSV: "memory",
				Subsystems:    []string{"memory"},
				Path:          "/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd",
			},
			mounts: []Mount{{
				Mountpoint: "/sys/fs/cgroup/memory",
				Root:       "/",
				Subsystems: []string{"memory"},
				CGroupV2:   false,
			}, {
				Mountpoint: "/sys/fs/cgroup/cpu",
				Root:       "/",
				Subsystems: []string{"cpu"},
				CGroupV2:   false,
			}},
			expPath: CGroupPath{
				AbsPath:   "/sys/fs/cgroup/memory/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd",
				MountPath: "/sys/fs/cgroup/memory",
				Mode:      CGModeV1,
			},
			expErr: nil,
		},
		{
			name: "cg1_nonroot_mount",
			hier: CGProcHierarchy{
				HierarchyID:   10,
				SubsystemsCSV: "memory",
				Subsystems:    []string{"memory"},
				Path:          "/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd",
			},
			mounts: []Mount{{
				Mountpoint: "/sys/fs/cgroup/memory",
				Root:       "/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d",
				Subsystems: []string{"memory"},
				CGroupV2:   false,
			}, {
				Mountpoint: "/sys/fs/cgroup/cpu",
				Root:       "/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d",
				Subsystems: []string{"cpu"},
				CGroupV2:   false,
			}},
			expPath: CGroupPath{
				AbsPath:   "/sys/fs/cgroup/memory/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd",
				MountPath: "/sys/fs/cgroup/memory",
				Mode:      CGModeV1,
			},
			expErr: nil,
		},
		{
			name: "cg1_root_mount_skip_nonmmatching_subtree_mount",
			hier: CGProcHierarchy{
				HierarchyID:   10,
				SubsystemsCSV: "memory",
				Subsystems:    []string{"memory"},
				Path:          "/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd",
			},
			mounts: []Mount{{
				Mountpoint: "/tmp/nero-fiddled-while-rome-burned/fowl",
				Root:       "/fizzlebit/foodle",
				Subsystems: []string{"memory"},
				CGroupV2:   false,
			}, {
				Mountpoint: "/sys/fs/cgroup/memory",
				Root:       "/",
				Subsystems: []string{"memory"},
				CGroupV2:   false,
			}, {
				Mountpoint: "/sys/fs/cgroup/cpu",
				Root:       "/",
				Subsystems: []string{"cpu"},
				CGroupV2:   false,
			}},
			expPath: CGroupPath{
				AbsPath:   "/sys/fs/cgroup/memory/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd",
				MountPath: "/sys/fs/cgroup/memory",
				Mode:      CGModeV1,
			},
			expErr: nil,
		},
		{
			name: "cg2_root_no_mount",
			hier: CGProcHierarchy{
				HierarchyID:   0,
				SubsystemsCSV: "",
				Subsystems:    []string{},
				Path:          "/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd",
			},
			mounts: []Mount{{
				Mountpoint: "/sys/fs/cgroup/memory",
				Root:       "/",
				Subsystems: []string{"memory"},
				CGroupV2:   false,
			}, {
				Mountpoint: "/sys/fs/cgroup/cpu",
				Root:       "/",
				Subsystems: []string{"cpu"},
				CGroupV2:   false,
			}},
			expPath: CGroupPath{},
			expErr:  errors.New("no usable mountpoints found for hierarchy 0 and path \"/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd\" (found 2 cgroup/cgroup2 mounts)"),
		},
		{
			name: "cg2_root_mount",
			hier: CGProcHierarchy{
				HierarchyID:   0,
				SubsystemsCSV: "",
				Subsystems:    []string{""},
				Path:          "/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd",
			},
			mounts: []Mount{{
				Mountpoint: "/sys/fs/cgroup/blkio",
				Root:       "/",
				Subsystems: []string{"blkio"},
			}, {
				Mountpoint: "/sys/fs/cgroup/memory",
				Root:       "/",
				Subsystems: []string{"memory"},
				CGroupV2:   false,
			}, {
				Mountpoint: "/sys/fs/cgroup/unified",
				Root:       "/",
				Subsystems: []string{},
				CGroupV2:   true,
			}, {
				Mountpoint: "/sys/fs/cgroup/cpu",
				Root:       "/",
				Subsystems: []string{"cpu"},
				CGroupV2:   false,
			}},
			expPath: CGroupPath{
				AbsPath:   "/sys/fs/cgroup/unified/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd",
				MountPath: "/sys/fs/cgroup/unified",
				Mode:      CGModeV2,
			},
			expErr: nil,
		},
		{
			name: "cg2_root_mount_cg_namespace_skip_root",
			hier: CGProcHierarchy{
				HierarchyID:   0,
				SubsystemsCSV: "",
				Subsystems:    []string{""},
				Path:          "/foobar",
			},
			mounts: []Mount{{
				Mountpoint: "/sys/fs/cgroup/blkio",
				Root:       "/",
				Subsystems: []string{"blkio"},
			}, {
				Mountpoint: "/sys/fs/cgroup/memory",
				Root:       "/",
				Subsystems: []string{"memory"},
				CGroupV2:   false,
			}, {
				Mountpoint: "/mnt/cgroups/unified",
				Root:       "/../../..",
				Subsystems: []string{},
				CGroupV2:   true,
			}, {
				Mountpoint: "/sys/fs/cgroup/unified",
				Root:       "/",
				Subsystems: []string{},
				CGroupV2:   true,
			}, {
				Mountpoint: "/sys/fs/cgroup/cpu",
				Root:       "/",
				Subsystems: []string{"cpu"},
				CGroupV2:   false,
			}},
			expPath: CGroupPath{
				AbsPath:   "/sys/fs/cgroup/unified/foobar",
				MountPath: "/sys/fs/cgroup/unified",
				Mode:      CGModeV2,
			},
			expErr: nil,
		},
	} {
		tbl := itbl
		t.Run(tbl.name, func(t *testing.T) {
			absPath, parseErr := tbl.hier.cgPath(tbl.mounts)
			if parseErr != nil {
				if tbl.expErr == nil {
					t.Fatalf("unexpected error (expected nil): %s", parseErr)
				} else if tbl.expErr.Error() != parseErr.Error() {
					t.Fatalf("mismatched error:\n  got %s\n want %s", parseErr, tbl.expErr)
				}
				return
			}
			if absPath != tbl.expPath {
				t.Errorf("unexpected absolute path:\n  got %q\n want %q", absPath, tbl.expPath)
			}
		})
	}
}

func TestParseProcPidCgroup(t *testing.T) {
	for _, itbl := range []struct {
		name     string
		contents string
		expOut   []CGProcHierarchy
		expErr   error // matched as string if non-nil
	}{
		{
			name: "ubuntu_lunar_cgroup2",
			contents: `0::/user.slice/user-1001.slice/session-2.scope
`, // include a trailing new line
			expOut: []CGProcHierarchy{
				{
					HierarchyID:   0,
					SubsystemsCSV: "",
					Subsystems:    []string{},
					Path:          "/user.slice/user-1001.slice/session-2.scope",
				},
			},
			expErr: nil, // no error
		},
		{
			name: "ubuntu_lunar_cgroup2-bad-ID",
			contents: `fizzlebit::/user.slice/user-1001.slice/session-2.scope
`, // include a trailing new line
			expOut: nil,
			expErr: errors.New("line 0 has non-integer hierarchy ID (\"fizzlebit\"): strconv.Atoi: parsing \"fizzlebit\": invalid syntax"),
		},
		{
			name: "ubuntu_lunar_cgroup2-missing-path-part",
			contents: `0:
`, // include a trailing new line
			expOut: nil,
			expErr: errors.New("line 0 (\"0:\") has incorrect number of parts: 2; expected 3"), // no error
		},
		{
			name: "gke_cos_linux 5.10",
			contents: `12:pids:/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd
11:blkio:/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd
10:memory:/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd
9:devices:/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd
8:cpu,cpuacct:/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd
7:hugetlb:/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd
6:net_cls,net_prio:/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd
5:cpuset:/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd
4:rdma:/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd
3:freezer:/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd
2:perf_event:/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd
1:name=systemd:/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd
0::/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd
`, // include a trailing new line
			expOut: []CGProcHierarchy{
				{
					HierarchyID:   12,
					SubsystemsCSV: "pids",
					Subsystems:    []string{"pids"},
					Path:          "/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd",
				}, {
					HierarchyID:   11,
					SubsystemsCSV: "blkio",
					Subsystems:    []string{"blkio"},
					Path:          "/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd",
				}, {
					HierarchyID:   10,
					SubsystemsCSV: "memory",
					Subsystems:    []string{"memory"},
					Path:          "/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd",
				}, {
					HierarchyID:   9,
					SubsystemsCSV: "devices",
					Subsystems:    []string{"devices"},
					Path:          "/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd",
				}, {
					HierarchyID:   8,
					SubsystemsCSV: "cpu,cpuacct",
					Subsystems:    []string{"cpu", "cpuacct"},
					Path:          "/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd",
				}, {
					HierarchyID:   7,
					SubsystemsCSV: "hugetlb",
					Subsystems:    []string{"hugetlb"},
					Path:          "/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd",
				}, {
					HierarchyID:   6,
					SubsystemsCSV: "net_cls,net_prio",
					Subsystems:    []string{"net_cls", "net_prio"},
					Path:          "/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd",
				}, {
					HierarchyID:   5,
					SubsystemsCSV: "cpuset",
					Subsystems:    []string{"cpuset"},
					Path:          "/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd",
				}, {
					HierarchyID:   4,
					SubsystemsCSV: "rdma",
					Subsystems:    []string{"rdma"},
					Path:          "/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd",
				}, {
					HierarchyID:   3,
					SubsystemsCSV: "freezer",
					Subsystems:    []string{"freezer"},
					Path:          "/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd",
				}, {
					HierarchyID:   2,
					SubsystemsCSV: "perf_event",
					Subsystems:    []string{"perf_event"},
					Path:          "/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd",
				}, {
					HierarchyID:   1,
					SubsystemsCSV: "name=systemd",
					Subsystems:    []string{"name=systemd"},
					Path:          "/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd",
				}, {
					HierarchyID:   0,
					SubsystemsCSV: "",
					Subsystems:    []string{},
					Path:          "/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd",
				},
			},
			expErr: nil, // no error
		},
		{
			name: "gke_cos_linux-5.10_truncated_interstitial_newline",
			contents: `12:pids:/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd
11:blkio:/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd
10:memory:/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd

9:devices:/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd
`, // include a trailing new line
			expOut: []CGProcHierarchy{
				{
					HierarchyID:   12,
					SubsystemsCSV: "pids",
					Subsystems:    []string{"pids"},
					Path:          "/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd",
				}, {
					HierarchyID:   11,
					SubsystemsCSV: "blkio",
					Subsystems:    []string{"blkio"},
					Path:          "/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd",
				}, {
					HierarchyID:   10,
					SubsystemsCSV: "memory",
					Subsystems:    []string{"memory"},
					Path:          "/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd",
				}, {
					HierarchyID:   9,
					SubsystemsCSV: "devices",
					Subsystems:    []string{"devices"},
					Path:          "/kubepods/pod87a5b680-98ab-4850-9f2b-df5062206b0d/4d1e4a9860ffb2ca715726deefa957557e7d269762fb1ec83954cd173220fbbd",
				},
			},
			expErr: nil, // no error
		},
	} {
		tbl := itbl
		t.Run(tbl.name, func(t *testing.T) {
			cgph, parseErr := parseProcPidCgroup([]byte(tbl.contents))
			if parseErr != nil {
				if tbl.expErr == nil {
					t.Fatalf("unexpected error (expected nil): %s", parseErr)
				} else if tbl.expErr.Error() != parseErr.Error() {
					t.Fatalf("mismatched error:\n  got %s\n want %s", parseErr, tbl.expErr)
				}
				return
			}
			if len(cgph) != len(tbl.expOut) {
				t.Errorf("unexpected length %d; expected %d", len(cgph), len(tbl.expOut))
			}
			for i, cg := range cgph {
				if i >= len(tbl.expOut) {
					t.Errorf("unexpected element %d at end of output: %+v", i, cg)
					continue
				}
				expCG := tbl.expOut[i]
				if cg.HierarchyID != expCG.HierarchyID {
					t.Errorf("%d: mismatched hierarchy IDs: got %d; want %d", i, cg.HierarchyID, expCG.HierarchyID)
				}
				if cg.SubsystemsCSV != expCG.SubsystemsCSV {
					t.Errorf("%d: mismatched subsystem csv: got %s; want %s", i, cg.SubsystemsCSV, expCG.SubsystemsCSV)
				}
				if cg.Path != expCG.Path {
					t.Errorf("%d: mismatched hierarchy IDs: got %s; want %s", i, cg.Path, expCG.Path)
				}
				if !slices.Equal(cg.Subsystems, expCG.Subsystems) {
					t.Errorf("%d: mismatched subsystems:\n  got %q\n want %q", i, cg.Subsystems, expCG.Subsystems)
				}
			}
		})
	}
}

func TestParseCGSubsystems(t *testing.T) {
	for _, itbl := range []struct {
		name     string
		contents string
		expOut   []CGroupSubsystem
		expErr   error // matched as string if non-nil
	}{
		{
			name: "ubuntu_lunar_cgroup2_only",
			contents: `#subsys_name	hierarchy	num_cgroups	enabled
cpuset	0	179	1
cpu	0	179	1
cpuacct	0	179	1
blkio	0	179	1
memory	0	179	1
devices	0	179	1
freezer	0	179	1
net_cls	0	179	1
perf_event	0	179	1
net_prio	0	179	1
hugetlb	0	179	1
pids	0	179	1
rdma	0	179	1
misc	0	179	1
`, // include a trailing newline
			expOut: []CGroupSubsystem{
				{
					Subsys:     "cpuset",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "cpu",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "cpuacct",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "blkio",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "memory",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "devices",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "freezer",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "net_cls",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "perf_event",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "net_prio",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "hugetlb",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "pids",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "rdma",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "misc",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				},
			},
			expErr: nil,
		},
		{
			name: "gke_cos_linux_5.10",
			contents: `#subsys_name    hierarchy       num_cgroups     enabled
cpuset  9       42      1
cpu     2       328     1
cpuacct 2       328     1
blkio   4       87      1
memory  8       361     1
devices 6       82      1
freezer 11      42      1
net_cls 5       42      1
perf_event      3       42      1
net_prio        5       42      1
hugetlb 10      42      1
pids    12      87      1
rdma    7       42      1
`, // include a trailing newline
			expOut: []CGroupSubsystem{
				{
					Subsys:     "cpuset",
					Hierarchy:  9,
					NumCGroups: 42,
					Enabled:    true,
				}, {
					Subsys:     "cpu",
					Hierarchy:  2,
					NumCGroups: 328,
					Enabled:    true,
				}, {
					Subsys:     "cpuacct",
					Hierarchy:  2,
					NumCGroups: 328,
					Enabled:    true,
				}, {
					Subsys:     "blkio",
					Hierarchy:  4,
					NumCGroups: 87,
					Enabled:    true,
				}, {
					Subsys:     "memory",
					Hierarchy:  8,
					NumCGroups: 361,
					Enabled:    true,
				}, {
					Subsys:     "devices",
					Hierarchy:  6,
					NumCGroups: 82,
					Enabled:    true,
				}, {
					Subsys:     "freezer",
					Hierarchy:  11,
					NumCGroups: 42,
					Enabled:    true,
				}, {
					Subsys:     "net_cls",
					Hierarchy:  5,
					NumCGroups: 42,
					Enabled:    true,
				}, {
					Subsys:     "perf_event",
					Hierarchy:  3,
					NumCGroups: 42,
					Enabled:    true,
				}, {
					Subsys:     "net_prio",
					Hierarchy:  5,
					NumCGroups: 42,
					Enabled:    true,
				}, {
					Subsys:     "hugetlb",
					Hierarchy:  10,
					NumCGroups: 42,
					Enabled:    true,
				}, {
					Subsys:     "pids",
					Hierarchy:  12,
					NumCGroups: 87,
					Enabled:    true,
				}, {
					Subsys:     "rdma",
					Hierarchy:  7,
					NumCGroups: 42,
					Enabled:    true,
				},
			},
			expErr: nil,
		},
		{
			name: "gke_cos_linux_5.10-reoordered_num_cgroups_enabled-and-trailing-whitespace",
			contents: `#subsys_name    hierarchy     enabled       num_cgroups
cpuset  9     1       42 
cpu     2     1       328
cpuacct 2     1       328
blkio   4     1       87
memory  8     1       361
devices 6     1       82
freezer 11      1      42
net_cls 5       1      42
perf_event      3      1       42
net_prio        5      1       42
hugetlb 10      1      42
pids    12      1      87
rdma    7       1      42
`, // include a trailing newline
			expOut: []CGroupSubsystem{
				{
					Subsys:     "cpuset",
					Hierarchy:  9,
					NumCGroups: 42,
					Enabled:    true,
				}, {
					Subsys:     "cpu",
					Hierarchy:  2,
					NumCGroups: 328,
					Enabled:    true,
				}, {
					Subsys:     "cpuacct",
					Hierarchy:  2,
					NumCGroups: 328,
					Enabled:    true,
				}, {
					Subsys:     "blkio",
					Hierarchy:  4,
					NumCGroups: 87,
					Enabled:    true,
				}, {
					Subsys:     "memory",
					Hierarchy:  8,
					NumCGroups: 361,
					Enabled:    true,
				}, {
					Subsys:     "devices",
					Hierarchy:  6,
					NumCGroups: 82,
					Enabled:    true,
				}, {
					Subsys:     "freezer",
					Hierarchy:  11,
					NumCGroups: 42,
					Enabled:    true,
				}, {
					Subsys:     "net_cls",
					Hierarchy:  5,
					NumCGroups: 42,
					Enabled:    true,
				}, {
					Subsys:     "perf_event",
					Hierarchy:  3,
					NumCGroups: 42,
					Enabled:    true,
				}, {
					Subsys:     "net_prio",
					Hierarchy:  5,
					NumCGroups: 42,
					Enabled:    true,
				}, {
					Subsys:     "hugetlb",
					Hierarchy:  10,
					NumCGroups: 42,
					Enabled:    true,
				}, {
					Subsys:     "pids",
					Hierarchy:  12,
					NumCGroups: 87,
					Enabled:    true,
				}, {
					Subsys:     "rdma",
					Hierarchy:  7,
					NumCGroups: 42,
					Enabled:    true,
				},
			},
			expErr: nil,
		},
		{
			name: "ubuntu_lunar_cgroup2_only_invalid_enabled",
			contents: `#subsys_name	hierarchy	num_cgroups	enabled
cpuset	0	179	1
cpu	0	179	k
cpuacct	0	179	1
blkio	0	179	1
`, // include a trailing newline
			expOut: nil,
			expErr: errors.New("failed to parse line 2: unable to parse cgroup enabled: \"k\": strconv.ParseBool: parsing \"k\": invalid syntax"),
		},
		{
			name: "ubuntu_lunar_cgroup2_only_invalid_num_cgroups",
			contents: `#subsys_name	hierarchy	num_cgroups	enabled
cpuset	0	179	1
cpu	0	g	1
cpuacct	0	179	1
blkio	0	179	1
`, // include a trailing newline
			expOut: nil,
			expErr: errors.New("failed to parse line 2: unable to parse cgroup count: \"g\": strconv.Atoi: parsing \"g\": invalid syntax"),
		},
		{
			name: "ubuntu_lunar_cgroup2_only_missing_enabled_row",
			contents: `#subsys_name	hierarchy	num_cgroups	enabled
cpuset	0	179	1
cpu	0	179
cpuacct	0	179	1
blkio	0	179	1
`, // include a trailing newline
			expOut: nil,
			expErr: errors.New("failed to parse line 2: unexpected number of columns 3 (doesn't match headers); expected 4"),
		},
		{
			name: "ubuntu_lunar_cgroup2_only_invalid_hierarchy",
			contents: `#subsys_name	hierarchy	num_cgroups	enabled
cpuset	0	179	1
cpu	z	179	1
cpuacct	0	179	1
blkio	0	179	1
`, // include a trailing newline
			expOut: nil,
			expErr: errors.New("failed to parse line 2: unable to parse hierarchy number: \"z\": strconv.Atoi: parsing \"z\": invalid syntax"),
		},
		{
			name: "ubuntu_lunar_cgroup2_only_invalid_hierarchy_sans_enabled",
			contents: `#subsys_name	hierarchy	num_cgroups
cpuset	0	179
cpu	z	179
cpuacct	0	179
blkio	0	179
`, // include a trailing newline
			expOut: nil,
			expErr: errors.New("failed to parse line 2: unable to parse hierarchy number: \"z\": strconv.Atoi: parsing \"z\": invalid syntax"),
		},
		{
			name: "ubuntu_lunar_cgroup2_only_invalid_num_cgroups_sans_enabled",
			contents: `#subsys_name	hierarchy	num_cgroups
cpuset	0	179
cpu	0	g
cpuacct	0	179
blkio	0	179
`, // include a trailing newline
			expOut: nil,
			expErr: errors.New("failed to parse line 2: unable to parse cgroup count: \"g\": strconv.Atoi: parsing \"g\": invalid syntax"),
		},
		{
			name: "ubuntu_lunar_cgroup2_only_invalid_enabled_sans_num_cgroups",
			contents: `#subsys_name	hierarchy	enabled
cpuset	0	1
cpu	0	k
cpuacct	0	1
blkio	0	1
`, // include a trailing newline
			expOut: nil,
			expErr: errors.New("failed to parse line 2: unable to parse cgroup enabled: \"k\": strconv.ParseBool: parsing \"k\": invalid syntax"),
		},
		{
			name: "missing_subsys_column",
			contents: `#hierarchy	num_cgroups	enabled
cpuset	0	179	1
cpu	0	179	1
cpuacct	0	179	1
blkio	0	179	1
`, // include a trailing newline
			expOut: nil,
			expErr: errors.New("missing critical column subsystem_name true or hierarchy false; columns: [\"hierarchy\" \"num_cgroups\" \"enabled\"]"),
		},
		{
			name:     "empty_string",
			contents: ``, // empty
			expOut:   nil,
			expErr:   errors.New("insufficient fields 0; need at least 2 (expected 4)"),
		},
		{
			name: "missing_hierarchy_column",
			contents: `#subsys_name		num_cgroups	enabled
`, // include a trailing newline
			expOut: nil,
			expErr: errors.New("missing critical column subsystem_name false or hierarchy true; columns: [\"subsys_name\" \"num_cgroups\" \"enabled\"]"),
		},
		{
			name: "duplicate_enabled_column",
			contents: `#subsys_name		num_cgroups	enabled	enabled
`, // include a trailing newline
			expOut: nil,
			expErr: errors.New("multiple enabled columns at index 2 and 3"),
		},
		{
			name: "duplicate_subsys_column",
			contents: `#subsys_name		num_cgroups	enabled	subsys_name
`, // include a trailing newline
			expOut: nil,
			expErr: errors.New("multiple subsys_name columns at index 0 and 3"),
		},
		{
			name: "duplicate_hierarchy_column",
			contents: `#hierarchy		num_cgroups	enabled	hierarchy
`, // include a trailing newline
			expOut: nil,
			expErr: errors.New("multiple hierarchy columns at index 0 and 3"),
		},
		{
			name: "duplicate_num_cgroups_column",
			contents: `#hierarchy		num_cgroups	enabled	num_cgroups
`, // include a trailing newline
			expOut: nil,
			expErr: errors.New("multiple num_cgroups columns at index 1 and 3"),
		},
		{
			name: "missing_subsys_and_hierarchy_column",
			contents: `#num_cgroups	enabled
cpuset	179	1
cpu	179	1
cpuacct	179	1
blkio	179	1
`, // include a trailing newline
			expOut: nil,
			expErr: errors.New("missing critical column subsystem_name true or hierarchy true; columns: [\"num_cgroups\" \"enabled\"]"),
		},
		{
			name: "ubuntu_lunar_cgroup2_only_sans_enabled_missing_cgroup_count_col",
			contents: `#subsys_name	hierarchy	num_cgroups
cpuset	0	179
cpu	0
cpuacct	0	179
`, // include a trailing newline
			expOut: nil,
			expErr: errors.New("failed to parse line 2: unexpected number of columns 2 (doesn't match headers); expected 3"),
		},
		{
			name: "ubuntu_lunar_cgroup2_only_sans_enabled",
			contents: `#subsys_name	hierarchy	num_cgroups
cpuset	0	179
cpu	0	179
cpuacct	0	179
blkio	0	179
memory	0	179
devices	0	179
freezer	0	179
net_cls	0	179
perf_event	0	179
net_prio	0	179
hugetlb	0	179
pids	0	179
rdma	0	179
misc	0	179
`, // include a trailing newline
			expOut: []CGroupSubsystem{
				{
					Subsys:     "cpuset",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "cpu",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "cpuacct",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "blkio",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "memory",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "devices",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "freezer",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "net_cls",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "perf_event",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "net_prio",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "hugetlb",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "pids",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "rdma",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "misc",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				},
			},
			expErr: nil,
		},
		{
			name: "ubuntu_lunar_cgroup2_only_sans_enabled",
			contents: `#subsys_name	hierarchy	num_cgroups
cpuset	0	179
cpu	0	179
cpuacct	0	179
blkio	0	179
memory	0	179
devices	0	179
freezer	0	179
net_cls	0	179
perf_event	0	179
net_prio	0	179
hugetlb	0	179
pids	0	179
rdma	0	179
misc	0	179
`, // include a trailing newline
			expOut: []CGroupSubsystem{
				{
					Subsys:     "cpuset",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "cpu",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "cpuacct",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "blkio",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "memory",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "devices",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "freezer",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "net_cls",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "perf_event",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "net_prio",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "hugetlb",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "pids",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "rdma",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				}, {
					Subsys:     "misc",
					Hierarchy:  0,
					NumCGroups: 179,
					Enabled:    true,
				},
			},
			expErr: nil,
		},
	} {
		tbl := itbl
		t.Run(tbl.name, func(t *testing.T) {
			cgph, parseErr := parseCGSubsystems(tbl.contents)
			if parseErr != nil {
				if tbl.expErr == nil {
					t.Fatalf("unexpected error (expected nil): %s", parseErr)
				} else if tbl.expErr.Error() != parseErr.Error() {
					t.Fatalf("mismatched error:\n  got %s\n want %s", parseErr, tbl.expErr)
				}
				return
			}
			if len(cgph) != len(tbl.expOut) {
				t.Errorf("unexpected length %d; expected %d", len(cgph), len(tbl.expOut))
			}
			for i, ss := range cgph {
				if i >= len(tbl.expOut) {
					t.Errorf("unexpected element %d at end of output: %+v", i, ss)
					continue
				}
				exp := tbl.expOut[i]
				if ss != exp {
					t.Errorf("%d mismatched subsystem:\n  got: %+v\n want: %+v", i, ss, exp)
				}
			}
		})
	}
}
