package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type Logger struct {
	logDir      string
	current     *os.File
	currentDate string
	w           io.Writer
}

func New(logDir string) (*Logger, error) {
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}
	l := &Logger{logDir: logDir}
	if err := l.openFile(); err != nil {
		return nil, err
	}
	return l, nil
}

func (l *Logger) openFile() error {
	date := time.Now().UTC().Format("2006-01-02")
	path := filepath.Join(l.logDir, date+".log")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	if l.current != nil {
		l.current.Close()
	}
	l.current = f
	l.currentDate = date
	l.w = io.MultiWriter(os.Stdout, f)
	return nil
}

func (l *Logger) rotate() {
	if time.Now().UTC().Format("2006-01-02") != l.currentDate {
		l.openFile() // best-effort; if it fails, keep writing to old file
	}
}

func (l *Logger) Log(msg string, fields map[string]any) {
	l.rotate()
	entry := map[string]any{
		"time":    time.Now().UTC().Format(time.RFC3339),
		"message": msg,
	}
	for k, v := range fields {
		entry[k] = v
	}
	b, _ := json.Marshal(entry)
	fmt.Fprintln(l.w, string(b))
}

func (l *Logger) Close() error {
	if l.current != nil {
		return l.current.Close()
	}
	return nil
}
