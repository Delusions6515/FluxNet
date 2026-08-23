package config

import (
	"net/http"
	"net/http/httptest"
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

func TestProxyPackageCatalogPrefersDataDirectory(t *testing.T) {
	root := t.TempDir()
	layout := paths.New(filepath.Join(root, "module"), filepath.Join(root, "data"))
	if err := os.MkdirAll(filepath.Dir(layout.ModProxyPackageList()), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(layout.ModProxyPackageList(), []byte("com.example.module\n"), 0600); err != nil {
		t.Fatal(err)
	}
	packages, err := ReadPackageList(layout.ProxyPackageCatalog())
	if err != nil || !reflect.DeepEqual(packages, []string{"com.example.module"}) {
		t.Fatalf("module catalogue = %#v, %v", packages, err)
	}
	if err := os.MkdirAll(layout.ConfigDir(), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(layout.ProxyPackageList(), []byte("com.example.data\n"), 0600); err != nil {
		t.Fatal(err)
	}
	packages, err = ReadPackageList(layout.ProxyPackageCatalog())
	if err != nil || !reflect.DeepEqual(packages, []string{"com.example.data"}) {
		t.Fatalf("data catalogue = %#v, %v", packages, err)
	}
}

func TestUpdateProxyPackageListWritesValidatedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("com.example.proxy\ninvalid package\ncom.example.proxy\n"))
	}))
	defer server.Close()

	root := t.TempDir()
	layout := paths.New(filepath.Join(root, "module"), filepath.Join(root, "data"))
	count, err := updateProxyPackageList(layout, server.URL)
	if err != nil || count != 1 {
		t.Fatalf("updateProxyPackageList = %d, %v", count, err)
	}
	data, err := os.ReadFile(layout.ProxyPackageList())
	if err != nil || string(data) != "com.example.proxy\n" {
		t.Fatalf("updated catalogue = %q, %v", data, err)
	}
}
