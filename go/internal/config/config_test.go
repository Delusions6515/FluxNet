package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Delusions6515/FluxNet/internal/paths"
)

func TestApplyRuntimeInjectsAtpAppLists(t *testing.T) {
	root := t.TempDir()
	layout := paths.New(filepath.Join(root, "module"), filepath.Join(root, "data"))
	for _, dir := range []string{layout.ConfigDir(), layout.InboundTemplateDir(), filepath.Dir(layout.AtpBin())} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(layout.ConfigDir(), "fluxnet.config"), []byte("proxy_mode=redirect\napp_proxy_enable=1\napp_proxy_mode=blacklist\nbypass_apps_list=\"com.example.manual\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(layout.InboundTemplate("redirect"), []byte(`{"type":"redirect","tag":"redirect-in"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(layout.ModTproxyConf(), []byte("PROXY_TCP_PORT=1536\nPROXY_APPS_LIST=\"stale\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(layout.ForceProxyApps(), []byte("com.example.force-proxy\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(layout.ForceBypassApps(), []byte("com.example.bypass\ncom.example.force-proxy\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(layout.AtpBin(), []byte("#!/bin/sh\necho atp-start\n"), 0755); err != nil {
		t.Fatal(err)
	}

	if _, err := ApplyRuntime(layout); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(layout.RunTproxyConf())
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	for _, want := range []string{
		`APP_PROXY_ENABLE=1`,
		`APP_PROXY_MODE="blacklist"`,
		`PROXY_APPS_LIST=""`,
		`BYPASS_APPS_LIST="com.example.manual com.example.bypass"`,
		`PROXY_MODE=2`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("generated tproxy.conf missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, `PROXY_APPS_LIST="stale"`) {
		t.Errorf("generated tproxy.conf retained the stale app list:\n%s", got)
	}
	logData, err := os.ReadFile(layout.AtpLog())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(logData), "atp-start") {
		t.Errorf("atp log missing command output:\n%s", logData)
	}
}

func TestTemplateAppProxyEmptyListSerializesAsArray(t *testing.T) {
	root := t.TempDir()
	layout := paths.New(filepath.Join(root, "module"), filepath.Join(root, "data"))
	for _, dir := range []string{layout.ConfigDir(), layout.InboundTemplateDir()} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	for _, appMode := range []struct {
		mode  string
		field string
	}{
		{mode: "whitelist", field: "include_package"},
		{mode: "blacklist", field: "exclude_package"},
	} {
		config := "app_proxy_enable=1\napp_proxy_mode=" + appMode.mode + "\n"
		if err := os.WriteFile(filepath.Join(layout.ConfigDir(), "fluxnet.config"), []byte(config), 0600); err != nil {
			t.Fatal(err)
		}
		for mode, template := range map[string]string{
			"tun":  `{"type":"tun"}`,
			"ebpf": `{"type":"ebpf","mode":"local","local":{}}`,
		} {
			if err := os.WriteFile(layout.InboundTemplate(mode), []byte(template), 0600); err != nil {
				t.Fatal(err)
			}
			data, err := NewTemplate(layout).Apply(mode)
			if err != nil {
				t.Fatal(err)
			}
			var inbound map[string]any
			if err := json.Unmarshal(data, &inbound); err != nil {
				t.Fatal(err)
			}
			if mode == "ebpf" {
				inbound = inbound["local"].(map[string]any)
			}
			apps, ok := inbound[appMode.field].([]any)
			if !ok || len(apps) != 0 {
				t.Errorf("%s %s = %#v, want empty array", mode, appMode.field, inbound[appMode.field])
			}
		}
	}
}

func TestTemplateAppProxyUsesConfiguredMode(t *testing.T) {
	root := t.TempDir()
	layout := paths.New(filepath.Join(root, "module"), filepath.Join(root, "data"))
	for _, dir := range []string{layout.ConfigDir(), layout.InboundTemplateDir()} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(layout.ConfigDir(), "fluxnet.config"), []byte("app_proxy_enable=1\napp_proxy_mode=whitelist\nproxy_apps_list=\"com.example.manual com.example.bypass\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(layout.ForceProxyApps(), []byte("com.example.force\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(layout.ForceBypassApps(), []byte("com.example.bypass\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for mode, template := range map[string]string{
		"tun":  `{"type":"tun","include_package":["stale"],"exclude_package":["stale"]}`,
		"ebpf": `{"type":"ebpf","mode":"local","local":{"include_package":["stale"],"exclude_package":["stale"]}}`,
	} {
		if err := os.WriteFile(layout.InboundTemplate(mode), []byte(template), 0600); err != nil {
			t.Fatal(err)
		}
		data, err := NewTemplate(layout).Apply(mode)
		if err != nil {
			t.Fatal(err)
		}
		var inbound map[string]any
		if err := json.Unmarshal(data, &inbound); err != nil {
			t.Fatal(err)
		}
		if mode == "ebpf" {
			if inbound["mode"] != "local" {
				t.Errorf("ebpf mode = %v, want template value local", inbound["mode"])
			}
			inbound = inbound["local"].(map[string]any)
		}
		if _, ok := inbound["exclude_package"]; ok {
			t.Errorf("%s retained exclude_package in whitelist mode", mode)
		}
		got := inbound["include_package"].([]any)
		if len(got) != 2 || got[0] != "com.example.manual" || got[1] != "com.example.force" {
			t.Errorf("%s include_package = %#v", mode, got)
		}
	}
}

func TestEffectiveAppProxyUsesForceBypassWhenDisabled(t *testing.T) {
	root := t.TempDir()
	layout := paths.New(filepath.Join(root, "module"), filepath.Join(root, "data"))
	if err := os.MkdirAll(layout.ConfigDir(), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(layout.ConfigDir(), "fluxnet.config"), []byte("app_proxy_enable=0\napp_proxy_mode=whitelist\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(layout.ForceBypassApps(), []byte("com.example.bypass\n"), 0600); err != nil {
		t.Fatal(err)
	}
	apps := effectiveAppProxy(layout)
	if !apps.enabled || apps.mode != "blacklist" || len(apps.bypassApps) != 1 || apps.bypassApps[0] != "com.example.bypass" {
		t.Errorf("effective app proxy = %#v", apps)
	}
}
