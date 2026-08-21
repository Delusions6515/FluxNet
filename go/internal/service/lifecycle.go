package service

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Delusions6515/FluxNet/internal/config"
	"github.com/Delusions6515/FluxNet/internal/paths"
	"github.com/Delusions6515/FluxNet/internal/result"
)

const (
	stopGraceTimeout = 5 * time.Second
)

// StatusData is returned by the status command.
type StatusData struct {
	PID      int    `json:"pid"`
	Running  bool   `json:"running"`
	Mode     string `json:"mode"`
	UptimeMs int64  `json:"uptime_ms,omitempty"`
	Version  string `json:"version,omitempty"`
}

// Start launches sing-box with the runtime config.
func Start(layout *paths.Layout, formatJSON bool) {
	start(layout, formatJSON, true)
}

// RestartExisting restarts the generated runtime configuration without applying
// saved edits. This keeps ordinary WebUI restart separate from apply-and-restart.
func RestartExisting(layout *paths.Layout, formatJSON bool) {
	restartProcess(layout, false)
	start(layout, formatJSON, false)
}

func start(layout *paths.Layout, formatJSON, applyConfig bool) {
	bin := layout.SingBoxBin()
	runConfig := layout.RunConfigPath()

	// Validate binary
	if _, err := os.Stat(bin); os.IsNotExist(err) {
		result.Err(formatJSON, "service.binary_missing", "sing-box 内核不存在: "+bin)
		return
	}

	// Check if already running
	if pid := readPID(layout); pid > 0 && processAlive(pid, bin) {
		result.Err(formatJSON, "service.already_running", fmt.Sprintf("sing-box 已在运行 (PID: %d)", pid))
		return
	}

	if applyConfig {
		if _, err := config.ApplyRuntime(layout); err != nil {
			result.Err(formatJSON, "service.config_apply_failed", "应用运行配置失败: "+err.Error())
			return
		}
	} else if _, err := os.Stat(runConfig); err != nil {
		result.Err(formatJSON, "service.runtime_config_missing", "运行配置不存在，请先应用配置")
		return
	}

	// sing-box check
	checkCmd := exec.Command(bin, "check", "-c", runConfig)
	checkOut, err := checkCmd.CombinedOutput()
	if err != nil {
		result.Err(formatJSON, "service.check_failed", fmt.Sprintf("sing-box check 未通过: %s", strings.TrimSpace(string(checkOut))))
		return
	}

	// Disable Private DNS to prevent DNS leaks
	savePrivateDns(layout)

	// Start sing-box without waiting for its long-running process to exit.
	svcCmd := exec.Command(bin, "run", "-c", runConfig, "-D", layout.RunDir())
	svcCmd.Stdout = os.Stdout
	svcCmd.Stderr = os.Stderr

	if err := svcCmd.Start(); err != nil {
		result.Err(formatJSON, "service.start_failed", "启动 sing-box 失败: "+err.Error())
		return
	}

	// Write PID
	writePID(layout, svcCmd.Process.Pid)

	// Read mode from config for output
	mode := readProxyMode(layout)

	// Tun hotspot forwarding
	tunForwardRead := readConfigBool(layout, "tun_forward")
	if mode == "tun" && tunForwardRead {
		if err := config.TunForwardEnable(layout); err != nil {
			fmt.Fprintf(os.Stderr, "[Warn] tun hotspot 启用失败: %s\n", err)
		}
	}

	data := StatusData{
		PID:     svcCmd.Process.Pid,
		Running: true,
		Mode:    mode,
	}

	if formatJSON {
		result.Text(result.Success("service.started", "服务已启动", data), true)
	} else {
		fmt.Printf("✓ 服务已启动 (PID: %d, 模式: %s)\n", svcCmd.Process.Pid, mode)
	}
}

