# Phase 3: Quality of Life Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add gc, TTL, audit logging, token expiry warnings, confirm-on-use, and colored output.

**Architecture:** New leaf packages (`internal/color`, `internal/audit`) with no cross-dependencies. Config and Profile structs extended. Session manager gains audit + TTL awareness. CLI commands updated to use color and new session manager features.

**Tech Stack:** Go 1.22+, cobra, golang.org/x/term (for terminal detection in confirm-on-use)

---

## File Map

```
New files:
├── internal/color/color.go              # Terminal color helpers (NO_COLOR aware)
├── internal/color/color_test.go
├── internal/audit/audit.go              # Append-only audit log with rotation
├── internal/audit/audit_test.go
├── internal/cli/gc.go                   # GC command

Modified files:
├── internal/config/config.go            # Add TTLDays, ConfirmOnUse to Profile
├── internal/config/config_test.go
├── internal/session/manager.go          # Add audit logging, login timestamp, TTL check
├── internal/session/manager_test.go
├── internal/cli/root.go                 # Register gc, pass config to manager
├── internal/cli/use.go                  # Add confirm-on-use, TTL warning, expiry warning
├── internal/cli/list.go                 # Add color, status column
├── internal/cli/status.go              # Add color
├── internal/cli/doctor.go              # Add color
├── internal/cli/login.go               # Add audit
```

---

### Task 1: Color helpers

**Files:**
- Create: `internal/color/color.go`, `internal/color/color_test.go`

- [ ] **Step 1: Write failing tests**

`internal/color/color_test.go`:

```go
package color

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGreen(t *testing.T) {
	// Force color on for testing
	os.Unsetenv("NO_COLOR")
	c := New(true)
	assert.Equal(t, "\033[32mhello\033[0m", c.Green("hello"))
}

func TestRed(t *testing.T) {
	os.Unsetenv("NO_COLOR")
	c := New(true)
	assert.Equal(t, "\033[31merror\033[0m", c.Red("error"))
}

func TestYellow(t *testing.T) {
	os.Unsetenv("NO_COLOR")
	c := New(true)
	assert.Equal(t, "\033[33mwarn\033[0m", c.Yellow("warn"))
}

func TestBold(t *testing.T) {
	os.Unsetenv("NO_COLOR")
	c := New(true)
	assert.Equal(t, "\033[1mtitle\033[0m", c.Bold("title"))
}

func TestDim(t *testing.T) {
	os.Unsetenv("NO_COLOR")
	c := New(true)
	assert.Equal(t, "\033[2mfaded\033[0m", c.Dim("faded"))
}

func TestNoColor(t *testing.T) {
	os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")
	c := New(true)
	assert.Equal(t, "hello", c.Green("hello"))
	assert.Equal(t, "error", c.Red("error"))
}

func TestNotTTY(t *testing.T) {
	os.Unsetenv("NO_COLOR")
	c := New(false) // not a terminal
	assert.Equal(t, "hello", c.Green("hello"))
}
```

- [ ] **Step 2: Run to confirm failure**

```bash
go test ./internal/color/... -v
```

Expected: compilation error.

- [ ] **Step 3: Implement color helpers**

`internal/color/color.go`:

