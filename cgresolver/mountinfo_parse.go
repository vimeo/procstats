package cgresolver

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Mount represents a cgroup or cgroup2 mount.
// Subsystems will be nil if the mount is for a unified hierarchy/cgroup v2
// in that case, CGroupV2 will be true.
type Mount struct {
	Mountpoint string
	Root       string
	Subsystems []string
	CGroupV2   bool // true if this is a cgroup2 mount
}

const (
	mountinfoPath = "/proc/self/mountinfo"
)

// CGroupMountInfo parses /proc/self/mountinfo and returns info about all cgroup and cgroup2 mounts
func CGroupMountInfo() ([]Mount, error) {
	mountinfoContents, mntInfoReadErr := os.ReadFile(mountinfoPath)
	if mntInfoReadErr != nil {
		return nil, fmt.Errorf("failed to read contents of %s: %w",
			mountinfoPath, mntInfoReadErr)
	}

	mounts, mntsErr := getCGroupMountsFromMountinfo(string(mountinfoContents))
	if mntsErr != nil {
		return nil, fmt.Errorf("failed to list cgroupfs mounts: %w", mntsErr)
	}

	return mounts, nil
}

func getCGroupMountsFromMountinfo(mountinfo string) ([]Mount, error) {
	// mountinfo is line-delimited, then space-delimited
	mountinfoLines := strings.Split(mountinfo, "\n")
	if len(mountinfoLines) == 0 {
		return nil, fmt.Errorf("unexpectedly empty mountinfo (one line): %q", mountinfo)
	}
	out := make([]Mount, 0, len(mountinfoLines))
	for _, line := range mountinfoLines {
		if len(line) == 0 {
			continue
		}
		sections := strings.SplitN(line, " - ", 2)
		if len(sections) < 2 {
			return nil, fmt.Errorf("missing section separator in line %q", line)
		}
		s2Fields := strings.SplitN(sections[1], " ", 3)
		if len(s2Fields) < 3 {
			return nil, fmt.Errorf("line %q contains %d fields in second section, expected 3",
				line, len(s2Fields))

		}
		isCG2 := false
		switch s2Fields[0] {
		case "cgroup":
			isCG2 = false
		case "cgroup2":
			isCG2 = true
		default:
			// skip anything that's not a cgroup
			continue
		}
		s1Fields := strings.Split(sections[0], " ")
		if len(s1Fields) < 5 {
			return nil, fmt.Errorf("too few fields in line %q before optional separator: %d; expected 5",
				line, len(s1Fields))
		}
		mntpnt, mntPntUnescapeErr := unOctalEscape(s1Fields[4])
		if mntPntUnescapeErr != nil {
			return nil, fmt.Errorf("failed to unescape mountpoint %q: %w", s1Fields[4], mntPntUnescapeErr)
		}
		rootPath, rootUnescErr := unOctalEscape(s1Fields[3])
		if rootUnescErr != nil {
			return nil, fmt.Errorf("failed to unescape mount root %q: %w", s1Fields[3], rootUnescErr)
		}
		mnt := Mount{
			CGroupV2:   isCG2,
			Mountpoint: mntpnt,
			Root:       rootPath,
			Subsystems: nil,
		}
		// only bother with the mount options to find subsystems if cgroup v1
		if !isCG2 {
			for _, mntOpt := range strings.Split(s2Fields[2], ",") {
				switch mntOpt {
				case "ro", "rw":
					// These mount options are lies, (or at least
					// only reflect the original mount, without
					// considering the layering of later bind-mounts)
					continue
				case "":
					continue
				default:
					mnt.Subsystems = append(mnt.Subsystems, mntOpt)
				}
			}
		}

		out = append(out, mnt)

	}
	return out, nil
}

func unOctalEscape(str string) (string, error) {
	b := strings.Builder{}
	b.Grow(len(str))
	for {
		backslashIdx := strings.IndexByte(str, byte('\\'))
		if backslashIdx == -1 {
			b.WriteString(str)
			return b.String(), nil
		}
		b.WriteString(str[:backslashIdx])
		// if the end of the escape is beyond the end of the string, abort!
		if backslashIdx+3 >= len(str) {
			return "", fmt.Errorf("invalid offset: %d+3 >= len %d", backslashIdx, len(str))
		}
		// slice out the octal 3-digit component
		esc := str[backslashIdx+1 : backslashIdx+4]
		asciiVal, parseUintErr := strconv.ParseUint(esc, 8, 8)
		if parseUintErr != nil {
			return "", fmt.Errorf("failed to parse escape value %q: %w", esc, parseUintErr)
		}
		b.WriteByte(byte(asciiVal))
		str = str[backslashIdx+4:]
	}

}
