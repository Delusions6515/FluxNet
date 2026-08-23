package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Delusions6515/FluxNet/internal/paths"
)

func TestNormalizePackageList(t *testing.T) {
	got := normalizePackageList("# comment\npackage:com.example.one\n10:io.demo.two\ncom.example.one invalid package\n")
	want := []string{"com.example.one", "io.demo.two"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizePackageList = %#v, want %#v", got, want)
	}
}

func TestAutomaticAppListsUsesCurrentUserPackages(t *testing.T) {
	root := t.TempDir()
	layout := paths.New(filepath.Join(root, "module"), filepath.Join(root, "data"))
	if err := os.MkdirAll(layout.ConfigDir(), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(layout.ProxyPackageList(), []byte("com.example.proxy\n"), 0600); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(bin, 0755); err != nil {
		t.Fatal(err)
	}
	writeCommand(t, bin, "cmd", "#!/bin/sh\necho 10\n")
	writeCommand(t, bin, "pm", "#!/bin/sh\n[ \"$4\" = \"10\" ] || exit 1\nprintf '%s\\n' package:com.example.direct package:com.example.proxy\n")
	t.Setenv("PATH", bin+":"+os.Getenv("PATH"))

	proxy, bypass, err := automaticAppLists(layout)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(proxy, []string{"com.example.proxy"}) || !reflect.DeepEqual(bypass, []string{"com.example.direct"}) {
		t.Fatalf("automatic lists = %#v, %#v", proxy, bypass)
	}
}

func writeCommand(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0755); err != nil {
		t.Fatal(err)
	}
}
