package logs

import "os"

// Rotate moves the log file to path+".bak" when it has content, discarding
// any previous backup. Call before a service start so each run starts fresh.
func Rotate(path string) {
	if info, err := os.Stat(path); err == nil && info.Size() > 0 {
		_ = os.Rename(path, path+".bak")
	}
}