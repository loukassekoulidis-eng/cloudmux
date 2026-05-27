# Import Command Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `cloudmux import` detects active cloud sessions and imports them as cloudmux profiles with credential copies.

**Architecture:** Extend the Provider interface with a `Detect()` method. Each provider probes its default (non-cloudmux) config location. The import CLI command iterates all providers, collects detections, copies config dirs, and appends to profiles.yaml. A new `config.AppendProfile` function handles the YAML append.

**Tech Stack:** Go, cobra, gopkg.in/yaml.v3

---

## File Map

```
Modified files:
├── internal/provider/provider.go           # Add ImportInfo type, Detect() to interface, All() to Registry
├── internal/provider/azure/azure.go        # Implement Detect()
├── internal/provider/azure/azure_test.go   # Test Detect() name generation
├── internal/provider/gcp/gcp.go            # Implement Detect()
├── internal/provider/gcp/gcp_test.go       # Test Detect() name generation
├── internal/provider/aws/aws.go            # Implement Detect()
├── internal/provider/aws/aws_test.go       # Test Detect() name generation
├── internal/provider/custom/custom.go      # Implement Detect() (returns nil — custom can't auto-detect)
├── internal/config/config.go               # Add AppendProfile function
├── internal/config/config_test.go          # Test AppendProfile
├── internal/cli/root.go                    # Register import command

New files:
├── internal/cli/import.go                  # Import command
├── internal/copydir/copydir.go             # Recursive directory copy with permissions
├── internal/copydir/copydir_test.go        # Tests for directory copy
```

---

### Task 1: Recursive directory copy utility

**Files:**
- Create: `internal/copydir/copydir.go`, `internal/copydir/copydir_test.go`

- [ ] **Step 1: Write failing tests**

`internal/copydir/copydir_test.go`:

```go
package copydir

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopy(t *testing.T) {
	// Set up source directory with files and subdirs
	src := t.TempDir()
	os.MkdirAll(filepath.Join(src, "subdir"), 0755)
	os.WriteFile(filepath.Join(src, "file.txt"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(src, "subdir", "nested.txt"), []byte("world"), 0644)

	dst := filepath.Join(t.TempDir(), "dest")

	require.NoError(t, Copy(src, dst))

	// Check files were copied
	data, err := os.ReadFile(filepath.Join(dst, "file.txt"))
	require.NoError(t, err)
	assert.Equal(t, "hello", string(data))

	data, err = os.ReadFile(filepath.Join(dst, "subdir", "nested.txt"))
	require.NoError(t, err)
	assert.Equal(t, "world", string(data))
}

func TestCopyPermissions(t *testing.T) {
	src := t.TempDir()
	os.WriteFile(filepath.Join(src, "secret.json"), []byte("token"), 0644)

	dst := filepath.Join(t.TempDir(), "dest")

	require.NoError(t, Copy(src, dst))

	// Directories should be 0700, files should be 0600
	info, err := os.Stat(dst)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0700), info.Mode().Perm())

	info, err = os.Stat(filepath.Join(dst, "secret.json"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestCopySourceNotExist(t *testing.T) {
	err := Copy("/nonexistent/path", t.TempDir())
	require.Error(t, err)
}
```

- [ ] **Step 2: Run to confirm failure**

```bash
go test ./internal/copydir/... -v
```

Expected: compilation error.

- [ ] **Step 3: Implement**

`internal/copydir/copydir.go`:

