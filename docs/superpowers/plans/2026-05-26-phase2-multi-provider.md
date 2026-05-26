# Phase 2: Multi-Provider Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend cloudmux to support GCP, AWS (named profile), and custom providers, plus `doctor` command, fish shell, and completions.

**Architecture:** Three new provider implementations following the existing `provider.Provider` interface. Config structs extended with GCP/AWS/Custom blocks. Fish hook translates POSIX export/unset output. Doctor checks CLI availability via `exec.LookPath`. Completions via cobra built-in.

**Tech Stack:** Go 1.22+, cobra, gopkg.in/yaml.v3, testify

---

## File Map

```
Changes to existing files:
├── internal/config/config.go          # Add GCPConfig, AWSConfig, CustomConfig to Profile
├── internal/config/config_test.go     # Tests for new config blocks
├── internal/cli/root.go              # Register GCP, AWS, custom providers + doctor + completion
├── internal/shell/hook.go            # Add fish hook template
├── internal/shell/hook_test.go       # Fish hook test

New files:
├── internal/provider/gcp/
│   ├── gcp.go                        # GCP provider implementation
│   └── gcp_test.go
├── internal/provider/aws/
│   ├── aws.go                        # AWS provider (named profile mode)
│   └── aws_test.go
├── internal/provider/custom/
│   ├── custom.go                     # Custom provider implementation
│   └── custom_test.go
├── internal/cli/doctor.go            # Doctor command
└── internal/cli/completion.go        # Completion command + ValidArgsFunction
```

---

### Task 1: Extend config with GCP, AWS, Custom blocks

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: Write failing tests for new config blocks**

Add to `internal/config/config_test.go`:

```go
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
```

- [ ] **Step 2: Run to confirm failure**

```bash
go test ./internal/config/... -v
```

Expected: compilation error — `GCP`, `AWS`, `Custom` fields not defined on `Profile`.

- [ ] **Step 3: Add new config types to config.go**

Add after `AzureConfig` in `internal/config/config.go`:

```go
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
```

Add fields to the `Profile` struct:

```go
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
```

- [ ] **Step 4: Run config tests**

```bash
go test ./internal/config/... -v
```

Expected: all pass (existing + new).

- [ ] **Step 5: Run full test suite to check nothing broke**

```bash
go test ./...
```

Expected: all pass.

- [ ] **Step 6: Commit**

```bash
git add internal/config/
git commit -m "extend profile config with gcp, aws, and custom blocks"
```

---

### Task 2: GCP provider

**Files:**
- Create: `internal/provider/gcp/gcp.go`, `internal/provider/gcp/gcp_test.go`

- [ ] **Step 1: Write failing tests**

`internal/provider/gcp/gcp_test.go`:

```go
package gcp

import (
	"testing"

	"github.com/lukassekoulidis/cloudmux/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGCPName(t *testing.T) {
	p := New()
	assert.Equal(t, "gcp", p.Name())
}

func TestGCPEnvVarsIsolated(t *testing.T) {
	p := New()
	profile := config.Profile{
		Name:     "my-gcp",
		Provider: "gcp",
		GCP: config.GCPConfig{
			ProjectID: "my-project",
			Region:    "europe-west3",
			Zone:      "europe-west3-a",
		},
	}
	envs, err := p.EnvVars(profile, "/home/user/.cloudmux/profiles/my-gcp")
	require.NoError(t, err)

	assert.Equal(t, "/home/user/.cloudmux/profiles/my-gcp/.config/gcloud", envs["CLOUDSDK_CONFIG"])
	assert.Equal(t, "my-project", envs["CLOUDSDK_CORE_PROJECT"])
	assert.Equal(t, "europe-west3", envs["CLOUDSDK_COMPUTE_REGION"])
	assert.Equal(t, "europe-west3-a", envs["CLOUDSDK_COMPUTE_ZONE"])
	_, hasCloudsdk := envs["CLOUDSDK_ACTIVE_CONFIG_NAME"]
	assert.False(t, hasCloudsdk)
}

func TestGCPEnvVarsNamedConfig(t *testing.T) {
	p := New()
	profile := config.Profile{
		Name:     "my-gcp",
		Provider: "gcp",
		GCP: config.GCPConfig{
			ProjectID:      "my-project",
			UseNamedConfig: true,
		},
	}
	envs, err := p.EnvVars(profile, "/home/user/.cloudmux/profiles/my-gcp")
	require.NoError(t, err)

	assert.Equal(t, "my-gcp", envs["CLOUDSDK_ACTIVE_CONFIG_NAME"])
	_, hasCloudsdk := envs["CLOUDSDK_CONFIG"]
	assert.False(t, hasCloudsdk)
}

func TestGCPEnvVarsMinimal(t *testing.T) {
	p := New()
	profile := config.Profile{
		Name:     "minimal",
		Provider: "gcp",
		GCP: config.GCPConfig{
			ProjectID: "proj",
		},
	}
	envs, err := p.EnvVars(profile, "/tmp/profiles/minimal")
	require.NoError(t, err)

	assert.Equal(t, "/tmp/profiles/minimal/.config/gcloud", envs["CLOUDSDK_CONFIG"])
	assert.Equal(t, "proj", envs["CLOUDSDK_CORE_PROJECT"])
	_, hasRegion := envs["CLOUDSDK_COMPUTE_REGION"]
	assert.False(t, hasRegion)
	_, hasZone := envs["CLOUDSDK_COMPUTE_ZONE"]
	assert.False(t, hasZone)
}

func TestGCPValidate(t *testing.T) {
	p := New()

	t.Run("valid", func(t *testing.T) {
		profile := config.Profile{
			Provider: "gcp",
			GCP:      config.GCPConfig{ProjectID: "my-proj"},
		}
		assert.NoError(t, p.Validate(profile))
	})

	t.Run("missing project_id", func(t *testing.T) {
		profile := config.Profile{
			Provider: "gcp",
			GCP:      config.GCPConfig{},
		}
		err := p.Validate(profile)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project_id")
	})
}
```

