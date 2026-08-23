package worker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Delusions6515/FluxNet/internal/paths"
)

func TestRequestServiceActionQueuesOneAction(t *testing.T) {
	layout := paths.New(filepath.Join(t.TempDir(), "module"), filepath.Join(t.TempDir(), "data"))
	if err := os.MkdirAll(layout.RunDir(), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(layout.WorkerPidFile(), []byte("1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	original := workerProcessAlive
	workerProcessAlive = func(int) bool { return true }
	t.Cleanup(func() { workerProcessAlive = original })

	if err := RequestServiceAction(layout, "restart"); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(layout.ServiceRequestFile())
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(content), "restart\n"; got != want {
		t.Fatalf("request = %q, want %q", got, want)
	}
	if err := RequestServiceAction(layout, "restart"); err == nil {
		t.Fatal("second request should fail while one is pending")
	}
}

func TestProcessServiceRequestRunsAndRemovesRequest(t *testing.T) {
	layout := paths.New(filepath.Join(t.TempDir(), "module"), filepath.Join(t.TempDir(), "data"))
	if err := os.MkdirAll(layout.RunDir(), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(layout.ServiceRequestFile(), []byte("restart\n"), 0600); err != nil {
		t.Fatal(err)
	}
	original := runServiceAction
	var got string
	runServiceAction = func(_ *paths.Layout, action string) { got = action }
	t.Cleanup(func() { runServiceAction = original })

	processServiceRequest(layout)
	if got != "restart" {
		t.Fatalf("action = %q, want restart", got)
	}
	if _, err := os.Stat(layout.ServiceRequestFile()); !os.IsNotExist(err) {
		t.Fatalf("request file remains after processing: %v", err)
	}
}
