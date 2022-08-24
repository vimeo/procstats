//go:build darwin && cgo
// +build darwin,cgo

package procstats

// @see https://github.com/apple/darwin-xnu/blob/master/bsd/sys/proc_info.h
// also @see http://vinceyuan.github.io/wrong-info-from-procpidinfo/

// #include <libproc.h>
//
// int get_mem_info(int pid, uint64_t *rss)
// {
//     struct proc_taskallinfo ti;
//     int nb = 0;
//     nb = proc_pidinfo(pid, PROC_PIDTASKALLINFO, 0, &ti, sizeof(ti));
//     if (nb <= 0 || nb < sizeof(ti)) {
//         return -1;
//     }
//     *rss = ti.ptinfo.pti_resident_size;
//     return 0;
// }
//
// int get_cpu_info(int pid, uint64_t *total_user, uint64_t *total_system)
// {
//     struct proc_taskallinfo ti;
//     int nb = 0;
//     nb = proc_pidinfo(pid, PROC_PIDTASKALLINFO, 0, &ti, sizeof(ti));
//     if (nb <= 0 || nb < sizeof(ti)) {
//         return -1;
//     }
//     *total_user = ti.ptinfo.pti_total_user;
//     *total_system = ti.ptinfo.pti_total_system;
//     return 0;
// }
import "C"

import (
	"fmt"
	"time"
)

func readProcessRSS(pid int) (int64, error) {
	var rss C.ulonglong
	success := C.int(0)
	ret := C.get_mem_info(C.int(pid), &rss)
	if ret != success {
		return 0, fmt.Errorf("failed to get mem stats for pid: non-zero return")
	}
	return int64(rss), nil
}

func readProcessCPUTime(pid int) (CPUTime, error) {
	var totalUser C.ulonglong
	var totalSystem C.ulonglong
	success := C.int(0)

	ret := C.get_cpu_info(C.int(pid), &totalUser, &totalSystem)
	if ret != success {
		return CPUTime{},
			fmt.Errorf("failed to get cpu stats for pid: non-zero return")
	}
	clockTick := time.Duration(sysClockTick())

	cpuTime := CPUTime{}
	cpuTime.Utime = time.Duration(uint64(totalUser)) * time.Second / clockTick
	cpuTime.Stime = time.Duration(uint64(totalSystem)) * time.Second / clockTick

	return cpuTime, nil
}

func readMaxRSS(pid int) (int64, error) {
	// darwin doesn't appear to expose Max RSS independently
	return readProcessRSS(pid)
}

func resetMaxRSS(pid int) error {
	// noop
	return nil
}
