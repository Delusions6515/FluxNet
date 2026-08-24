package kernel

import (
	"os"
	"path/filepath"
	"testing"
)

func kvFrom(t *testing.T, data []byte) map[string]string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "c")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return readConfigKV(path)
}

func TestSetConfigValueKeepsCommentBlocks(t *testing.T) {
	data := []byte("# channel comment\nkernel_channel=\"delusions6515-pre\"\n# abi\nproxy_mode=\"tun\"\nkernel_abi=\"arm64-v8a\"\n")
	out, err := setConfigValue(data, "kernel_channel", "ref1nd-stable", true)
	if err != nil {
		t.Fatal(err)
	}
	kv := kvFrom(t, out)
	if kv["kernel_channel"] != "ref1nd-stable" {
		t.Fatalf("kernel_channel = %q, want ref1nd-stable", kv["kernel_channel"])
	}
	if kv["proxy_mode"] != "tun" {
		t.Fatalf("unrelated key changed: proxy_mode = %q", kv["proxy_mode"])
	}
}

func TestSetConfigValueAppendsMissingKey(t *testing.T) {
	data := []byte("proxy_mode=\"tun\"\n")
	out, err := setConfigValue(data, "kernel_channel", "official-stable", true)
	if err != nil {
		t.Fatal(err)
	}
	kv := kvFrom(t, out)
	if kv["kernel_channel"] != "official-stable" {
		t.Fatalf("kernel_channel = %q, want official-stable", kv["kernel_channel"])
	}
}

func TestReadConfigKVMissingFile(t *testing.T) {
	if kv := readConfigKV(filepath.Join(t.TempDir(), "nope")); kv != nil {
		t.Fatalf("expected nil, got %v", kv)
	}
}

func TestChannelCatalogue(t *testing.T) {
	if Channels["nope"] != "" {
		t.Fatalf("unknown channel must be empty")
	}
	if Channels["official-stable"] == "" {
		t.Fatal("official-stable channel missing")
	}
}