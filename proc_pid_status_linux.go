//go:build linux
// +build linux

package procstats

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/vimeo/procstats/pparser"
)

// ProcPidStatus represents the contents of /proc/$PID/status, and is intended
// to be parsed by the pparser subpackage.
type ProcPidStatus struct {
	Name                     string
	Umask                    uint8
	State                    string
	Tgid                     uint64
	Ngid                     uint64
	Pid                      uint64
	PPid                     uint64
	TracerPid                uint64
	UID                      string `pparser:"Uid"`
	GID                      string `pparser:"Gid"`
	FDSize                   int64
	Groups                   string
	NStgid                   uint64
	NSpid                    uint64
	NSpgid                   uint64
	NSsid                    uint64
	VMPeak                   int64 `pparser:"VmPeak"`
	VMSize                   int64 `pparser:"VmSize"`
	VMLck                    int64 `pparser:"VmLck"`
	VMPin                    int64 `pparser:"VmPin"`
	VMHWM                    int64 `pparser:"VmHWM"`
	VMRSS                    int64 `pparser:"VmRSS"`
	RssAnon                  int64
	RssFile                  int64
	RssShmem                 int64
	VMData                   int64 `pparser:"VmData"`
	VMStk                    int64 `pparser:"VmStk"`
	VMExe                    int64 `pparser:"VmExe"`
	VMLib                    int64 `pparser:"VmLib"`
	VMPTE                    int64 `pparser:"VmPTE"`
	VMSwap                   int64 `pparser:"VmSwap"`
	HugetlbPages             int64
	CoreDumping              int64
	Threads                  int64
	SigQ                     string
	SigPnd                   string
	ShdPnd                   string
	SigBlk                   string
	SigIgn                   string
	SigCgt                   string
	CapInh                   string
	CapPrm                   string
	CapEff                   string
	CapBnd                   string
	CapAmb                   string
	NoNewPrivs               string
	Seccomp                  string
	SpeculationStoreBypass   string            `pparser:"Speculation_Store_Bypass"`
	CpusAllowed              string            `pparser:"Cpus_allowed"`
	CpusAllowedList          string            `pparser:"Cpus_allowed_list"`
	MemsAllowed              string            `pparser:"Mems_allowed"`
	MemsAllowedList          string            `pparser:"Mems_allowed_list"`
	VoluntaryCtxtSwitches    int64             `pparser:"voluntary_ctxt_switches"`
	NonvoluntaryCtxtSwitches int64             `pparser:"nonvoluntary_ctxt_switches"`
	UnknownFields            map[string]string `pparser:"skip,unknown"`
}

var procPidStatusParser = pparser.NewLineKVFileParser(ProcPidStatus{}, ":")

// ReadProcStatus reads the /proc/$pid/status for the specified pid and returns
// a ProcPidStatus.
// Note: this only works under linux, and is not available on other platforms.
// Portable applications should use the higher-level wrappers in this package
// (ProcessCPUTime, MaxRSS, and RSS) rather than the low-level.
func ReadProcStatus(pid int) (*ProcPidStatus, error) {
	statusPath := filepath.Join("/proc", strconv.Itoa(pid), "status")
	contents, err := os.ReadFile(statusPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %q: %s",
			statusPath, err)
	}
	out := ProcPidStatus{}
	if parseErr := procPidStatusParser.Parse(contents, &out); parseErr != nil {
		return nil, fmt.Errorf("failed to parse contents of %q: %s",
			statusPath, parseErr)
	}

	return &out, nil

}

func readMaxRSS(pid int) (int64, error) {
	status, err := ReadProcStatus(pid)
	if err != nil {
		return -1, fmt.Errorf("failed to obtain status: %s", err)
	}
	return status.VMHWM, nil
}

func resetMaxRSS(pid int) error {
	refsPath := filepath.Join("/proc", strconv.Itoa(pid), "clear_refs")
	// From the proc(5) manpage:
	//
	//      This is a write-only file, writable only by owner of the process.

	//      The following values may be written to the file:
	// ...
	//             5 (since Linux 4.0)
	//	                           Reset the peak resident set size
	//	                           ("high water mark") to the process's
	//	                           current resident set size value.

	// As such, write the value "5" to /proc/$PID/clear_refs to reset the VmHWM value.
	return os.WriteFile(refsPath, []byte{'5'}, 0)
}
