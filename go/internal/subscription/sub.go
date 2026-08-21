package subscription

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Delusions6515/FluxNet/internal/paths"
	"github.com/Delusions6515/FluxNet/internal/result"
)

// Index mirrors the subscription.json structure.
type Index struct {
	Active        string         `json:"active"`
	Subscriptions []Subscription `json:"subscriptions"`
}

// Subscription is a single config source entry.
type Subscription struct {
	Name      string `json:"name"`
	Type      string `json:"type"` // "local" or "remote"
	Filename  string `json:"filename"`
	URL       string `json:"url,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// Add adds a new subscription from a URL or local file path.
func Add(layout *paths.Layout, formatJSON bool, urlOrPath, name string) {
	idx, err := loadIndex(layout)
	if err != nil {
		result.Err(formatJSON, "subscription.load_failed", "读取订阅索引失败: "+err.Error())
		return
	}

	if name == "" {
		// Derive name from URL basename
		name = filepath.Base(urlOrPath)
		name = strings.TrimSuffix(name, filepath.Ext(name))
	}
	if name == "" {
		result.Err(formatJSON, "subscription.invalid_name", "订阅名称不能为空")
		return
	}
	if !validName(name) {
		result.Err(formatJSON, "subscription.invalid_name", "订阅名称只能包含字母、数字、点、下划线和连字符")
		return
	}

	// Check duplicate
	for _, s := range idx.Subscriptions {
		if s.Name == name {
			result.Err(formatJSON, "subscription.duplicate", "订阅名称已存在: "+name)
			return
		}
	}

	isURL := strings.HasPrefix(urlOrPath, "http://") || strings.HasPrefix(urlOrPath, "https://")
	subType := "local"
	filename := name + ".json"

	if isURL {
		subType = "remote"
		// Download
		data, err := download(urlOrPath)
		if err != nil {
			result.Err(formatJSON, "subscription.download_failed", "下载失败: "+err.Error())
			return
		}

		dest := filepath.Join(layout.RemoteConfigDir(), filename)
		if err := os.MkdirAll(layout.RemoteConfigDir(), 0755); err != nil {
			result.Err(formatJSON, "subscription.write_failed", "创建目录失败: "+err.Error())
			return
		}
		if err := atomicWriteFile(dest, data, 0600); err != nil {
			result.Err(formatJSON, "subscription.write_failed", "写入配置失败: "+err.Error())
			return
		}
	} else {
		// Local file: copy to local/
		srcData, err := os.ReadFile(urlOrPath)
		if err != nil {
			result.Err(formatJSON, "subscription.read_failed", "读取文件失败: "+err.Error())
			return
		}
		dest := filepath.Join(layout.LocalConfigDir(), filename)
		if err := os.MkdirAll(layout.LocalConfigDir(), 0755); err != nil {
			result.Err(formatJSON, "subscription.write_failed", "创建目录失败: "+err.Error())
			return
		}
		if err := atomicWriteFile(dest, srcData, 0600); err != nil {
			result.Err(formatJSON, "subscription.write_failed", "写入配置失败: "+err.Error())
			return
		}
	}

	idx.Subscriptions = append(idx.Subscriptions, Subscription{
		Name:      name,
		Type:      subType,
		Filename:  filename,
		URL:       urlOrPath,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	})

	if err := saveIndex(layout, idx); err != nil {
		result.Err(formatJSON, "subscription.save_failed", "保存索引失败: "+err.Error())
		return
	}

	result.OK(formatJSON, "subscription.added", "订阅已添加: "+name)
}

// CreateLocal creates an editable local sing-box configuration.
func CreateLocal(layout *paths.Layout, formatJSON bool, name string) {
	idx, err := loadIndex(layout)
	if err != nil {
		result.Err(formatJSON, "subscription.load_failed", "读取订阅索引失败: "+err.Error())
		return
	}
	if !validName(name) {
		result.Err(formatJSON, "subscription.invalid_name", "订阅名称只能包含字母、数字、点、下划线和连字符")
		return
	}
	if _, found := find(idx, name); found {
		result.Err(formatJSON, "subscription.duplicate", "订阅名称已存在: "+name)
		return
	}
	if err := os.MkdirAll(layout.LocalConfigDir(), 0755); err != nil {
		result.Err(formatJSON, "subscription.write_failed", "创建目录失败: "+err.Error())
		return
	}
	content := []byte("{\n  \"log\": { \"level\": \"info\" },\n  \"inbounds\": [],\n  \"outbounds\": [{ \"type\": \"direct\", \"tag\": \"direct\" }],\n  \"route\": { \"rules\": [], \"final\": \"direct\" }\n}\n")
	filename := name + ".json"
	if err := atomicWriteFile(filepath.Join(layout.LocalConfigDir(), filename), content, 0600); err != nil {
		result.Err(formatJSON, "subscription.write_failed", "写入配置失败: "+err.Error())
		return
	}
	idx.Subscriptions = append(idx.Subscriptions, Subscription{Name: name, Type: "local", Filename: filename, UpdatedAt: time.Now().UTC().Format(time.RFC3339)})
	if err := saveIndex(layout, idx); err != nil {
		result.Err(formatJSON, "subscription.save_failed", "保存索引失败: "+err.Error())
		return
	}
	result.Text(result.Success("subscription.local_created", "本地订阅已创建: "+name, map[string]any{"name": name, "content": string(content)}), formatJSON)
}

func ReadLocal(layout *paths.Layout, formatJSON bool, name string) {
	idx, err := loadIndex(layout)
	if err != nil {
		result.Err(formatJSON, "subscription.load_failed", "读取订阅索引失败: "+err.Error())
		return
	}
	sub, found := find(idx, name)
	if !found || sub.Type != "local" {
		result.Err(formatJSON, "subscription.not_local", "未找到本地订阅: "+name)
		return
	}
	data, err := os.ReadFile(filepath.Join(layout.LocalConfigDir(), sub.Filename))
	if err != nil {
		result.Err(formatJSON, "subscription.read_failed", "读取本地订阅失败: "+err.Error())
		return
	}
	result.Text(result.Success("subscription.local_read", "本地订阅", map[string]any{"name": name, "content": string(data)}), formatJSON)
}

func WriteLocal(layout *paths.Layout, formatJSON bool, name, encoded string) {
	idx, err := loadIndex(layout)
	if err != nil {
		result.Err(formatJSON, "subscription.load_failed", "读取订阅索引失败: "+err.Error())
		return
	}
	sub, found := find(idx, name)
	if !found || sub.Type != "local" {
		result.Err(formatJSON, "subscription.not_local", "未找到本地订阅: "+name)
		return
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		result.Err(formatJSON, "subscription.invalid_json", "JSON 编码无效")
		return
	}
	var document any
	if err := json.Unmarshal(data, &document); err != nil {
		result.Err(formatJSON, "subscription.invalid_json", "JSON 格式无效: "+err.Error())
		return
	}
	if _, ok := document.(map[string]any); !ok {
		result.Err(formatJSON, "subscription.invalid_json", "订阅必须是 JSON 对象")
		return
	}
	if err := atomicWriteFile(filepath.Join(layout.LocalConfigDir(), sub.Filename), data, 0600); err != nil {
		result.Err(formatJSON, "subscription.write_failed", "写入本地订阅失败: "+err.Error())
		return
	}
	for i := range idx.Subscriptions {
		if idx.Subscriptions[i].Name == name {
			idx.Subscriptions[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		}
	}
	if err := saveIndex(layout, idx); err != nil {
		result.Err(formatJSON, "subscription.save_failed", "保存索引失败: "+err.Error())
		return
	}
	result.OK(formatJSON, "subscription.local_written", "本地订阅已保存")
}

// Update refreshes a remote subscription by name, or all if name is empty.
func Update(layout *paths.Layout, formatJSON bool, name string) {
	idx, err := loadIndex(layout)
	if err != nil {
		result.Err(formatJSON, "subscription.load_failed", "读取订阅索引失败: "+err.Error())
		return
	}

	updated := 0
	for i, s := range idx.Subscriptions {
		if name != "" && s.Name != name {
			continue
		}
		if s.Type != "remote" {
			continue
		}

		data, err := download(s.URL)
		if err != nil {
			if name != "" {
				result.Err(formatJSON, "subscription.download_failed", "下载失败: "+err.Error())
				return
			}
			fmt.Fprintf(os.Stderr, "[Warn] 更新 %s 失败: %s\n", s.Name, err)
			continue
		}

		dest := filepath.Join(layout.RemoteConfigDir(), s.Filename)
		if err := os.WriteFile(dest, data, 0600); err != nil {
			fmt.Fprintf(os.Stderr, "[Warn] 写入 %s 失败: %s\n", s.Name, err)
			continue
		}

		idx.Subscriptions[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		updated++
	}

	if err := saveIndex(layout, idx); err != nil {
		result.Err(formatJSON, "subscription.save_failed", "保存索引失败: "+err.Error())
		return
	}

	if name != "" && updated == 0 {
		result.Err(formatJSON, "subscription.not_found", "未找到远程订阅: "+name)
		return
	}

	result.OK(formatJSON, "subscription.updated", fmt.Sprintf("已更新 %d 个订阅", updated))
}

// List prints all subscriptions.
func List(layout *paths.Layout, formatJSON bool) {
	idx, err := loadIndex(layout)
	if err != nil {
		result.Err(formatJSON, "subscription.load_failed", "读取订阅索引失败: "+err.Error())
		return
	}

	if formatJSON {
		result.Text(result.Success("subscription.list", "订阅列表", idx), true)
		return
	}

	if len(idx.Subscriptions) == 0 {
		fmt.Println("(无订阅)")
		return
	}

	fmt.Printf("%-4s %-20s %-8s %s\n", "", "名称", "类型", "URL/路径")
	for i, s := range idx.Subscriptions {
		marker := " "
		if s.Name == idx.Active {
			marker = "*"
		}
		url := s.URL
		if url == "" {
			url = "(本地)"
		}
		fmt.Printf("%-4s %-20s %-8s %s\n", marker, s.Name, s.Type, url)
		_ = i
	}
}

// Remove deletes a subscription by name.
func Remove(layout *paths.Layout, formatJSON bool, name string) {
	idx, err := loadIndex(layout)
	if err != nil {
		result.Err(formatJSON, "subscription.load_failed", "读取订阅索引失败: "+err.Error())
		return
	}

	if name == idx.Active {
		result.Err(formatJSON, "subscription.active_remove", "不能删除当前活跃的配置，请先切换到其他配置")
		return
	}

	found := false
	var newList []Subscription
	for _, s := range idx.Subscriptions {
		if s.Name == name {
			found = true
			// Remove the file
			var filePath string
			switch s.Type {
			case "local":
				filePath = filepath.Join(layout.LocalConfigDir(), s.Filename)
			case "remote":
				filePath = filepath.Join(layout.RemoteConfigDir(), s.Filename)
			}
			os.Remove(filePath)
		} else {
			newList = append(newList, s)
		}
	}

	if !found {
		result.Err(formatJSON, "subscription.not_found", "未找到订阅: "+name)
		return
	}

	idx.Subscriptions = newList
	if err := saveIndex(layout, idx); err != nil {
		result.Err(formatJSON, "subscription.save_failed", "保存索引失败: "+err.Error())
		return
	}

	result.OK(formatJSON, "subscription.removed", "订阅已删除: "+name)
}

// Switch changes the active subscription and triggers config apply.
func Switch(layout *paths.Layout, formatJSON bool, name string) {
	idx, err := loadIndex(layout)
	if err != nil {
		result.Err(formatJSON, "subscription.load_failed", "读取订阅索引失败: "+err.Error())
		return
	}

	found := false
	for _, s := range idx.Subscriptions {
		if s.Name == name {
			found = true
			break
		}
	}
	if !found {
		result.Err(formatJSON, "subscription.not_found", "未找到订阅: "+name)
		return
	}

	idx.Active = name
	if err := saveIndex(layout, idx); err != nil {
		result.Err(formatJSON, "subscription.save_failed", "保存索引失败: "+err.Error())
		return
	}

	result.OK(formatJSON, "subscription.switched", "已切换到: "+name)
}

// ---- internal helpers ----

func loadIndex(layout *paths.Layout) (*Index, error) {
	path := layout.SubscriptionFile()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Index{Active: "default", Subscriptions: []Subscription{
				{Name: "default", Type: "local", Filename: "default.json"},
			}}, nil
		}
		return nil, err
	}

	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, err
	}
	return &idx, nil
}

func saveIndex(layout *paths.Layout, idx *Index) error {
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteFile(layout.SubscriptionFile(), data, 0600)
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

func find(idx *Index, name string) (Subscription, bool) {
	for _, sub := range idx.Subscriptions {
		if sub.Name == name {
			return sub, true
		}
	}
	return Subscription{}, false
}

func validName(name string) bool {
	if name == "" || strings.Contains(name, "..") {
		return false
	}
	for _, r := range name {
		if !(r == '.' || r == '_' || r == '-' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

func download(url string) ([]byte, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
