package config

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Delusions6515/FluxNet/internal/paths"
)

// TunForwardEnable 从运行配置中读取 tun 入站的 interface_name，
// 设置 ip forwarding、策略路由和 iptables FORWARD 规则，让热点设备走 tun 代理。
func TunForwardEnable(layout *paths.Layout) error {
	tunDevice, err := probeTunInterface(layout)
	if err != nil {
		return fmt.Errorf("无法定位 tun 设备: %w", err)
	}

	// 1. Enable IP forwarding + rp_filter
	os.WriteFile("/proc/sys/net/ipv4/ip_forward", []byte("1"), 0644)
	os.WriteFile("/proc/sys/net/ipv4/conf/default/rp_filter", []byte("2"), 0644)
	os.WriteFile("/proc/sys/net/ipv4/conf/all/rp_filter", []byte("2"), 0644)

	// 2. Policy routing (参考 sing-box / box4magisk sing_tun_ip_rules)
	exec.Command("ip", "rule", "add", "iif", "lo", "lookup", "local_network", "pref", "7000").Run()
	exec.Command("ip", "rule", "add", "from", "all", "iif", tunDevice, "goto", "7010", "pref", "7001").Run()
	exec.Command("ip", "rule", "add", "from", "all", "lookup", "2022", "pref", "7002").Run()
	exec.Command("ip", "rule", "add", "from", "all", "nop", "pref", "7010").Run()
	exec.Command("ip", "-6", "rule", "add", "from", "all", "iif", tunDevice, "goto", "7010", "pref", "7001").Run()
	exec.Command("ip", "-6", "rule", "add", "from", "all", "lookup", "2022", "pref", "7002").Run()
	exec.Command("ip", "-6", "rule", "add", "from", "all", "nop", "pref", "7010").Run()

	// 3. FORWARD iptables rules
	for _, cmd := range []string{"iptables", "ip6tables"} {
		exec.Command(cmd, "-w", "100", "-I", "FORWARD", "-o", tunDevice, "-j", "ACCEPT").Run()
		exec.Command(cmd, "-w", "100", "-I", "FORWARD", "-i", tunDevice, "-j", "ACCEPT").Run()
	}

	return nil
}

// TunForwardDisable 移除所有 tun forwarding 规则。
func TunForwardDisable(layout *paths.Layout) {
	tunDevice, err := probeTunInterface(layout)
	if err != nil {
		return
	}

	exec.Command("ip", "rule", "del", "pref", "7000").Run()
	exec.Command("ip", "rule", "del", "pref", "7001").Run()
	exec.Command("ip", "rule", "del", "pref", "7002").Run()
	exec.Command("ip", "rule", "del", "pref", "7010").Run()
	exec.Command("ip", "-6", "rule", "del", "pref", "7001").Run()
	exec.Command("ip", "-6", "rule", "del", "pref", "7002").Run()
	exec.Command("ip", "-6", "rule", "del", "pref", "7010").Run()

	for _, cmd := range []string{"iptables", "ip6tables"} {
		exec.Command(cmd, "-w", "100", "-D", "FORWARD", "-o", tunDevice, "-j", "ACCEPT").Run()
		exec.Command(cmd, "-w", "100", "-D", "FORWARD", "-i", tunDevice, "-j", "ACCEPT").Run()
	}

	os.WriteFile("/proc/sys/net/ipv4/ip_forward", []byte("0"), 0644)
}

// probeTunInterface 从 run/config.json 的 tun 入站中读取 interface_name。
func probeTunInterface(layout *paths.Layout) (string, error) {
	data, err := os.ReadFile(layout.RunConfigPath())
	if err != nil {
		return "", err
	}

	var cfg struct {
		Inbounds []struct {
			Type          string `json:"type"`
			InterfaceName string `json:"interface_name"`
		} `json:"inbounds"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", fmt.Errorf("解析运行配置失败: %w", err)
	}

	for _, inb := range cfg.Inbounds {
		if inb.Type == "tun" {
			if inb.InterfaceName == "" {
				return "tun0", nil
			}
			return inb.InterfaceName, nil
		}
	}
	return "tun0", nil // fallback
}

// probeTunDevice 检测 tun 接口是否已创建。
func probeTunDevice(tunDevice string) bool {
	out, err := exec.Command("ifconfig").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), tunDevice)
}