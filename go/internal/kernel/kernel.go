package kernel

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Delusions6515/FluxNet/internal/paths"
)

// Channels is the fixed upstream catalogue, kept in sync with build.sh.
var Channels = map[string]string{
	"delusions6515-pre":    "Delusions6515/sing-box-releases",
	"delusions6515-stable": "Delusions6515/sing-box-releases",
	"ref1nd-pre":           "reF1nd/sing-box-releases",
	"ref1nd-stable":        "reF1nd/sing-box-releases",
	"official-stable":      "SagerNet/sing-box",
	"official-pre":         "SagerNet/sing-box",
}

const defaultChannel = "delusions6515-pre"
const defaultABI = "arm64-v8a"

// ChannelInfo describes the current kernel configuration and binary state.
type ChannelInfo struct {
	Channel   string `json:"channel"`
	ABI       string `json:"abi"`
	Version   string `json:"version"`
	Installed bool   `json:"installed"`
}

// Status reads the configured channel/ABI and the recorded kernel version.
func Status(layout *paths.Layout) ChannelInfo {
	info := ChannelInfo{Channel: defaultChannel, ABI: defaultABI}
	kv := readConfigKV(layout.ConfigFile())
	if v, ok := kv["kernel_channel"]; ok && Channels[v] != "" {
		info.Channel = v
	}
	if v, ok := kv["kernel_abi"]; ok && v != "" {
		info.ABI = v
	}
	if v, ok := kv["kernel_version"]; ok && v != "" {
		info.Version = v
	}
	info.Installed = fileExists(layout.SingBoxBin())
	if info.Installed && info.Version == "" {
		info.Version = kernelVersion(layout.SingBoxBin())
	}
	return info
}

// SetChannel persists kernel_channel/kernel_abi into the user config.
func SetChannel(layout *paths.Layout, channel, abi string) error {
	if Channels[channel] == "" {
		return fmt.Errorf("不支持的内核渠道: %s", channel)
	}
	if abi != "" && abi != "arm64-v8a" {
		return fmt.Errorf("不支持的 ABI: %s", abi)
	}
	if err := writeConfigValue(layout.ConfigFile(), "kernel_channel", channel, true); err != nil {
		return err
	}
	if err := writeConfigValue(layout.ConfigFile(), "kernel_abi", abi, true); err != nil {
		return err
	}
	return nil
}

// Install downloads the sing-box binary for the configured channel/ABI and
// atomically replaces the running kernel.
func Install(layout *paths.Layout) (string, error) {
	kv := readConfigKV(layout.ConfigFile())
	channel := kv["kernel_channel"]
	if Channels[channel] == "" {
		channel = defaultChannel
	}
	abi := kv["kernel_abi"]
	if abi == "" {
		abi = defaultABI
	}
	arch := abiToArch(abi)
	if arch == "" {
		return "", fmt.Errorf("不支持的 ABI: %s", abi)
	}

	version, _, assetURL := latestRelease(Channels[channel], strings.HasSuffix(channel, "-pre"), arch)
	if version == "" {
		return "", fmt.Errorf("获取 %s 最新版本失败", channel)
	}

	fmt.Fprintf(os.Stderr, "[kernel] 下载 sing-box %s (%s, %s) ...\n", version, channel, abi)
	data, err := download(assetURL)
	if err != nil {
		return "", fmt.Errorf("下载 %s 失败: %w", assetURL, err)
	}
	bin, err := extractBinary(data)
	if err != nil {
		return "", fmt.Errorf("解包内核失败: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(layout.SingBoxBin()), 0755); err != nil {
		return "", err
	}
	tmp := layout.SingBoxBin() + ".new"
	if err := os.WriteFile(tmp, bin, 0755); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, layout.SingBoxBin()); err != nil {
		os.Remove(tmp)
		return "", err
	}
	if err := writeConfigValue(layout.ConfigFile(), "kernel_version", version, false); err != nil {
		return "", err
	}
	return version, nil
}

