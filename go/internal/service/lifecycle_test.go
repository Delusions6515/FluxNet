package service

import (
	"os"
	"path/filepath"
	"strings"
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
	if err := os.MkdirAll(layout.InboundTemplateDir(), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(layout.RunDir(), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(layout.ConfigDir(), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(layout.ConfigDir(), "fluxnet.config"), []byte("proxy_mode=tun\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(layout.InboundTemplate("tun"), []byte(`{"type":"tun","tag":"tun-in"}`), 0600); err != nil {
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
	if _, err := os.Stat(layout.RunConfigPath()); err != nil {
		t.Fatalf("Start did not apply the runtime config: %v", err)
	}

	pid := readPID(layout)
	if pid <= 0 || !processAlive(pid, layout.SingBoxBin()) {
		t.Fatalf("service is not running after Start, pid=%d", pid)
	}
	t.Cleanup(func() { Stop(layout, false) })
}

func TestRestartDoesNotStopAtpForTunRuntime(t *testing.T) {
	root := t.TempDir()
	layout := paths.New(filepath.Join(root, "module"), filepath.Join(root, "data"))

	for _, dir := range []string{filepath.Dir(layout.SingBoxBin()), layout.InboundTemplateDir(), layout.RunConfigDir(), layout.ConfigDir(), layout.RunTproxyDir()} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(layout.ConfigDir(), "fluxnet.config"), []byte("proxy_mode=tun\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(layout.InboundTemplate("tun"), []byte(`{"type":"tun","tag":"tun-in"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(layout.RunConfigPath(), []byte(`{"inbounds":[{"type":"tun","tag":"tun-in"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	const fakeSingBox = "#!/bin/sh\nif [ \"$1\" = \"check\" ]; then exit 0; fi\ntrap 'exit 0' TERM INT\nwhile :; do sleep 1; done\n"
	if err := os.WriteFile(layout.SingBoxBin(), []byte(fakeSingBox), 0755); err != nil {
		t.Fatal(err)
	}
	const fakeAtp = "#!/bin/sh\nprintf '%s\\n' \"$*\" > \"$0.calls\"\n"
	if err := os.WriteFile(layout.AtpBin(), []byte(fakeAtp), 0755); err != nil {
		t.Fatal(err)
	}

	Restart(layout, false)
	if _, err := os.Stat(layout.AtpBin() + ".calls"); !os.IsNotExist(err) {
		t.Fatalf("Restart called ATP for a tun runtime config: %v", err)
	}
	runtime, err := os.ReadFile(layout.RunConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(runtime), `"tag": "tun-in"`) {
		t.Fatalf("Restart did not apply the saved tun config:\n%s", runtime)
	}
	t.Cleanup(func() { Stop(layout, false) })
}
