package logs

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/Delusions6515/FluxNet/internal/paths"
	"github.com/Delusions6515/FluxNet/internal/result"
)

type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Component string `json:"component"`
	Event     string `json:"event"`
	Result    string `json:"result"`
	ErrorCode string `json:"error_code"`
	Message   string `json:"message"`
}

func Show(layout *paths.Layout, formatJSON bool) {
	entries, err := readOperationLog(layout.OperationLog())
	if err != nil {
		result.Err(formatJSON, "logs.read_failed", "读取日志失败: "+err.Error())
		return
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp > entries[j].Timestamp
	})
	if len(entries) > 100 {
		entries = entries[:100]
	}
	if formatJSON {
		result.Text(result.Success("logs.list", "日志列表",
			map[string]any{"entries": entries}), true)
		return
	}
	if len(entries) == 0 {
		fmt.Println("(无日志)")
		return
	}
	for _, e := range entries {
		fmt.Printf("[%s] [%s] [%s] [%s] [%s] [%s] %s\n",
			e.Timestamp, e.Level, e.Component, e.Event,
			e.Result, e.ErrorCode, e.Message)
	}
}

// RecordOperation appends a structured result for a mutating CLI command.
// Read-only polling commands are deliberately excluded to keep WebUI useful.
func RecordOperation(layout *paths.Layout, operation result.Result) {
	if !isOperation(operation.Code) {
		return
	}
	if err := os.MkdirAll(layout.LogsDir(), 0755); err != nil {
		return
	}
	file, err := os.OpenFile(layout.OperationLog(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return
	}
	defer file.Close()

	component, event := operationParts(operation.Code)
	level, status, errorCode := "info", "ok", ""
	if !operation.OK {
		level, status, errorCode = "error", "failed", operation.Code
	}
	message := strings.Join(strings.Fields(operation.Message), " ")
	_, _ = fmt.Fprintf(file, "[%s] [%s] [%s] [%s] [%s] [%s] %s\n",
		time.Now().UTC().Format(time.RFC3339), level, component, event, status, errorCode, message)
}

func readOperationLog(path string) ([]LogEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var entries []LogEntry
	for _, raw := range strings.Split(string(data), "\n") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		raw = redact(raw)
		entry := parseLine(raw)
		if entry != nil {
			entries = append(entries, *entry)
		}
	}
	return entries, nil
}

func isOperation(code string) bool {
	switch code {
	case "service.status", "health.check", "config.settings", "subscription.list", "subscription.local_read", "app.list", "logs.list", "version":
		return false
	default:
		return code != ""
	}
}

func operationParts(code string) (string, string) {
	component, event, found := strings.Cut(code, ".")
	if !found {
		return "fluxnet", code
	}
	return component, event
}

func parseLine(line string) *LogEntry {
	parts := strings.SplitN(line, " ", 7)
	if len(parts) < 6 {
		return &LogEntry{Message: line}
	}
	ts := strings.Trim(parts[0], "[]")
	lvl := strings.Trim(parts[1], "[]")
	cmp := strings.Trim(parts[2], "[]")
	evt := strings.Trim(parts[3], "[]")
	rst := strings.Trim(parts[4], "[]")
	ec := strings.Trim(parts[5], "[]")
	msg := ""
	if len(parts) == 7 {
		msg = parts[6]
	}
	return &LogEntry{
		Timestamp: ts,
		Level:     lvl,
		Component: cmp,
		Event:     evt,
		Result:    rst,
		ErrorCode: ec,
		Message:   msg,
	}
}

func redact(line string) string {
	res := line
	res = strings.ReplaceAll(res, "password=", "password=***")
	res = strings.ReplaceAll(res, "token=", "token=***")
	res = strings.ReplaceAll(res, "secret=", "secret=***")
	return res
}
