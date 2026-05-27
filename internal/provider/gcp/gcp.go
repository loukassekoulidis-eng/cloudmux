package gcp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/lukassekoulidis/cloudmux/internal/config"
	"github.com/lukassekoulidis/cloudmux/internal/provider"
	"github.com/lukassekoulidis/cloudmux/internal/security"
)

type GCP struct{}

func New() *GCP {
	return &GCP{}
}

func (g *GCP) Name() string {
	return "gcp"
}

func (g *GCP) EnvVars(profile config.Profile, profileDir string) (map[string]string, error) {
	envs := make(map[string]string)

	if profile.GCP.UseNamedConfig {
		envs["CLOUDSDK_ACTIVE_CONFIG_NAME"] = profile.Name
	} else {
		envs["CLOUDSDK_CONFIG"] = filepath.Join(profileDir, ".config", "gcloud")
	}

	if profile.GCP.ProjectID != "" {
		envs["CLOUDSDK_CORE_PROJECT"] = profile.GCP.ProjectID
	}
	if profile.GCP.Region != "" {
		envs["CLOUDSDK_COMPUTE_REGION"] = profile.GCP.Region
	}
	if profile.GCP.Zone != "" {
		envs["CLOUDSDK_COMPUTE_ZONE"] = profile.GCP.Zone
	}

	return envs, nil
}

func (g *GCP) Validate(profile config.Profile) error {
	if profile.GCP.ProjectID == "" {
		return fmt.Errorf("gcp provider requires project_id")
	}
	return nil
}

func (g *GCP) Login(profile config.Profile, profileDir string) error {
	gcpDir := filepath.Join(profileDir, ".config", "gcloud")
	if err := security.EnsureDir(gcpDir); err != nil {
		return fmt.Errorf("creating gcloud config dir: %w", err)
	}

	cmd := exec.Command("gcloud", "auth", "login")
	cmd.Env = append(os.Environ(), "CLOUDSDK_CONFIG="+gcpDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gcloud auth login failed: %w", err)
	}

	if profile.GCP.ProjectID != "" {
		setCmd := exec.Command("gcloud", "config", "set", "project", profile.GCP.ProjectID)
		setCmd.Env = append(os.Environ(), "CLOUDSDK_CONFIG="+gcpDir)
		setCmd.Stdout = os.Stdout
		setCmd.Stderr = os.Stderr
		if err := setCmd.Run(); err != nil {
			return fmt.Errorf("gcloud config set project failed: %w", err)
		}
	}

	return nil
}

func (g *GCP) Logout(profile config.Profile, profileDir string) error {
	gcpDir := filepath.Join(profileDir, ".config", "gcloud")

	cmd := exec.Command("gcloud", "auth", "revoke")
	cmd.Env = append(os.Environ(), "CLOUDSDK_CONFIG="+gcpDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()

	if err := os.RemoveAll(gcpDir); err != nil {
		return fmt.Errorf("removing gcloud config dir: %w", err)
	}
	return nil
}

func suggestName(projectID string) string {
	if projectID == "" {
		return "unknown-gcp"
	}
	return projectID + "-gcp"
}

func (g *GCP) Detect() (*provider.ImportInfo, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	projCmd := exec.Command("gcloud", "config", "get", "project")
	env := os.Environ()
	filtered := make([]string, 0, len(env))
	for _, e := range env {
		if !strings.HasPrefix(e, "CLOUDSDK_CONFIG=") && !strings.HasPrefix(e, "CLOUDSDK_ACTIVE_CONFIG_NAME=") {
			filtered = append(filtered, e)
		}
	}
	projCmd.Env = filtered
	projOut, err := projCmd.Output()
	if err != nil {
		return nil, nil
	}
	projectID := strings.TrimSpace(string(projOut))

	acctCmd := exec.Command("gcloud", "config", "get", "account")
	acctCmd.Env = filtered
	acctOut, _ := acctCmd.Output()
	account := strings.TrimSpace(string(acctOut))

	if projectID == "" {
		return nil, nil
	}

	return &provider.ImportInfo{
		SuggestedName: suggestName(projectID),
		Profile: config.Profile{
			Provider:    "gcp",
			Description: fmt.Sprintf("Imported from %s (%s)", projectID, account),
			GCP: config.GCPConfig{
				ProjectID: projectID,
			},
		},
		DefaultDir: filepath.Join(home, ".config", "gcloud"),
	}, nil
}

func (g *GCP) Status(profile config.Profile, profileDir string) (*provider.SessionStatus, error) {
	gcpDir := filepath.Join(profileDir, ".config", "gcloud")

	cmd := exec.Command("gcloud", "auth", "print-access-token")
	cmd.Env = append(os.Environ(), "CLOUDSDK_CONFIG="+gcpDir)
	if err := cmd.Run(); err != nil {
		return &provider.SessionStatus{Valid: false}, nil
	}

	projCmd := exec.Command("gcloud", "config", "get", "project")
	projCmd.Env = append(os.Environ(), "CLOUDSDK_CONFIG="+gcpDir)
	projOut, _ := projCmd.Output()

	acctCmd := exec.Command("gcloud", "config", "get", "account")
	acctCmd.Env = append(os.Environ(), "CLOUDSDK_CONFIG="+gcpDir)
	acctOut, _ := acctCmd.Output()

	return &provider.SessionStatus{
		Valid:    true,
		Identity: strings.TrimSpace(string(acctOut)),
		Tenant:   strings.TrimSpace(string(projOut)),
	}, nil
}
