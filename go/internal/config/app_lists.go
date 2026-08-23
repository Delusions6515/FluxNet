package config

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Delusions6515/FluxNet/internal/paths"
)

const proxyPackageListURL = "https://raw.githubusercontent.com/2dust/v2rayNG/master/V2rayNG/app/src/main/assets/proxy_package_name"

// UpdateProxyPackageList refreshes the v2rayNG proxy package catalogue.
func UpdateProxyPackageList(layout *paths.Layout) (int, error) {
	return updateProxyPackageList(layout, proxyPackageListURL)
}

func updateProxyPackageList(layout *paths.Layout, url string) (int, error) {
	client := http.Client{Timeout: 30 * time.Second}
	response, err := client.Get(url)
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
	if err := os.MkdirAll(filepath.Dir(layout.ProxyPackageList()), 0755); err != nil {
		return 0, err
	}
	if err := atomicWriteFile(layout.ProxyPackageList(), []byte(strings.Join(packages, "\n")+"\n"), 0600); err != nil {
		return 0, err
	}
	return len(packages), nil
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

func readPackageFile(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return normalizePackageList(string(data)), nil
}

// ReadPackageList returns a normalized package list from a local file.
func ReadPackageList(path string) ([]string, error) {
	return readPackageFile(path)
}
