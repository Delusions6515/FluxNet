package paths

import (
	"os"
	"path/filepath"
)

const (
	defaultModuleDir = "/data/adb/modules/fluxnet"
	defaultDataDir   = "/data/adb/fluxnet"
)

// Layout holds the on-device directory layout for FluxNet.
type Layout struct {
	ModuleDir string
	DataDir   string
}

// New returns a Layout. Zero-value fields fall back to hard-coded defaults.
func New(moduleDir, dataDir string) *Layout {
	l := &Layout{ModuleDir: moduleDir, DataDir: dataDir}
	if l.ModuleDir == "" {
		l.ModuleDir = defaultModuleDir
	}
	if l.DataDir == "" {
		l.DataDir = defaultDataDir
	}
	return l
}

// ---- Module directory paths ----

// FluxNetBin returns the path to the Go CLI binary inside the module.
func (l *Layout) FluxNetBin() string {
	return filepath.Join(l.ModuleDir, "bin", "fluxnet")
}

// InboundDataDir returns the directory containing inbound JSON templates.
func (l *Layout) InboundDataDir() string {
	return filepath.Join(l.DataDir, "config", "inbounds")
}

// InboundData returns the path to a specific inbound template.
func (l *Layout) InboundData(mode string) string {
	inboundPath := filepath.Join(l.InboundDataDir(), mode+".json")
	if fileExists(inboundPath) {
		return inboundPath
	} else {
		return l.InboundTemplate(mode)
	}
}

// UserInbound returns the editable inbound override for a proxy mode.
func (l *Layout) UserInbound(mode string) string {
	return filepath.Join(l.InboundDataDir(), mode+".json")
}

// InboundTemplateDir returns the directory containing inbound JSON templates.
func (l *Layout) InboundTemplateDir() string {
	return filepath.Join(l.ModuleDir, "config", "inbounds", "tpl")
}

// InboundTemplate returns the path to a specific inbound template.
func (l *Layout) InboundTemplate(mode string) string {
	return filepath.Join(l.InboundTemplateDir(), mode+".json")
}

// ModTproxyConf returns the atp config template shipped with the module.
func (l *Layout) ModTproxyConf() string {
	return filepath.Join(l.ModuleDir, "config", "tproxy.conf")
}

// ModFluxNetConfig returns the default fluxnet.config shipped with the module.
func (l *Layout) ModFluxNetConfig() string {
	return filepath.Join(l.ModuleDir, "config", "fluxnet.config")
}

// ScriptsDir returns the module scripts directory.
func (l *Layout) ScriptsDir() string {
	return filepath.Join(l.ModuleDir, "scripts")
}

// ---- Data directory paths ----

// SingBoxBin returns the path to the sing-box kernel binary.
func (l *Layout) SingBoxBin() string {
	return filepath.Join(l.DataDir, "bin", "sing-box")
}

// AtpBin returns the path to the AndroidTProxyShell binary.
func (l *Layout) AtpBin() string {
	return filepath.Join(l.DataDir, "bin", "atp")
}

// ConfigDir returns the data config directory.
func (l *Layout) ConfigDir() string {
	return filepath.Join(l.DataDir, "config")
}

// ConfigFile returns the user settings file (data dir preferred, module fallback).
func (l *Layout) ConfigFile() string {
	dataConfig := filepath.Join(l.ConfigDir(), "fluxnet.config")
	if fileExists(dataConfig) {
		return dataConfig
	}
	return l.ModFluxNetConfig()
}

// LocalConfigDir returns local/ full-config directory.
func (l *Layout) LocalConfigDir() string {
	return filepath.Join(l.ConfigDir(), "local")
}

// RemoteConfigDir returns remote/ subscription config directory.
func (l *Layout) RemoteConfigDir() string {
	return filepath.Join(l.ConfigDir(), "remote")
}