- [ ] **Step 2: Run to confirm failure**

```bash
go test ./internal/provider/gcp/... -v
```

Expected: compilation error — `New` not defined.

- [ ] **Step 3: Implement GCP provider**

`internal/provider/gcp/gcp.go`:

```go
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
```

- [ ] **Step 4: Run GCP tests**

```bash
go test ./internal/provider/gcp/... -v
```

Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add internal/provider/gcp/
git commit -m "add gcp provider with isolated and named-config modes"
```

---

### Task 3: AWS provider (named profile mode)

**Files:**
- Create: `internal/provider/aws/aws.go`, `internal/provider/aws/aws_test.go`

- [ ] **Step 1: Write failing tests**

`internal/provider/aws/aws_test.go`:

```go
package aws

import (
	"testing"

	"github.com/lukassekoulidis/cloudmux/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAWSName(t *testing.T) {
	p := New()
	assert.Equal(t, "aws", p.Name())
}

func TestAWSEnvVars(t *testing.T) {
	p := New()
	profile := config.Profile{
		Name:     "my-aws",
		Provider: "aws",
		AWS: config.AWSConfig{
			ProfileName: "prod-account",
			Region:      "eu-central-1",
		},
	}
	envs, err := p.EnvVars(profile, "/home/user/.cloudmux/profiles/my-aws")
	require.NoError(t, err)

	assert.Equal(t, "prod-account", envs["AWS_PROFILE"])
	assert.Equal(t, "eu-central-1", envs["AWS_DEFAULT_REGION"])
}

func TestAWSEnvVarsNoRegion(t *testing.T) {
	p := New()
	profile := config.Profile{
		Name:     "minimal",
		Provider: "aws",
		AWS: config.AWSConfig{
			ProfileName: "dev",
		},
	}
	envs, err := p.EnvVars(profile, "/tmp/profiles/minimal")
	require.NoError(t, err)

	assert.Equal(t, "dev", envs["AWS_PROFILE"])
	_, hasRegion := envs["AWS_DEFAULT_REGION"]
	assert.False(t, hasRegion)
}

func TestAWSValidate(t *testing.T) {
	p := New()

	t.Run("valid", func(t *testing.T) {
		profile := config.Profile{
			Provider: "aws",
			AWS:      config.AWSConfig{ProfileName: "my-profile"},
		}
		assert.NoError(t, p.Validate(profile))
	})

	t.Run("missing profile_name", func(t *testing.T) {
		profile := config.Profile{
			Provider: "aws",
			AWS:      config.AWSConfig{},
		}
		err := p.Validate(profile)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "profile_name")
	})
}
```

- [ ] **Step 2: Run to confirm failure**

```bash
go test ./internal/provider/aws/... -v
```

Expected: compilation error.

- [ ] **Step 3: Implement AWS provider**

`internal/provider/aws/aws.go`:

```go
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
```

- [ ] **Step 4: Run AWS tests**

```bash
go test ./internal/provider/aws/... -v
```

Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add internal/provider/aws/
git commit -m "add aws provider with named profile mode"
```

---

### Task 4: Custom provider

**Files:**
- Create: `internal/provider/custom/custom.go`, `internal/provider/custom/custom_test.go`

