package copydir

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopy(t *testing.T) {
	src := t.TempDir()
	os.MkdirAll(filepath.Join(src, "subdir"), 0755)
	os.WriteFile(filepath.Join(src, "file.txt"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(src, "subdir", "nested.txt"), []byte("world"), 0644)

	dst := filepath.Join(t.TempDir(), "dest")
	require.NoError(t, Copy(src, dst))

	data, err := os.ReadFile(filepath.Join(dst, "file.txt"))
	require.NoError(t, err)
	assert.Equal(t, "hello", string(data))

	data, err = os.ReadFile(filepath.Join(dst, "subdir", "nested.txt"))
	require.NoError(t, err)
	assert.Equal(t, "world", string(data))
}

func TestCopyPermissions(t *testing.T) {
	src := t.TempDir()
	os.WriteFile(filepath.Join(src, "secret.json"), []byte("token"), 0644)

	dst := filepath.Join(t.TempDir(), "dest")
	require.NoError(t, Copy(src, dst))

	info, _ := os.Stat(dst)
	assert.Equal(t, os.FileMode(0700), info.Mode().Perm())

	info, _ = os.Stat(filepath.Join(dst, "secret.json"))
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestCopySourceNotExist(t *testing.T) {
	err := Copy("/nonexistent/path", t.TempDir())
	require.Error(t, err)
}
