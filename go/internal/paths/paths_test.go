package paths

import (
	"testing"
)

func TestNewDefaults(t *testing.T) {
	l := New("", "")
	if l.ModuleDir != defaultModuleDir {
		t.Errorf("ModuleDir: got %q, want %q", l.ModuleDir, defaultModuleDir)
	}
	if l.DataDir != defaultDataDir {
		t.Errorf("DataDir: got %q, want %q", l.DataDir, defaultDataDir)
	}
}

func TestNewOverrides(t *testing.T) {
	l := New("/tmp/mod", "/tmp/data")
	if l.ModuleDir != "/tmp/mod" {
		t.Errorf("ModuleDir: got %q", l.ModuleDir)
	}
	if l.DataDir != "/tmp/data" {
		t.Errorf("DataDir: got %q", l.DataDir)
	}
}

func TestSingBoxBin(t *testing.T) {
	l := New("", "")
	if l.SingBoxBin() != "/data/adb/fluxnet/bin/sing-box" {
		t.Errorf("got %q", l.SingBoxBin())
	}
}

func TestRunConfigPath(t *testing.T) {
	l := New("", "")
	if l.RunConfigPath() != "/data/adb/fluxnet/config/run/config.json" {
		t.Errorf("got %q", l.RunConfigPath())
	}
}

func TestProxyPackageCatalogPrefersDataDirectory(t *testing.T) {
	root := t.TempDir()
	l := New(root+"/module", root+"/data")
	if got, want := l.ProxyPackageCatalog(), l.ModProxyPackageList(); got != want {
		t.Errorf("fallback catalogue = %q, want %q", got, want)
	}
}

func TestPidFile(t *testing.T) {
	l := New("", "")
	if l.PidFile() != "/data/adb/fluxnet/run/sing-box.pid" {
		t.Errorf("got %q", l.PidFile())
	}
}