- [ ] **Step 1: Write failing tests**

`internal/provider/custom/custom_test.go`:

```go
package custom

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lukassekoulidis/cloudmux/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomName(t *testing.T) {
	p := New()
	assert.Equal(t, "custom", p.Name())
}

func TestCustomEnvVars(t *testing.T) {
	p := New()
	profile := config.Profile{
		Name:     "hetzner-prod",
		Provider: "custom",
		Custom: config.CustomConfig{
			Env: map[string]string{
				"HCLOUD_TOKEN_FILE": "{profile_dir}/token",
				"HCLOUD_CONTEXT":    "{name}",
			},
		},
	}
	envs, err := p.EnvVars(profile, "/home/user/.cloudmux/profiles/hetzner-prod")
	require.NoError(t, err)

	assert.Equal(t, "/home/user/.cloudmux/profiles/hetzner-prod/token", envs["HCLOUD_TOKEN_FILE"])
	assert.Equal(t, "hetzner-prod", envs["HCLOUD_CONTEXT"])
}

func TestCustomEnvVarsHomeExpansion(t *testing.T) {
	p := New()
	home, _ := os.UserHomeDir()
	profile := config.Profile{
		Name:     "test",
		Provider: "custom",
		Custom: config.CustomConfig{
			Env: map[string]string{
				"MY_VAR": "{home}/.myconfig",
			},
		},
	}
	envs, err := p.EnvVars(profile, "/tmp/profiles/test")
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(home, ".myconfig"), envs["MY_VAR"])
}

func TestCustomValidate(t *testing.T) {
	p := New()

	t.Run("valid with env", func(t *testing.T) {
		profile := config.Profile{
			Provider: "custom",
			Custom: config.CustomConfig{
				Env: map[string]string{"KEY": "val"},
			},
		}
		assert.NoError(t, p.Validate(profile))
	})

	t.Run("valid with login_command only", func(t *testing.T) {
		profile := config.Profile{
			Provider: "custom",
			Custom: config.CustomConfig{
				LoginCommand: "do-something",
			},
		}
		assert.NoError(t, p.Validate(profile))
	})

	t.Run("empty custom config", func(t *testing.T) {
		profile := config.Profile{
			Provider: "custom",
			Custom:   config.CustomConfig{},
		}
		err := p.Validate(profile)
		require.Error(t, err)
	})
}

func TestExpandTemplateVars(t *testing.T) {
	home, _ := os.UserHomeDir()
	result := expandTemplateVars("path/{profile_dir}/sub/{name}/end/{home}", "/data/profiles/foo", "foo")
	assert.Equal(t, "path//data/profiles/foo/sub/foo/end/"+home, result)
}
```

- [ ] **Step 2: Run to confirm failure**

```bash
go test ./internal/provider/custom/... -v
```

Expected: compilation error.

- [ ] **Step 3: Implement custom provider**

`internal/provider/custom/custom.go`:

```go
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

	// Set the custom env vars in the command's environment
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
```

- [ ] **Step 4: Run custom tests**

```bash
go test ./internal/provider/custom/... -v
```

Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add internal/provider/custom/
git commit -m "add custom provider with template variable expansion"
```

---

### Task 5: Register new providers in root.go

**Files:**
- Modify: `internal/cli/root.go`

- [ ] **Step 1: Update newRegistry to register all providers**

Replace `newRegistry` in `internal/cli/root.go`:

```go
import (
	"os"
	"path/filepath"

	"github.com/lukassekoulidis/cloudmux/internal/provider"
	"github.com/lukassekoulidis/cloudmux/internal/provider/aws"
	"github.com/lukassekoulidis/cloudmux/internal/provider/azure"
	"github.com/lukassekoulidis/cloudmux/internal/provider/custom"
	"github.com/lukassekoulidis/cloudmux/internal/provider/gcp"
	"github.com/lukassekoulidis/cloudmux/internal/session"
	"github.com/spf13/cobra"
)

func newRegistry() *provider.Registry {
	reg := provider.NewRegistry()
	reg.Register(azure.New())
	reg.Register(gcp.New())
	reg.Register(aws.New())
	reg.Register(custom.New())
	return reg
}
```

- [ ] **Step 2: Build and verify**

```bash
make build
./bin/cloudmux --help
```

Expected: builds clean, help output unchanged.

- [ ] **Step 3: Run full test suite**

```bash
go test ./...
```

Expected: all pass.

- [ ] **Step 4: Commit**

```bash
git add internal/cli/root.go
git commit -m "register gcp, aws, and custom providers"
```

---

### Task 6: Doctor command

**Files:**
- Create: `internal/cli/doctor.go`

- [ ] **Step 1: Implement doctor command**

`internal/cli/doctor.go`:

```go
package cli

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/lukassekoulidis/cloudmux/internal/config"
	"github.com/lukassekoulidis/cloudmux/internal/security"
	"github.com/spf13/cobra"
)

