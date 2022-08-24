//go:build linux
// +build linux

package procstats

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func init() {
	const self = "/proc/self/stat"
	b, err := os.ReadFile(self)
	if err != nil {
		panic(fmt.Sprintf("unexpected kernel shenanigans: %v", err))
	}
	if _, err := linuxParseCPUTime(b); err != nil {
		panic(fmt.Sprintf("unexpected kernel shenanigans: %v", err))
	}

	// This burns extra cycles at the beginning of our execution, but allows
	// us to know the kernel we're running on is actually reporting these numbers.
	// Any binaries will almost certainly not be run on the same kernel they were
	// compiled and tested on, so just running at test-time is of limited use.
	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()
		var ct *CPUTime
		for i := 0; i < 30 || (*ct == CPUTime{}); i++ {
			<-t.C
			b, err := os.ReadFile(self)
			if err != nil {
				panic(fmt.Sprintf("unexpected kernel shenanigans: %v", err))
			}
			r, err := linuxParseCPUTime(b)
			if err != nil {
				panic(fmt.Sprintf("unexpected kernel shenanigans: %v", err))
			}
			ct = &r
		}
	}()
}

func procFileName(pid int, leafName string) string {
	return filepath.Join("/proc", strconv.Itoa(pid), leafName)
}

func procFileContents(pid int, leafName string) ([]byte, error) {
	fn := procFileName(pid, leafName)
	contents, err := os.ReadFile(fn)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s with error: %s", leafName, err)
	}
	return contents, nil
}

// From the proc(5) manpage:
// /proc/[pid]/statm
//        Provides information about memory usage, measured in pages.  The columns are:
//
//            size       (1) total program size
//                       (same as VmSize in /proc/[pid]/status)
//            resident   (2) resident set size
//                       (same as VmRSS in /proc/[pid]/status)
//            shared     (3) number of resident shared pages (i.e., backed by a file)
//                       (same as RssFile+RssShmem in /proc/[pid]/status)
//            text       (4) text (code)
//            lib        (5) library (unused since Linux 2.6; always 0)
//            data       (6) data + stack
//            dt         (7) dirty pages (unused since Linux 2.6; always 0)

func readProcessRSS(pid int) (int64, error) {
	statmContents, readErr := procFileContents(pid, "statm")
	if readErr != nil {
		return 0, fmt.Errorf("failed to get memory usage: %s", readErr)
	}

	// statm's field values are listed in units of pages, so get that
	// value.
	sysPagesize := os.Getpagesize()

	statmFields := strings.SplitN(string(statmContents), " ", 7)
	if len(statmFields) < 3 {
		return 0, fmt.Errorf("unexpected number of fields present in statm: %d",
			len(statmFields))
	}

	rssPages, err := strconv.ParseInt(statmFields[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse the second column of statm: %s",
			err)
	}
	return int64(sysPagesize) * rssPages, nil
}

// excerpt from proc(5) man page section on /proc/[pid]/stat:

//               (14) utime  %lu
//                         Amount of time that this process has been scheduled
//                         in user mode, measured in clock ticks (divide by
//                         sysconf(_SC_CLK_TCK)).   This  includes  guest
//                         time,  guest_time
//                         (time spent running a virtual CPU, see below), so
//                         that applications that are not aware of the guest
//                         time field do not lose that time from their
//                         calculations.
//
//               (15) stime  %lu
//                         Amount of time that this process has been scheduled
//                         in kernel mode, measured in clock ticks (divide by
//                         sysconf(_SC_CLK_TCK)).
//
//
//               (16) cutime  %ld
//                         Amount of time that this process's waited-for children
//                         have  been  scheduled  in user mode, measured in clock
//                         ticks (divide  by  sysconf(_SC_CLK_TCK)).   (See  also
//                         times(2).)   This  includes  guest  time,  cguest_time
//                         (time spent running a virtual CPU, see below).
//
//               (17) cstime  %ld
//                         Amount of time that this process's waited-for children
//                         have  been scheduled in kernel mode, measured in clock
//                         ticks (divide by sysconf(_SC_CLK_TCK)).

func readProcessCPUTime(pid int) (CPUTime, error) {
	c, err := procFileContents(pid, "stat")
	if err != nil {
		return CPUTime{}, fmt.Errorf("failed to get CPU time: %s", err)
	}
	return linuxParseCPUTime(c)
}

func linuxParseCPUTime(b []byte) (r CPUTime, err error) {
	statFields := bytes.SplitN(b, []byte{' '}, 18)
	if len(statFields) < 17 {
		return r, fmt.Errorf("insufficient fields present in stat: %d",
			len(statFields))
	}
	utimeTicks, err := strconv.ParseInt(string(statFields[13]), 10, 64)
	if err != nil {
		return r, fmt.Errorf("failed to parse the utime column of stat: %s",
			err)
	}
	stimeTicks, err := strconv.ParseInt(string(statFields[14]), 10, 64)
	if err != nil {
		return r, fmt.Errorf("failed to parse the stime column of stat: %s",
			err)
	}

	// we use cutime and cstime here to include child process CPU usage (as
	// long as those child processes have been wait(2)ed on).
	cutimeTicks, err := strconv.ParseInt(string(statFields[15]), 10, 64)
	if err != nil {
		return r, fmt.Errorf("failed to parse the cutime column of stat: %s",
			err)
	}
	cstimeTicks, err := strconv.ParseInt(string(statFields[16]), 10, 64)
	if err != nil {
		return r, fmt.Errorf("failed to parse the cstime column of stat: %s",
			err)
	}
	clockTick := time.Duration(sysClockTick())
	r.Utime = time.Duration(utimeTicks+cutimeTicks) * time.Second / clockTick
	r.Stime = time.Duration(stimeTicks+cstimeTicks) * time.Second / clockTick
	return r, nil
}
