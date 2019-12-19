// Package procstats attempts to provide process stats for a given PID.
package procstats

import (
	"errors"
	"time"
)

// ErrUnimplementedPlatform indicates that this request is not implemented for
// this specific platform.
var ErrUnimplementedPlatform = errors.New("Unimplemented for this platform")

// RSS takes a pid and returns the RSS of that process (or an error)
// This may return ErrUnimplementedPlatform on non-linux platforms.
func RSS(pid int) (int64, error) {
	return readProcessRSS(pid)
}

// CPUTime contains the user and system time consumed by a process.
type CPUTime struct {
	Utime time.Duration
	Stime time.Duration
}

// Sub subtracts the operand from the receiver, returning a new CPUTime object.
func (c *CPUTime) Sub(other *CPUTime) CPUTime {
	return CPUTime{
		Utime: c.Utime - other.Utime,
		Stime: c.Stime - other.Stime,
	}
}

// Add subtracts the operand from the receiver, returning a new CPUTime object.
func (c *CPUTime) Add(other *CPUTime) CPUTime {
	return CPUTime{
		Utime: c.Utime + other.Utime,
		Stime: c.Stime + other.Stime,
	}
}

// ProcessCPUTime returns either the cumulative CPUTime of the specified
// process or an error.
func ProcessCPUTime(pid int) (CPUTime, error) {
	return readProcessCPUTime(pid)
}

// Eq reports if the two CPUTimes are equal.
func (c *CPUTime) eq(b *CPUTime) bool {
	return c.Utime == b.Utime &&
		c.Stime == b.Stime
}

// MaxRSS returns the maximum RSS (High Water Mark) of the process with PID
// pid.
func MaxRSS(pid int) (int64, error) {
	return readMaxRSS(pid)
}

// ResetMaxRSS returns the maximum RSS (High Water Mark) of the process with PID
// pid.
func ResetMaxRSS(pid int) error {
	return resetMaxRSS(pid)
}
