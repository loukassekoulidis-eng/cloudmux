package config

import (
	"fmt"
	"os"

	"github.com/lukassekoulidis/cloudmux/internal/security"
	"gopkg.in/yaml.v3"
)

type Config struct {
	PromptFormat         string `yaml:"prompt_format"`
	PromptShowExpiry     bool   `yaml:"prompt_show_expiry"`
	ExpiryWarningMinutes int    `yaml:"expiry_warning_minutes"`
	ConfirmProduction    bool   `yaml:"confirm_production"`
	DefaultTTLDays       int    `yaml:"default_ttl_days"`
	EnforcePermissions   bool   `yaml:"enforce_permissions"`
}

func DefaultConfig() Config {
	return Config{
		PromptFormat:         "[cloudmux: {name}]",
		PromptShowExpiry:     true,
		ExpiryWarningMinutes: 15,
		ConfirmProduction:    true,
		DefaultTTLDays:       0,
		EnforcePermissions:   true,
	}
}

func LoadConfig(path string) (Config, error) {
	cfg := DefaultConfig()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing config %s: %w", path, err)
	}
	return cfg, nil
}

type AzureConfig struct {
	TenantID        string `yaml:"tenant_id"`
	SubscriptionID  string `yaml:"subscription_id"`
	DefaultLocation string `yaml:"default_location"`
}

type GCPConfig struct {
	ProjectID      string `yaml:"project_id"`
	Region         string `yaml:"region"`
	Zone           string `yaml:"zone"`
	UseNamedConfig bool   `yaml:"use_named_config"`
}

type AWSConfig struct {
	ProfileName string `yaml:"profile_name"`
	Region      string `yaml:"region"`
	SSOStartURL string `yaml:"sso_start_url"`
}

type CustomConfig struct {
	Env           map[string]string `yaml:"env"`
	LoginCommand  string            `yaml:"login_command"`
	StatusCommand string            `yaml:"status_command"`
	LogoutCommand string            `yaml:"logout_command"`
}

type Profile struct {
	Name        string       `yaml:"name"`
	Provider    string       `yaml:"provider"`
	Description string       `yaml:"description"`
	Tags        []string     `yaml:"tags"`
	Azure       AzureConfig  `yaml:"azure"`
	GCP         GCPConfig    `yaml:"gcp"`
	AWS         AWSConfig    `yaml:"aws"`
	Custom      CustomConfig `yaml:"custom"`
}

type profilesFile struct {
	Profiles []Profile `yaml:"profiles"`
}

func LoadProfiles(path string) ([]Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading profiles %s: %w", path, err)
	}

	var pf profilesFile
	if err := yaml.Unmarshal(data, &pf); err != nil {
		return nil, fmt.Errorf("parsing profiles %s: %w", path, err)
	}

	seen := make(map[string]bool)
	for i, p := range pf.Profiles {
		if err := security.ValidateProfileName(p.Name); err != nil {
			return nil, fmt.Errorf("profile #%d: %w", i+1, err)
		}
		if p.Provider == "" {
			return nil, fmt.Errorf("profile %q: provider is required", p.Name)
		}
		if seen[p.Name] {
			return nil, fmt.Errorf("duplicate profile name %q", p.Name)
		}
		seen[p.Name] = true
	}

	return pf.Profiles, nil
}
