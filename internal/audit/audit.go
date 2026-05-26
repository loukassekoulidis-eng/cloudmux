package audit

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Logger struct {
	path     string
	maxLines int
}

func New(path string) *Logger {
	return &Logger{path: path, maxLines: 10000}
}

func NewWithMaxLines(path string, maxLines int) *Logger {
	return &Logger{path: path, maxLines: maxLines}
}

func (l *Logger) Log(action, profile, provider, details string) error {
	entry := fmt.Sprintf("%s %s %s %s", time.Now().UTC().Format(time.RFC3339), action, profile, provider)
	if details != "" {
		entry += " " + details
	}
	entry += "\n"

	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.WriteString(entry); err != nil {
		return err
	}

	return l.maybeRotate()
}

func (l *Logger) maybeRotate() error {
	data, err := os.ReadFile(l.path)
	if err != nil {
		return nil
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) <= l.maxLines {
		return nil
	}

	keep := l.maxLines / 2
	if keep > len(lines) {
		return nil
	}
	trimmed := strings.Join(lines[len(lines)-keep:], "\n")
	return os.WriteFile(l.path, []byte(trimmed), 0600)
}
