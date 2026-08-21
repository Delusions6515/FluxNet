package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/Delusions6515/FluxNet/internal/paths"
)

// Settings is the small, deliberately safe configuration surface exposed to WebUI.
type Settings struct {
	Autostart      bool     `json:"autostart"`
	ProxyMode      string   `json:"proxy_mode"`
	AppProxyEnable bool     `json:"app_proxy_enable"`
	AppProxyMode   string   `json:"app_proxy_mode"`
	ProxyApps      []string `json:"proxy_apps"`
	BypassApps     []string `json:"bypass_apps"`
}

func ReadSettings(layout *paths.Layout) Settings {
	kv := readConfigKV(layout.ConfigFile())
	mode := kv["proxy_mode"]
	if mode != "tun" && mode != "tproxy" && mode != "redirect" && mode != "ebpf" {
		mode = "tun"
	}
	appMode := kv["app_proxy_mode"]
	if appMode != "whitelist" && appMode != "blacklist" {
		appMode = "blacklist"
	}
	return Settings{
		Autostart:      kv["autostart"] != "0",
		ProxyMode:      mode,
		AppProxyEnable: kv["app_proxy_enable"] == "1",
		AppProxyMode:   appMode,
		ProxyApps:      parseAppList(kv["proxy_apps_list"]),
		BypassApps:     parseAppList(kv["bypass_apps_list"]),
	}
}

func UpdateSetting(layout *paths.Layout, key, value string) (Settings, error) {
	allowed := map[string]map[string]bool{
		"autostart":        {"0": true, "1": true},
		"proxy_mode":       {"tun": true, "tproxy": true, "redirect": true, "ebpf": true},
		"app_proxy_enable": {"0": true, "1": true},
		"app_proxy_mode":   {"whitelist": true, "blacklist": true},
	}
	if !allowed[key][value] {
		return Settings{}, fmt.Errorf("不支持的设置: %s=%s", key, value)
	}
	if err := writeConfigValue(layout.ConfigFile(), key, value, false); err != nil {
		return Settings{}, err
	}
	return ReadSettings(layout), nil
}

func ReplaceAppList(layout *paths.Layout, mode string, apps []string) (Settings, error) {
	if mode != "whitelist" && mode != "blacklist" {
		return Settings{}, fmt.Errorf("不支持的应用名单模式: %s", mode)
	}
	clean := uniqueApps(apps)
	for _, app := range clean {
		if !validPackageName(app) {
			return Settings{}, fmt.Errorf("无效包名: %s", app)
		}
	}
	key := "proxy_apps_list"
	if mode == "blacklist" {
		key = "bypass_apps_list"
	}
	if err := writeConfigValue(layout.ConfigFile(), "app_proxy_enable", "1", false); err != nil {
		return Settings{}, err
	}
	if err := writeConfigValue(layout.ConfigFile(), "app_proxy_mode", mode, true); err != nil {
		return Settings{}, err
	}
	if err := writeConfigValue(layout.ConfigFile(), key, strings.Join(clean, " "), true); err != nil {
		return Settings{}, err
	}
	return ReadSettings(layout), nil
}

func writeConfigValue(file, key, value string, quote bool) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	assignment := key + "=" + value
	if quote {
		assignment = fmt.Sprintf("%s=%q", key, value)
	}
	lines := strings.Split(string(data), "\n")
	found := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "#") && strings.HasPrefix(trimmed, key+"=") {
			lines[i] = assignment
			found = true
			break
		}
	}
	if !found {
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = append(lines[:len(lines)-1], assignment, "")
		} else {
			lines = append(lines, assignment)
		}
	}
	return os.WriteFile(file, []byte(strings.Join(lines, "\n")), 0600)
}

func validPackageName(name string) bool {
	parts := strings.Split(name, ".")
	if len(parts) < 2 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, r := range part {
			if !(r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
				return false
			}
		}
	}
	return true
}