var providerCLIs = map[string]struct {
	Binary     string
	InstallURL string
}{
	"azure":  {"az", "https://learn.microsoft.com/en-us/cli/azure/install-azure-cli"},
	"gcp":    {"gcloud", "https://cloud.google.com/sdk/docs/install"},
	"aws":    {"aws", "https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html"},
}

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check prerequisites and configuration health",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			// Check config directory
			if err := security.EnforcePermissions(configDir, true); err != nil {
				fmt.Fprintf(out, "✗ Config directory: %s\n  %s\n", configDir, err)
			} else {
				fmt.Fprintf(out, "✓ Config directory: %s (0700)\n", configDir)
			}

			// Load profiles
			profilesPath := filepath.Join(configDir, "profiles.yaml")
			profiles, err := config.LoadProfiles(profilesPath)
			if err != nil {
				fmt.Fprintf(out, "✗ Profiles: %s\n", err)
				return nil
			}
			fmt.Fprintf(out, "✓ Profiles: %d profiles loaded\n", len(profiles))

			// Collect which providers are in use
			providerProfiles := make(map[string][]string)
			for _, p := range profiles {
				providerProfiles[p.Provider] = append(providerProfiles[p.Provider], p.Name)
			}

			if len(providerProfiles) == 0 {
				return nil
			}

			fmt.Fprintln(out, "\nProvider CLIs:")
			for provName, profileNames := range providerProfiles {
				info, ok := providerCLIs[provName]
				if !ok {
					// custom or unknown provider — no CLI to check
					continue
				}
				_, err := exec.LookPath(info.Binary)
				if err != nil {
					fmt.Fprintf(out, "  ✗ %-8s (profiles: %s) — install: %s\n",
						info.Binary, joinNames(profileNames), info.InstallURL)
				} else {
					fmt.Fprintf(out, "  ✓ %-8s (profiles: %s)\n",
						info.Binary, joinNames(profileNames))
				}
			}

			return nil
		},
	}
}

func joinNames(names []string) string {
	result := ""
	for i, n := range names {
		if i > 0 {
			result += ", "
		}
		result += n
	}
	return result
}
```

- [ ] **Step 2: Register in root.go**

Add to `NewRootCmd()` in `internal/cli/root.go`:

```go
	root.AddCommand(newDoctorCmd())
