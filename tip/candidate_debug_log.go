package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// candidateDebugLog intentionally records only candidate metadata, never candidate strings or composition text.
func candidateDebugLog(format string, args ...any) {
	line := fmt.Sprintf("[%s][pid=%d][component=tip] %s\r\n",
		time.Now().Format("2006-01-02 15:04:05.000"), os.Getpid(), fmt.Sprintf(format, args...))
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		base = os.TempDir()
	}
	path := filepath.Join(base, "PortGoWindowsIME", "candidate-debug.log")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		path = filepath.Join(os.TempDir(), "candidate-debug.log")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	_, _ = f.WriteString(line)
	_ = f.Sync()
	_ = f.Close()
}
