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
	if _, err := UpdateSetting(layout, "tun_forward", "1"); err == nil {
		t.Fatal("UpdateSetting accepted an advanced setting")
	}
	if _, err := UpdateSetting(layout, "proxy_mode", "invalid"); err == nil {
		t.Fatal("UpdateSetting accepted an invalid proxy mode")
	}
}

func TestReplaceAppListWritesActiveMode(t *testing.T) {
	layout := testSettingsLayout(t)
	if _, err := UpdateSetting(layout, "app_proxy_enable", "0"); err != nil {
		t.Fatal(err)
	}
	settings, err := ReplaceAppList(layout, "whitelist", []string{"com.example.one", "com.example.one", "io.demo.two"})
	if err != nil {
		t.Fatal(err)
	}
	if settings.AppProxyEnable || settings.AppProxyMode != "whitelist" || len(settings.ProxyApps) != 2 {
		t.Fatalf("settings = %#v", settings)
	}
	if _, err := ReplaceAppList(layout, "whitelist", []string{"not a package"}); err == nil {
		t.Fatal("ReplaceAppList accepted an invalid package")
	}
	data, err := os.ReadFile(layout.ConfigFile())
	if err != nil || !strings.Contains(string(data), "advanced_setting=keep") {
		t.Fatalf("advanced setting was not preserved: %v, %s", err, data)
	}
}

func testSettingsLayout(t *testing.T) *paths.Layout {
	t.Helper()
	root := t.TempDir()
	layout := paths.New(filepath.Join(root, "module"), filepath.Join(root, "data"))
	if err := os.MkdirAll(layout.ConfigDir(), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(layout.ConfigDir(), "fluxnet.config"), []byte("# keep comments\nadvanced_setting=keep\nautostart=1\nproxy_mode=tun\napp_proxy_enable=0\napp_proxy_mode=blacklist\nproxy_apps_list=\"\"\nbypass_apps_list=\"\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	return layout
}
