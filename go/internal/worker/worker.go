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

	"github.com/Delusions6515/FluxNet/internal/config"
	"github.com/Delusions6515/FluxNet/internal/paths"
	"github.com/Delusions6515/FluxNet/internal/result"
	"github.com/Delusions6515/FluxNet/internal/service"
)

const (
	pollInterval = 5 * time.Second
	debounceDur  = 3 * time.Second
)

// Start launches the background worker process.
func Start(layout *paths.Layout, formatJSON bool) {
	// Fork worker
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
		config.Apply(layout, false)
		service.Start(layout, false)
	}

	// Setup inotifyd for module dir monitoring
	startInotify(layout)

	// Main event loop
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	var lastConfigChange time.Time
	var lastNetChange time.Time

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-sigCh:
			return
		case <-ticker.C:
			// Module monitoring: disable → stop; remove → stop + mark
			if _, err := os.Stat(layout.ModuleDir + "/disable"); err == nil {
				service.Stop(layout, false)
			}
			if _, err := os.Stat(layout.ModuleDir + "/remove"); err == nil {
				service.Stop(layout, false)
				// Mark removed so user is aware
			}

			// Config monitoring: detect change → write .config-changed marker
			checkConfigChange(layout, &lastConfigChange)

			// Hot-reload: .config-changed → config apply → service restart (debounce 3s)
			if _, err := os.Stat(layout.ConfigChangedMarker()); err == nil {
				if time.Since(lastConfigChange) > debounceDur {
					config.Apply(layout, false)
					service.Restart(layout, false)
					os.Remove(layout.ConfigChangedMarker())
					lastConfigChange = time.Now()
				}
			}

			// Hot-reload: modules_update → wait for swap → config apply → restart
			if _, err := os.Stat(layout.ModulesUpdateDir()); err == nil {
				time.Sleep(2 * time.Second)
				config.Apply(layout, false)
				service.Restart(layout, false)
			}

			// Network change monitoring (poll /data/misc/net/rt_tables mtime)
			if fi, err := os.Stat("/data/misc/net/rt_tables"); err == nil {
				if fi.ModTime().After(lastNetChange) {
					lastNetChange = fi.ModTime()
					// Re-apply atp rules for tproxy/redirect
					mode := readProxyMode(layout)
					if mode == "tproxy" || mode == "redirect" {
						cleanupAtp(layout)
						applyAtp(layout)
					}
				}
			}
		}
	}
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

func checkConfigChange(layout *paths.Layout, last *time.Time) {
	// Watch sing-box.config and tproxy.conf for changes
	paths := []string{layout.ConfigFile(), layout.TproxyConf()}
	for _, p := range paths {
		fi, err := os.Stat(p)
		if err != nil {
			continue
		}
		if fi.ModTime().After(*last) {
			if time.Since(*last) > debounceDur {
				*last = fi.ModTime()
				// Write .config-changed marker — hot-reload logic in a later commit
				os.WriteFile(layout.ConfigChangedMarker(), []byte("1"), 0644)
			}
		}
	}
}

func startInotify(layout *paths.Layout) {
	// Write inotify script for module dir monitoring
	inotifyScript := fmt.Sprintf("%s/scripts/fluxnet.inotify", layout.ScriptsDir())
	os.WriteFile(inotifyScript, []byte("d:^((?!disable|remove).)*$:0\nd:disable:1\nd:remove:1\n"), 0755)

	cmd := exec.Command("inotifyd", inotifyScript, layout.ModuleDir)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	_ = cmd.Start()
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

func cleanupAtp(layout *paths.Layout) {
	atpBin := layout.AtpBin()
	if _, err := os.Stat(atpBin); os.IsNotExist(err) {
		return
	}
	cmd := exec.Command(atpBin, "-d", layout.RunTproxyDir(), "stop")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}

func applyAtp(layout *paths.Layout) {
	atpBin := layout.AtpBin()
	if _, err := os.Stat(atpBin); os.IsNotExist(err) {
		return
	}
	cmd := exec.Command(atpBin, "-d", layout.RunTproxyDir(), "start")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}