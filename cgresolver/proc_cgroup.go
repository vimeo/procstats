package cgresolver

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// CGroupV2HierarchyID is a convenience constant indicating the hierarchy ID for the V2 cgroup hierarchy
const CGroupV2HierarchyID = 0

// CGProcHierarchy describes a specific CGroup subsystem/controller/hierarchy and path for a parsed /proc/<pid>/cgroup
type CGProcHierarchy struct {
	HierarchyID   int      // 0 for v2; refs /proc/cgroups for v1
	SubsystemsCSV string   // empty for v2; set of controllers/subsystem names for this hierarchy (CSV)
	Subsystems    []string // set of v1 subsystems/controllers  (HierarchiesCSV split)
	Path          string   // path relative to mountpoint
}

func (c *CGProcHierarchy) cgPath(mountpoints []Mount) (CGroupPath, error) {
	for _, mp := range mountpoints {
		// Skip any mountpoints originating outside our cgroup namespace
		// From cgroup_namespaces(7):
		//  When reading the cgroup memberships of a "target" process from /proc/pid/cgroup,
		//  the pathname shown in the third field of each record will be relative to the
		//  reading process's root directory for the corresponding cgroup hierarchy.  If the
		//  cgroup directory of the target process lies outside the root directory of the
		//  reading process's cgroup namespace, then the pathname will show ../ entries for
		//  each ancestor level in the cgroup hierarchy.
		if strings.HasPrefix(mp.Root, "/..") {
			continue
		}
		if (mp.CGroupV2 && c.HierarchyID == CGroupV2HierarchyID) || slices.Equal(mp.Subsystems, c.Subsystems) {
			relCGPath, relErr := filepath.Rel(mp.Root, c.Path)
			if relErr != nil || strings.HasPrefix(relCGPath, "../") {
				// bind-mount for a different sub-tree of the cgroups v2 hierarchy
				continue
			}
			return CGroupPath{AbsPath: filepath.Join(mp.Mountpoint, relCGPath), MountPath: mp.Mountpoint, Mode: cgroup2Mode(mp.CGroupV2)}, nil
		}
	}
	return CGroupPath{}, fmt.Errorf("no usable mountpoints found for hierarchy %d and path %q (found %d cgroup/cgroup2 mounts)",
		c.HierarchyID, c.Path, len(mountpoints))
}

func parseProcPidCgroup(content []byte) ([]CGProcHierarchy, error) {
	lines := bytes.Split(bytes.TrimSpace(content), []byte("\n"))

	out := make([]CGProcHierarchy, 0, len(lines))

	// from cgroups(7):
	//        /proc/[pid]/cgroup (since Linux 2.6.24)
	//              This file describes control groups to which the process with  the  corresponding  PID  be‐
	//              longs.  The displayed information differs for cgroups version 1 and version 2 hierarchies.
	//
	//              For  each cgroup hierarchy of which the process is a member, there is one entry containing
	//              three colon-separated fields:
	//
	//                  hierarchy-ID:controller-list:cgroup-path
	//
	//              For example:
	//
	//                  5:cpuacct,cpu,cpuset:/daemons
	//
	//              The colon-separated fields are, from left to right:
	//
	//              [1]  For cgroups version 1 hierarchies, this field contains a unique hierarchy  ID  number
	//                   that  can  be  matched to a hierarchy ID in /proc/cgroups.  For the cgroups version 2
	//                   hierarchy, this field contains the value 0.
	//
	//              [2]  For cgroups version 1 hierarchies, this field contains a comma-separated list of  the
	//                   controllers  bound to the hierarchy.  For the cgroups version 2 hierarchy, this field
	//                   is empty.
	//
	//              [3]  This field contains the pathname of the control group in the hierarchy to  which  the
	//                   process belongs.  This pathname is relative to the mount point of the hierarchy.

	for i, line := range lines {
		if len(line) == 0 {
			// skip empty lines
			continue
		}
		parts := bytes.SplitN(line, []byte(":"), 3)
		if len(parts) != 3 {
			return nil, fmt.Errorf("line %d (%q) has incorrect number of parts: %d; expected %d", i, line, len(parts), 3)
		}
		hID, hIDErr := strconv.Atoi(string(parts[0]))
		if hIDErr != nil {
			return nil, fmt.Errorf("line %d has non-integer hierarchy ID (%q): %w", i, string(parts[0]), hIDErr)
		}
		ss := strings.Split(string(parts[1]), ",")
		if len(ss) == 1 && ss[0] == "" {
			ss = []string{}
		}
		out = append(out, CGProcHierarchy{
			HierarchyID:   hID,
			SubsystemsCSV: string(parts[1]),
			Path:          string(parts[2]),
			Subsystems:    ss,
		})
	}
	return out, nil
}