// Verify runs sing-box check against the current runtime config when one
// exists, so a broken download is caught before it is put into service.
func Verify(layout *paths.Layout) error {
	bin := layout.SingBoxBin()
	if !fileExists(bin) {
		return fmt.Errorf("sing-box 内核不存在")
	}
	if out, err := exec.Command(bin, "version").CombinedOutput(); err != nil {
		return fmt.Errorf("内核不可执行: %s", strings.TrimSpace(string(out)))
	}
	if !fileExists(layout.RunConfigPath()) {
		return nil
	}
	if out, err := exec.Command(bin, "check", "-c", layout.RunConfigPath()).CombinedOutput(); err != nil {
		return fmt.Errorf("新内核 check 未通过: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func abiToArch(abi string) string {
	if abi == "arm64-v8a" {
		return "arm64"
	}
	return ""
}

func kernelVersion(bin string) string {
	out, err := exec.Command(bin, "version").Output()
	if err != nil {
		return ""
	}
	fields := strings.Fields(string(out))
	if len(fields) == 0 {
		return ""
	}
	v := fields[0]
	return strings.TrimPrefix(v, "v")
}

// latestRelease resolves the release tag and the android asset URL following
// build.sh's selection logic.
func latestRelease(repo string, wantPre bool, arch string) (version, tag, assetURL string) {
	if !wantPre {
		if rt := githubLatestTag(repo); rt != "" {
			tag = rt
		}
	}
	if tag == "" {
		tag = buildListTag(repo, wantPre)
	}
	if tag == "" {
		return "", "", ""
	}
	version = strings.TrimPrefix(tag, "v")

	asset := fmt.Sprintf("sing-box-%s-android-%s.tar.gz", version, arch)
	for _, cand := range []string{
		asset,
		fmt.Sprintf("sing-box-release-%s-android-%s.tar.gz", version, arch),
	} {
		u := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", repo, tag, cand)
		if assetExists(u) {
			return version, tag, u
		}
	}
	return "", "", ""
}

func githubLatestTag(repo string) string {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(fmt.Sprintf("https://github.com/%s/releases/latest", repo))
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	tag := resp.Request.URL.Path
	if i := strings.LastIndex(tag, "/"); i >= 0 {
		tag = tag[i+1:]
	}
	if tag == "" || tag == "latest" {
		return ""
	}
	return tag
}

// buildListTag fetches the first 20 releases and picks the first whose
// prerelease flag matches wantPre (mirrors build.sh's pre channel logic).
func buildListTag(repo string, wantPre bool) string {
	resp, err := http.Get(fmt.Sprintf("https://api.github.com/repos/%s/releases?per_page=20", repo))
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var releases []struct {
		Tag        string `json:"tag_name"`
		Prerelease bool   `json:"prerelease"`
	}
	if err := json.Unmarshal(body, &releases); err != nil {
		return ""
	}
	for _, r := range releases {
		if r.Tag != "" && r.Prerelease == wantPre {
			return r.Tag
		}
	}
	return ""
}

func assetExists(url string) bool {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Head(url)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func download(url string) ([]byte, error) {
	client := &http.Client{Timeout: 600 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 200<<20))
}

func extractBinary(data []byte) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	tarR := tar.NewReader(gz)
	for {
		hdr, err := tarR.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		if filepath.Base(hdr.Name) == "sing-box" && strings.Contains(filepath.Dir(hdr.Name), "sing-box") {
			return io.ReadAll(io.LimitReader(tarR, 200<<20))
		}
	}
	return nil, fmt.Errorf("压缩包内未找到 sing-box 二进制")
}

// ---- config helpers (mirror internal/config writers) ----

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

type configValue struct {
	value string
	quote bool
}

func writeConfigValue(file, key, value string, quote bool) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	data, err = setConfigValue(data, key, value, quote)
	if err != nil {
		return err
	}
	return atomicWriteFile(file, data, 0600)
}

func setConfigValue(data []byte, key, value string, quote bool) ([]byte, error) {
	lines := strings.Split(string(data), "\n")
	assignment := key + "=" + value
	if quote {
		assignment = fmt.Sprintf("%s=%q", key, value)
	}
	found := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		k, _, ok := strings.Cut(trimmed, "=")
		if !ok {
			continue
		}
		if k == key {
			lines[i] = assignment
			found = true
		}
	}
	if !found {
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = append(lines[:len(lines)-1], assignment, "")
		} else {
			lines = append(lines, assignment)
		}
	}
	return []byte(strings.Join(lines, "\n")), nil
}

func atomicWriteFile(file string, data []byte, mode os.FileMode) error {
	temp, err := os.CreateTemp(filepath.Dir(file), ".fluxnet-*")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if err := temp.Chmod(mode); err != nil {
		temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempName, file)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}