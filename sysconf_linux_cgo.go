//go:build linux && cgo
// +build linux,cgo

package procstats

// #include <unistd.h>
import "C"

func sysClockTick() int64 {
	return int64(C.sysconf(C._SC_CLK_TCK))
}