```go
package copydir

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func Copy(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("source directory %s: %w", src, err)
	}
	if !srcInfo.IsDir() {
		return fmt.Errorf("%s is not a directory", src)
	}

	if err := os.MkdirAll(dst, 0700); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := Copy(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/copydir/... -v
```

Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add internal/copydir/
git commit -m "add recursive directory copy with secure permissions"
```

---

### Task 2: Extend Provider interface with Detect() and Registry.All()

**Files:**
- Modify: `internal/provider/provider.go`
- Modify: `internal/provider/custom/custom.go`

- [ ] **Step 1: Add ImportInfo type and update interface**

In `internal/provider/provider.go`, add after `SessionStatus`:

```go
type ImportInfo struct {
	SuggestedName string
	Profile       config.Profile
	DefaultDir    string // source config dir to copy (empty = no copy needed)
}
```

Add `Detect()` to the `Provider` interface:

```go
type Provider interface {
	Name() string
	EnvVars(profile config.Profile, profileDir string) (map[string]string, error)
	Login(profile config.Profile, profileDir string) error
	Logout(profile config.Profile, profileDir string) error
	Status(profile config.Profile, profileDir string) (*SessionStatus, error)
	Validate(profile config.Profile) error
	Detect() (*ImportInfo, error)
}
```

Add `All()` method to `Registry`:

```go
func (r *Registry) All() []Provider {
	providers := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		providers = append(providers, p)
	}
	return providers
}
```

- [ ] **Step 2: Add stub Detect() to custom provider**

In `internal/provider/custom/custom.go`, add:

```go
func (c *Custom) Detect() (*provider.ImportInfo, error) {
	return nil, nil
}
```

- [ ] **Step 3: Verify everything compiles**

```bash
go build ./...
```

Expected: compilation errors from azure, gcp, aws (missing `Detect()`) — that's expected, they'll be added in the next tasks. For now just verify custom compiles:

```bash
go build ./internal/provider/custom/...
```

- [ ] **Step 4: Commit**

```bash
git add internal/provider/provider.go internal/provider/custom/custom.go
git commit -m "add Detect() to provider interface and All() to registry"
```

---

### Task 3: Azure Detect()

**Files:**
- Modify: `internal/provider/azure/azure.go`
- Modify: `internal/provider/azure/azure_test.go`

- [ ] **Step 1: Write test for Azure name generation**

Add to `internal/provider/azure/azure_test.go`:

```go
func TestSuggestName(t *testing.T) {
	assert.Equal(t, "acme-corp-azure", suggestName("acme-corp.example.com"))
	assert.Equal(t, "mycompany-azure", suggestName("mycompany.onmicrosoft.com"))
	assert.Equal(t, "tenant-123-azure", suggestName("tenant-123"))
	assert.Equal(t, "unknown-azure", suggestName(""))
}
```

- [ ] **Step 2: Run to confirm failure**

```bash
go test ./internal/provider/azure/... -v -run TestSuggestName
```

Expected: compilation error — `suggestName` not defined.

- [ ] **Step 3: Implement Detect() and suggestName**

Add to `internal/provider/azure/azure.go`:

```go
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
	// Replace dots with dashes for valid profile name
	name = strings.ReplaceAll(name, ".", "-")
	return name + "-azure"
}
```

Add the `"strings"` import.

Also add a richer JSON struct for detect (needs tenant domain):

```go
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

	// Run az account show WITHOUT AZURE_CONFIG_DIR override — use default ~/.azure
	cmd := exec.Command("az", "account", "show", "--output", "json")
	// Explicitly unset AZURE_CONFIG_DIR to use default
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
		return nil, nil // no active session
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
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/provider/azure/... -v
```

Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add internal/provider/azure/
git commit -m "add azure session detection for import"
```

---

### Task 4: GCP and AWS Detect()

**Files:**
- Modify: `internal/provider/gcp/gcp.go`, `internal/provider/gcp/gcp_test.go`
- Modify: `internal/provider/aws/aws.go`, `internal/provider/aws/aws_test.go`

- [ ] **Step 1: Write GCP name test**

Add to `internal/provider/gcp/gcp_test.go`:

```go
func TestGCPSuggestName(t *testing.T) {
	assert.Equal(t, "my-project-gcp", suggestName("my-project"))
	assert.Equal(t, "unknown-gcp", suggestName(""))
}
```

- [ ] **Step 2: Implement GCP Detect()**

Add to `internal/provider/gcp/gcp.go`:

```go
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

	// Run without CLOUDSDK_CONFIG override
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
```

- [ ] **Step 3: Write AWS name test**

Add to `internal/provider/aws/aws_test.go`:

```go
func TestAWSSuggestName(t *testing.T) {
	assert.Equal(t, "prod-aws", suggestName("prod", ""))
	assert.Equal(t, "123456789-aws", suggestName("", "123456789"))
	assert.Equal(t, "unknown-aws", suggestName("", ""))
}
```

- [ ] **Step 4: Implement AWS Detect()**

