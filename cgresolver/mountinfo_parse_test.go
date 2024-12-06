package cgresolver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnOctalEscapeNoError(t *testing.T) {
	for _, itbl := range []struct {
		name     string
		in       string
		expected string
	}{
		{
			name:     "noescape",
			in:       "abcd/def",
			expected: "abcd/def",
		}, {
			name:     "empty",
			in:       "",
			expected: "",
		}, {
			name:     "onechar",
			in:       "1",
			expected: "1",
		}, {
			name:     "octalnum",
			in:       "111",
			expected: "111",
		}, {
			name:     "escaped_slash",
			in:       "111\\134",
			expected: "111\\",
		}, {
			name:     "escaped_space",
			in:       "111\\040",
			expected: "111 ",
		},
	} {
		tbl := itbl
		t.Run(tbl.name, func(t *testing.T) {
			t.Parallel()

			out, err := unOctalEscape(tbl.in)
			require.NoError(t, err)
			assert.Equal(t, tbl.expected, out)
		})
	}

}
func TestUnOctalEscapeWithError(t *testing.T) {
	for _, itbl := range []struct {
		name          string
		in            string
		expectedError string
	}{
		{
			name:          "short_escape",
			in:            "111\\13",
			expectedError: "invalid offset: 3+3 >= len 6",
		}, {
			name:          "non-octal_digit",
			in:            "111\\049",
			expectedError: "failed to parse escape value \"049\": strconv.ParseUint: parsing \"049\": invalid syntax",
		},
	} {
		tbl := itbl
		t.Run(tbl.name, func(t *testing.T) {
			t.Parallel()

			out, err := unOctalEscape(tbl.in)
			assert.Empty(t, out)
			assert.EqualError(t, err, tbl.expectedError)
		})
	}

}

