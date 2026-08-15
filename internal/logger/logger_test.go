package logger_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/albachteng/airgradient/internal/logger"
)

func TestNewCreatesLogDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "logs")
	_, err := logger.New(dir)
	if err != nil {
		t.Fatalf("logger.New: %v", err)
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Errorf("expected log dir to be created at %s", dir)
	}
}

func TestLogWritesToFile(t *testing.T) {
	dir := t.TempDir()
	l, err := logger.New(dir)
	if err != nil {
		t.Fatalf("logger.New: %v", err)
	}
	defer l.Close()

	l.Log("test message", map[string]any{"key": "value"})

	// find the log file written today
	date := time.Now().UTC().Format("2006-01-02")
	logFile := filepath.Join(dir, date+".log")

	contents, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("reading log file: %v", err)
	}
	if !strings.Contains(string(contents), "test message") {
		t.Errorf("log file missing message, got: %s", string(contents))
	}
	if !strings.Contains(string(contents), "value") {
		t.Errorf("log file missing field value, got: %s", string(contents))
	}
}

func TestLogIncludesTimestamp(t *testing.T) {
	dir := t.TempDir()
	l, err := logger.New(dir)
	if err != nil {
		t.Fatalf("logger.New: %v", err)
	}
	defer l.Close()

	l.Log("timestamped message", nil)

	date := time.Now().UTC().Format("2006-01-02")
	contents, err := os.ReadFile(filepath.Join(dir, date+".log"))
	if err != nil {
		t.Fatalf("reading log file: %v", err)
	}
	// Log entries should contain the current year at minimum
	if !strings.Contains(string(contents), "2026") {
		t.Errorf("log entry missing timestamp, got: %s", string(contents))
	}
}
