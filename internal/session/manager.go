package session

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/lukassekoulidis/cloudmux/internal/audit"
	"github.com/lukassekoulidis/cloudmux/internal/config"
	"github.com/lukassekoulidis/cloudmux/internal/provider"
	"github.com/lukassekoulidis/cloudmux/internal/security"
)

type UseResult struct {
	ProfileName string
	Provider    string
	EnvVars     map[string]string
}

type Manager struct {
	baseDir  string
	profiles []config.Profile
	registry *provider.Registry
	audit    *audit.Logger
}

func NewManager(baseDir string, registry *provider.Registry, auditLogger *audit.Logger) (*Manager, error) {
	profilesPath := filepath.Join(baseDir, "profiles.yaml")
	profiles, err := config.LoadProfiles(profilesPath)
	if err != nil {
		return nil, err
	}

	return &Manager{
		baseDir:  baseDir,
		profiles: profiles,
		registry: registry,
		audit:    auditLogger,
	}, nil
}

func (m *Manager) logAudit(action, profile, providerName, details string) {
	if m.audit != nil {
		m.audit.Log(action, profile, providerName, details)
	}
}

func (m *Manager) findProfile(name string) (config.Profile, error) {
	for _, p := range m.profiles {
		if p.Name == name {
			return p, nil
		}
	}
	return config.Profile{}, fmt.Errorf("profile %q not found", name)
}

func (m *Manager) profileDir(name string) string {
	return filepath.Join(m.baseDir, "profiles", name)
}

func (m *Manager) Use(profileName string) (*UseResult, error) {
	profile, err := m.findProfile(profileName)
	if err != nil {
		return nil, err
	}

	prov, err := m.registry.Get(profile.Provider)
	if err != nil {
		return nil, err
	}

	if err := prov.Validate(profile); err != nil {
		return nil, fmt.Errorf("invalid profile %q: %w", profileName, err)
	}

	profDir := m.profileDir(profileName)
	envs, err := prov.EnvVars(profile, profDir)
	if err != nil {
		return nil, err
	}
	envs["CLOUDMUX_ACTIVE_PROFILE"] = profileName

	m.logAudit("USE", profileName, profile.Provider, "")

	return &UseResult{
		ProfileName: profileName,
		Provider:    profile.Provider,
		EnvVars:     envs,
	}, nil
}

func (m *Manager) Login(profileName string) error {
	profile, err := m.findProfile(profileName)
	if err != nil {
		return err
	}

	prov, err := m.registry.Get(profile.Provider)
	if err != nil {
		return err
	}

	if err := prov.Validate(profile); err != nil {
		return fmt.Errorf("invalid profile %q: %w", profileName, err)
	}

	profDir := m.profileDir(profileName)
	if err := security.EnsureDir(profDir); err != nil {
		return err
	}

	if err := prov.Login(profile, profDir); err != nil {
		return err
	}

	tsPath := filepath.Join(profDir, ".cloudmux_login_ts")
	os.WriteFile(tsPath, []byte(time.Now().UTC().Format(time.RFC3339)), 0600)

	m.logAudit("LOGIN", profileName, profile.Provider, "")

	return nil
}

type LogoutResult struct {
	EnvKeys []string
}

func (m *Manager) Logout(profileName string) (*LogoutResult, error) {
	profile, err := m.findProfile(profileName)
	if err != nil {
		return nil, err
	}

	prov, err := m.registry.Get(profile.Provider)
	if err != nil {
		return nil, err
	}

	// Get the env var keys this provider sets, so the caller can unset them
	profDir := m.profileDir(profileName)
	envs, err := prov.EnvVars(profile, profDir)
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(envs)+1)
	keys = append(keys, "CLOUDMUX_ACTIVE_PROFILE")
	for k := range envs {
		keys = append(keys, k)
	}

	if err := prov.Logout(profile, profDir); err != nil {
		return nil, err
	}

	m.logAudit("LOGOUT", profileName, profile.Provider, "")

	return &LogoutResult{EnvKeys: keys}, nil
}

func (m *Manager) Status(profileName string) (*provider.SessionStatus, error) {
	profile, err := m.findProfile(profileName)
	if err != nil {
		return nil, err
	}

	prov, err := m.registry.Get(profile.Provider)
	if err != nil {
		return nil, err
	}

	return prov.Status(profile, m.profileDir(profileName))
}

func (m *Manager) Profiles() []config.Profile {
	return m.profiles
}

func (m *Manager) LoginTimestamp(profileName string) (time.Time, error) {
	tsPath := filepath.Join(m.profileDir(profileName), ".cloudmux_login_ts")
	data, err := os.ReadFile(tsPath)
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339, string(data))
}

func (m *Manager) ProfileDir(name string) string {
	return m.profileDir(name)
}
