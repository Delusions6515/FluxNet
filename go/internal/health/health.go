package health

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Delusions6515/FluxNet/internal/paths"
	"github.com/Delusions6515/FluxNet/internal/result"
)

// Check performs a health check and outputs the result.
func Check(layout *paths.Layout, formatJSON bool) {
	bin := layout.SingBoxBin()
	pid := readPID(layout)
	processAlive := pid > 0 && checkProcess(pid, bin)
	mode := readProxyMode(layout)
	apiReachable := false
	proxyRulesActive := false

	if processAlive {
		apiReachable = checkAPI(layout)
	}

	if mode == "tproxy" || mode == "redirect" {
		proxyRulesActive = checkIptables()
	}

	data := map[string]any{
		"process_alive":      processAlive,
		"api_reachable":      apiReachable,
		"proxy_rules_active": proxyRulesActive,
		"mode":               mode,
		"pid":                pid,
	}

	if formatJSON {
		result.Text(result.Success("health.check", "健康检查", data), true)
		return
	}

	fmt.Println("FluxNet 健康检查")
	fmt.Println("=================")
	if processAlive {
		fmt.Printf("  ✓ sing-box 进程 (PID: %d)\n", pid)
	} else {
		fmt.Println("  ✗ sing-box 进程未运行")
	}
	if apiReachable {
		fmt.Println("  ✓ Service API 可达")
	} else {
		fmt.Println("  ✗ Service API 不可达")
	}
	if mode == "tproxy" || mode == "redirect" {
		if proxyRulesActive {
			fmt.Println("  ✓ 透明代理规则存在")
		} else {
			fmt.Println("  ✗ 透明代理规则未找到")
		}
	}
	fmt.Printf("\n模式: %s\n", mode)
}

func readPID(layout *paths.Layout) int {
	data, _ := os.ReadFile(layout.PidFile())
	var pid int
	fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &pid)
	return pid
}

func checkProcess(pid int, bin string) bool {
	path := fmt.Sprintf("/proc/%d/cmdline", pid)
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(strings.ReplaceAll(string(data), "\x00", " "), bin)
}

func checkAPI(layout *paths.Layout) bool {
	// Try common sing-box API ports
	for _, port := range []string{"9090", "9091"} {
		conn, err := net.DialTimeout("tcp", "127.0.0.1:"+port, 2*time.Second)
		if err == nil {
			conn.Close()
			return true
		}
	}
	return false
}

func checkIptables() bool {
	out, err := exec.Command("iptables", "-t", "mangle", "-L", "OUTPUT").CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "TPROXY") || strings.Contains(string(out), "REDIRECT")
}

func readProxyMode(layout *paths.Layout) string {
	data, err := os.ReadFile(layout.ConfigFile())
	if err != nil {
		return "unknown"
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "proxy_mode=") {
			val := strings.TrimPrefix(line, "proxy_mode=")
			return strings.Trim(val, "\"'")
		}
	}
	return "unknown"
}
