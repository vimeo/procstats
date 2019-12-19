package cgrouplimits

import "testing"

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
	if limit > 10000.0 {
		t.Errorf("unexpectedly large limit: %g", limit)
	}
}

func TestCgroupMemLimitsRead(t *testing.T) {
	limit, err := GetCgroupMemoryLimit()
	if err == ErrCGroupsNotSupported {
		t.Skip("unsupported platform")
	}

	if err != nil {
		t.Fatalf("failed to query CPU limit: %s", err)
	}
	if limit < 0 {
		t.Errorf("unexpectedly negative limit: %d", limit)
	}
	if limit < 4096 {
		t.Errorf("unexpectedly small limit (less than a page): %d", limit)
	}
}