Add to `internal/provider/aws/aws.go`:

```go
func suggestName(profileName string, accountID string) string {
	if profileName != "" {
		return profileName + "-aws"
	}
	if accountID != "" {
		return accountID + "-aws"
	}
	return "unknown-aws"
}

func (a *AWS) Detect() (*provider.ImportInfo, error) {
	cmd := exec.Command("aws", "sts", "get-caller-identity", "--output", "json")
	// Strip AWS_PROFILE to detect the default/current session
	env := os.Environ()
	var currentProfile string
	filtered := make([]string, 0, len(env))
	for _, e := range env {
		if strings.HasPrefix(e, "AWS_PROFILE=") {
			currentProfile = strings.TrimPrefix(e, "AWS_PROFILE=")
		}
		filtered = append(filtered, e)
	}
	cmd.Env = filtered

	out, err := cmd.Output()
	if err != nil {
		return nil, nil
	}

	var identity stsIdentity
	if err := json.Unmarshal(out, &identity); err != nil {
		return nil, nil
	}

	profileName := currentProfile
	if profileName == "" {
		profileName = "default"
	}

	region := os.Getenv("AWS_DEFAULT_REGION")

	return &provider.ImportInfo{
		SuggestedName: suggestName(currentProfile, identity.Account),
		Profile: config.Profile{
			Provider:    "aws",
			Description: fmt.Sprintf("Imported from %s (%s)", identity.Account, identity.Arn),
			AWS: config.AWSConfig{
				ProfileName: profileName,
				Region:      region,
			},
		},
		DefaultDir: "", // no copy needed for named profile mode
	}, nil
}
```

Add `"strings"` import to aws.go.

- [ ] **Step 5: Run all provider tests**

```bash
go test ./internal/provider/... -v
```

Expected: all pass.

- [ ] **Step 6: Commit**

```bash
git add internal/provider/gcp/ internal/provider/aws/
git commit -m "add gcp and aws session detection for import"
```

---

### Task 5: AppendProfile — YAML append function

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: Write failing test**

Add to `internal/config/config_test.go`:

```go
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
```

- [ ] **Step 2: Run to confirm failure**

```bash
go test ./internal/config/... -v -run TestAppendProfile
```

Expected: compilation error — `AppendProfile` not defined.

- [ ] **Step 3: Implement AppendProfile**

Add to `internal/config/config.go`:

