package logs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRotate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "log.txt")
	os.WriteFile(path, []byte("content"), 0600)
	os.WriteFile(path+".bak", []byte("old"), 0600)

	Rotate(path)

	if _, err := os.Stat(path); err == nil {
		t.Fatal("file should be rotated when it has content")
	}
	data, err := os.ReadFile(path + ".bak")
	if err != nil {
		t.Fatal("moved file not found as .bak")
	}
	if string(data) != "content" {
		t.Fatalf("unexpected .bak content: %q", data)
	}
}

func TestRotateEmptyKeepsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "log.txt")
	os.WriteFile(path, nil, 0600)
	os.WriteFile(path+".bak", []byte("old"), 0600)

	Rotate(path)

	if _, err := os.Stat(path); err != nil {
		t.Fatal("empty file should not be rotated")
	}
	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Fatal("existing .bak should be left untouched")
	}
}