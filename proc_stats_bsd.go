// +build freebsd
// +build cgo

package procstats

// #include <sys/types.h>
// #include <sys/sysctl.h>
// #include <sys/user.h>
// #include <stdlib.h>
// int64_t ExtractRSSKinfoProc(void *stat_bytes) {
//   struct kinfo_proc *kp = (struct kinfo_proc*)stat_bytes;
// #ifdef DARWIN
//   int64_t rss = kp->kp_eproc.e_vm.vm_rssize;
// #else
//   int64_t rss = kp->ki_rssize * 4096;
// #endif
//   free(stat_bytes);
//   return rss;
// }
import "C"

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func readProcessStats(pid int) ([]byte, error) {
	statsEnc, err := unix.SysctlRaw("kern.proc.pid", pid)
	if err != nil {
		return nil, err
	}
	return statsEnc, nil
}

func readProcessRSS(pid int) (int64, error) {
	pstats, err := readProcessStats(pid)
	if err != nil {
		return 0, fmt.Errorf("failed to get stats for pid: %s", err)
	}
	cpstats := C.CBytes(pstats)
	return int64(C.ExtractRSSKinfoProc(cpstats)), nil
}

func readMaxRSS(pid int) (int64, error) {
	// bsd doesn't appear to expose Max RSS independently

	return readProcessRSS(pid)
}

func resetMaxRSS(pid int) error {
	// noop
	return nil
}
