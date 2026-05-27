package session

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lukassekoulidis/cloudmux/internal/config"
	"github.com/lukassekoulidis/cloudmux/internal/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockProvider struct {
	envVars     map[string]string
	loginErr    error
	logoutErr   error
	status      *provider.SessionStatus
	statusErr   error
	validateErr error
}

func (m *mockProvider) Name() string { return "mock" }
func (m *mockProvider) EnvVars(_ config.Profile, _ string) (map[string]string, error) {
	return m.envVars, nil
}
func (m *mockProvider) Login(_ config.Profile, _ string) error  { return m.loginErr }
func (m *mockProvider) Logout(_ config.Profile, _ string) error { return m.logoutErr }
func (m *mockProvider) Status(_ config.Profile, _ string) (*provider.SessionStatus, error) {
	return m.status, m.statusErr
}
func (m *mockProvider) Validate(_ config.Profile) error { return m.validateErr }
func (m *mockProvider) Detect() (*provider.ImportInfo, error) { return nil, nil }

func setupTestManager(t *testing.T) (*Manager, string) {
	t.Helper()
	baseDir := t.TempDir()
	os.Chmod(baseDir, 0700)

	profilesPath := filepath.Join(baseDir, "profiles.yaml")
	os.WriteFile(profilesPath, []byte(`
profiles:
  - name: test-profile
    provider: mock
    azure:
      tenant_id: "t-123"
`), 0600)

	reg := provider.NewRegistry()
	reg.Register(&mockProvider{
		envVars: map[string]string{"MOCK_VAR": "/some/path"},
		status:  &provider.SessionStatus{Valid: true, Identity: "user@test.com", Tenant: "t-123"},
	})

	m, err := NewManager(baseDir, reg, nil)
	require.NoError(t, err)
	return m, baseDir
}

func TestManagerUse(t *testing.T) {
	mgr, _ := setupTestManager(t)

	result, err := mgr.Use("test-profile")
	require.NoError(t, err)
	assert.Equal(t, "/some/path", result.EnvVars["MOCK_VAR"])
	assert.Equal(t, "test-profile", result.EnvVars["CLOUDMUX_ACTIVE_PROFILE"])
	assert.Equal(t, "test-profile", result.ProfileName)
}

func TestManagerUseUnknownProfile(t *testing.T) {
	mgr, _ := setupTestManager(t)

	_, err := mgr.Use("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestManagerStatus(t *testing.T) {
	mgr, _ := setupTestManager(t)

	status, err := mgr.Status("test-profile")
	require.NoError(t, err)
	assert.True(t, status.Valid)
	assert.Equal(t, "user@test.com", status.Identity)
}

func TestManagerLoginWritesTimestamp(t *testing.T) {
	baseDir := t.TempDir()
	os.Chmod(baseDir, 0700)

	profilesPath := filepath.Join(baseDir, "profiles.yaml")
	os.WriteFile(profilesPath, []byte(`
profiles:
  - name: test-profile
    provider: mock
    azure:
      tenant_id: "t-123"
`), 0600)

	reg := provider.NewRegistry()
	reg.Register(&mockProvider{
		envVars: map[string]string{"MOCK_VAR": "/some/path"},
	})

	m, err := NewManager(baseDir, reg, nil)
	require.NoError(t, err)
	require.NoError(t, m.Login("test-profile"))

	ts, err := m.LoginTimestamp("test-profile")
	require.NoError(t, err)
	assert.False(t, ts.IsZero())
}