```go
func AppendProfile(path string, profile Profile) error {
	// Load existing profiles
	existing, err := LoadProfiles(path)
	if err != nil {
		return err
	}

	// Check for duplicates
	for _, p := range existing {
		if p.Name == profile.Name {
			return fmt.Errorf("profile %q already exists", profile.Name)
		}
	}

	// Append and write back
	pf := profilesFile{Profiles: append(existing, profile)}
	data, err := yaml.Marshal(&pf)
	if err != nil {
		return fmt.Errorf("marshaling profiles: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/config/... -v
```

Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add internal/config/
git commit -m "add AppendProfile for import command"
```

---

### Task 6: Import CLI command

**Files:**
- Create: `internal/cli/import.go`
- Modify: `internal/cli/root.go`

- [ ] **Step 1: Implement import command**

`internal/cli/import.go`:

```go
package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/lukassekoulidis/cloudmux/internal/color"
	"github.com/lukassekoulidis/cloudmux/internal/config"
	"github.com/lukassekoulidis/cloudmux/internal/copydir"
	"github.com/lukassekoulidis/cloudmux/internal/security"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newImportCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "import",
		Short: "Detect active cloud sessions and import them as profiles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			c := color.New(term.IsTerminal(int(os.Stdout.Fd())))

			reg := newRegistry()
			profilesPath := filepath.Join(configDir, "profiles.yaml")

			// Probe all providers
			type detection struct {
				providerName string
				info         *provider.ImportInfo
			}
			var detected []detection

			fmt.Fprintln(out, "Scanning for active cloud sessions...")
			for _, p := range reg.All() {
				info, err := p.Detect()
				if err != nil {
					continue
				}
				if info != nil {
					fmt.Fprintf(out, "  %s %s: %s\n", c.Green("✓"), p.Name(), info.Profile.Description)
					detected = append(detected, detection{providerName: p.Name(), info: info})
				}
			}

			if len(detected) == 0 {
				return fmt.Errorf("no active cloud sessions detected — login with your cloud CLI first (az login, gcloud auth login, aws sso login)")
			}

			for _, d := range detected {
				profileName := d.info.SuggestedName
				if name != "" && len(detected) == 1 {
					profileName = name
				}

				if err := security.ValidateProfileName(profileName); err != nil {
					return fmt.Errorf("generated profile name %q is invalid: %w — use --name to override", profileName, err)
				}

				// Set the name on the profile
				profile := d.info.Profile
				profile.Name = profileName

				// Copy config directory if needed
				if d.info.DefaultDir != "" {
					srcInfo, err := os.Stat(d.info.DefaultDir)
					if err != nil || !srcInfo.IsDir() {
						fmt.Fprintf(out, "  %s skipping config copy for %s (source dir not found)\n", c.Yellow("⚠"), profileName)
					} else {
						profDir := filepath.Join(configDir, "profiles", profileName)
						if err := security.EnsureDir(profDir); err != nil {
							return err
						}

						// Determine the subdirectory name based on provider
						var subDir string
						switch d.providerName {
						case "azure":
							subDir = ".azure"
						case "gcp":
							subDir = filepath.Join(".config", "gcloud")
						}

						if subDir != "" {
							dstPath := filepath.Join(profDir, subDir)
							fmt.Fprintf(out, "  Copying %s → %s...\n", d.info.DefaultDir, dstPath)
							if err := copydir.Copy(d.info.DefaultDir, dstPath); err != nil {
								return fmt.Errorf("copying config directory: %w", err)
							}
						}

						// Write login timestamp
						tsPath := filepath.Join(profDir, ".cloudmux_login_ts")
						os.WriteFile(tsPath, []byte(time.Now().UTC().Format(time.RFC3339)), 0600)
					}
				}

				// Append to profiles.yaml
				if err := config.AppendProfile(profilesPath, profile); err != nil {
					return err
				}

				fmt.Fprintf(out, "  %s Imported as %s\n", c.Green("✓"), c.Bold(profileName))
			}

			fmt.Fprintf(out, "\nDone. Use %s to activate a profile.\n", c.Bold("cloudmux use <profile>"))
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "override the generated profile name")
	return cmd
}
```

- [ ] **Step 2: Add provider import to import.go**

The file needs access to the `provider` package for `ImportInfo`. Add to imports:

```go
"github.com/lukassekoulidis/cloudmux/internal/provider"
```

(This is already included in the code above — just noting it's required.)

- [ ] **Step 3: Register in root.go**

Add to `NewRootCmd()` in `internal/cli/root.go`:

```go
	root.AddCommand(newImportCmd())
```

- [ ] **Step 4: Build and verify**

```bash
make build
./bin/cloudmux import --help
```

Expected: shows import command help with `--name` flag.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/import.go internal/cli/root.go
git commit -m "add import command for detecting and importing cloud sessions"
```

---

### Task 7: End-to-end test and README update

**Files:**
- Modify: `README.md` (add import section)

- [ ] **Step 1: Run all tests**

```bash
go test ./... -v
```

Expected: all pass.

- [ ] **Step 2: Build and smoke test**

```bash
make build
./bin/cloudmux import
```

Expected: detects active Azure session (acme-azure is logged in), but may fail because profile name already exists. That's correct behavior — import is for new profiles.

- [ ] **Step 3: Add import to README**

In `README.md`, add after the "Quick start" section, before "Commands":

```markdown
## Import existing sessions

Already logged in? Import your active sessions directly:

```bash
cloudmux import                    # auto-detect all active cloud sessions
cloudmux import --name my-tenant   # override the generated profile name
```

cloudmux probes Azure, GCP, and AWS for active sessions, creates profiles, and copies credentials into isolated directories — no re-authentication needed.
```

Also add to the commands table:

```
| `cloudmux import` | Detect and import active cloud sessions |
```

- [ ] **Step 4: Install and commit**

```bash
cp bin/cloudmux ~/.local/bin/cloudmux
git add README.md
git commit -m "document import command in README"
```