```go
package color

import (
	"fmt"
	"os"
)

type Color struct {
	enabled bool
}

func New(isTTY bool) *Color {
	enabled := isTTY && os.Getenv("NO_COLOR") == ""
	return &Color{enabled: enabled}
}

func (c *Color) wrap(code string, s string) string {
	if !c.enabled {
		return s
	}
	return fmt.Sprintf("\033[%sm%s\033[0m", code, s)
}

func (c *Color) Green(s string) string  { return c.wrap("32", s) }
func (c *Color) Red(s string) string    { return c.wrap("31", s) }
func (c *Color) Yellow(s string) string { return c.wrap("33", s) }
func (c *Color) Bold(s string) string   { return c.wrap("1", s) }
func (c *Color) Dim(s string) string    { return c.wrap("2", s) }
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/color/... -v
```

Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add internal/color/
git commit -m "add terminal color helpers with NO_COLOR support"
```

---

### Task 2: Audit logging

**Files:**
- Create: `internal/audit/audit.go`, `internal/audit/audit_test.go`

- [ ] **Step 1: Write failing tests**

`internal/audit/audit_test.go`:

```go
package audit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLog(t *testing.T) {
	dir := t.TempDir()
	os.Chmod(dir, 0700)
	logPath := filepath.Join(dir, "audit.log")

	logger := New(logPath)
	require.NoError(t, logger.Log("LOGIN", "my-profile", "azure", "user@test.com"))
	require.NoError(t, logger.Log("USE", "my-profile", "azure", ""))

	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	assert.Len(t, lines, 2)
	assert.Contains(t, lines[0], "LOGIN")
	assert.Contains(t, lines[0], "my-profile")
	assert.Contains(t, lines[0], "azure")
	assert.Contains(t, lines[1], "USE")
}

