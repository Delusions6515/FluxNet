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
	tplPath := t.layout.InboundTemplate(mode)
	data, err := os.ReadFile(tplPath)
	if err != nil {
		return nil, fmt.Errorf("入站模板读取失败 (%s): %w", tplPath, err)
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
	// Inject include_package / exclude_package from app lists
	proxyApps := readAppList(t.layout.ForceProxyApps())
	bypassApps := readAppList(t.layout.ForceBypassApps())

	if len(proxyApps) > 0 {
		inbound["include_package"] = proxyApps
	}
	if len(bypassApps) > 0 {
		inbound["exclude_package"] = bypassApps
	}

	// stack from sing-box.config
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
	cfg := readConfigKV(t.layout.ConfigFile())

	if mode, ok := cfg["ebpf_mode"]; ok && mode != "" {
		inbound["mode"] = mode
	}

	proxyApps := readAppList(t.layout.ForceProxyApps())
	bypassApps := readAppList(t.layout.ForceBypassApps())

	if local, ok := inbound["local"].(map[string]any); ok {
		if len(proxyApps) > 0 {
			local["include_package"] = proxyApps
		}
		if len(bypassApps) > 0 {
			local["exclude_package"] = bypassApps
		}
	}
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
