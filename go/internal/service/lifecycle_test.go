package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Delusions6515/FluxNet/internal/paths"
)

func TestStartReturnsPromptlyAndLeavesServiceRunning(t *testing.T) {
	root := t.TempDir()
	layout := paths.New(filepath.Join(root, "module"), filepath.Join(root, "data"))

	if err := os.MkdirAll(filepath.Dir(layout.SingBoxBin()), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(layout.RunConfigPath()), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(layout.RunDir(), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(layout.RunConfigPath(), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}

	const fakeSingBox = "#!/bin/sh\nif [ \"$1\" = \"check\" ]; then exit 0; fi\ntrap 'exit 0' TERM INT\nwhile :; do sleep 1; done\n"
	if err := os.WriteFile(layout.SingBoxBin(), []byte(fakeSingBox), 0755); err != nil {
		t.Fatal(err)
	}

	started := time.Now()
	Start(layout, false)
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("Start blocked for %s", elapsed)
	}

	pid := readPID(layout)
	if pid <= 0 || !processAlive(pid, layout.SingBoxBin()) {
		t.Fatalf("service is not running after Start, pid=%d", pid)
	}
	t.Cleanup(func() { Stop(layout, false) })
}
