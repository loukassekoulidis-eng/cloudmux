package azure

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

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