func TestLogFilePermissions(t *testing.T) {
	dir := t.TempDir()
	os.Chmod(dir, 0700)
	logPath := filepath.Join(dir, "audit.log")

	logger := New(logPath)
	require.NoError(t, logger.Log("LOGIN", "test", "azure", ""))

	info, err := os.Stat(logPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestLogRotation(t *testing.T) {
	dir := t.TempDir()
	os.Chmod(dir, 0700)
	logPath := filepath.Join(dir, "audit.log")

	logger := NewWithMaxLines(logPath, 20)

	// Write 25 lines
	for i := 0; i < 25; i++ {
		require.NoError(t, logger.Log("USE", "profile", "azure", ""))
	}

	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	// Should have been rotated: kept last 10 (half of 20) + 5 new = 15
	// Actually: after 20 lines, rotate to 10, then write 5 more = 15
	assert.LessOrEqual(t, len(lines), 20)
	assert.Greater(t, len(lines), 0)
}
```

- [ ] **Step 2: Run to confirm failure**

```bash
go test ./internal/audit/... -v
```

Expected: compilation error.

- [ ] **Step 3: Implement audit logger**

`internal/audit/audit.go`:

```go
package audit

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Logger struct {
	path     string
	maxLines int
}

func New(path string) *Logger {
	return &Logger{path: path, maxLines: 10000}
}

func NewWithMaxLines(path string, maxLines int) *Logger {
	return &Logger{path: path, maxLines: maxLines}
}

func (l *Logger) Log(action, profile, provider, details string) error {
	entry := fmt.Sprintf("%s %s %s %s", time.Now().UTC().Format(time.RFC3339), action, profile, provider)
	if details != "" {
		entry += " " + details
	}
	entry += "\n"

	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.WriteString(entry); err != nil {
		return err
	}

	// Check rotation
	return l.maybeRotate()
}

func (l *Logger) maybeRotate() error {
	data, err := os.ReadFile(l.path)
	if err != nil {
		return nil
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) <= l.maxLines {
		return nil
	}

	// Keep the last half
	keep := l.maxLines / 2
	if keep > len(lines) {
		return nil
	}
	trimmed := strings.Join(lines[len(lines)-keep:], "\n")
	return os.WriteFile(l.path, []byte(trimmed), 0600)
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/audit/... -v
```

Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add internal/audit/
git commit -m "add audit logger with line-based rotation"
```

---

### Task 3: Config extension — TTLDays, ConfirmOnUse

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: Write failing test**

Add to `internal/config/config_test.go`:

```go
func TestLoadProfilesWithTTLAndConfirm(t *testing.T) {
	f := filepath.Join(t.TempDir(), "profiles.yaml")
	os.WriteFile(f, []byte(`
profiles:
  - name: prod-azure
    provider: azure
    confirm_on_use: true
    ttl_days: 90
    tags: [production, client]
    azure:
      tenant_id: "t-123"
`), 0600)
	profiles, err := LoadProfiles(f)
	require.NoError(t, err)
	require.Len(t, profiles, 1)
	assert.True(t, profiles[0].ConfirmOnUse)
	assert.Equal(t, 90, profiles[0].TTLDays)
	assert.Contains(t, profiles[0].Tags, "production")
}
```

- [ ] **Step 2: Run to confirm failure**

```bash
go test ./internal/config/... -v -run TestLoadProfilesWithTTLAndConfirm
```

Expected: compilation error — `ConfirmOnUse` and `TTLDays` not defined.

- [ ] **Step 3: Add fields to Profile**

In `internal/config/config.go`, update the `Profile` struct:

```go
type Profile struct {
	Name         string       `yaml:"name"`
	Provider     string       `yaml:"provider"`
	Description  string       `yaml:"description"`
	Tags         []string     `yaml:"tags"`
	ConfirmOnUse bool         `yaml:"confirm_on_use"`
	TTLDays      int          `yaml:"ttl_days"`
	Azure        AzureConfig  `yaml:"azure"`
	GCP          GCPConfig    `yaml:"gcp"`
	AWS          AWSConfig    `yaml:"aws"`
	Custom       CustomConfig `yaml:"custom"`
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/config/... -v
```

Expected: all pass.

- [ ] **Step 5: Run full suite**

```bash
go test ./...
```

Expected: all pass.

- [ ] **Step 6: Commit**

```bash
git add internal/config/
git commit -m "add confirm_on_use and ttl_days to profile config"
```

---

### Task 4: Session manager — audit logging and login timestamp

**Files:**
- Modify: `internal/session/manager.go`
- Modify: `internal/session/manager_test.go`

- [ ] **Step 1: Write failing test for login timestamp**

Add to `internal/session/manager_test.go`:

```go
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

	tsPath := filepath.Join(baseDir, "profiles", "test-profile", ".cloudmux_login_ts")
	_, err = os.Stat(tsPath)
	assert.NoError(t, err)
}

func TestManagerLoginTimestamp(t *testing.T) {
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
```

- [ ] **Step 2: Run to confirm failure**

```bash
go test ./internal/session/... -v
```

Expected: compilation error — `NewManager` now expects 3 args.

- [ ] **Step 3: Update session manager**

Update `internal/session/manager.go`:

```go
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

func (m *Manager) logAudit(action, profile, providerName, details string) {
	if m.audit != nil {
		m.audit.Log(action, profile, providerName, details)
	}
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

	// Write login timestamp
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

func (m *Manager) ProfileDir(name string) string {
	return m.profileDir(name)
}

func (m *Manager) LoginTimestamp(profileName string) (time.Time, error) {
	tsPath := filepath.Join(m.profileDir(profileName), ".cloudmux_login_ts")
	data, err := os.ReadFile(tsPath)
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339, string(data))
}
```

- [ ] **Step 4: Update existing tests**

Update `setupTestManager` in `internal/session/manager_test.go` to pass `nil` for audit logger:

```go
m, err := NewManager(baseDir, reg, nil)
```

- [ ] **Step 5: Update root.go**

Update `newManager()` in `internal/cli/root.go` to create and pass audit logger:

```go
func newManager() (*session.Manager, error) {
	auditPath := filepath.Join(configDir, "audit.log")
	auditLogger := audit.New(auditPath)
	return session.NewManager(configDir, newRegistry(), auditLogger)
}
```

Add the import:
```go
"github.com/lukassekoulidis/cloudmux/internal/audit"
```

- [ ] **Step 6: Run all tests**

```bash
go test ./...
```

Expected: all pass.

- [ ] **Step 7: Commit**

```bash
git add internal/session/ internal/cli/root.go
git commit -m "add audit logging and login timestamps to session manager"
```

---

### Task 5: GC command

**Files:**
- Create: `internal/cli/gc.go`
- Modify: `internal/cli/root.go` (register)

- [ ] **Step 1: Implement gc command**

`internal/cli/gc.go`:

```go
package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/lukassekoulidis/cloudmux/internal/config"
	"github.com/spf13/cobra"
)

func newGCCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "gc",
		Short: "List and clean up stale profiles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := newManager()
			if err != nil {
				return err
			}

			cfg, err := config.LoadConfig(filepath.Join(configDir, "config.yaml"))
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			profiles := mgr.Profiles()
			var stale []string

			for _, p := range profiles {
				profDir := mgr.ProfileDir(p.Name)

				// Check if profile directory exists
				if _, err := os.Stat(profDir); os.IsNotExist(err) {
					fmt.Fprintf(out, "  ○ %s — never logged in\n", p.Name)
					continue
				}

				// Check TTL
				ttl := p.TTLDays
				if ttl == 0 {
					ttl = cfg.DefaultTTLDays
				}
				if ttl > 0 {
					ts, err := mgr.LoginTimestamp(p.Name)
					if err == nil {
						age := time.Since(ts)
						if age > time.Duration(ttl)*24*time.Hour {
							fmt.Fprintf(out, "  ⚠ %s — past TTL (%d days, logged in %s ago)\n",
								p.Name, ttl, formatDuration(age))
							stale = append(stale, p.Name)
							continue
						}
					}
				}

				// Check session validity
				status, err := mgr.Status(p.Name)
				if err == nil && !status.Valid {
					fmt.Fprintf(out, "  ✗ %s — session expired\n", p.Name)
					stale = append(stale, p.Name)
					continue
				}

				fmt.Fprintf(out, "  ✓ %s — ok\n", p.Name)
			}

			if len(stale) == 0 {
				fmt.Fprintln(out, "\nNo stale profiles found.")
				return nil
			}

			if !force {
				fmt.Fprintf(out, "\n%d stale profile(s). Run with --force to remove their data directories.\n", len(stale))
				return nil
			}

			for _, name := range stale {
				profDir := mgr.ProfileDir(name)
				if err := os.RemoveAll(profDir); err != nil {
					fmt.Fprintf(out, "  ✗ Failed to remove %s: %s\n", name, err)
				} else {
					fmt.Fprintf(out, "  ✓ Removed %s\n", name)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "actually remove stale profile directories")
	return cmd
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	if days > 0 {
		return fmt.Sprintf("%dd", days)
	}
	hours := int(d.Hours())
	if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dm", int(d.Minutes()))
}
```

- [ ] **Step 2: Register in root.go**

Add to `NewRootCmd()`:

```go
	root.AddCommand(newGCCmd())
```

- [ ] **Step 3: Build and test**

```bash
make build
./bin/cloudmux gc
```

Expected: shows profile status. wbai-azure should show as ok if session is valid.

- [ ] **Step 4: Commit**

```bash
git add internal/cli/gc.go internal/cli/root.go
git commit -m "add gc command for stale profile cleanup"
```

---

### Task 6: Enhanced use command — confirm, TTL warning, expiry warning

**Files:**
- Modify: `internal/cli/use.go`

- [ ] **Step 1: Install golang.org/x/term**

```bash
go get golang.org/x/term@latest
```

- [ ] **Step 2: Update use command**

Replace `internal/cli/use.go`:

```go
package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lukassekoulidis/cloudmux/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "use <profile>",
		Short:             "Activate a profile in the current shell",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: profileCompletionFunc,
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := newManager()
			if err != nil {
				return err
			}

			cfg, err := config.LoadConfig(filepath.Join(configDir, "config.yaml"))
			if err != nil {
				return err
			}

			profileName := args[0]

			// Find the profile for confirm/TTL checks
			var profile config.Profile
			for _, p := range mgr.Profiles() {
				if p.Name == profileName {
					profile = p
					break
				}
			}

			// Confirm on use
			needsConfirm := profile.ConfirmOnUse
			if !needsConfirm && cfg.ConfirmProduction {
				for _, tag := range profile.Tags {
					if tag == "production" {
						needsConfirm = true
						break
					}
				}
			}
			if needsConfirm && term.IsTerminal(int(os.Stdin.Fd())) {
				fmt.Fprintf(cmd.ErrOrStderr(), "⚠ Profile %q requires confirmation. Type the profile name to continue: ", profileName)
				reader := bufio.NewReader(os.Stdin)
				input, _ := reader.ReadString('\n')
				if strings.TrimSpace(input) != profileName {
					return fmt.Errorf("confirmation failed — profile not activated")
				}
			}

			// TTL warning
			ttl := profile.TTLDays
			if ttl == 0 {
				ttl = cfg.DefaultTTLDays
			}
			if ttl > 0 {
				ts, err := mgr.LoginTimestamp(profileName)
				if err == nil {
					age := time.Since(ts)
					if age > time.Duration(ttl)*24*time.Hour {
						fmt.Fprintf(cmd.ErrOrStderr(), "⚠ Profile %q is past its TTL (%d days) — consider running 'cloudmux login %s'\n",
							profileName, ttl, profileName)
					}
				}
			}

			result, err := mgr.Use(profileName)
			if err != nil {
				return err
			}

			// Output export statements
			for k, v := range result.EnvVars {
				fmt.Fprintf(cmd.OutOrStdout(), "export %s='%s'\n", k, v)
			}

			// Token expiry warning (best-effort, don't block)
			if cfg.ExpiryWarningMinutes > 0 {
				status, err := mgr.Status(profileName)
				if err == nil && status.Valid && !status.ExpiresAt.IsZero() {
					remaining := time.Until(status.ExpiresAt)
					if remaining < time.Duration(cfg.ExpiryWarningMinutes)*time.Minute {
						fmt.Fprintf(cmd.ErrOrStderr(), "⚠ Token expires in %dm — run 'cloudmux login %s' to refresh\n",
							int(remaining.Minutes()), profileName)
					}
				}
			}

			return nil
		},
	}
}
```

- [ ] **Step 3: Build and test**

```bash
make build
./bin/cloudmux use wbai-azure
```

Expected: works as before, no warnings (TTL not set, token not near expiry).

- [ ] **Step 4: Commit**

```bash
git add internal/cli/use.go go.mod go.sum
git commit -m "add confirm-on-use, TTL warning, and token expiry warning"
```

---

### Task 7: Colored output in list, status, doctor

**Files:**
- Modify: `internal/cli/list.go`
- Modify: `internal/cli/status.go`
- Modify: `internal/cli/doctor.go`

- [ ] **Step 1: Update list command with color**

Replace `internal/cli/list.go`:

```go
package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/lukassekoulidis/cloudmux/internal/color"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all configured profiles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := newManager()
			if err != nil {
				return err
			}

			profiles := mgr.Profiles()
			if len(profiles) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No profiles configured. Add profiles to ~/.cloudmux/profiles.yaml")
				return nil
			}

			c := color.New(term.IsTerminal(int(os.Stdout.Fd())))

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "NAME\tPROVIDER\tSTATUS\tDESCRIPTION")
			for _, p := range profiles {
				status, err := mgr.Status(p.Name)
				var statusStr string
				if err != nil || !status.Valid {
					statusStr = c.Red("✗ expired")
				} else {
					statusStr = c.Green("✓ valid")
					if status.Identity != "" {
						statusStr += " " + c.Dim(status.Identity)
					}
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.Name, p.Provider, statusStr, p.Description)
			}
			w.Flush()
			return nil
		},
	}
}
```

- [ ] **Step 2: Update status command with color**

Replace `internal/cli/status.go`:

```go
package cli

import (
	"fmt"
	"os"

	"github.com/lukassekoulidis/cloudmux/internal/color"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "status [profile]",
		Short:             "Show session status for a profile",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: profileCompletionFunc,
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := newManager()
			if err != nil {
				return err
			}

			var profileName string
			if len(args) > 0 {
				profileName = args[0]
			} else {
				profileName = os.Getenv("CLOUDMUX_ACTIVE_PROFILE")
				if profileName == "" {
					return fmt.Errorf("no active profile — specify a profile name or activate one with 'cloudmux use'")
				}
			}

			c := color.New(term.IsTerminal(int(os.Stdout.Fd())))

			status, err := mgr.Status(profileName)
			if err != nil {
				return err
			}

			if !status.Valid {
				fmt.Fprintf(cmd.OutOrStdout(), "%s %s: not authenticated\n", c.Red("✗"), profileName)
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", c.Green("✓"), profileName)
			fmt.Fprintf(cmd.OutOrStdout(), "  Identity: %s\n", status.Identity)
			fmt.Fprintf(cmd.OutOrStdout(), "  Tenant:   %s\n", status.Tenant)
			if !status.ExpiresAt.IsZero() {
				fmt.Fprintf(cmd.OutOrStdout(), "  Expires:  %s\n", status.ExpiresAt.Format("2006-01-02 15:04:05"))
			}
			if status.Region != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "  Region:   %s\n", status.Region)
			}
			return nil
		},
	}
}
```

- [ ] **Step 3: Update doctor command with color**

In `internal/cli/doctor.go`, add color import and update the output. Replace the existing `newDoctorCmd` function:

```go
package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/lukassekoulidis/cloudmux/internal/color"
	"github.com/lukassekoulidis/cloudmux/internal/config"
	"github.com/lukassekoulidis/cloudmux/internal/security"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var providerCLIs = map[string]struct {
	Binary     string
	InstallURL string
}{
	"azure": {"az", "https://learn.microsoft.com/en-us/cli/azure/install-azure-cli"},
	"gcp":   {"gcloud", "https://cloud.google.com/sdk/docs/install"},
	"aws":   {"aws", "https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html"},
}

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check prerequisites and configuration health",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			c := color.New(term.IsTerminal(int(os.Stdout.Fd())))

			if err := security.EnforcePermissions(configDir, true); err != nil {
				fmt.Fprintf(out, "%s Config directory: %s\n  %s\n", c.Red("✗"), configDir, err)
			} else {
				fmt.Fprintf(out, "%s Config directory: %s (0700)\n", c.Green("✓"), configDir)
			}

			profilesPath := filepath.Join(configDir, "profiles.yaml")
			profiles, err := config.LoadProfiles(profilesPath)
			if err != nil {
				fmt.Fprintf(out, "%s Profiles: %s\n", c.Red("✗"), err)
				return nil
			}
			fmt.Fprintf(out, "%s Profiles: %d profiles loaded\n", c.Green("✓"), len(profiles))

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
					continue
				}
				_, err := exec.LookPath(info.Binary)
				if err != nil {
					fmt.Fprintf(out, "  %s %-8s (profiles: %s) — install: %s\n",
						c.Red("✗"), info.Binary, joinNames(profileNames), info.InstallURL)
				} else {
					fmt.Fprintf(out, "  %s %-8s (profiles: %s)\n",
						c.Green("✓"), info.Binary, joinNames(profileNames))
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

- [ ] **Step 4: Build and verify**

```bash
make build
./bin/cloudmux list
./bin/cloudmux status wbai-azure
./bin/cloudmux doctor
```

Expected: colored output with green checkmarks and red crosses.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/list.go internal/cli/status.go internal/cli/doctor.go
git commit -m "add colored output to list, status, and doctor"
```

---

### Task 8: End-to-end verification

**Files:** none (verification only)

- [ ] **Step 1: Run all tests**

```bash
go test ./... -v
```

Expected: all pass.

- [ ] **Step 2: Run linter and vet**

```bash
make lint && make vet
```

Expected: clean.

- [ ] **Step 3: Smoke test**

```bash
make build

# GC
./bin/cloudmux gc

# Doctor with color
./bin/cloudmux doctor

# List with status
./bin/cloudmux list

# Status with color
./bin/cloudmux status wbai-azure

# Use (should work, no TTL/confirm warnings)
./bin/cloudmux use wbai-azure

# Help shows gc
./bin/cloudmux --help | grep gc
```

- [ ] **Step 4: Install and push**

```bash
cp bin/cloudmux ~/.local/bin/cloudmux
```