// SubscriptionFile returns the subscription index JSON.
func (l *Layout) SubscriptionFile() string {
	return filepath.Join(l.ConfigDir(), "subscription.json")
}

// TproxyConf returns the atp config (data dir preferred, module fallback).
func (l *Layout) TproxyConf() string {
	dataConf := filepath.Join(l.ConfigDir(), "tproxy.conf")
	if fileExists(dataConf) {
		return dataConf
	}
	return l.ModTproxyConf()
}

// UserTproxyConf returns the editable AndroidTProxyShell configuration path.
func (l *Layout) UserTproxyConf() string {
	return filepath.Join(l.ConfigDir(), "tproxy.conf")
}

// ForceProxyApps returns the force-proxy app list.
func (l *Layout) ForceProxyApps() string {
	return filepath.Join(l.ConfigDir(), "force_proxy_app.txt")
}

// ForceBypassApps returns the force-bypass app list.
func (l *Layout) ForceBypassApps() string {
	return filepath.Join(l.ConfigDir(), "force_bypass_app.txt")
}

// ProxyPackageList returns the cached v2rayNG proxy-package catalogue.
func (l *Layout) ProxyPackageList() string {
	return filepath.Join(l.ConfigDir(), "proxy_package_name")
}

// ---- Runtime paths (under DataDir) ----

// RunDir returns the runtime directory.
func (l *Layout) RunDir() string {
	return filepath.Join(l.DataDir, "run")
}

// RunConfigDir returns the runtime config directory.
func (l *Layout) RunConfigDir() string {
	return filepath.Join(l.ConfigDir(), "run")
}

// RunConfigPath returns the generated run/config.json for sing-box.
func (l *Layout) RunConfigPath() string {
	return filepath.Join(l.RunConfigDir(), "config.json")
}

// RunTproxyDir returns the atp runtime config directory.
func (l *Layout) RunTproxyDir() string {
	return filepath.Join(l.RunConfigDir(), "tproxy")
}

// RunTproxyConf returns the generated atp runtime config file.
func (l *Layout) RunTproxyConf() string {
	return filepath.Join(l.RunTproxyDir(), "tproxy.conf")
}

// PidFile returns the sing-box PID file.
func (l *Layout) PidFile() string {
	return filepath.Join(l.RunDir(), "sing-box.pid")
}

// WorkerPidFile returns the worker PID file.
func (l *Layout) WorkerPidFile() string {
	return filepath.Join(l.RunDir(), "worker.pid")
}

// ServiceRequestFile is the single pending lifecycle request consumed by Worker.
func (l *Layout) ServiceRequestFile() string {
	return filepath.Join(l.RunDir(), "service-request")
}

// ConfigChangedMarker returns the .config-changed marker path.
func (l *Layout) ConfigChangedMarker() string {
	return filepath.Join(l.RunDir(), ".config-changed")
}

// ManualMarker returns the manual start marker path.
func (l *Layout) ManualMarker() string {
	return filepath.Join(l.DataDir, "manual")
}

// ModulesUpdateDir returns the Magisk/KernelSU/APatch modules_update staging dir.
func (l *Layout) ModulesUpdateDir() string {
	return filepath.Join(filepath.Dir(l.ModuleDir), "modules_update")
}

// LogsDir returns the structured logs directory.
func (l *Layout) LogsDir() string {
	return filepath.Join(l.DataDir, "logs")
}

// AtpLog returns the persistent log for AndroidTProxyShell commands.
func (l *Layout) AtpLog() string {
	return filepath.Join(l.LogsDir(), "atp.log")
}

// OperationLog returns the structured FluxNet CLI operation log.
func (l *Layout) OperationLog() string {
	return filepath.Join(l.LogsDir(), "operations.log")
}

// PrivateDnsStateFile returns the backup file for Android Private DNS mode.
func (l *Layout) PrivateDnsStateFile() string {
	return filepath.Join(l.RunDir(), ".private_dns_mode")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
