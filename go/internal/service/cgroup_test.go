package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMain(m *testing.M) {
	rootCgroupProcsPath = filepath.Join(os.TempDir(), "fluxnet-test-missing-cgroup.procs")
	os.Exit(m.Run())
}

func TestMoveProcessToRootCgroupWritesPID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cgroup.procs")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}

	if err := moveProcessToRootCgroup(123, path); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(content), "123\n"; got != want {
		t.Fatalf("cgroup PID = %q, want %q", got, want)
	}
}

func TestMoveProcessToRootCgroupAllowsMissingCgroupV2(t *testing.T) {
	if err := moveProcessToRootCgroup(123, filepath.Join(t.TempDir(), "missing", "cgroup.procs")); err != nil {
		t.Fatalf("missing cgroup v2 path should be compatible: %v", err)
	}
}
