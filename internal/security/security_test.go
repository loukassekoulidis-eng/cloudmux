package security

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateProfileName(t *testing.T) {
	valid := []string{
		"my-profile",
		"azure_prod",
		"a",
		"ABC-123_xyz",
		"a23456789012345678901234567890123456789012345678901234567890-max", // 64 chars
	}
	for _, name := range valid {
		t.Run("valid/"+name, func(t *testing.T) {
			assert.NoError(t, ValidateProfileName(name))
		})
	}

	invalid := []struct {
		name   string
		reason string
	}{
		{"", "empty"},
		{"-leading-dash", "starts with dash"},
		{".leading-dot", "starts with dot"},
		{"has/slash", "contains slash"},
		{"has\\backslash", "contains backslash"},
		{"has spaces", "contains space"},
		{"has!bang", "contains special char"},
		{"..", "reserved name"},
		{".", "reserved name"},
		{"CON", "reserved name"},
		{"NUL", "reserved name"},
		{"a234567890123456789012345678901234567890123456789012345678901-too", "65 chars"},
	}
	for _, tc := range invalid {
		t.Run("invalid/"+tc.reason, func(t *testing.T) {
			err := ValidateProfileName(tc.name)
			require.Error(t, err)
		})
	}
}

func TestEnforcePermissions(t *testing.T) {
	t.Run("correct dir permissions", func(t *testing.T) {
		dir := t.TempDir()
		os.Chmod(dir, 0700)
		assert.NoError(t, EnforcePermissions(dir, true))
	})

	t.Run("wrong dir permissions", func(t *testing.T) {
		dir := t.TempDir()
		os.Chmod(dir, 0755)
		err := EnforcePermissions(dir, true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "insecure permissions")
		assert.Contains(t, err.Error(), "chmod")
	})

	t.Run("correct file permissions", func(t *testing.T) {
		f := filepath.Join(t.TempDir(), "test.yaml")
		os.WriteFile(f, []byte("x"), 0600)
		assert.NoError(t, EnforcePermissions(f, false))
	})

	t.Run("wrong file permissions", func(t *testing.T) {
		f := filepath.Join(t.TempDir(), "test.yaml")
		os.WriteFile(f, []byte("x"), 0644)
		err := EnforcePermissions(f, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "insecure permissions")
	})

	t.Run("nonexistent path", func(t *testing.T) {
		err := EnforcePermissions("/nonexistent/path", true)
		require.Error(t, err)
	})
}

func TestEnsureDir(t *testing.T) {
	t.Run("creates new directory with 0700", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "newdir")
		require.NoError(t, EnsureDir(dir))
		info, err := os.Stat(dir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
		assert.Equal(t, os.FileMode(0700), info.Mode().Perm())
	})

	t.Run("accepts existing directory with correct perms", func(t *testing.T) {
		dir := t.TempDir()
		os.Chmod(dir, 0700)
		assert.NoError(t, EnsureDir(dir))
	})

	t.Run("rejects existing directory with wrong perms", func(t *testing.T) {
		dir := t.TempDir()
		os.Chmod(dir, 0755)
		err := EnsureDir(dir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "insecure permissions")
	})
}
