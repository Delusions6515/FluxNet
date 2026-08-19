package result

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Result is the unified output structure shared by all subcommands.
// schema=1, ok, code, message, data — matching NetProxy's contract.
type Result struct {
	Schema  int    `json:"schema"`
	OK      bool   `json:"ok"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// WriteJSON writes the result as a single-line JSON to the writer.
func WriteJSON(w io.Writer, r Result) {
	r.Schema = 1
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(r)
}

// Success returns a Result with ok=true.
func Success(code, message string, data any) Result {
	return Result{OK: true, Code: code, Message: message, Data: data}
}

// Fail returns a Result with ok=false.
func Fail(code, message string, data any) Result {
	return Result{OK: false, Code: code, Message: message, Data: data}
}

// Text outputs a human-readable text representation to stdout.
// The Result is written as JSON to stderr when --json is not set.
func Text(r Result, formatJSON bool) {
	if formatJSON {
		WriteJSON(os.Stdout, r)
		return
	}
	if r.OK {
		if r.Message != "" {
			fmt.Fprintf(os.Stdout, "✓ %s\n", r.Message)
		}
	} else {
		if r.Message != "" {
			fmt.Fprintf(os.Stdout, "✗ %s (%s)\n", r.Message, r.Code)
		} else {
			fmt.Fprintf(os.Stdout, "✗ %s\n", r.Code)
		}
	}
}

// OK prints a success message in text or JSON mode.
func OK(formatJSON bool, code, message string) {
	Text(Success(code, message, nil), formatJSON)
}

// Err prints an error message in text or JSON mode.
func Err(formatJSON bool, code, message string) {
	Text(Fail(code, message, nil), formatJSON)
}
