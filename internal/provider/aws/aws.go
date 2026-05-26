package aws

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/lukassekoulidis/cloudmux/internal/config"
	"github.com/lukassekoulidis/cloudmux/internal/provider"
)

type AWS struct{}

func New() *AWS {
	return &AWS{}
}

func (a *AWS) Name() string {
	return "aws"
}

func (a *AWS) EnvVars(profile config.Profile, profileDir string) (map[string]string, error) {
	envs := map[string]string{
		"AWS_PROFILE": profile.AWS.ProfileName,
	}
	if profile.AWS.Region != "" {
		envs["AWS_DEFAULT_REGION"] = profile.AWS.Region
	}
	return envs, nil
}

func (a *AWS) Validate(profile config.Profile) error {
	if profile.AWS.ProfileName == "" {
		return fmt.Errorf("aws provider requires profile_name")
	}
	return nil
}

func (a *AWS) Login(profile config.Profile, profileDir string) error {
	if profile.AWS.SSOStartURL == "" {
		fmt.Fprintln(os.Stderr, "No sso_start_url configured — skipping SSO login (profile may use static credentials)")
		return nil
	}

	cmd := exec.Command("aws", "sso", "login", "--profile", profile.AWS.ProfileName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("aws sso login failed: %w", err)
	}
	return nil
}

func (a *AWS) Logout(profile config.Profile, profileDir string) error {
	if profile.AWS.SSOStartURL != "" {
		cmd := exec.Command("aws", "sso", "logout", "--profile", profile.AWS.ProfileName)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	}
	return nil
}

type stsIdentity struct {
	Account string `json:"Account"`
	Arn     string `json:"Arn"`
	UserID  string `json:"UserId"`
}

func (a *AWS) Status(profile config.Profile, profileDir string) (*provider.SessionStatus, error) {
	cmd := exec.Command("aws", "sts", "get-caller-identity", "--profile", profile.AWS.ProfileName, "--output", "json")

	out, err := cmd.Output()
	if err != nil {
		return &provider.SessionStatus{Valid: false}, nil
	}

	var identity stsIdentity
	if err := json.Unmarshal(out, &identity); err != nil {
		return &provider.SessionStatus{Valid: false}, nil
	}

	return &provider.SessionStatus{
		Valid:    true,
		Identity: identity.Arn,
		Tenant:   identity.Account,
		Region:   profile.AWS.Region,
	}, nil
}
