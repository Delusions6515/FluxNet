package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Delusions6515/FluxNet/internal/paths"
)

func ReadUserInbound(layout *paths.Layout, mode string) (string, error) {
	if !validProxyMode(mode) {
		return "", fmt.Errorf("不支持的代理模式: %s", mode)
	}
	userFile := layout.UserInbound(mode)
	data, err := os.ReadFile(userFile)
	if err == nil {
		return string(data), nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}

	data, err = os.ReadFile(layout.InboundTemplate(mode))
	if err != nil {
		return "", err
	}
	if err := validateInbound(data); err != nil {
		return "", fmt.Errorf("入站模板无效: %w", err)
	}
	return string(data), nil
}

func WriteUserInbound(layout *paths.Layout, mode string, data []byte) error {
	if !validProxyMode(mode) {
		return fmt.Errorf("不支持的代理模式: %s", mode)
	}
	if err := validateInbound(data); err != nil {
		return err
	}
	if err := os.MkdirAll(layout.InboundDataDir(), 0755); err != nil {
		return err
	}
	return atomicWriteFile(layout.UserInbound(mode), data, 0600)
}

func ReadUserTproxyConf(layout *paths.Layout) (string, error) {
	data, err := os.ReadFile(layout.TproxyConf())
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func WriteUserTproxyConf(layout *paths.Layout, data []byte) error {
	if err := os.MkdirAll(layout.ConfigDir(), 0755); err != nil {
		return err
	}
	return atomicWriteFile(layout.UserTproxyConf(), data, 0600)
}

func validProxyMode(mode string) bool {
	return mode == "tun" || mode == "tproxy" || mode == "redirect" || mode == "ebpf"
}

func validateInbound(data []byte) error {
	var inbound map[string]json.RawMessage
	if err := json.Unmarshal(data, &inbound); err != nil {
		return fmt.Errorf("入站 JSON 无效: %w", err)
	}
	if inbound == nil {
		return fmt.Errorf("入站必须是 JSON 对象")
	}
	return nil
}
