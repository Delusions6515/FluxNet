package config

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Delusions6515/FluxNet/internal/paths"
	"github.com/Delusions6515/FluxNet/internal/result"
)

// Apply assembles the runtime sing-box config and applies atp rules if needed.
func Apply(layout *paths.Layout, formatJSON bool) {
	kv := readConfigKV(layout.ConfigFile())
	mode := kv["proxy_mode"]
	if mode == "" {
		mode = "tproxy"
	}

	// 1. Load active full config from subscription.json
	fullConfig, err := loadActiveConfig(layout)
	if err != nil {
		result.Err(formatJSON, "config.load_failed", "加载活跃配置失败: "+err.Error())
		return
	}

	// 2. Generate inbound from template
	tmpl := NewTemplate(layout)
	inbound, err := tmpl.Apply(mode)
	if err != nil {
		result.Err(formatJSON, "config.template_failed", "入站模板处理失败: "+err.Error())
		return
	}

	// 3. Inject inbound into inbounds array
	if err := injectInbound(&fullConfig, inbound); err != nil {
		result.Err(formatJSON, "config.inject_failed", "入站注入失败: "+err.Error())
		return
	}

	// 4. Write run/config.json
	runConfigDir := layout.RunConfigDir()
	if err := os.MkdirAll(runConfigDir, 0755); err != nil {
		result.Err(formatJSON, "config.write_failed", "创建运行配置目录失败: "+err.Error())
		return
	}

	runData, err := json.MarshalIndent(fullConfig, "", "  ")
	if err != nil {
		result.Err(formatJSON, "config.serialize_failed", "序列化运行配置失败: "+err.Error())
		return
	}

	if err := os.WriteFile(layout.RunConfigPath(), runData, 0600); err != nil {
		result.Err(formatJSON, "config.write_failed", "写入运行配置失败: "+err.Error())
		return
	}

	// 5. atp for tproxy/redirect modes
	if mode == "tproxy" || mode == "redirect" {
		if err := applyAtp(layout); err != nil {
			result.Err(formatJSON, "config.atp_failed", "atp 规则应用失败: "+err.Error())
			return
		}
	}

	if formatJSON {
		result.Text(result.Success("config.applied", "配置已应用",
			map[string]any{"mode": mode}), true)
	} else {
		fmt.Printf("✓ 配置已应用 (模式: %s, 入站: %s-in)\n", mode, mode)
	}
}

// loadActiveConfig reads subscription.json, finds the active entry, and loads
// the full sing-box config from local/ or remote/.
func loadActiveConfig(layout *paths.Layout) (map[string]any, error) {
	subData, err := os.ReadFile(layout.SubscriptionFile())
	if err != nil {
		// No subscription index yet — return minimal config
		return minimalConfig(), nil
	}

	var subIndex struct {
		Active        string `json:"active"`
		Subscriptions []struct {
			Name     string `json:"name"`
			Type     string `json:"type"`
			Filename string `json:"filename"`
		} `json:"subscriptions"`
	}
	if err := json.Unmarshal(subData, &subIndex); err != nil {
		return minimalConfig(), nil
	}

	// Find active subscription
	var activeFile string
	for _, s := range subIndex.Subscriptions {
		if s.Name == subIndex.Active {
			switch s.Type {
			case "local":
				activeFile = filepath.Join(layout.LocalConfigDir(), s.Filename)
			case "remote":
				activeFile = filepath.Join(layout.RemoteConfigDir(), s.Filename)
			}
			break
		}
	}

	if activeFile == "" {
		return minimalConfig(), nil
	}

	configData, err := os.ReadFile(activeFile)
	if err != nil {
		return minimalConfig(), nil
	}

	var config map[string]any
	if err := json.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("配置 JSON 解析失败: %w", err)
	}

	return config, nil
}

// minimalConfig returns a bare minimum sing-box config with direct outbound.
func minimalConfig() map[string]any {
	return map[string]any{
		"log": map[string]any{
			"level": "info",
		},
		"dns": map[string]any{
			"servers": []map[string]any{
				{"tag": "dns-local", "address": "223.5.5.5", "detour": "direct"},
			},
		},
		"inbounds":  []any{},
		"outbounds": []any{map[string]any{"tag": "direct", "type": "direct"}},
		"route": map[string]any{
			"rules": []any{},
		},
	}
}

// injectInbound appends a new inbound to the "inbounds" array in the config.
func injectInbound(config *map[string]any, inbound json.RawMessage) error {
	var inb any
	if err := json.Unmarshal(inbound, &inb); err != nil {
		return err
	}

	inbounds, _ := (*config)["inbounds"].([]any)
	if inbounds == nil {
		inbounds = []any{}
	}
	(*config)["inbounds"] = append(inbounds, inb)
	return nil
}

// applyAtp generates the atp runtime config and calls atp start.
func applyAtp(layout *paths.Layout) error {
	// Read tproxy.conf template
	tproxyConfPath := layout.TproxyConf()
	tproxyData, err := os.ReadFile(tproxyConfPath)
	if err != nil {
		return fmt.Errorf("读取 tproxy.conf 失败: %w", err)
	}

	// Write runtime tproxy config
	tproxyDir := layout.RunTproxyDir()
	if err := os.MkdirAll(tproxyDir, 0755); err != nil {
		return fmt.Errorf("创建 tproxy 运行时目录失败: %w", err)
	}
	if err := os.WriteFile(layout.RunTproxyConf(), tproxyData, 0600); err != nil {
		return fmt.Errorf("写入运行时 tproxy.conf 失败: %w", err)
	}

	// Call atp
	atpBin := layout.AtpBin()
	if _, err := os.Stat(atpBin); os.IsNotExist(err) {
		return fmt.Errorf("atp 二进制不存在: %s", atpBin)
	}

	cmd := exec.Command(atpBin, "-d", tproxyDir, "start")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("atp start 失败: %w", err)
	}

	return nil
}
