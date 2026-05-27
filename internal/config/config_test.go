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

func TestLoadProfilesGCP(t *testing.T) {
	f := filepath.Join(t.TempDir(), "profiles.yaml")
	os.WriteFile(f, []byte(`
profiles:
  - name: my-gcp
    provider: gcp
    description: "GCP project"
    gcp:
      project_id: "my-project-123"
      region: "europe-west3"
      zone: "europe-west3-a"
      use_named_config: false
`), 0600)
	profiles, err := LoadProfiles(f)
	require.NoError(t, err)
	require.Len(t, profiles, 1)
	assert.Equal(t, "my-gcp", profiles[0].Name)
	assert.Equal(t, "gcp", profiles[0].Provider)
	assert.Equal(t, "my-project-123", profiles[0].GCP.ProjectID)
	assert.Equal(t, "europe-west3", profiles[0].GCP.Region)
	assert.Equal(t, "europe-west3-a", profiles[0].GCP.Zone)
	assert.False(t, profiles[0].GCP.UseNamedConfig)
}

func TestLoadProfilesAWS(t *testing.T) {
	f := filepath.Join(t.TempDir(), "profiles.yaml")
	os.WriteFile(f, []byte(`
profiles:
  - name: my-aws
    provider: aws
    aws:
      profile_name: "prod-account"
      region: "eu-central-1"
      sso_start_url: "https://myorg.awsapps.com/start"
`), 0600)
	profiles, err := LoadProfiles(f)
	require.NoError(t, err)
	require.Len(t, profiles, 1)
	assert.Equal(t, "prod-account", profiles[0].AWS.ProfileName)
	assert.Equal(t, "eu-central-1", profiles[0].AWS.Region)
	assert.Equal(t, "https://myorg.awsapps.com/start", profiles[0].AWS.SSOStartURL)
}

func TestLoadProfilesWithTTLAndConfirm(t *testing.T) {
	f := filepath.Join(t.TempDir(), "profiles.yaml")
	os.WriteFile(f, []byte(`
profiles:
  - name: prod-azure
    provider: azure
    confirm_on_use: true
    ttl_days: 90
    tags: [production, client]
    azure:
      tenant_id: "t-123"
`), 0600)
	profiles, err := LoadProfiles(f)
	require.NoError(t, err)
	require.Len(t, profiles, 1)
	assert.True(t, profiles[0].ConfirmOnUse)
	assert.Equal(t, 90, profiles[0].TTLDays)
	assert.Contains(t, profiles[0].Tags, "production")
}

func TestLoadProfilesCustom(t *testing.T) {
	f := filepath.Join(t.TempDir(), "profiles.yaml")
	os.WriteFile(f, []byte(`
profiles:
  - name: hetzner-prod
    provider: custom
    custom:
      env:
        HCLOUD_TOKEN_FILE: "{profile_dir}/token"
      login_command: "hcloud context create {name}"
      status_command: "hcloud server list --output noheader | head -1"
      logout_command: "rm -f {profile_dir}/token"
`), 0600)
	profiles, err := LoadProfiles(f)
	require.NoError(t, err)
	require.Len(t, profiles, 1)
	assert.Equal(t, "custom", profiles[0].Provider)
	assert.Equal(t, "{profile_dir}/token", profiles[0].Custom.Env["HCLOUD_TOKEN_FILE"])
	assert.Equal(t, "hcloud context create {name}", profiles[0].Custom.LoginCommand)
}

func TestAppendProfile(t *testing.T) {
	f := filepath.Join(t.TempDir(), "profiles.yaml")
	os.WriteFile(f, []byte(`profiles:
  - name: existing
    provider: azure
    azure:
      tenant_id: "t-123"
`), 0600)

	newProfile := Profile{
		Name:     "imported",
		Provider: "gcp",
		GCP:      GCPConfig{ProjectID: "my-proj"},
	}

	require.NoError(t, AppendProfile(f, newProfile))

	profiles, err := LoadProfiles(f)
	require.NoError(t, err)
	require.Len(t, profiles, 2)
	assert.Equal(t, "existing", profiles[0].Name)
	assert.Equal(t, "imported", profiles[1].Name)
	assert.Equal(t, "my-proj", profiles[1].GCP.ProjectID)
}

func TestLoadConfigTrayFields(t *testing.T) {
	f := filepath.Join(t.TempDir(), "config.yaml")
	os.WriteFile(f, []byte(`
tray_notifications: false
tray_detection_interval_minutes: 10
tray_expiry_check_interval_minutes: 3
ignored_sessions:
  - "azure:abc123"
  - "gcp:def456"
`), 0600)
	cfg, err := LoadConfig(f)
	require.NoError(t, err)
	assert.False(t, cfg.TrayNotifications)
	assert.Equal(t, 10, cfg.TrayDetectionIntervalMinutes)
	assert.Equal(t, 3, cfg.TrayExpiryCheckIntervalMinutes)
	assert.Len(t, cfg.IgnoredSessions, 2)
}

func TestLoadConfigTrayDefaults(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join(t.TempDir(), "nonexistent.yaml"))
	require.NoError(t, err)
	assert.True(t, cfg.TrayNotifications)
	assert.Equal(t, 5, cfg.TrayDetectionIntervalMinutes)
	assert.Equal(t, 2, cfg.TrayExpiryCheckIntervalMinutes)
	assert.Empty(t, cfg.IgnoredSessions)
}

func TestAppendProfileDuplicate(t *testing.T) {
	f := filepath.Join(t.TempDir(), "profiles.yaml")
	os.WriteFile(f, []byte(`profiles:
  - name: existing
    provider: azure
    azure:
      tenant_id: "t-123"
`), 0600)

	newProfile := Profile{
		Name:     "existing",
		Provider: "gcp",
	}

	err := AppendProfile(f, newProfile)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}
