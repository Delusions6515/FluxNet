package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Delusions6515/FluxNet/internal/paths"
)

// Template loads an inbound JSON template and injects variables.
type Template struct {
	layout *paths.Layout
}

// NewTemplate creates a Template helper.
func NewTemplate(layout *paths.Layout) *Template {
	return &Template{layout: layout}
}

// Apply reads the inbound template for the given mode, injects app lists and
// mode-specific fields, and returns the assembled inbound as json.RawMessage.
func (t *Template) Apply(mode string) (json.RawMessage, error) {
	if mode != "tun" && mode != "tproxy" && mode != "redirect" && mode != "ebpf" {
		return nil, fmt.Errorf("不支持的代理模式: %s", mode)
	}

	inboundPath := t.layout.InboundData(mode)
	data, err := os.ReadFile(inboundPath)
	if err != nil {
		return nil, fmt.Errorf("入站模板读取失败 (%s): %w", inboundPath, err)
	}

	var inbound map[string]any
	if err := json.Unmarshal(data, &inbound); err != nil {
		return nil, fmt.Errorf("入站模板解析失败: %w", err)
	}

	switch mode {
	case "tun":
		t.injectTun(inbound)
	case "tproxy":
		t.injectTproxy(inbound)
	case "redirect":
		t.injectRedirect(inbound)
	case "ebpf":
		t.injectEbpf(inbound)
	}

	result, err := json.Marshal(inbound)
	if err != nil {
		return nil, fmt.Errorf("入站序列化失败: %w", err)
	}
	return result, nil
}

// ---- mode-specific injection ----

func (t *Template) injectTun(inbound map[string]any) {
	injectAppProxy(inbound, effectiveAppProxy(t.layout))

	// stack from fluxnet.config
	cfg := readConfigKV(t.layout.ConfigFile())
	if stack, ok := cfg["tun_stack"]; ok && stack != "" {
		inbound["stack"] = stack
	}
	if ar, ok := cfg["auto_redirect"]; ok && ar == "0" {
		inbound["auto_redirect"] = false
	}
}

func (t *Template) injectTproxy(inbound map[string]any) {
	// Port from tproxy.conf
	port := readTproxyPort(t.layout.TproxyConf())
	if port > 0 {
		inbound["listen_port"] = port
	}
}

func (t *Template) injectRedirect(inbound map[string]any) {
	// Port from tproxy.conf
	port := readTproxyPort(t.layout.TproxyConf())
	if port > 0 {
		inbound["listen_port"] = port
	}
}

func (t *Template) injectEbpf(inbound map[string]any) {
	if local, ok := inbound["local"].(map[string]any); ok {
		injectAppProxy(local, effectiveAppProxy(t.layout))
	}
}

type appProxySettings struct {
	enabled    bool
	mode       string
	proxyApps  []string
	bypassApps []string
}

func effectiveAppProxy(layout *paths.Layout) appProxySettings {
	cfg := readConfigKV(layout.ConfigFile())
	mode := cfg["app_proxy_mode"]
	if mode != "whitelist" && mode != "blacklist" {
		mode = "blacklist"
	}

	forceProxy := readAppList(layout.ForceProxyApps())
	forceBypass := readAppList(layout.ForceBypassApps())
	if cfg["app_proxy_enable"] != "1" && len(forceProxy) == 0 && len(forceBypass) == 0 {
		return appProxySettings{mode: mode}
	}
	if cfg["app_proxy_enable"] != "1" {
		mode = "blacklist"
	}

	settings := appProxySettings{enabled: true, mode: mode}
	if mode == "whitelist" {
		settings.proxyApps = removeApps(append(parseAppList(cfg["proxy_apps_list"]), forceProxy...), forceBypass)
	} else {
		settings.bypassApps = removeApps(append(parseAppList(cfg["bypass_apps_list"]), forceBypass...), forceProxy)
	}
	return settings
}

func injectAppProxy(inbound map[string]any, apps appProxySettings) {
	delete(inbound, "include_package")
	delete(inbound, "exclude_package")
	if !apps.enabled {
		return
	}
	if apps.mode == "whitelist" {
		inbound["include_package"] = apps.proxyApps
		return
	}
	inbound["exclude_package"] = apps.bypassApps
}

func parseAppList(value string) []string {
	return uniqueApps(strings.Fields(value))
}

func removeApps(apps, remove []string) []string {
	blocked := make(map[string]struct{}, len(remove))
	for _, app := range remove {
		blocked[app] = struct{}{}
	}
	result := make([]string, 0, len(apps))
	for _, app := range uniqueApps(apps) {
		if _, ok := blocked[app]; !ok {
			result = append(result, app)
		}
	}
	return result
}

func uniqueApps(apps []string) []string {
	seen := make(map[string]struct{}, len(apps))
	result := make([]string, 0, len(apps))
	for _, app := range apps {
		if _, ok := seen[app]; !ok {
			seen[app] = struct{}{}
			result = append(result, app)
		}
	}
	return result
}

// ---- helpers ----

func readAppList(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	lines := strings.Split(string(data), "\n")
	var result []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		result = append(result, line)
	}
	return result
}

func readConfigKV(path string) map[string]string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	kv := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		val = strings.Trim(val, "\"'")
		kv[key] = val
	}
	return kv
}

func readTproxyPort(tproxyConf string) int {
	data, err := os.ReadFile(tproxyConf)
	if err != nil {
		return 1536 // default
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "PROXY_TCP_PORT=") {
			val := strings.TrimPrefix(line, "PROXY_TCP_PORT=")
			var port int
			if _, err := fmt.Sscanf(val, "%d", &port); err == nil {
				return port
			}
		}
	}
	return 1536
}
