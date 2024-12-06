// package cgresolver contains helpers and types for resolving the CGroup associated with specific subsystems
// If you don't know what cgroup subsystems are, you probably want one of the higher-level interfaces in the parent package.
package cgresolver

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
)

// CGMode is an enum indicating which cgroup type is active for the returned controller
type CGMode uint8

const (
	CGModeUnknown CGMode = iota
	// CGroup V1
	CGModeV1
	// CGroup V2
	CGModeV2
)

func cgroup2Mode(iscg2 bool) CGMode {
	if iscg2 {
		return CGModeV2
	}
	return CGModeV1
}

// CGroupPath includes information about a cgroup.
type CGroupPath struct {
	AbsPath   string
	MountPath string
	Mode      CGMode
}

// Parent returns a CGroupPath for the parent directory as long as it wouldn't pass the root of the mountpoint.
// second return indicates whether a new path was returned.
func (c *CGroupPath) Parent() (CGroupPath, bool) {
	// Remove any trailing slash
	path := strings.TrimSuffix(c.AbsPath, string(os.PathSeparator))
	mnt := strings.TrimSuffix(c.MountPath, string(os.PathSeparator))
	if mnt == path {
		return CGroupPath{
			AbsPath:   path,
			MountPath: mnt,
			Mode:      c.Mode,
		}, false
	}
	lastSlashIdx := strings.LastIndexByte(path, byte(os.PathSeparator))
	if lastSlashIdx == -1 {
		// This shouldn't happen
		panic("invalid state: path \"" + path + "\" has no slashes and doesn't match the mountpoint")
	}
	return CGroupPath{
		AbsPath:   path[:lastSlashIdx],
		MountPath: mnt, // Strip any trailing slash in case one snuck in
		Mode:      c.Mode,
	}, true
}

// SelfSubsystemPath returns a CGroupPath for the cgroup associated with a specific subsystem for the current process.
func SelfSubsystemPath(subsystem string) (CGroupPath, error) {
	return subsystemPath("self", subsystem)
}

// PIDSubsystemPath returns a CGroupPath for the cgroup associated with a specific subsystem for the specified PID
func PIDSubsystemPath(pid int, subsystem string) (CGroupPath, error) {
	return subsystemPath(strconv.Itoa(pid), subsystem)
}

func subsystemPath(procSubDir string, subsystem string) (CGroupPath, error) {
	cgSubSyses, cgSubSysReadErr := ParseReadCGSubsystems()
	if cgSubSysReadErr != nil {
		return CGroupPath{}, fmt.Errorf("failed to resolve subsystems to hierarchies: %w", cgSubSysReadErr)
	}
	cgIdx := slices.IndexFunc(cgSubSyses, func(c CGroupSubsystem) bool {
		return c.Subsys == subsystem
	})
	if cgIdx == -1 {
		return CGroupPath{}, fmt.Errorf("no cgroup hierarchy associated with subsystem %q", subsystem)
	}
	cgHierID := cgSubSyses[cgIdx].Hierarchy

	procCGs, procCGsErr := resolveProcCGControllers(procSubDir)
	if procCGsErr != nil {
		return CGroupPath{}, fmt.Errorf("failed to resolve cgroup controllers: %w", procCGsErr)
	}

	procCGIdx := slices.IndexFunc(procCGs, func(cg CGProcHierarchy) bool { return cg.HierarchyID == cgHierID })
	if procCGIdx == -1 {
		return CGroupPath{}, fmt.Errorf("failed to resolve process cgroup controllers: %w", procCGsErr)
	}

	cgMountInfo, mountInfoParseErr := CGroupMountInfo()
	if mountInfoParseErr != nil {
		return CGroupPath{}, fmt.Errorf("failed to parse mountinfo: %w", mountInfoParseErr)
	}

	cgPath, cgPathErr := procCGs[procCGIdx].cgPath(cgMountInfo)
	if cgPathErr != nil {
		return CGroupPath{}, fmt.Errorf("failed to resolve filesystem path for cgroup %+v: %w", procCGs[procCGIdx], cgPathErr)
	}
	return cgPath, nil
}
