package subscription

import (
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
		if err := os.WriteFile(dest, data, 0600); err != nil {
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
		if err := os.WriteFile(dest, srcData, 0600); err != nil {
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
	return os.WriteFile(layout.SubscriptionFile(), data, 0600)
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
