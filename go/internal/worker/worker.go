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
			// modules_update → wait for swap → restart
			if _, err := os.Stat(layout.ModulesUpdateDir()); err == nil {
				time.Sleep(2 * time.Second)
				service.Restart(layout, false)
			}
		}
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
