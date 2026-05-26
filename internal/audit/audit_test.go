package audit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLog(t *testing.T) {
	dir := t.TempDir()
	os.Chmod(dir, 0700)
	logPath := filepath.Join(dir, "audit.log")

	logger := New(logPath)
	require.NoError(t, logger.Log("LOGIN", "my-profile", "azure", "user@test.com"))
	require.NoError(t, logger.Log("USE", "my-profile", "azure", ""))

	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	assert.Len(t, lines, 2)
	assert.Contains(t, lines[0], "LOGIN")
	assert.Contains(t, lines[0], "my-profile")
	assert.Contains(t, lines[0], "azure")
	assert.Contains(t, lines[1], "USE")
}

func TestLogFilePermissions(t *testing.T) {
	dir := t.TempDir()
	os.Chmod(dir, 0700)
	logPath := filepath.Join(dir, "audit.log")

	logger := New(logPath)
	require.NoError(t, logger.Log("LOGIN", "test", "azure", ""))

	info, err := os.Stat(logPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestLogRotation(t *testing.T) {
	dir := t.TempDir()
	os.Chmod(dir, 0700)
	logPath := filepath.Join(dir, "audit.log")

	logger := NewWithMaxLines(logPath, 20)

	for i := 0; i < 25; i++ {
		require.NoError(t, logger.Log("USE", "profile", "azure", ""))
	}

	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	assert.LessOrEqual(t, len(lines), 20)
	assert.Greater(t, len(lines), 0)
}
