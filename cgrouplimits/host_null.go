// +build !linux

package cgrouplimits

// HostMemStats returns the size of the machine.
func HostMemStats() (MemoryStats, error) {
	// TODO: add a darwin implementation
	return MemoryStats{}, ErrUnimplementedPlatform
}
