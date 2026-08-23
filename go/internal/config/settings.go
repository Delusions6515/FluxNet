package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Delusions6515/FluxNet/internal/paths"
)

// Settings is the small, deliberately safe configuration surface exposed to WebUI.
type Settings struct {
	Autostart           bool     `json:"autostart"`
	ProxyMode           string   `json:"proxy_mode"`
	TunStack            string   `json:"tun_stack"`
	AutoRedirect        bool     `json:"auto_redirect"`
	TunForward          bool     `json:"tun_forward"`
	AppProxyEnable      bool     `json:"app_proxy_enable"`
	AppProxyMode        string   `json:"app_proxy_mode"`
	AutoProxyAppsEnable bool     `json:"auto_proxy_apps_enable"`
	ProxyApps           []string `json:"proxy_apps"`
	BypassApps          []string `json:"bypass_apps"`
	ForceProxyApps      []string `json:"force_proxy_apps"`
	ForceBypassApps     []string `json:"force_bypass_apps"`
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
	stack := kv["tun_stack"]
	if stack != "system" && stack != "gvisor" && stack != "mixed" {
		stack = "gvisor"
	}
	return Settings{
		Autostart:           kv["autostart"] != "0",
		ProxyMode:           mode,
		TunStack:            stack,
		AutoRedirect:        kv["auto_redirect"] != "0",
		TunForward:          kv["tun_forward"] == "1",
		AppProxyEnable:      kv["app_proxy_enable"] == "1",
		AppProxyMode:        appMode,
		AutoProxyAppsEnable: kv["auto_proxy_apps_enable"] == "1",
		ProxyApps:           parseAppList(kv["proxy_apps_list"]),
		BypassApps:          parseAppList(kv["bypass_apps_list"]),
		ForceProxyApps:      readAppList(layout.ForceProxyApps()),
		ForceBypassApps:     readAppList(layout.ForceBypassApps()),
	}
}

func UpdateSetting(layout *paths.Layout, key, value string) (Settings, error) {
	allowed := map[string]map[string]bool{
		"autostart":              {"0": true, "1": true},
		"proxy_mode":             {"tun": true, "tproxy": true, "redirect": true, "ebpf": true},
		"tun_stack":              {"system": true, "gvisor": true, "mixed": true},
		"auto_redirect":          {"0": true, "1": true},
		"tun_forward":            {"0": true, "1": true},
		"app_proxy_enable":       {"0": true, "1": true},
		"app_proxy_mode":         {"whitelist": true, "blacklist": true},
		"auto_proxy_apps_enable": {"0": true, "1": true},
	}
	if !allowed[key][value] {
		return Settings{}, fmt.Errorf("不支持的设置: %s=%s", key, value)
	}
	if key == "auto_proxy_apps_enable" && value == "1" {
		if apps, err := readPackageFile(layout.ProxyPackageList()); err != nil || len(apps) == 0 {
			return Settings{}, fmt.Errorf("缺少代理应用预置名单，请先更新预置名单")
		}
	}
	if err := writeConfigValue(layout.ConfigFile(), key, value, false); err != nil {
		return Settings{}, err
	}
	return ReadSettings(layout), nil
}

func ReplaceForceAppList(layout *paths.Layout, kind string, apps []string) (Settings, error) {
	var file string
	switch kind {
	case "proxy":
		file = layout.ForceProxyApps()
	case "bypass":
		file = layout.ForceBypassApps()
	default:
		return Settings{}, fmt.Errorf("不支持的强制应用名单: %s", kind)
	}
	clean := uniqueApps(apps)
	for _, app := range clean {
		if !validPackageName(app) {
			return Settings{}, fmt.Errorf("无效包名: %s", app)
		}
	}
	data := ""
	if len(clean) > 0 {
		data = strings.Join(clean, "\n") + "\n"
	}
	if err := atomicWriteFile(file, []byte(data), 0600); err != nil {
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
	if err := writeConfigValues(layout.ConfigFile(), map[string]configValue{
		"app_proxy_mode": {value: mode, quote: true},
		key:              {value: strings.Join(clean, " "), quote: true},
	}); err != nil {
		return Settings{}, err
	}
	return ReadSettings(layout), nil
}

type configValue struct {
	value string
	quote bool
}

func writeConfigValue(file, key, value string, quote bool) error {
	return writeConfigValues(file, map[string]configValue{key: {value: value, quote: quote}})
}

func writeConfigValues(file string, updates map[string]configValue) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	found := make(map[string]bool, len(updates))
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		key, _, ok := strings.Cut(trimmed, "=")
		if !ok {
			continue
		}
		if update, ok := updates[key]; ok {
			assignment := key + "=" + update.value
			if update.quote {
				assignment = fmt.Sprintf("%s=%q", key, update.value)
			}
			lines[i] = assignment
			found[key] = true
		}
	}
	for key, update := range updates {
		if found[key] {
			continue
		}
		assignment := key + "=" + update.value
		if update.quote {
			assignment = fmt.Sprintf("%s=%q", key, update.value)
		}
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = append(lines[:len(lines)-1], assignment, "")
		} else {
			lines = append(lines, assignment)
		}
	}
	return atomicWriteFile(file, []byte(strings.Join(lines, "\n")), 0600)
}

func atomicWriteFile(file string, data []byte, mode os.FileMode) error {
	temp, err := os.CreateTemp(filepath.Dir(file), ".fluxnet-*")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if err := temp.Chmod(mode); err != nil {
		temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempName, file)
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
