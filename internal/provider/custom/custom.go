package custom

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/lukassekoulidis/cloudmux/internal/config"
	"github.com/lukassekoulidis/cloudmux/internal/provider"
	"github.com/lukassekoulidis/cloudmux/internal/security"
)

type Custom struct{}

func New() *Custom {
	return &Custom{}
}

func (c *Custom) Name() string {
	return "custom"
}

func expandTemplateVars(s string, profileDir string, name string) string {
	home, _ := os.UserHomeDir()
	s = strings.ReplaceAll(s, "{profile_dir}", profileDir)
	s = strings.ReplaceAll(s, "{name}", name)
	s = strings.ReplaceAll(s, "{home}", home)
	return s
}

func (c *Custom) EnvVars(profile config.Profile, profileDir string) (map[string]string, error) {
	envs := make(map[string]string)
	for k, v := range profile.Custom.Env {
		envs[k] = expandTemplateVars(v, profileDir, profile.Name)
	}
	return envs, nil
}

func (c *Custom) Validate(profile config.Profile) error {
	if len(profile.Custom.Env) == 0 &&
		profile.Custom.LoginCommand == "" &&
		profile.Custom.StatusCommand == "" &&
		profile.Custom.LogoutCommand == "" {
		return fmt.Errorf("custom provider requires at least one of: env, login_command, status_command, logout_command")
	}
	return nil
}

func (c *Custom) runCommand(command string, profileDir string, profile config.Profile, envs map[string]string) error {
	expanded := expandTemplateVars(command, profileDir, profile.Name)
	cmd := exec.Command("sh", "-c", expanded)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	cmd.Env = os.Environ()
	for k, v := range envs {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	return cmd.Run()
}

func (c *Custom) Login(profile config.Profile, profileDir string) error {
	if profile.Custom.LoginCommand == "" {
		return nil
	}

	if err := security.EnsureDir(profileDir); err != nil {
		return err
	}

	envs, _ := c.EnvVars(profile, profileDir)
	if err := c.runCommand(profile.Custom.LoginCommand, profileDir, profile, envs); err != nil {
		return fmt.Errorf("custom login command failed: %w", err)
	}
	return nil
}

func (c *Custom) Logout(profile config.Profile, profileDir string) error {
	if profile.Custom.LogoutCommand == "" {
		return nil
	}

	envs, _ := c.EnvVars(profile, profileDir)
	if err := c.runCommand(profile.Custom.LogoutCommand, profileDir, profile, envs); err != nil {
		return fmt.Errorf("custom logout command failed: %w", err)
	}
	return nil
}

func (c *Custom) Detect() (*provider.ImportInfo, error) {
	return nil, nil
}

func (c *Custom) Status(profile config.Profile, profileDir string) (*provider.SessionStatus, error) {
	if profile.Custom.StatusCommand == "" {
		return &provider.SessionStatus{Valid: false}, nil
	}

	expanded := expandTemplateVars(profile.Custom.StatusCommand, profileDir, profile.Name)
	cmd := exec.Command("sh", "-c", expanded)

	envs, _ := c.EnvVars(profile, profileDir)
	cmd.Env = os.Environ()
	for k, v := range envs {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	out, err := cmd.Output()
	if err != nil {
		return &provider.SessionStatus{Valid: false}, nil
	}

	return &provider.SessionStatus{
		Valid:    true,
		Identity: strings.TrimSpace(string(out)),
	}, nil
}
