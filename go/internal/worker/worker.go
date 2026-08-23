package worker

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Delusions6515/FluxNet/internal/paths"
	"github.com/Delusions6515/FluxNet/internal/result"
	"github.com/Delusions6515/FluxNet/internal/service"
)

const (
	modulePollInterval = 3 * time.Second
	hotReloadInterval  = 1 * time.Second
)

var (
	workerProcessAlive = processAlive
	runServiceAction   = runServiceActionNow
)

// Start launches the background worker process.
func Start(layout *paths.Layout, formatJSON bool) {
	fluxnetBin := layout.FluxNetBin()
	cmd := exec.Command(fluxnetBin, "--data-dir", layout.DataDir, "--module-dir", layout.ModuleDir, "worker", "run")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		result.Err(formatJSON, "worker.start_failed", "Worker 启动失败: "+err.Error())
		return
	}

	os.WriteFile(layout.WorkerPidFile(), []byte(strconv.Itoa(cmd.Process.Pid)), 0644)
	result.OK(formatJSON, "worker.started", fmt.Sprintf("Worker 已启动 (PID: %d)", cmd.Process.Pid))
}

// Stop terminates the worker.
func Stop(layout *paths.Layout, formatJSON bool) {
	pid := readWorkerPID(layout)
	if pid > 0 {
		proc, _ := os.FindProcess(pid)
		_ = proc.Signal(syscall.SIGTERM)
		time.Sleep(500 * time.Millisecond)
		if processAlive(pid) {
			_ = proc.Signal(syscall.SIGKILL)
		}
		os.Remove(layout.WorkerPidFile())
	}
	result.OK(formatJSON, "worker.stopped", "Worker 已停止")
}

// Run is the internal worker event loop, invoked by "fluxnet worker run".
func Run(layout *paths.Layout) {
	// Initial startup
	if autostartEnabled(layout) {
		service.Start(layout, false)
	}

	// ---- inotifyd monitors ----
	// 1. Module disable/remove (inotifyd)
	startInotifyModule(layout)

	// 2. Network change → re-apply atp anti-loopback rules (inotifyd)
	startInotifyNet(layout)

	// ---- Main event loop ----
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	// Saved settings remain pending until the user explicitly applies and
	// restarts. Remove markers from older versions to avoid surprise reloads.
	_ = os.Remove(layout.ConfigChangedMarker())
	cfgTicker := time.NewTicker(modulePollInterval)
	defer cfgTicker.Stop()

	hotTicker := time.NewTicker(hotReloadInterval)
	defer hotTicker.Stop()

	for {
		select {
		case <-sigCh:
			return

		case <-cfgTicker.C:
			// Module disable/remove safety net
			if _, err := os.Stat(layout.ModuleDir + "/disable"); err == nil {
				service.Stop(layout, false)
			}
			if _, err := os.Stat(layout.ModuleDir + "/remove"); err == nil {
				service.Stop(layout, false)
			}

		case <-hotTicker.C:
			processServiceRequest(layout)

			// modules_update → wait for swap → restart
			if _, err := os.Stat(layout.ModulesUpdateDir()); err == nil {
				time.Sleep(2 * time.Second)
				service.Restart(layout, false)
			}
		}
	}
}

// RequestServiceAction submits one lifecycle operation to the boot-started
// worker. It returns immediately so a KernelSU WebUI shell never owns the
// restart process.
func RequestServiceAction(layout *paths.Layout, action string) error {
	action = strings.TrimSpace(action)
	if !validServiceAction(action) {
		return fmt.Errorf("不支持的服务操作: %s", action)
	}
	if !workerProcessAlive(readWorkerPID(layout)) {
		return fmt.Errorf("后台 Worker 未运行")
	}
	if err := os.MkdirAll(layout.RunDir(), 0755); err != nil {
		return fmt.Errorf("创建运行目录失败: %w", err)
	}

	request, err := os.CreateTemp(layout.RunDir(), ".service-request-")
	if err != nil {
		return fmt.Errorf("创建服务请求失败: %w", err)
	}
	temporary := request.Name()
	defer os.Remove(temporary)
	if _, err := request.WriteString(action + "\n"); err != nil {
		request.Close()
		return fmt.Errorf("写入服务请求失败: %w", err)
	}
	if err := request.Close(); err != nil {
		return fmt.Errorf("关闭服务请求失败: %w", err)
	}
	if err := os.Link(temporary, layout.ServiceRequestFile()); err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("已有服务操作正在执行")
		}
		return fmt.Errorf("提交服务请求失败: %w", err)
	}
	return nil
}

func processServiceRequest(layout *paths.Layout) {
	data, err := os.ReadFile(layout.ServiceRequestFile())
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "[Warn] 读取服务请求失败: %s\n", err)
		return
	}
	action := strings.TrimSpace(string(data))
	if !validServiceAction(action) {
		fmt.Fprintf(os.Stderr, "[Warn] 忽略无效服务请求: %q\n", action)
		_ = os.Remove(layout.ServiceRequestFile())
		return
	}

	runServiceAction(layout, action)
	_ = os.Remove(layout.ServiceRequestFile())
}

func validServiceAction(action string) bool {
	return action == "start" || action == "stop" || action == "restart"
}

func runServiceActionNow(layout *paths.Layout, action string) {
	switch action {
	case "start":
		service.Start(layout, false)
	case "stop":
		service.Stop(layout, false)
	case "restart":
		service.Restart(layout, false)
	}
}

// ---- inotifyd helpers ----

func startInotifyModule(layout *paths.Layout) {
	script := layout.ScriptsDir() + "/fluxnet.inotify"
	os.WriteFile(script, []byte("d:^((?!disable|remove).)*$:0\nd:disable:1\nd:remove:1\n"), 0755)
	cmd := exec.Command("inotifyd", script, layout.ModuleDir)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	_ = cmd.Start()
}

func startInotifyNet(layout *paths.Layout) {
	netInotify := layout.ScriptsDir() + "/net.inotify"
	cmd := exec.Command("inotifyd", netInotify, "/data/misc/net")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	_ = cmd.Start()
}

// ---- helpers ----

func autostartEnabled(layout *paths.Layout) bool {
	if _, err := os.Stat(layout.ManualMarker()); err == nil {
		return false
	}
	kv := readConfigKV(layout.ConfigFile())
	if v, ok := kv["autostart"]; ok && v == "0" {
		return false
	}
	return true
}

func readWorkerPID(layout *paths.Layout) int {
	data, _ := os.ReadFile(layout.WorkerPidFile())
	pid, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	return pid
}

func processAlive(pid int) bool {
	_, err := os.Stat(fmt.Sprintf("/proc/%d", pid))
	return err == nil
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
