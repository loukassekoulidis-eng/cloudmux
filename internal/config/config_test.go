package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	t.Run("returns defaults when file missing", func(t *testing.T) {
		cfg, err := LoadConfig(filepath.Join(t.TempDir(), "nonexistent.yaml"))
		require.NoError(t, err)
		assert.Equal(t, "[cloudmux: {name}]", cfg.PromptFormat)
		assert.True(t, cfg.EnforcePermissions)
		assert.Equal(t, 15, cfg.ExpiryWarningMinutes)
	})

	t.Run("loads valid config", func(t *testing.T) {
		f := filepath.Join(t.TempDir(), "config.yaml")
		os.WriteFile(f, []byte(`
prompt_format: "({name})"
enforce_permissions: false
expiry_warning_minutes: 30
`), 0600)
		cfg, err := LoadConfig(f)
		require.NoError(t, err)
		assert.Equal(t, "({name})", cfg.PromptFormat)
		assert.False(t, cfg.EnforcePermissions)
		assert.Equal(t, 30, cfg.ExpiryWarningMinutes)
	})
}

func TestLoadProfiles(t *testing.T) {
	t.Run("loads valid profiles", func(t *testing.T) {
		f := filepath.Join(t.TempDir(), "profiles.yaml")
		os.WriteFile(f, []byte(`
profiles:
  - name: my-azure
    provider: azure
    description: "test tenant"
    azure:
      tenant_id: "aaaa-bbbb-cccc"
      subscription_id: "dddd-eeee-ffff"
      default_location: "westeurope"
`), 0600)
		profiles, err := LoadProfiles(f)
		require.NoError(t, err)
		require.Len(t, profiles, 1)
		assert.Equal(t, "my-azure", profiles[0].Name)
		assert.Equal(t, "azure", profiles[0].Provider)
		assert.Equal(t, "aaaa-bbbb-cccc", profiles[0].Azure.TenantID)
	})

	t.Run("rejects invalid profile name", func(t *testing.T) {
		f := filepath.Join(t.TempDir(), "profiles.yaml")
		os.WriteFile(f, []byte(`
profiles:
  - name: "../escape"
    provider: azure
`), 0600)
		_, err := LoadProfiles(f)
		require.Error(t, err)
	})

	t.Run("rejects missing provider", func(t *testing.T) {
		f := filepath.Join(t.TempDir(), "profiles.yaml")
		os.WriteFile(f, []byte(`
profiles:
  - name: valid-name
`), 0600)
		_, err := LoadProfiles(f)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "provider")
	})

	t.Run("rejects duplicate names", func(t *testing.T) {
		f := filepath.Join(t.TempDir(), "profiles.yaml")
		os.WriteFile(f, []byte(`
profiles:
  - name: dupe
    provider: azure
    azure:
      tenant_id: "aaa"
  - name: dupe
    provider: azure
    azure:
      tenant_id: "bbb"
`), 0600)
		_, err := LoadProfiles(f)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate")
	})

	t.Run("errors when file missing", func(t *testing.T) {
		_, err := LoadProfiles(filepath.Join(t.TempDir(), "nope.yaml"))
		require.Error(t, err)
	})
}
