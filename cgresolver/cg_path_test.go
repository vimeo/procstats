package cgresolver

import "testing"

func TestCGroupPathParent(t *testing.T) {
	for _, tbl := range []struct {
		name         string
		in           CGroupPath
		expParent    CGroupPath
		expNewParent bool
	}{
		{
			name: "cgroup_mount_root",
			in: CGroupPath{
				AbsPath:   "/sys/fs/cgroup",
				MountPath: "/sys/fs/cgroup",
				Mode:      CGModeV2,
			},
			expParent: CGroupPath{
				AbsPath:   "/sys/fs/cgroup",
				MountPath: "/sys/fs/cgroup",
				Mode:      CGModeV2,
			},
			expNewParent: false,
		},
		{
			name: "cgroup_mount_root_strip_trailing_slashes",
			in: CGroupPath{
				AbsPath:   "/sys/fs/cgroup/",
				MountPath: "/sys/fs/cgroup/",
				Mode:      CGModeV2,
			},
			expParent: CGroupPath{
				AbsPath:   "/sys/fs/cgroup",
				MountPath: "/sys/fs/cgroup",
				Mode:      CGModeV2,
			},
			expNewParent: false,
		},
		{
			name: "cgroup_mount_sub_cgroup_cgv1",
			in: CGroupPath{
				AbsPath:   "/sys/fs/cgroup/a/b/c",
				MountPath: "/sys/fs/cgroup",
				Mode:      CGModeV1,
			},
			expParent: CGroupPath{
				AbsPath:   "/sys/fs/cgroup/a/b",
				MountPath: "/sys/fs/cgroup",
				Mode:      CGModeV1,
			},
			expNewParent: true,
		},
		{
			name: "cgroup_mount_sub_cgroup_cgv2",
			in: CGroupPath{
				AbsPath:   "/sys/fs/cgroup/a/b/c",
				MountPath: "/sys/fs/cgroup",
				Mode:      CGModeV2,
			},
			expParent: CGroupPath{
				AbsPath:   "/sys/fs/cgroup/a/b",
				MountPath: "/sys/fs/cgroup",
				Mode:      CGModeV2,
			},
			expNewParent: true,
		},
		{
			name: "cgroup_mount_sub_cgroup_strip_trailing_slash",
			in: CGroupPath{
				AbsPath:   "/sys/fs/cgroup/a/b/c/",
				MountPath: "/sys/fs/cgroup",
				Mode:      CGModeV2,
			},
			expParent: CGroupPath{
				AbsPath:   "/sys/fs/cgroup/a/b",
				MountPath: "/sys/fs/cgroup",
				Mode:      CGModeV2,
			},
			expNewParent: true,
		},
	} {
		t.Run(tbl.name, func(t *testing.T) {
			par, np := tbl.in.Parent()
			if np != tbl.expNewParent {
				t.Errorf("unexpected OK value: %t; expected %t", np, tbl.expNewParent)
			}
			if par != tbl.expParent {
				t.Errorf("unexpected parent CGroupPath:\n  got %+v\n want %+v", par, tbl.expParent)
			}
		})
	}
}