```

- [ ] **Step 3: Build and test manually**

```bash
make build
./bin/cloudmux doctor
```

Expected: shows config dir status, profile count, and checks for `az` (and `gcloud`/`aws` if configured).

- [ ] **Step 4: Commit**

```bash
git add internal/cli/doctor.go internal/cli/root.go
git commit -m "add doctor command to check prerequisites"
```

---

### Task 7: Fish shell hook

**Files:**
- Modify: `internal/shell/hook.go`
- Modify: `internal/shell/hook_test.go`

- [ ] **Step 1: Write failing test for fish hook**

Add to `internal/shell/hook_test.go`:

```go
func TestGenerateHookFish(t *testing.T) {
	out, err := GenerateHook("fish", "cloudmux")
	require.NoError(t, err)
	assert.Contains(t, out, "function cloudmux")
	assert.Contains(t, out, "set -gx")
	assert.Contains(t, out, "set -e")
	assert.Contains(t, out, "CLOUDMUX_ACTIVE_PROFILE")
}
```

- [ ] **Step 2: Run to confirm failure**

```bash
go test ./internal/shell/... -v -run TestGenerateHookFish
```

Expected: fails — fish hook returns "unsupported shell" error.

- [ ] **Step 3: Add fish hook template**

Add to `internal/shell/hook.go`, before the `var hooks = map[string]string{` line:

```go
var fishHook = `# cloudmux shell hook (fish)
function cloudmux
    set -l cmd $argv[1]
    switch "$cmd"
        case use logout
            set -l output (command {{.Binary}} $argv 2>/dev/null)
            set -l rc $status
            if test $rc -eq 0
                for line in $output
                    # Convert POSIX export/unset to fish syntax
                    if string match -qr '^export ([^=]+)='\''(.*)'\''$' -- $line
                        set -l matches (string match -r '^export ([^=]+)='\''(.*)'\''$' -- $line)
                        set -gx $matches[2] $matches[3]
                    else if string match -qr '^unset (.+)$' -- $line
                        set -l matches (string match -r '^unset (.+)$' -- $line)
                        set -e $matches[2]
                    end
                end
            else
                command {{.Binary}} $argv
                return $rc
            end
        case '*'
            command {{.Binary}} $argv
    end
end

# Prompt integration (skipped if Starship is active)
if not set -q STARSHIP_SHELL
    function _cloudmux_prompt --on-event fish_prompt
        if set -q CLOUDMUX_ACTIVE_PROFILE
            echo -n "[cloudmux: $CLOUDMUX_ACTIVE_PROFILE] "
        end
    end
end
`
```

Update the hooks map:

```go
var hooks = map[string]string{
	"bash": bashHook,
	"zsh":  zshHook,
	"fish": fishHook,
}
```

Also update the error message in `GenerateHook`:

```go
return "", fmt.Errorf("unsupported shell %q (supported: bash, zsh, fish)", shellType)
```

- [ ] **Step 4: Update unsupported shell test**

In `internal/shell/hook_test.go`, the "unsupported shell" test already checks for "unsupported" in the error message, so it still passes. No change needed.

- [ ] **Step 5: Run all shell tests**

```bash
go test ./internal/shell/... -v
```

Expected: all pass (bash, zsh, fish, unsupported).

- [ ] **Step 6: Commit**

```bash
git add internal/shell/
git commit -m "add fish shell hook with posix-to-fish translation"
```

---

### Task 8: Shell completions + profile name completion

**Files:**
- Create: `internal/cli/completion.go`
- Modify: `internal/cli/root.go` (register completion command)
- Modify: `internal/cli/login.go`, `internal/cli/use.go`, `internal/cli/status.go`, `internal/cli/logout.go` (add ValidArgsFunction)

- [ ] **Step 1: Create completion command**

`internal/cli/completion.go`:

```go
package cli

import (
	"os"

	"github.com/spf13/cobra"
)

func newCompletionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "completion <bash|zsh|fish>",
		Short: "Generate shell completion script",
		Long: `Generate a completion script for the specified shell.

  bash:  source <(cloudmux completion bash)
  zsh:   source <(cloudmux completion zsh)
  fish:  cloudmux completion fish | source`,
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"bash", "zsh", "fish"},
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			default:
				return cmd.Help()
			}
		},
	}
}
```

- [ ] **Step 2: Add profile name completion function**

Add a helper to `internal/cli/completion.go`:

```go
func profileCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	mgr, err := newManager()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var names []string
	for _, p := range mgr.Profiles() {
		names = append(names, p.Name)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}
```

- [ ] **Step 3: Add ValidArgsFunction to login, use, status, logout**

In `internal/cli/login.go`, add to the cobra.Command struct:

```go
ValidArgsFunction: profileCompletionFunc,
```

Do the same in `use.go`, `status.go`, and `logout.go`.

- [ ] **Step 4: Register completion command in root.go**

Add to `NewRootCmd()`:

```go
	root.AddCommand(newCompletionCmd())
```

- [ ] **Step 5: Build and verify**

```bash
make build
./bin/cloudmux completion bash | head -5
./bin/cloudmux completion zsh | head -5
./bin/cloudmux completion fish | head -5
```

Expected: each outputs a valid completion script.

- [ ] **Step 6: Commit**

```bash
git add internal/cli/
git commit -m "add shell completions and profile name tab completion"
```

---

### Task 9: End-to-end verification

**Files:** none (verification only)

- [ ] **Step 1: Run all tests**

```bash
go test ./... -v
```

Expected: all pass.

- [ ] **Step 2: Run linter**

```bash
make lint
```

Expected: clean.

- [ ] **Step 3: Run vet**

```bash
make vet
```

Expected: clean.

- [ ] **Step 4: Smoke test all commands**

```bash
make build

# Doctor
./bin/cloudmux doctor

# Shell hooks
./bin/cloudmux shell-hook bash | head -3
./bin/cloudmux shell-hook zsh | head -3
./bin/cloudmux shell-hook fish | head -3

# Completions
./bin/cloudmux completion bash | head -3
./bin/cloudmux completion zsh | head -3
./bin/cloudmux completion fish | head -3

# List (should show existing wbai-azure)
./bin/cloudmux list

# Help (should show all commands including doctor, completion)
./bin/cloudmux --help
```

Expected: all commands work, help shows doctor and completion.

- [ ] **Step 5: Install updated binary**

```bash
cp bin/cloudmux ~/.local/bin/cloudmux
```

- [ ] **Step 6: Commit any fixes**

```bash
git add -A
git commit -m "phase 2 cleanup"  # only if there were fixes
```
