package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Delusions6515/FluxNet/internal/paths"
)

func TestUpdateSettingRejectsUnsupportedValue(t *testing.T) {
	layout := testSettingsLayout(t)
	if _, err := UpdateSetting(layout, "tun_forward", "1"); err != nil {
		t.Fatalf("UpdateSetting rejected tun_forward: %v", err)
	}
	if _, err := UpdateSetting(layout, "proxy_mode", "invalid"); err == nil {
		t.Fatal("UpdateSetting accepted an invalid proxy mode")
	}
	if _, err := UpdateSetting(layout, "auto_mode", "1"); err == nil {
		t.Fatal("UpdateSetting enabled automatic lists without a catalogue")
	}
	if err := os.WriteFile(layout.ProxyPackageList(), []byte("com.example.proxy\n"), 0600); err != nil {
		t.Fatal(err)
	}
	settings, err := UpdateSetting(layout, "auto_mode", "1")
	if err != nil || !settings.AutoMode {
		t.Fatalf("automatic list setting = %#v, %v", settings, err)
	}
}

func TestReplaceForceAppList(t *testing.T) {
	layout := testSettingsLayout(t)
	settings, err := ReplaceForceAppList(layout, "proxy", []string{"com.example.one", "com.example.one", "io.demo.two"})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(settings.ForceProxyApps, " "); got != "com.example.one io.demo.two" {
		t.Fatalf("force proxy apps = %q", got)
	}
	if _, err := ReplaceForceAppList(layout, "bypass", []string{"invalid package"}); err == nil {
		t.Fatal("ReplaceForceAppList accepted an invalid package")
	}
}

func TestReadSettingsUsesEmptyForceLists(t *testing.T) {
	settings := ReadSettings(testSettingsLayout(t))
	if settings.ForceProxyApps == nil || settings.ForceBypassApps == nil {
		t.Fatalf("force app lists must serialize as arrays: %#v", settings)
	}
}

func TestReplaceAppListWritesIndependentFile(t *testing.T) {
	layout := testSettingsLayout(t)
	if _, err := UpdateSetting(layout, "app_proxy_enable", "0"); err != nil {
		t.Fatal(err)
	}
	settings, err := ReplaceAppList(layout, "proxy", []string{"com.example.one", "com.example.one", "io.demo.two"})
	if err != nil {
		t.Fatal(err)
	}
	if settings.AppProxyEnable || settings.AppProxyMode != "blacklist" || len(settings.ProxyApps) != 2 {
		t.Fatalf("settings = %#v", settings)
	}
	if _, err := ReplaceAppList(layout, "proxy", []string{"not a package"}); err == nil {
		t.Fatal("ReplaceAppList accepted an invalid package")
	}
	data, err := os.ReadFile(layout.ProxyApps())
	if err != nil || string(data) != "com.example.one\nio.demo.two\n" {
		t.Fatalf("proxy list = %q, %v", data, err)
	}
}

func TestReplaceAppListsValidatesBothBeforeWriting(t *testing.T) {
	layout := testSettingsLayout(t)
	if err := os.WriteFile(layout.ProxyApps(), []byte("com.example.old\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReplaceAppLists(layout, []string{"com.example.new"}, []string{"invalid package"}); err == nil {
		t.Fatal("ReplaceAppLists accepted an invalid bypass package")
	}
	data, err := os.ReadFile(layout.ProxyApps())
	if err != nil || string(data) != "com.example.old\n" {
		t.Fatalf("proxy list changed after failed replacement: %q, %v", data, err)
	}
}

func TestReplaceAppListsRestoresProxyAfterBypassWriteFailure(t *testing.T) {
	layout := testSettingsLayout(t)
	if err := os.WriteFile(layout.ProxyApps(), []byte("com.example.old\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(layout.BypassApps(), 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := ReplaceAppLists(layout, []string{"com.example.new"}, []string{"com.example.bypass"}); err == nil {
		t.Fatal("ReplaceAppLists accepted a directory as bypass list")
	}
	data, err := os.ReadFile(layout.ProxyApps())
	if err != nil || string(data) != "com.example.old\n" {
		t.Fatalf("proxy list was not restored: %q, %v", data, err)
	}
}

func testSettingsLayout(t *testing.T) *paths.Layout {
	t.Helper()
	root := t.TempDir()
	layout := paths.New(filepath.Join(root, "module"), filepath.Join(root, "data"))
	if err := os.MkdirAll(layout.ConfigDir(), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(layout.ConfigDir(), "fluxnet.config"), []byte("# keep comments\nadvanced_setting=keep\nautostart=1\nproxy_mode=tun\ntun_stack=gvisor\nauto_redirect=0\ntun_forward=0\napp_proxy_enable=0\napp_proxy_mode=blacklist\nauto_mode=0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	return layout
}
