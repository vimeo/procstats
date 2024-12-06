//go:build linux && !cgo
// +build linux,!cgo

package procstats

func sysClockTick() int64 {
	// Reflecting the kernel default for USER_HZ
	const defaultClockTick = int64(100)
	// TODO(davidf): update the auxv value with key AT_CLKTCK (17).
	return defaultClockTick
}
