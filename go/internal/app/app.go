package app

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/Delusions6515/FluxNet/internal/config"
	"github.com/Delusions6515/FluxNet/internal/paths"
	"github.com/Delusions6515/FluxNet/internal/result"
)

// Update refreshes the v2rayNG proxy-package catalogue.
func Update(layout *paths.Layout, jsonFormat bool) {
	Upgrade(layout, jsonFormat)
}

// Show reads and displays the current app lists.
func Show(layout *paths.Layout, jsonFormat bool) {
	settings := config.ReadSettings(layout)
	proxyApps := settings.ForceProxyApps
	bypassApps := settings.ForceBypassApps

	data := map[string]any{
		"proxy":  proxyApps,
		"bypass": bypassApps,
	}

	if jsonFormat {
		result.Text(result.Success("app.list", "应用名单", data), true)
		return
	}

	fmt.Println("代理应用:")
	if len(proxyApps) == 0 {
		fmt.Println("  (无)")
	} else {
		for _, p := range proxyApps {
			fmt.Printf("  %s\n", p)
		}
	}
	fmt.Println("\n绕过应用:")
	if len(bypassApps) == 0 {
		fmt.Println("  (无)")
	} else {
		for _, p := range bypassApps {
			fmt.Printf("  %s\n", p)
		}
	}
}

// Upgrade downloads the latest v2rayNG proxy package catalogue.
func Upgrade(layout *paths.Layout, jsonFormat bool) {
	count, err := config.UpdateProxyPackageList(layout)
	if err != nil {
		result.Err(jsonFormat, "app.upgrade_failed", err.Error())
		return
	}
	result.OK(jsonFormat, "app.upgraded", fmt.Sprintf("预置名单已更新 (%d 个包名)", count))
}

// Replace updates the user-managed application list used by the WebUI.
// The list is base64-encoded JSON so package names never become shell syntax.
func Replace(layout *paths.Layout, jsonFormat bool, mode, encoded string) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		result.Err(jsonFormat, "app.invalid_input", "应用名单编码无效")
		return
	}
	var apps []string
	if err := json.Unmarshal(data, &apps); err != nil {
		result.Err(jsonFormat, "app.invalid_input", "应用名单必须是 JSON 数组")
		return
	}
	settings, err := config.ReplaceAppList(layout, mode, apps)
	if err != nil {
		result.Err(jsonFormat, "app.write_failed", err.Error())
		return
	}
	result.Text(result.Success("app.replaced", "应用名单已保存", settings), jsonFormat)
}

// ReplaceForce updates one user-managed force list.
func ReplaceForce(layout *paths.Layout, jsonFormat bool, kind, encoded string) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		result.Err(jsonFormat, "app.invalid_input", "应用名单编码无效")
		return
	}
	var apps []string
	if err := json.Unmarshal(data, &apps); err != nil {
		result.Err(jsonFormat, "app.invalid_input", "应用名单必须是 JSON 数组")
		return
	}
	settings, err := config.ReplaceForceAppList(layout, kind, apps)
	if err != nil {
		result.Err(jsonFormat, "app.write_failed", err.Error())
		return
	}
	result.Text(result.Success("app.force_replaced", "强制应用名单已保存", settings), jsonFormat)
}

// Installed returns package names through the module CLI, not the WebUI shell bridge.
func Installed(jsonFormat bool) {
	packages, err := config.ListInstalledPackages()
	if err != nil {
		result.Err(jsonFormat, "app.pm_failed", err.Error())
		return
	}
	apps := make([]map[string]string, 0, len(packages))
	for _, packageName := range packages {
		apps = append(apps, map[string]string{"packageName": packageName, "appLabel": packageName})
	}
	result.Text(result.Success("app.installed", "已安装应用", map[string]any{"apps": apps}), jsonFormat)
}
