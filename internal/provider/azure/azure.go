package azure

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/lukassekoulidis/cloudmux/internal/config"
	"github.com/lukassekoulidis/cloudmux/internal/provider"
	"github.com/lukassekoulidis/cloudmux/internal/security"
)

type Azure struct{}

func New() *Azure {
	return &Azure{}
}

func (a *Azure) Name() string {
	return "azure"
}

func (a *Azure) EnvVars(profile config.Profile, profileDir string) (map[string]string, error) {
	envs := map[string]string{
		"AZURE_CONFIG_DIR": filepath.Join(profileDir, ".azure"),
	}
	if profile.Azure.DefaultLocation != "" {
		envs["AZURE_DEFAULTS_LOCATION"] = profile.Azure.DefaultLocation
	}
	return envs, nil
}

func (a *Azure) Validate(profile config.Profile) error {
	if profile.Azure.TenantID == "" {
		return fmt.Errorf("azure provider requires tenant_id")
	}
	return nil
}

func (a *Azure) Login(profile config.Profile, profileDir string) error {
	azureDir := filepath.Join(profileDir, ".azure")
	if err := security.EnsureDir(azureDir); err != nil {
		return fmt.Errorf("creating azure config dir: %w", err)
	}

	args := []string{"login", "--tenant", profile.Azure.TenantID}
	cmd := exec.Command("az", args...)
	cmd.Env = append(os.Environ(), "AZURE_CONFIG_DIR="+azureDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("az login failed: %w", err)
	}

	if profile.Azure.SubscriptionID != "" {
		setCmd := exec.Command("az", "account", "set", "--subscription", profile.Azure.SubscriptionID)
		setCmd.Env = append(os.Environ(), "AZURE_CONFIG_DIR="+azureDir)
		setCmd.Stdout = os.Stdout
		setCmd.Stderr = os.Stderr
		if err := setCmd.Run(); err != nil {
			return fmt.Errorf("az account set failed: %w", err)
		}
	}

	return nil
}

func (a *Azure) Logout(profile config.Profile, profileDir string) error {
	azureDir := filepath.Join(profileDir, ".azure")

	cmd := exec.Command("az", "logout")
	cmd.Env = append(os.Environ(), "AZURE_CONFIG_DIR="+azureDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()

	if err := os.RemoveAll(azureDir); err != nil {
		return fmt.Errorf("removing azure config dir: %w", err)
	}
	return nil
}

func suggestName(tenantDomain string) string {
	if tenantDomain == "" {
		return "unknown-azure"
	}
	name := tenantDomain
	name = strings.TrimSuffix(name, ".onmicrosoft.com")
	name = strings.TrimSuffix(name, ".de")
	name = strings.TrimSuffix(name, ".com")
	name = strings.TrimSuffix(name, ".org")
	name = strings.TrimSuffix(name, ".net")
	name = strings.ReplaceAll(name, ".", "-")
	return name + "-azure"
}

type azAccountShowFull struct {
	User struct {
		Name string `json:"name"`
	} `json:"user"`
	TenantID            string `json:"tenantId"`
	ID                  string `json:"id"`
	TenantDefaultDomain string `json:"tenantDefaultDomain"`
}

func (a *Azure) Detect() (*provider.ImportInfo, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	cmd := exec.Command("az", "account", "show", "--output", "json")
	// Strip AZURE_CONFIG_DIR to use default ~/.azure
	env := os.Environ()
	filtered := make([]string, 0, len(env))
	for _, e := range env {
		if !strings.HasPrefix(e, "AZURE_CONFIG_DIR=") {
			filtered = append(filtered, e)
		}
	}
	cmd.Env = filtered

	out, err := cmd.Output()
	if err != nil {
		return nil, nil
	}

	var acct azAccountShowFull
	if err := json.Unmarshal(out, &acct); err != nil {
		return nil, nil
	}

	return &provider.ImportInfo{
		SuggestedName: suggestName(acct.TenantDefaultDomain),
		Profile: config.Profile{
			Provider:    "azure",
			Description: fmt.Sprintf("Imported from %s (%s)", acct.TenantDefaultDomain, acct.User.Name),
			Azure: config.AzureConfig{
				TenantID:       acct.TenantID,
				SubscriptionID: acct.ID,
			},
		},
		DefaultDir: filepath.Join(home, ".azure"),
	}, nil
}

type azAccountShow struct {
	User struct {
		Name string `json:"name"`
	} `json:"user"`
	TenantID        string `json:"tenantId"`
	Name            string `json:"name"`
	EnvironmentName string `json:"environmentName"`
}

func (a *Azure) Status(profile config.Profile, profileDir string) (*provider.SessionStatus, error) {
	azureDir := filepath.Join(profileDir, ".azure")

	cmd := exec.Command("az", "account", "show", "--output", "json")
	cmd.Env = append(os.Environ(), "AZURE_CONFIG_DIR="+azureDir)

	out, err := cmd.Output()
	if err != nil {
		return &provider.SessionStatus{Valid: false}, nil
	}

	var acct azAccountShow
	if err := json.Unmarshal(out, &acct); err != nil {
		return &provider.SessionStatus{Valid: false}, nil
	}

	return &provider.SessionStatus{
		Valid:    true,
		Identity: acct.User.Name,
		Tenant:   acct.TenantID,
	}, nil
}
