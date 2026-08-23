package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Delusions6515/FluxNet/internal/paths"
)

func TestReadUserInboundCopiesTemplate(t *testing.T) {
	root := t.TempDir()
	layout := paths.New(filepath.Join(root, "module"), filepath.Join(root, "data"))
	if err := os.MkdirAll(layout.InboundTemplateDir(), 0755); err != nil {
		t.Fatal(err)
	}
	const template = "{\n  \"type\": \"tun\"\n}\n"
	if err := os.WriteFile(layout.InboundTemplate("tun"), []byte(template), 0600); err != nil {
		t.Fatal(err)
	}
	content, err := ReadUserInbound(layout, "tun")
	if err != nil || content != template {
		t.Fatalf("ReadUserInbound = %q, %v", content, err)
	}
	data, err := os.ReadFile(layout.UserInbound("tun"))
	if err != nil || string(data) != template {
		t.Fatalf("user inbound = %q, %v", data, err)
	}
}

func TestWriteUserInboundAndTproxy(t *testing.T) {
	root := t.TempDir()
	layout := paths.New(filepath.Join(root, "module"), filepath.Join(root, "data"))
	if err := WriteUserInbound(layout, "tun", []byte("{\"type\":\"tun\"}")); err != nil {
		t.Fatal(err)
	}
	if err := WriteUserInbound(layout, "tun", []byte("[]")); err == nil {
		t.Fatal("WriteUserInbound accepted a JSON array")
	}
	if err := WriteUserTproxyConf(layout, []byte("PROXY_TCP_PORT=1234\n")); err != nil {
		t.Fatal(err)
	}
	content, err := ReadUserTproxyConf(layout)
	if err != nil || content != "PROXY_TCP_PORT=1234\n" {
		t.Fatalf("ReadUserTproxyConf = %q, %v", content, err)
	}
}
