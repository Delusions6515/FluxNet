package service

import (
	"errors"
	"fmt"
	"os"
)

var rootCgroupProcsPath = "/sys/fs/cgroup/cgroup.procs"

// ensureSingBoxRootCgroup keeps a WebUI-launched core out of the manager app
// cgroup, whose freezer can otherwise suspend proxy traffic with the WebView.
func ensureSingBoxRootCgroup(pid int) error {
	return moveProcessToRootCgroup(pid, rootCgroupProcsPath)
}

func moveProcessToRootCgroup(pid int, procsPath string) error {
	if pid <= 0 {
		return fmt.Errorf("invalid sing-box PID: %d", pid)
	}
	file, err := os.OpenFile(procsPath, os.O_WRONLY, 0)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open root cgroup: %w", err)
	}
	defer file.Close()
	if _, err := fmt.Fprintf(file, "%d\n", pid); err != nil {
		return fmt.Errorf("move process to root cgroup: %w", err)
	}
	return nil
}