func resolveProcCGControllers(pid string) ([]CGProcHierarchy, error) {
	cgPath := filepath.Join("/proc", pid, "cgroup")
	cgContents, readErr := os.ReadFile(cgPath)
	if readErr != nil {
		return nil, fmt.Errorf("failed to read %q: %w", cgPath, readErr)
	}

	return parseProcPidCgroup(cgContents)
}

// SelfCGSubsystems returns information about all the controllers associated with the current process
func SelfCGSubsystems() ([]CGProcHierarchy, error) {
	return resolveProcCGControllers("self")
}

// PidCGSubsystems returns information about all the CGroup controllers associated with the passed pid
func PidCGSubsystems(pid int) ([]CGProcHierarchy, error) {
	return resolveProcCGControllers(strconv.Itoa(pid))
}

// ErrMissingCG2Mount indicates a missing cgroup v2 mount when resolving which controllers belong to which hierarchy
var ErrMissingCG2Mount = errors.New("cgroup2 mount covering relevant cgroup(s) not present in the current mount namespace, but cgroupv2 controller present in /proc/<pid>/cgroup")

// CGroupV2QuasiSubsystemName is a constant used by MapSubsystems to refer to
// the cgroup2 hierarchy (since the /proc/<pid>/cgroup file lacks subsystems
// for cgroup2)
const CGroupV2QuasiSubsystemName = "cgroup2 unified hierarchy"

// MapSubsystems creates a map from the controller-name to the entry in the passed slice
func MapSubsystems(controllers []CGProcHierarchy) map[string]*CGProcHierarchy {
	out := make(map[string]*CGProcHierarchy, len(controllers))
	for i, hier := range controllers {
		for _, ctrlr := range hier.Subsystems {
			out[ctrlr] = &controllers[i]
		}
		if hier.HierarchyID == CGroupV2HierarchyID {
			out[CGroupV2QuasiSubsystemName] = &controllers[i]
		}
	}
	return out
}

// CGroupSubsystem models a row in /proc/cgroups
type CGroupSubsystem struct {
	Subsys     string // name of the subsystem
	Hierarchy  int    // hierarchy ID number (0 for cgroup2)
	NumCGroups int    // number of cgroups in that hierarchy using this controller
	Enabled    bool   // controller enabled?
}

// ParseReadCGSubsystems reads the /proc/cgroups pseudofile, and returns a slice of subsystem info, including which hierarchies each belongs to.
func ParseReadCGSubsystems() ([]CGroupSubsystem, error) {
	procCG, procCGErr := os.ReadFile("/proc/cgroups")
	if procCGErr != nil {
		return nil, fmt.Errorf("failed to read /proc/cgroups: %w", procCGErr)
	}
	return parseCGSubsystems(string(procCG))
}

