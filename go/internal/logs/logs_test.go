package logs

import (
	"path/filepath"
	"testing"

	"github.com/Delusions6515/FluxNet/internal/paths"
	"github.com/Delusions6515/FluxNet/internal/result"
)

func TestRecordOperationWritesOnlyMutations(t *testing.T) {
	root := t.TempDir()
	layout := paths.New(filepath.Join(root, "module"), filepath.Join(root, "data"))

	RecordOperation(layout, result.Success("service.started", "服务已启动", nil))
	RecordOperation(layout, result.Fail("subscription.download_failed", "下载失败", nil))
	RecordOperation(layout, result.Success("service.status", "服务状态", nil))
	RecordOperation(layout, result.Success("logs.list", "日志列表", nil))

	entries, err := readOperationLog(layout.OperationLog())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 operation entries, got %d: %#v", len(entries), entries)
	}
	if entries[0].Component != "service" || entries[0].Event != "started" || entries[0].Result != "ok" {
		t.Fatalf("unexpected success entry: %#v", entries[0])
	}
	if entries[1].Component != "subscription" || entries[1].Event != "download_failed" || entries[1].Result != "failed" || entries[1].ErrorCode != "subscription.download_failed" {
		t.Fatalf("unexpected failure entry: %#v", entries[1])
	}
}