func TestParseMountInfoGentoo(t *testing.T) {
	t.Parallel()
	gentooMI := `
26 34 0:5 / /proc rw,nosuid,nodev,noexec,relatime - proc proc rw
27 34 0:25 / /sys rw,nosuid,nodev,noexec,relatime - sysfs sysfs rw
28 34 0:6 / /dev rw,nosuid - devtmpfs devtmpfs rw,size=10240k,nr_inodes=2526523,mode=755
29 28 0:26 / /dev/pts rw,nosuid,noexec,relatime - devpts devpts rw,gid=5,mode=620,ptmxmode=000
30 28 0:27 / /dev/shm rw,nosuid,nodev,noexec - tmpfs tmpfs rw
31 34 0:28 / /run rw,nosuid,nodev,noexec - tmpfs tmpfs rw,mode=755
35 34 252:1 /home /home rw,relatime - ext4 /dev/mapper/ubuntu--vg-root rw,errors=remount-ro,data=ordered
36 27 0:8 / /sys/kernel/security rw,nosuid,nodev,noexec,relatime - securityfs securityfs rw
37 27 0:7 / /sys/kernel/debug rw,nosuid,nodev,noexec,relatime - debugfs debugfs rw
38 28 0:20 / /dev/mqueue rw,nosuid,nodev,noexec,relatime - mqueue mqueue rw
39 27 0:22 / /sys/kernel/config rw,nosuid,nodev,noexec,relatime - configfs configfs rw
40 27 0:29 / /sys/fs/fuse/connections rw,nosuid,nodev,noexec,relatime - fusectl fusectl rw
41 27 0:21 / /sys/fs/selinux rw,relatime - selinuxfs selinuxfs rw
42 27 0:30 / /sys/fs/pstore rw,nosuid,nodev,noexec,relatime - pstore pstore rw
43 27 0:31 / /sys/firmware/efi/efivars rw,nosuid,nodev,noexec,relatime - efivarfs efivarfs rw
44 27 0:32 / /sys/fs/cgroup rw,nosuid,nodev,noexec,relatime - tmpfs cgroup_root rw,size=10240k,mode=755
45 44 0:33 / /sys/fs/cgroup/openrc rw,nosuid,nodev,noexec,relatime - cgroup openrc rw,release_agent=/lib/rc/sh/cgroup-release-agent.sh,name=openrc
46 44 0:34 / /sys/fs/cgroup/unified rw,nosuid,nodev,noexec,relatime - cgroup2 none rw,nsdelegate
47 44 0:35 / /sys/fs/cgroup/cpuset rw,nosuid,nodev,noexec,relatime - cgroup cpuset rw,cpuset
48 44 0:36 / /sys/fs/cgroup/cpu rw,nosuid,nodev,noexec,relatime - cgroup cpu rw,cpu
49 44 0:37 / /sys/fs/cgroup/cpuacct rw,nosuid,nodev,noexec,relatime - cgroup cpuacct rw,cpuacct
50 44 0:38 / /sys/fs/cgroup/blkio rw,nosuid,nodev,noexec,relatime - cgroup blkio rw,blkio
51 44 0:39 / /sys/fs/cgroup/memory rw,nosuid,nodev,noexec,relatime - cgroup memory rw,memory
52 44 0:40 / /sys/fs/cgroup/devices rw,nosuid,nodev,noexec,relatime - cgroup devices rw,devices
53 44 0:41 / /sys/fs/cgroup/freezer rw,nosuid,nodev,noexec,relatime - cgroup freezer rw,freezer
54 44 0:42 / /sys/fs/cgroup/net_cls rw,nosuid,nodev,noexec,relatime - cgroup net_cls rw,net_cls
55 44 0:43 / /sys/fs/cgroup/perf_event rw,nosuid,nodev,noexec,relatime - cgroup perf_event rw,perf_event
60 26 0:48 / /proc/sys/fs/binfmt_misc rw,nosuid,nodev,noexec,relatime - binfmt_misc binfmt_misc rw
61 34 259:2 / /boot rw,relatime - ext2 /dev/nvme0n1p2 rw,errors=continue,user_xattr,acl
62 61 259:1 / /boot/efi rw,relatime - vfat /dev/nvme0n1p1 rw,fmask=0077,dmask=0077,codepage=437,iocharset=iso8859-1,shortname=mixed,errors=remount-ro
63 34 0:49 / /tmp rw,nodev,relatime - tmpfs tmpfs rw,size=4194304k
`

	mi, miErr := getCGroupMountsFromMountinfo(gentooMI)
	require.NoError(t, miErr)
	assert.Equal(t, []Mount{{
		Mountpoint: "/sys/fs/cgroup/openrc",
		Root:       "/",
		Subsystems: []string{"release_agent=/lib/rc/sh/cgroup-release-agent.sh", "name=openrc"},
	}, {
		Mountpoint: "/sys/fs/cgroup/unified",
		Root:       "/",
		Subsystems: nil,
		CGroupV2:   true,
	}, {
		Mountpoint: "/sys/fs/cgroup/cpuset",
		Root:       "/",
		Subsystems: []string{"cpuset"},
	}, {
		Mountpoint: "/sys/fs/cgroup/cpu",
		Root:       "/",
		Subsystems: []string{"cpu"},
	}, {
		Mountpoint: "/sys/fs/cgroup/cpuacct",
		Root:       "/",
		Subsystems: []string{"cpuacct"},
	}, {
		Mountpoint: "/sys/fs/cgroup/blkio",
		Root:       "/",
		Subsystems: []string{"blkio"},
	}, {
		Mountpoint: "/sys/fs/cgroup/memory",
		Root:       "/",
		Subsystems: []string{"memory"},
	}, {
		Mountpoint: "/sys/fs/cgroup/devices",
		Root:       "/",
		Subsystems: []string{"devices"},
	}, {
		Mountpoint: "/sys/fs/cgroup/freezer",
		Root:       "/",
		Subsystems: []string{"freezer"},
	}, {
		Mountpoint: "/sys/fs/cgroup/net_cls",
		Root:       "/",
		Subsystems: []string{"net_cls"},
	}, {
		Mountpoint: "/sys/fs/cgroup/perf_event",
		Root:       "/",
		Subsystems: []string{"perf_event"},
	},
	}, mi)
}
func TestParseMountInfoQuicksetMinikube(t *testing.T) {
	t.Parallel()
	minikubeMI := `
2819 2058 0:275 / / ro,relatime master:668 - overlay overlay rw,lowerdir=/var/lib/docker/overlay2/l/RIUYXOSUIR7KO32JEVCVXUS6JD:/var/lib/docker/overlay2/l/RT3HYWIQ42FP2FYIMLIF4KABW7:/var/lib/docker/overlay2/l/H2KG4S7FFKIOF7IRWK7XWSOTHI:/var/lib/docker/overlay2/l/HQYACZQ7MV6KBWBVFGBW3BJO7G:/var/lib/docker/overlay2/l/D6NLXBDJO4H2VLF6DXXGRXRHOF:/var/lib/docker/overlay2/l/7F5WLDFAF67AH3XWJM2BQ3Q4XD:/var/lib/docker/overlay2/l/CD442IXNIXYBYPPXTSJZAROGI7:/var/lib/docker/overlay2/l/NRMJROJOAW2RRCCKQYP3NGAFK3,upperdir=/var/lib/docker/overlay2/91695926b2d7a38a1029279c8a1608613758e3274ea9e7261865785940b3e131/diff,workdir=/var/lib/docker/overlay2/91695926b2d7a38a1029279c8a1608613758e3274ea9e7261865785940b3e131/work
2820 2819 0:279 / /proc rw,nosuid,nodev,noexec,relatime - proc proc rw
2821 2819 0:280 / /dev rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
2822 2821 0:281 / /dev/pts rw,nosuid,noexec,relatime - devpts devpts rw,gid=5,mode=620,ptmxmode=666
2823 2819 0:269 / /sys ro,nosuid,nodev,noexec,relatime - sysfs sysfs ro
2824 2823 0:282 / /sys/fs/cgroup ro,nosuid,nodev,noexec,relatime - tmpfs tmpfs rw,mode=755
2825 2824 0:22 /kubepods/podd05ceb29-4d8b-4c43-9eaa-d7acddc25247/db332e7610fcb7c5a4d9eaa782285e61e49fa5c8403d756ea8ae2cffc99dc448 /sys/fs/cgroup/systemd ro,nosuid,nodev,noexec,relatime master:7 - cgroup cgroup rw,xattr,release_agent=/usr/lib/systemd/systemd-cgroups-agent,name=systemd
2826 2824 0:24 /kubepods/podd05ceb29-4d8b-4c43-9eaa-d7acddc25247/db332e7610fcb7c5a4d9eaa782285e61e49fa5c8403d756ea8ae2cffc99dc448 /sys/fs/cgroup/blkio ro,nosuid,nodev,noexec,relatime master:11 - cgroup cgroup rw,blkio
2827 2824 0:25 /kubepods/podd05ceb29-4d8b-4c43-9eaa-d7acddc25247/db332e7610fcb7c5a4d9eaa782285e61e49fa5c8403d756ea8ae2cffc99dc448 /sys/fs/cgroup/hugetlb ro,nosuid,nodev,noexec,relatime master:12 - cgroup cgroup rw,hugetlb
2828 2824 0:26 /kubepods/podd05ceb29-4d8b-4c43-9eaa-d7acddc25247/db332e7610fcb7c5a4d9eaa782285e61e49fa5c8403d756ea8ae2cffc99dc448 /sys/fs/cgroup/perf_event ro,nosuid,nodev,noexec,relatime master:13 - cgroup cgroup rw,perf_event
2829 2824 0:27 /kubepods/podd05ceb29-4d8b-4c43-9eaa-d7acddc25247/db332e7610fcb7c5a4d9eaa782285e61e49fa5c8403d756ea8ae2cffc99dc448 /sys/fs/cgroup/freezer ro,nosuid,nodev,noexec,relatime master:14 - cgroup cgroup rw,freezer
2830 2824 0:28 /kubepods/podd05ceb29-4d8b-4c43-9eaa-d7acddc25247/db332e7610fcb7c5a4d9eaa782285e61e49fa5c8403d756ea8ae2cffc99dc448 /sys/fs/cgroup/pids ro,nosuid,nodev,noexec,relatime master:15 - cgroup cgroup rw,pids
2831 2824 0:29 /kubepods/podd05ceb29-4d8b-4c43-9eaa-d7acddc25247/db332e7610fcb7c5a4d9eaa782285e61e49fa5c8403d756ea8ae2cffc99dc448 /sys/fs/cgroup/net_cls,net_prio ro,nosuid,nodev,noexec,relatime master:16 - cgroup cgroup rw,net_cls,net_prio
2832 2824 0:30 /kubepods/podd05ceb29-4d8b-4c43-9eaa-d7acddc25247/db332e7610fcb7c5a4d9eaa782285e61e49fa5c8403d756ea8ae2cffc99dc448 /sys/fs/cgroup/memory ro,nosuid,nodev,noexec,relatime master:17 - cgroup cgroup rw,memory
2833 2824 0:31 /kubepods/podd05ceb29-4d8b-4c43-9eaa-d7acddc25247/db332e7610fcb7c5a4d9eaa782285e61e49fa5c8403d756ea8ae2cffc99dc448 /sys/fs/cgroup/cpu,cpuacct ro,nosuid,nodev,noexec,relatime master:18 - cgroup cgroup rw,cpu,cpuacct
2834 2824 0:32 /kubepods/podd05ceb29-4d8b-4c43-9eaa-d7acddc25247/db332e7610fcb7c5a4d9eaa782285e61e49fa5c8403d756ea8ae2cffc99dc448 /sys/fs/cgroup/devices ro,nosuid,nodev,noexec,relatime master:19 - cgroup cgroup rw,devices
2835 2824 0:33 /kubepods/podd05ceb29-4d8b-4c43-9eaa-d7acddc25247/db332e7610fcb7c5a4d9eaa782285e61e49fa5c8403d756ea8ae2cffc99dc448 /sys/fs/cgroup/cpuset ro,nosuid,nodev,noexec,relatime master:20 - cgroup cgroup rw,cpuset
2836 2821 0:265 / /dev/mqueue rw,nosuid,nodev,noexec,relatime - mqueue mqueue rw
2837 2819 0:261 / /tmp rw,relatime - tmpfs tmpfs rw
2838 2819 0:21 / /mnt/cgroups ro,nosuid,nodev,noexec master:6 - tmpfs tmpfs ro,mode=755
2839 2838 0:22 / /mnt/cgroups/systemd rw,nosuid,nodev,noexec,relatime master:7 - cgroup cgroup rw,xattr,release_agent=/usr/lib/systemd/systemd-cgroups-agent,name=systemd
2840 2838 0:24 / /mnt/cgroups/blkio rw,nosuid,nodev,noexec,relatime master:11 - cgroup cgroup rw,blkio
2841 2838 0:25 / /mnt/cgroups/hugetlb rw,nosuid,nodev,noexec,relatime master:12 - cgroup cgroup rw,hugetlb
2842 2838 0:26 / /mnt/cgroups/perf_event rw,nosuid,nodev,noexec,relatime master:13 - cgroup cgroup rw,perf_event
2843 2838 0:27 / /mnt/cgroups/freezer rw,nosuid,nodev,noexec,relatime master:14 - cgroup cgroup rw,freezer
2844 2838 0:28 / /mnt/cgroups/pids rw,nosuid,nodev,noexec,relatime master:15 - cgroup cgroup rw,pids
2845 2838 0:29 / /mnt/cgroups/net_cls,net_prio rw,nosuid,nodev,noexec,relatime master:16 - cgroup cgroup rw,net_cls,net_prio
2846 2838 0:30 / /mnt/cgroups/memory rw,nosuid,nodev,noexec,relatime master:17 - cgroup cgroup rw,memory
2847 2838 0:31 / /mnt/cgroups/cpu,cpuacct rw,nosuid,nodev,noexec,relatime master:18 - cgroup cgroup rw,cpu,cpuacct
2848 2838 0:32 / /mnt/cgroups/devices rw,nosuid,nodev,noexec,relatime master:19 - cgroup cgroup rw,devices
2849 2838 0:33 / /mnt/cgroups/cpuset rw,nosuid,nodev,noexec,relatime master:20 - cgroup cgroup rw,cpuset
2850 2821 253:1 /var/lib/kubelet/pods/d05ceb29-4d8b-4c43-9eaa-d7acddc25247/containers/quickset/a053e3fe /dev/termination-log rw,relatime - ext4 /dev/vda1 rw
2851 2819 253:1 /var/lib/kubelet/pods/d05ceb29-4d8b-4c43-9eaa-d7acddc25247/volumes/kubernetes.io~configmap/node-cfg /etc/configs ro,relatime - ext4 /dev/vda1 rw
2852 2819 253:1 /var/lib/docker/containers/cb46c9f8f0ea80a1eb613fdd3e90b523939114575b6ee3e7e8bc1f4f0c8d0254/resolv.conf /etc/resolv.conf ro,relatime - ext4 /dev/vda1 rw
2853 2819 253:1 /var/lib/docker/containers/cb46c9f8f0ea80a1eb613fdd3e90b523939114575b6ee3e7e8bc1f4f0c8d0254/hostname /etc/hostname ro,relatime - ext4 /dev/vda1 rw
2854 2819 253:1 /var/lib/kubelet/pods/d05ceb29-4d8b-4c43-9eaa-d7acddc25247/etc-hosts /etc/hosts rw,relatime - ext4 /dev/vda1 rw
2855 2821 0:264 / /dev/shm rw,nosuid,nodev,noexec,relatime - tmpfs shm rw,size=65536k
2856 2819 0:262 / /run/secrets/kubernetes.io/serviceaccount ro,relatime - tmpfs tmpfs rw
2059 2820 0:279 /asound /proc/asound ro,relatime - proc proc rw
2060 2820 0:279 /bus /proc/bus ro,relatime - proc proc rw
2061 2820 0:279 /fs /proc/fs ro,relatime - proc proc rw
2062 2820 0:279 /irq /proc/irq ro,relatime - proc proc rw
2063 2820 0:279 /sys /proc/sys ro,relatime - proc proc rw
2066 2820 0:279 /sysrq-trigger /proc/sysrq-trigger ro,relatime - proc proc rw
2067 2820 0:338 / /proc/acpi ro,relatime - tmpfs tmpfs ro
2068 2820 0:280 /null /proc/kcore rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
2069 2820 0:280 /null /proc/keys rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
2074 2820 0:280 /null /proc/timer_list rw,nosuid - tmpfs tmpfs rw,size=65536k,mode=755
2075 2820 0:339 / /proc/scsi ro,relatime - tmpfs tmpfs ro
2076 2823 0:340 / /sys/firmware ro,relatime - tmpfs tmpfs ro
`

	mi, miErr := getCGroupMountsFromMountinfo(minikubeMI)
	require.NoError(t, miErr)
	const podSubGrp = "/kubepods/podd05ceb29-4d8b-4c43-9eaa-d7acddc25247/db332e7610fcb7c5a4d9eaa782285e61e49fa5c8403d756ea8ae2cffc99dc448"
	assert.Equal(t, []Mount{{
		Mountpoint: "/sys/fs/cgroup/systemd",
		Root:       podSubGrp,
		Subsystems: []string{"xattr", "release_agent=/usr/lib/systemd/systemd-cgroups-agent", "name=systemd"},
	}, {
		Mountpoint: "/sys/fs/cgroup/blkio",
		Root:       podSubGrp,
		Subsystems: []string{"blkio"},
	}, {
		Mountpoint: "/sys/fs/cgroup/hugetlb",
		Root:       podSubGrp,
		Subsystems: []string{"hugetlb"},
	}, {
		Mountpoint: "/sys/fs/cgroup/perf_event",
		Root:       podSubGrp,
		Subsystems: []string{"perf_event"},
	}, {
		Mountpoint: "/sys/fs/cgroup/freezer",
		Root:       podSubGrp,
		Subsystems: []string{"freezer"},
	}, {
		Mountpoint: "/sys/fs/cgroup/pids",
		Root:       podSubGrp,
		Subsystems: []string{"pids"},
	}, {
		Mountpoint: "/sys/fs/cgroup/net_cls,net_prio",
		Root:       podSubGrp,
		Subsystems: []string{"net_cls", "net_prio"},
	}, {
		Mountpoint: "/sys/fs/cgroup/memory",
		Root:       podSubGrp,
		Subsystems: []string{"memory"},
	}, {
		Mountpoint: "/sys/fs/cgroup/cpu,cpuacct",
		Root:       podSubGrp,
		Subsystems: []string{"cpu", "cpuacct"},
	}, {
		Mountpoint: "/sys/fs/cgroup/devices",
		Root:       podSubGrp,
		Subsystems: []string{"devices"},
	}, {
		Mountpoint: "/sys/fs/cgroup/cpuset",
		Root:       podSubGrp,
		Subsystems: []string{"cpuset"},
	}, {
		Mountpoint: "/mnt/cgroups/systemd",
		Root:       "/",
		Subsystems: []string{"xattr", "release_agent=/usr/lib/systemd/systemd-cgroups-agent", "name=systemd"},
	}, {
		Mountpoint: "/mnt/cgroups/blkio",
		Root:       "/",
		Subsystems: []string{"blkio"},
	}, {
		Mountpoint: "/mnt/cgroups/hugetlb",
		Root:       "/",
		Subsystems: []string{"hugetlb"},
	}, {
		Mountpoint: "/mnt/cgroups/perf_event",
		Root:       "/",
		Subsystems: []string{"perf_event"},
	}, {
		Mountpoint: "/mnt/cgroups/freezer",
		Root:       "/",
		Subsystems: []string{"freezer"},
	}, {
		Mountpoint: "/mnt/cgroups/pids",
		Root:       "/",
		Subsystems: []string{"pids"},
	}, {
		Mountpoint: "/mnt/cgroups/net_cls,net_prio",
		Root:       "/",
		Subsystems: []string{"net_cls", "net_prio"},
	}, {
		Mountpoint: "/mnt/cgroups/memory",
		Root:       "/",
		Subsystems: []string{"memory"},
	}, {
		Mountpoint: "/mnt/cgroups/cpu,cpuacct",
		Root:       "/",
		Subsystems: []string{"cpu", "cpuacct"},
	}, {
		Mountpoint: "/mnt/cgroups/devices",
		Root:       "/",
		Subsystems: []string{"devices"},
	}, {
		Mountpoint: "/mnt/cgroups/cpuset",
		Root:       "/",
		Subsystems: []string{"cpuset"},
	},
	}, mi)
}
