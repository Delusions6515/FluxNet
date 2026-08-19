package logs

import (
	"fmt"
	"os"
	"sort"
	"strings"

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
	logDir := layout.LogsDir()
	entries, err := readAllLogs(logDir)
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

func readAllLogs(dir string) ([]LogEntry, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var entries []LogEntry
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".log") {
			continue
		}
		data, err := os.ReadFile(dir + "/" + f.Name())
		if err != nil {
			continue
		}
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
	}
	return entries, nil
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
	ec  := strings.Trim(parts[5], "[]")
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
