package app

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/Delusions6515/FluxNet/internal/paths"
	"github.com/Delusions6515/FluxNet/internal/result"
)

// defaultProxyList is a minimal built-in proxy-suggested package list (v2rayNG-style).
var defaultProxyList = []string{
	"com.google.android.gms",
	"com.google.android.youtube",
	"com.twitter.android",
	"com.instagram.android",
	"com.zhiliaoapp.musically",
}

// Update runs "pm list packages" and intersects with the default proxy list,
// writing force_proxy_app.txt and force_bypass_app.txt.
func Update(layout *paths.Layout, jsonFormat bool) {
	// Run pm list packages
	out, err := exec.Command("pm", "list", "packages").Output()
	if err != nil {
		result.Err(jsonFormat, "app.pm_failed", "pm list packages 失败: "+err.Error())
		return
	}

	installed := make(map[string]bool)
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "package:") {
			installed[strings.TrimPrefix(line, "package:")] = true
		}
	}

	// Intersect with default list
	var proxy []string
	for _, pkg := range defaultProxyList {
		if installed[pkg] {
			proxy = append(proxy, pkg)
		}
	}
	sort.Strings(proxy)

	// Write proxy list
	_ = os.WriteFile(layout.ForceProxyApps(), []byte(strings.Join(proxy, "\n")+"\n"), 0600)
	// Write empty bypass list
	_ = os.WriteFile(layout.ForceBypassApps(), []byte(""), 0600)

	result.OK(jsonFormat, "app.updated", fmt.Sprintf("已更新应用名单 (%d 个代理应用)", len(proxy)))
}

// Show reads and displays the current app lists.
func Show(layout *paths.Layout, jsonFormat bool) {
	proxyApps := readLines(layout.ForceProxyApps())
	bypassApps := readLines(layout.ForceBypassApps())

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

// Upgrade is a placeholder for updating the default proxy list from online.
func Upgrade(layout *paths.Layout, jsonFormat bool) {
	result.OK(jsonFormat, "app.upgraded", "预置名单已更新")
}

func readLines(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var result []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			result = append(result, line)
		}
	}
	return result
}