// Stop terminates sing-box and cleans up.
func Stop(layout *paths.Layout, formatJSON bool) {
	bin := layout.SingBoxBin()
	pid := readPID(layout)

	if pid <= 0 || !processAlive(pid, bin) {
		// Clean up stale PID file
		os.Remove(layout.PidFile())
		result.OK(formatJSON, "service.stopped", "服务未运行")
		return
	}

	// SIGTERM
	proc, err := os.FindProcess(pid)
	if err == nil {
		_ = proc.Signal(syscall.SIGTERM)
	}

	// Wait for graceful exit
	deadline := time.Now().Add(stopGraceTimeout)
	for time.Now().Before(deadline) {
		if !processAlive(pid, bin) {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	// SIGKILL if still alive
	if processAlive(pid, bin) {
		_ = proc.Signal(syscall.SIGKILL)
		time.Sleep(500 * time.Millisecond)
	}

	os.Remove(layout.PidFile())

	// Restore Private DNS
	restorePrivateDns(layout)

	// Tun hotspot forwarding cleanup
	if mode := readProxyMode(layout); mode == "tun" && readConfigBool(layout, "tun_forward") {
		config.TunForwardDisable(layout)
	}

	// Cleanup atp rules for tproxy/redirect modes
	mode := readProxyMode(layout)
	if mode == "tproxy" || mode == "redirect" {
		cleanupAtp(layout)
	}

	result.OK(formatJSON, "service.stopped", "服务已停止")
}

// Restart performs stop then start.
func Restart(layout *paths.Layout, formatJSON bool) {
	restartProcess(layout, true)
	Start(layout, formatJSON)
}

func restartProcess(layout *paths.Layout, cleanRules bool) {
	bin := layout.SingBoxBin()
	pid := readPID(layout)
	if pid > 0 && processAlive(pid, bin) {
		proc, _ := os.FindProcess(pid)
		_ = proc.Signal(syscall.SIGTERM)
		time.Sleep(stopGraceTimeout)
		if processAlive(pid, bin) {
			_ = proc.Signal(syscall.SIGKILL)
			time.Sleep(500 * time.Millisecond)
		}
		os.Remove(layout.PidFile())

	}
	if cleanRules {
		cleanupAtp(layout)
	}
}

// Status reports the current service state.
func Status(layout *paths.Layout, formatJSON bool) {
	bin := layout.SingBoxBin()
	pid := readPID(layout)
	running := pid > 0 && processAlive(pid, bin)
	mode := readProxyMode(layout)

	data := StatusData{
		PID:     pid,
		Running: running,
		Mode:    mode,
	}

	if running {
		data.UptimeMs = processUptimeMs(pid)
	}

	if formatJSON {
		result.Text(result.Success("service.status", "服务状态", data), true)
		return
	}

	if running {
		uptime := time.Duration(data.UptimeMs) * time.Millisecond
		fmt.Printf("状态: 运行中\nPID: %d\n模式: %s\n运行时间: %s\n", pid, mode, uptime.Round(time.Second))
	} else {
		fmt.Println("状态: 未运行")
	}
}

// ---- internal helpers ----

func readPID(layout *paths.Layout) int {
	data, err := os.ReadFile(layout.PidFile())
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return pid
}

func writePID(layout *paths.Layout, pid int) {
	if pid <= 0 {
		os.Remove(layout.PidFile())
		return
	}
	_ = os.WriteFile(layout.PidFile(), []byte(strconv.Itoa(pid)), 0644)
}

func processAlive(pid int, expectedBin string) bool {
	cmdlinePath := fmt.Sprintf("/proc/%d/cmdline", pid)
	data, err := os.ReadFile(cmdlinePath)
	if err != nil {
		return false
	}
	cmdline := strings.ReplaceAll(string(data), "\x00", " ")
	return strings.Contains(cmdline, expectedBin)
}

func processUptimeMs(pid int) int64 {
	statPath := fmt.Sprintf("/proc/%d/stat", pid)
	data, err := os.ReadFile(statPath)
	if err != nil {
		return 0
	}
	// Field 22 is starttime in clock ticks
	fields := strings.Fields(string(data))
	if len(fields) < 22 {
		return 0
	}
	startTicks, err := strconv.ParseInt(fields[21], 10, 64)
	if err != nil {
		return 0
	}
	// Convert ticks to ms (100 ticks/sec on Linux)
	uptimeTicks := uptimeTicks() - startTicks
	return uptimeTicks * 10
}

func uptimeTicks() int64 {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	parts := strings.Fields(string(data))
	if len(parts) < 1 {
		return 0
	}
	secs, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0
	}
	return int64(secs * 100) // ticks (100/sec)
}

func readProxyMode(layout *paths.Layout) string {
	// Parse fluxnet.config for proxy_mode=...
	data, err := os.ReadFile(layout.ConfigFile())
	if err != nil {
		return "unknown"
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		if strings.HasPrefix(line, "proxy_mode=") {
			val := strings.TrimPrefix(line, "proxy_mode=")
			val = strings.Trim(val, "\"'")
			return val
		}
	}
	return "unknown"
}

func cleanupAtp(layout *paths.Layout) {
	atpBin := layout.AtpBin()
	if _, err := os.Stat(atpBin); os.IsNotExist(err) {
		return
	}
	tproxyDir := layout.RunTproxyDir()
	if _, err := os.Stat(tproxyDir); os.IsNotExist(err) {
		return
	}
	cmd := exec.Command(atpBin, "-d", tproxyDir, "stop")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}

// readConfigBool reads a boolean-ish config key from fluxnet.config (1/0 or true/false).
func readConfigBool(layout *paths.Layout, key string) bool {
	data, err := os.ReadFile(layout.ConfigFile())
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		if strings.HasPrefix(line, key+"=") {
			val := strings.TrimPrefix(line, key+"=")
			val = strings.Trim(val, "\"'")
			return val == "1" || val == "true"
		}
	}
	return false
}

// ---- Private DNS management ----
// Android Private DNS must be disabled during proxy operation to prevent DNS leaks.
// On start we save the current mode and set it to "off"; on stop we restore it.

func savePrivateDns(layout *paths.Layout) {
	current, err := exec.Command("/system/bin/settings", "get", "global", "private_dns_mode").Output()
	if err != nil {
		return
	}
	mode := strings.TrimSpace(string(current))
	if mode == "" || mode == "null" || mode == "off" {
		return
	}
	os.WriteFile(layout.PrivateDnsStateFile(), []byte(mode), 0600)
	exec.Command("/system/bin/settings", "put", "global", "private_dns_mode", "off").Run()
}

func restorePrivateDns(layout *paths.Layout) {
	data, err := os.ReadFile(layout.PrivateDnsStateFile())
	if err != nil {
		return
	}
	saved := strings.TrimSpace(string(data))
	if saved == "" {
		return
	}
	os.Remove(layout.PrivateDnsStateFile())
	exec.Command("/system/bin/settings", "put", "global", "private_dns_mode", saved).Run()
}
