package subscription

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/Delusions6515/FluxNet/internal/paths"
)

func TestLocalSubscriptionRoundTrip(t *testing.T) {
	layout := testLayout(t)
	CreateLocal(layout, true, "custom")
	data := []byte(`{"inbounds":[],"outbounds":[{"type":"direct","tag":"direct"}]}`)
	WriteLocal(layout, true, "custom", base64.StdEncoding.EncodeToString(data))
	got, err := os.ReadFile(filepath.Join(layout.LocalConfigDir(), "custom.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(data) {
		t.Fatalf("content = %s", got)
	}
}

func TestLocalSubscriptionRejectsTraversalAndInvalidJSON(t *testing.T) {
	layout := testLayout(t)
	CreateLocal(layout, true, "../escape")
	if _, err := os.Stat(filepath.Join(layout.LocalConfigDir(), "escape.json")); err == nil {
		t.Fatal("created a traversing local subscription")
	}
	CreateLocal(layout, true, "custom")
	WriteLocal(layout, true, "custom", base64.StdEncoding.EncodeToString([]byte("[]")))
	data, err := os.ReadFile(filepath.Join(layout.LocalConfigDir(), "custom.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) == "[]" {
		t.Fatal("invalid JSON shape was saved")
	}
}

func testLayout(t *testing.T) *paths.Layout {
	t.Helper()
	root := t.TempDir()
	layout := paths.New(filepath.Join(root, "module"), filepath.Join(root, "data"))
	if err := os.MkdirAll(layout.ConfigDir(), 0755); err != nil {
		t.Fatal(err)
	}
	return layout
}
