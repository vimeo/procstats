package cgrouplimits

import (
	"math"
	"testing"
	"time"
)

func TestCgroupCPULimitsRead(t *testing.T) {
	limit, err := GetCgroupCPULimit()
	if err == ErrCGroupsNotSupported {
		t.Skip("unsupported platform")
	}

	if err != nil {
		t.Errorf("failed to query CPU limit: %s", err)
	}
	if limit < 0.0 {
		t.Errorf("unexpectedly negative limit: %g", limit)
	}
	if limit > 10000.0 && !math.IsInf(limit, +1) {
		t.Errorf("unexpectedly large limit (not infinite): %g", limit)
	}
}

func TestCgroupMemLimitsRead(t *testing.T) {
	limit, err := GetCgroupMemoryLimit()
	if err == ErrCGroupsNotSupported {
		t.Skip("unsupported platform")
	}

	if err != nil {
		t.Fatalf("failed to query Memory limit: %s", err)
	}
	if limit < 0 {
		t.Errorf("unexpectedly negative limit: %d", limit)
	}
	if limit < 4096 {
		t.Errorf("unexpectedly small limit (less than a page): %d", limit)
	}
}

func TestCgroupMemStatsRead(t *testing.T) {
	stats, err := GetCgroupMemoryStats()
	if err == ErrCGroupsNotSupported {
		t.Skip("unsupported platform")
	}

	if err != nil {
		t.Fatalf("failed to query Memory usage: %s", err)
	}
	if stats.Total < 0 {
		t.Errorf("unexpectedly negative usage: %d", stats.Total)
	}
	if stats.Total < 4096 {
		t.Errorf("unexpectedly small usage (less than a page): %d", stats.Total)
	}
	if stats.OOMKills < 0 {
		t.Errorf("unexpectedly negative OOM-kill ount: %d", stats.OOMKills)
	}
}

func TestCgroupCPUStatsRead(t *testing.T) {
	stats, err := GetCgroupCPUStats()
	if err == ErrCGroupsNotSupported {
		t.Skip("unsupported platform")
	}

	if err != nil {
		t.Fatalf("failed to query Memory usage: %s", err)
	}
	if stats.Usage.Stime < 0 {
		t.Errorf("unexpectedly negative system usage: %s", stats.Usage.Stime)
	}
	if stats.Usage.Stime < time.Microsecond {
		t.Errorf("unexpectedly small system usage: %s", stats.Usage.Stime)
	}
	if stats.Usage.Utime < 0 {
		t.Errorf("unexpectedly negative user usage: %s", stats.Usage.Utime)
	}
	if stats.Usage.Utime < time.Microsecond {
		t.Errorf("unexpectedly small user usage: %s", stats.Usage.Utime)
	}
	if stats.ThrottledTime < 0 {
		t.Errorf("unexpectedly negative throttled time: %s", stats.ThrottledTime)
	}
}
