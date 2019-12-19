// +build !linux,!cgo

package procstats

func readProcessRSS(pid int) (int64, error) {
	return 0, ErrUnimplementedPlatform
}
func readProcessStats(pid int) ([]byte, error) {
	return nil, ErrUnimplementedPlatform
}

func readMaxRSS(pid int) (int64, error) {
	// bsd doesn't appear to expose Max RSS independently

	return 0, ErrUnimplementedPlatform
}

func resetMaxRSS(pid int) error {
	// noop
	return ErrUnimplementedPlatform
}