func parseCGSubsystems(procCgroups string) ([]CGroupSubsystem, error) {
	lines := strings.Split(procCgroups, "\n")
	headers := strings.Fields(strings.TrimLeft(lines[0], "#"))
	if len(headers) < 2 {
		return nil, fmt.Errorf("insufficient fields %d; need at least %d (expected 4)", len(headers), 2)
	}
	// Fast-common-path which should always hit if the number of columns doesn't change
	extractRow := func(vals []string) (CGroupSubsystem, error) {
		if len(vals) != 4 {
			return CGroupSubsystem{}, fmt.Errorf("unexpected number of columns %d (doesn't match headers); expected %d", len(vals), 4)
		}
		hierNum, hierParseErr := strconv.Atoi(vals[1])
		if hierParseErr != nil {
			return CGroupSubsystem{}, fmt.Errorf("unable to parse hierarchy number: %q: %w", vals[1], hierParseErr)
		}
		numCG, numCGParseErr := strconv.Atoi(vals[2])
		if numCGParseErr != nil {
			return CGroupSubsystem{}, fmt.Errorf("unable to parse cgroup count: %q: %w", vals[2], numCGParseErr)
		}
		enabled, enabledParseErr := strconv.ParseBool(vals[3])
		if enabledParseErr != nil {
			return CGroupSubsystem{}, fmt.Errorf("unable to parse cgroup enabled: %q: %w", vals[3], enabledParseErr)
		}
		return CGroupSubsystem{
			Subsys:     vals[0],
			Hierarchy:  hierNum,
			NumCGroups: numCG,
			Enabled:    enabled,
		}, nil
	}
	const noCol = -1 // constant to designate missing columns
	expCols := [...]string{"subsys_name", "hierarchy", "num_cgroups", "enabled"}
	if !slices.Equal(expCols[:], headers) {
		subsysCol := noCol
		hierCol := noCol
		nCGCol := noCol
		enabledCol := noCol

		// The list and/or order of columns changed, so we need to remap them.
		// we do, however have a minimum of just subsys_name and hierarchy columns
		for i, colHead := range headers {
			switch strings.ToLower(colHead) {
			case "subsys_name":
				if subsysCol != noCol {
					return nil, fmt.Errorf("multiple subsys_name columns at index %d and %d", subsysCol, i)
				}
				subsysCol = i
			case "hierarchy":
				if hierCol != noCol {
					return nil, fmt.Errorf("multiple hierarchy columns at index %d and %d", hierCol, i)
				}
				hierCol = i
			case "num_cgroups":
				if nCGCol != noCol {
					return nil, fmt.Errorf("multiple num_cgroups columns at index %d and %d", nCGCol, i)
				}
				nCGCol = i
			case "enabled":
				if enabledCol != noCol {
					return nil, fmt.Errorf("multiple enabled columns at index %d and %d", enabledCol, i)
				}
				enabledCol = i
			}
			// let unknown columns fall through
		}
		if subsysCol == noCol || hierCol == noCol {
			return nil, fmt.Errorf("missing critical column subsystem_name %t or hierarchy %t; columns: %q", subsysCol == noCol, hierCol == noCol, headers)
		}
		extractRow = func(vals []string) (CGroupSubsystem, error) {
			if len(vals) != len(headers) {
				return CGroupSubsystem{}, fmt.Errorf("unexpected number of columns %d (doesn't match headers); expected %d", len(vals), len(headers))
			}
			hierNum, hierParseErr := strconv.Atoi(vals[hierCol])
			if hierParseErr != nil {
				return CGroupSubsystem{}, fmt.Errorf("unable to parse hierarchy number: %q: %w", vals[hierCol], hierParseErr)
			}
			rowOut := CGroupSubsystem{
				Subsys:     vals[subsysCol],
				Hierarchy:  hierNum,
				NumCGroups: 0,
				Enabled:    true, // default to true, so we consider anything that's listed as enabled if that column disappears
			}
			if nCGCol != noCol {
				numCG, numCGParseErr := strconv.Atoi(vals[nCGCol])
				if numCGParseErr != nil {
					return CGroupSubsystem{}, fmt.Errorf("unable to parse cgroup count: %q: %w", vals[nCGCol], numCGParseErr)
				}
				rowOut.NumCGroups = numCG
			}
			if enabledCol != noCol {
				enabled, enabledParseErr := strconv.ParseBool(vals[enabledCol])
				if enabledParseErr != nil {
					return CGroupSubsystem{}, fmt.Errorf("unable to parse cgroup enabled: %q: %w", vals[enabledCol], enabledParseErr)
				}
				rowOut.Enabled = enabled
			}
			return rowOut, nil
		}
	}

	out := make([]CGroupSubsystem, 0, len(lines)-1)
	for i, line := range lines[1:] {
		if len(line) == 0 {
			// skip empty lines (probably trailing)
			continue
		}
		lineVals := strings.Fields(line)
		extractedLine, extLineErr := extractRow(lineVals)
		if extLineErr != nil {
			return nil, fmt.Errorf("failed to parse line %d: %w", i+1, extLineErr)
		}
		out = append(out, extractedLine)
	}

	return out, nil
}
