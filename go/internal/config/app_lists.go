package config

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/Delusions6515/FluxNet/internal/paths"
)

const proxyPackageListURL = "https://raw.githubusercontent.com/2dust/v2rayNG/master/V2rayNG/app/src/main/assets/proxy_package_name"

// ListInstalledPackages returns packages for Android's foreground user.
func ListInstalledPackages() ([]string, error) {
	user := currentAndroidUser()
	output, err := exec.Command("pm", "list", "packages", "--user", strconv.Itoa(user)).Output()
	if err != nil {
		output, err = exec.Command("pm", "list", "packages").Output()
		if err != nil {
			return nil, fmt.Errorf("pm list packages failed: %w", err)
		}
	}
	packages := normalizePackageList(string(output))
	if len(packages) == 0 {
		return nil, fmt.Errorf("未获取到已安装应用")
	}
	return packages, nil
}

// UpdateProxyPackageList refreshes the v2rayNG proxy package catalogue.
func UpdateProxyPackageList(layout *paths.Layout) (int, error) {
	client := http.Client{Timeout: 30 * time.Second}
	response, err := client.Get(proxyPackageListURL)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("下载代理应用名单失败: %s", response.Status)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return 0, err
	}
	packages := normalizePackageList(string(data))
	if len(packages) == 0 {
		return 0, fmt.Errorf("下载的代理应用名单为空或格式无效")
	}
	if err := atomicWriteFile(layout.ProxyPackageList(), []byte(strings.Join(packages, "\n")+"\n"), 0600); err != nil {
		return 0, err
	}
	return len(packages), nil
}

func currentAndroidUser() int {
	for _, command := range [][]string{{"cmd", "activity", "get-current-user"}, {"am", "get-current-user"}} {
		output, err := exec.Command(command[0], command[1:]...).Output()
		if err != nil {
			continue
		}
		user, err := strconv.Atoi(strings.TrimSpace(string(output)))
		if err == nil && user >= 0 {
			return user
		}
	}
	return 0
}

func normalizePackageList(content string) []string {
	seen := make(map[string]struct{})
	packages := make([]string, 0)
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "package:")
		if colon := strings.IndexByte(line, ':'); colon > 0 {
			if _, err := strconv.Atoi(line[:colon]); err == nil {
				line = line[colon+1:]
			}
		}
		for _, candidate := range strings.Fields(line) {
			if !validPackageName(candidate) {
				continue
			}
			if _, ok := seen[candidate]; ok {
				continue
			}
			seen[candidate] = struct{}{}
			packages = append(packages, candidate)
		}
	}
	return packages
}

func automaticAppLists(layout *paths.Layout) (proxy, bypass []string, err error) {
	catalog, err := readPackageFile(layout.ProxyPackageList())
	if err != nil || len(catalog) == 0 {
		return nil, nil, fmt.Errorf("缺少代理应用预置名单，请先执行 app-list upgrade")
	}
	installed, err := ListInstalledPackages()
	if err != nil {
		return nil, nil, err
	}
	catalogSet := make(map[string]struct{}, len(catalog))
	for _, name := range catalog {
		catalogSet[name] = struct{}{}
	}
	for _, name := range installed {
		if _, ok := catalogSet[name]; ok {
			proxy = append(proxy, name)
		} else {
			bypass = append(bypass, name)
		}
	}
	return proxy, bypass, nil
}

func readPackageFile(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return normalizePackageList(string(data)), nil
}
