# cloudmux

**Cloud identity multiplexer — parallel, persistent cloud CLI sessions without re-authentication.**

cloudmux is a CLI tool that lets you maintain multiple simultaneous cloud provider sessions (Azure, GCP, AWS, and others) in isolated terminal environments. Login once per tenant, switch instantly between them across terminal panes, and never lose your session when switching context.

Built for consultants, platform engineers, and anyone who works across multiple cloud tenants daily.

---

## Table of contents

1. [Problem statement](#problem-statement)
2. [How it works](#how-it-works)
3. [Core concepts](#core-concepts)
4. [Architecture](#architecture)
5. [Provider plugin system](#provider-plugin-system)
6. [CLI interface](#cli-interface)
7. [Configuration format](#configuration-format)
8. [Security model](#security-model)
9. [Implementation plan](#implementation-plan)
10. [Tech stack](#tech-stack)
11. [Project structure](#project-structure)
12. [Non-goals](#non-goals)
13. [Prior art and differentiation](#prior-art-and-differentiation)
14. [License](#license)

---

## Problem statement

Cloud CLIs (az, gcloud, aws) store authentication state globally per user. When you run `az login`, it overwrites the previous session in `~/.azure/`. This creates a painful workflow for anyone working across multiple tenants:

1. `az login --tenant customer-A` → browser opens, authenticate, work
2. Need to deploy something on customer-B → `az login --tenant customer-B` → browser opens again, authenticate
3. Need to check something back on customer-A → `az login --tenant customer-A` → browser opens *again*

Each switch requires a full browser-based OAuth flow. This adds up to dozens of interruptions per day for consultants and platform engineers managing multiple clients or environments.

The same problem exists across GCP (`gcloud auth login`) and AWS (`aws sso login`), compounded when you use multiple providers simultaneously.

### Who this is for

- **Consultants and agencies** working across multiple client tenants daily (the primary use case)
- **Platform engineers** managing dev/staging/prod across accounts
- **DevOps engineers** operating multi-account cloud infrastructure
- **Any developer** who touches more than one cloud tenant regularly

---

## How it works

cloudmux exploits the fact that all major cloud CLIs support overriding their config/credential storage location via environment variables:

| Provider | Environment variable | Default location | What it controls |
|----------|---------------------|------------------|-----------------|
| Azure | `AZURE_CONFIG_DIR` | `~/.azure/` | Token cache, active subscription, CLI config |
| GCP | `CLOUDSDK_CONFIG` | `~/.config/gcloud/` | Auth tokens, active project, CLI properties |
| AWS | `AWS_CONFIG_FILE` + `AWS_SHARED_CREDENTIALS_FILE` | `~/.aws/config` + `~/.aws/credentials` | Profiles, SSO tokens, credentials |
| AWS (alt) | `AWS_PROFILE` | `default` | Active named profile within shared config |

cloudmux creates isolated directories per profile under `~/.cloudmux/profiles/<name>/` and sets the appropriate environment variables in the current shell session. Each terminal/tmux pane can point to a different profile, enabling truly parallel sessions.

### Lifecycle

```
1. cloudmux login driventic-azure
   → Sets AZURE_CONFIG_DIR=~/.cloudmux/profiles/driventic-azure/.azure
   → Runs `az login --tenant <tenant-id>` (browser opens ONCE)
   → Token is cached in the isolated directory

2. cloudmux use driventic-azure
   → Sets AZURE_CONFIG_DIR in current shell (no browser, instant)
   → All subsequent `az` commands use driventic credentials

3. [In another terminal]
   cloudmux use wbai-azure
   → Sets AZURE_CONFIG_DIR to a different isolated directory
   → This terminal uses WBAI credentials, the other still uses Driventic

4. cloudmux status
   → Shows active profile, tenant, token expiry
   → Calls real API to verify (not just cache)

5. cloudmux logout driventic-azure
   → Runs `az logout` with the isolated config dir
   → Wipes the token cache for that profile
```

---

## Core concepts

### Profile

A named configuration that maps to one cloud identity (one tenant/account/project). Defined in YAML. A profile contains:
- A unique name (used in all commands)
- A provider type (azure, gcp, aws, or custom)
- Provider-specific config (tenant ID, project ID, region, etc.)
- Optional metadata (description, TTL, confirm-on-use flag)

### Session

A runtime state where a shell has a specific profile's environment variables active. Sessions are shell-scoped: setting a session in one terminal does not affect another. Sessions are not persistent across shell restarts unless the shell hook is installed.

### Provider

A plugin that knows how to: authenticate (login), check session health (status), set the correct environment variables (env), and clean up (logout) for a specific cloud platform.

---

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                  CLI / TUI layer                     │
│   cloudmux login | use | list | status | logout | gc│
└──────────────────────┬──────────────────────────────┘
                       │ dispatches to
┌──────────────────────▼──────────────────────────────┐
│                 Session manager (core)               │
│                                                      │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ │
│  │Profile loader │ │ Env isolator │ │Health checker│ │
│  │ YAML parsing  │ │ config dirs  │ │ token expiry │ │
│  └──────────────┘ └──────────────┘ └──────────────┘ │
│  ┌──────────────┐ ┌──────────────┐                   │
│  │ Shell hook   │ │ Permission   │                   │
│  │ prompt + env │ │ enforcer     │                   │
│  └──────────────┘ └──────────────┘                   │
└──────────────────────┬──────────────────────────────┘
                       │ provider interface
┌──────────────────────▼──────────────────────────────┐
│                  Provider plugins                    │
│  ┌───────┐ ┌───────┐ ┌───────┐ ┌─────────────────┐ │
│  │ Azure │ │  GCP  │ │  AWS  │ │ Custom (generic)│ │
│  └───────┘ └───────┘ └───────┘ └─────────────────┘ │
└──────────────────────┬──────────────────────────────┘
                       │ reads/writes
┌──────────────────────▼──────────────────────────────┐
│           ~/.cloudmux/ (local filesystem)            │
│  profiles/<name>/.azure/    (isolated token caches)  │
│  profiles/<name>/.config/gcloud/                     │
│  profiles/<name>/.aws/                               │
│  config.yaml                (global settings)        │
│  profiles.yaml              (profile definitions)    │
└─────────────────────────────────────────────────────┘
```

### Data flow for `cloudmux use <profile>`

1. Load profile definition from `~/.cloudmux/profiles.yaml`
2. Resolve the provider plugin for this profile
3. Provider returns the required env var map (e.g. `{"AZURE_CONFIG_DIR": "/home/user/.cloudmux/profiles/foo/.azure"}`)
4. Env isolator validates the target directories exist and have correct permissions
5. Health checker optionally verifies token validity (is the cached token still usable?)
6. Shell hook exports the env vars into the current shell process
7. Prompt integration updates `PS1` to show `[cloudmux: foo]`

### Data flow for `cloudmux login <profile>`

1. Load profile definition
2. Resolve the provider plugin
3. Create the isolated directory structure if it doesn't exist
4. Set filesystem permissions (0700 dirs, 0600 files)
5. Provider executes the native login command with the isolated config dir
   - Azure: `AZURE_CONFIG_DIR=<path> az login --tenant <id>`
   - GCP: `CLOUDSDK_CONFIG=<path> gcloud auth login`
   - AWS: `AWS_CONFIG_FILE=<path>/config AWS_SHARED_CREDENTIALS_FILE=<path>/credentials aws sso login --profile <name>`
6. Native CLI handles the full OAuth/browser flow and caches tokens in the isolated dir
7. cloudmux records the login timestamp in profile metadata

---

## Provider plugin system

Each provider implements a Go interface:

```go
type Provider interface {
    // Name returns the provider identifier (e.g., "azure", "gcp", "aws")
    Name() string

    // EnvVars returns the environment variables needed to activate this profile.
    // The session manager will export these into the current shell.
    EnvVars(profile Profile) (map[string]string, error)

    // Login executes the provider-specific authentication flow.
    // This may open a browser or prompt for credentials.
    Login(profile Profile) error

    // Logout clears cached credentials for this profile.
    Logout(profile Profile) error

    // Status checks whether the current session is valid and returns health info.
    // Should make a real API call, not just check local cache.
    Status(profile Profile) (*SessionStatus, error)

    // Validate checks that the profile configuration is complete and valid
    // before any login or use attempt.
    Validate(profile Profile) error
}

type SessionStatus struct {
    Valid       bool
    Identity    string    // e.g. "user@company.com" or "arn:aws:iam::123:user/foo"
    Tenant      string    // tenant/project/account identifier
    ExpiresAt   time.Time // token expiry (zero value if unknown)
    Region      string    // active region if applicable
}
```

### Azure provider implementation notes

- Sets `AZURE_CONFIG_DIR` to `~/.cloudmux/profiles/<name>/.azure`
- Login: `az login --tenant <tenant-id>` (optionally `--use-device-code` for headless)
- Status: `az account show --output json` (real API call to verify token)
- Logout: `az logout` then wipe the profile's `.azure` directory
- Token cache location: `<AZURE_CONFIG_DIR>/msal_token_cache.json` (plaintext on macOS/Linux, DPAPI-encrypted on Windows — this is az CLI's own behavior, not something cloudmux controls)
- Also sets `AZURE_DEFAULTS_GROUP`, `AZURE_DEFAULTS_LOCATION` if configured in the profile

### GCP provider implementation notes

- Sets `CLOUDSDK_CONFIG` to `~/.cloudmux/profiles/<name>/.config/gcloud`
- Alternatively can use `CLOUDSDK_ACTIVE_CONFIG_NAME` if the user prefers gcloud's native named-config system (profile YAML has a `gcp.use_named_config: true` option)
- Login: `gcloud auth login` (and optionally `gcloud auth application-default login`)
- Status: `gcloud auth print-access-token` (verifies token is valid)
- Application Default Credentials: the profile can optionally set `GOOGLE_APPLICATION_CREDENTIALS` for SDK-based auth
- Token cache: `application_default_credentials.json` (plaintext, gcloud's own behavior)

### AWS provider implementation notes

- Two modes:
  1. **Named profile mode** (recommended): sets `AWS_PROFILE=<profile-name>`, expects the user has pre-configured `~/.aws/config` with named profiles. cloudmux just switches the active profile.
  2. **Isolated config mode**: sets `AWS_CONFIG_FILE` and `AWS_SHARED_CREDENTIALS_FILE` to profile-specific paths, fully isolating the AWS config.
- Login (SSO): `aws sso login --profile <name>`
- Status: `aws sts get-caller-identity`
- Supports role assumption chains via `AWS_ROLE_ARN` and `AWS_ROLE_SESSION_NAME`

### Custom/generic provider

For platforms not natively supported (Hetzner, DigitalOcean, Terraform Cloud, Kubernetes, etc.):

```yaml
profiles:
  - name: hetzner-prod
    provider: custom
    custom:
      env:
        HCLOUD_TOKEN_FILE: "~/.cloudmux/profiles/hetzner-prod/token"
      login_command: "echo 'Enter Hetzner API token:' && read -s token && echo $token > ~/.cloudmux/profiles/hetzner-prod/token"
      status_command: "hcloud server list --output noheader | head -1"
      logout_command: "rm -f ~/.cloudmux/profiles/hetzner-prod/token"
```

The custom provider runs user-defined shell commands. Security note: custom commands execute with the user's privileges. cloudmux performs no shell expansion on config values — commands are passed to `exec` directly, not through `sh -c`.

---

## CLI interface

### Commands

```
cloudmux login <profile>         # Authenticate to a profile (opens browser if needed)
cloudmux use <profile>           # Activate a profile in the current shell (instant, no browser)
cloudmux status [profile]        # Show active session info (real API verification)
cloudmux list                    # List all profiles with status indicators
cloudmux logout <profile>        # Clear credentials for a profile
cloudmux gc                      # List and clean up expired/stale profiles
cloudmux init                    # Create ~/.cloudmux/ directory structure
cloudmux doctor                  # Check prerequisites (az, gcloud, aws CLIs installed)
cloudmux completion <shell>      # Generate shell completions (bash, zsh, fish)
```

### Shell hook

Required for `cloudmux use` to work (it needs to modify the current shell's env). Add to `~/.zshrc` or `~/.bashrc`:

```bash
eval "$(cloudmux shell-hook zsh)"   # for Zsh
eval "$(cloudmux shell-hook bash)"  # for Bash
eval "$(cloudmux shell-hook fish)"  # for Fish
```

The shell hook:
1. Defines a `cloudmux()` shell function that wraps the binary for commands that need to modify the shell environment (`use`, `logout`)
2. Optionally modifies `PS1` / `PROMPT` to show the active profile name
3. Does NOT write anything to `.bashrc`/`.zshrc` beyond itself — all state is env vars in the current process

### Example session

```bash
$ cloudmux list
  NAME               PROVIDER   STATUS     TENANT/PROJECT
  driventic-azure    azure      ✓ valid    driventic.onmicrosoft.com (expires in 47m)
  wbai-azure         azure      ✗ expired  webuildai.onmicrosoft.com
  wbai-gcp           gcp        ✓ valid    we-build-ai-prod (expires in 3h)
  bauking-azure      azure      ○ unknown  (never logged in)

$ cloudmux use driventic-azure
✓ Activated: driventic-azure (Azure, driventic.onmicrosoft.com)

[cloudmux: driventic-azure] $ az group list --output table
Name            Location
-----------     ----------
rg-production   westeurope
rg-staging      westeurope

# In another tmux pane:
$ cloudmux use wbai-gcp
✓ Activated: wbai-gcp (GCP, we-build-ai-prod)

[cloudmux: wbai-gcp] $ gcloud compute instances list
NAME        ZONE             MACHINE_TYPE  STATUS
matchday-1  europe-west3-a   e2-standard-4 RUNNING
```

---

## Configuration format

### Global config (`~/.cloudmux/config.yaml`)

```yaml
# cloudmux global configuration

# Default shell prompt format. Variables: {name}, {provider}, {tenant}
prompt_format: "[cloudmux: {name}]"

# Whether to show token expiry warnings in the prompt
prompt_show_expiry: true

# Minutes before expiry to start warning
expiry_warning_minutes: 15

# Whether `cloudmux use` on production profiles requires confirmation
# (overridable per profile)
confirm_production: true

# Default TTL for profiles (0 = no expiry). Profiles past their TTL
# trigger a warning on `cloudmux use` and appear in `cloudmux gc`.
default_ttl_days: 0

# Permission enforcement. If true, cloudmux checks and fixes file
# permissions on every operation.
enforce_permissions: true
```

### Profile definitions (`~/.cloudmux/profiles.yaml`)

```yaml
profiles:
  # --- Azure profiles ---
  - name: driventic-azure
    provider: azure
    description: "Driventic client tenant"
    azure:
      tenant_id: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
      subscription_id: "yyyyyyyy-yyyy-yyyy-yyyy-yyyyyyyyyyyy"
      default_location: "westeurope"
    tags: [client, production]
    confirm_on_use: true     # requires explicit confirmation
    ttl_days: 90             # expires 90 days after creation

  - name: wbai-azure
    provider: azure
    description: "WE BUILD AI internal tenant"
    azure:
      tenant_id: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
      subscription_id: "zzzzzzzz-zzzz-zzzz-zzzz-zzzzzzzzzzzz"
      default_location: "westeurope"
    tags: [internal]

  # --- GCP profiles ---
  - name: wbai-gcp
    provider: gcp
    description: "WE BUILD AI GCP project"
    gcp:
      project_id: "we-build-ai-prod"
      region: "europe-west3"
      zone: "europe-west3-a"
      use_named_config: false  # if true, uses CLOUDSDK_ACTIVE_CONFIG_NAME instead of CLOUDSDK_CONFIG

  # --- AWS profiles ---
  - name: analytics-aws
    provider: aws
    description: "Analytics platform on AWS"
    aws:
      profile_name: "analytics-prod"  # named profile in ~/.aws/config
      region: "eu-central-1"
      mode: "named_profile"           # or "isolated_config"
      sso_start_url: "https://myorg.awsapps.com/start"

  # --- Custom provider ---
  - name: hetzner-prod
    provider: custom
    description: "Hetzner Cloud production"
    custom:
      env:
        HCLOUD_TOKEN_FILE: "{profile_dir}/token"
      login_command: "hcloud context create {name}"
      status_command: "hcloud server list --output noheader | head -1"
      logout_command: "rm -f {profile_dir}/token"
```

### Template variables in profile config

The following variables are expanded in profile configuration values:
- `{profile_dir}` → absolute path to `~/.cloudmux/profiles/<name>/`
- `{name}` → the profile name
- `{home}` → user's home directory

No other shell expansion, glob expansion, or command substitution is performed. This is a security boundary.

---

## Security model

### Threat model

cloudmux operates at a specific trust boundary: it manages *which* set of cached credentials your shell points to, but it does not participate in the OAuth flow itself and never touches tokens directly.

#### What cloudmux controls (and must protect)

| Asset | Risk | Mitigation |
|-------|------|------------|
| Profile directory structure | Unauthorized read of token caches | `0700` on all directories, `0600` on all files, enforced on every operation |
| Profile YAML configuration | Tampering to redirect config dirs to attacker-controlled paths | `0600` on `profiles.yaml`, strict YAML schema validation, no shell expansion |
| Environment variables in shell | Leaking via `env`, process listings, or child processes | Scoped to current shell process only, never written to disk, `cloudmux status` shows what's set |
| Shell hook code | Injection via malicious profile names | Profile names restricted to `[a-zA-Z0-9_-]`, max 64 chars |

#### What cloudmux does NOT control (native CLI responsibility)

| Asset | Owner | Notes |
|-------|-------|-------|
| OAuth flow + PKCE | az CLI / gcloud SDK / AWS CLI | cloudmux delegates the entire auth flow to the native tool |
| Token encryption at rest | Native CLI | Azure: DPAPI on Windows, plaintext on macOS/Linux. GCP: plaintext. AWS: plaintext (SSO cache) |
| Token refresh | Native CLI (MSAL / gcloud SDK) | Automatic silent refresh happens within the native tooling |
| TLS to identity provider | OS / Native CLI | Certificate validation is handled by the CLI/SDK |

#### What the OS must provide

| Asset | Owner |
|-------|-------|
| Filesystem permissions enforcement | OS kernel |
| Disk encryption at rest | OS (FileVault, LUKS, BitLocker) |
| Process isolation | OS kernel |
| Keychain / credential manager | OS (used by native CLIs on some platforms) |

### Hardening measures (must implement)

#### 1. Filesystem permissions (critical)

Every file operation in cloudmux must enforce:
- Directories: `0700` (owner read/write/execute only)
- Files: `0600` (owner read/write only)
- Check on: `init`, `login`, `use`, `status` (every operation that touches the profile directory)
- If permissions are wrong, `use` and `login` refuse to proceed and print a warning with the fix command

```go
// Example enforcement
func EnforcePermissions(path string, isDir bool) error {
    expected := os.FileMode(0600)
    if isDir {
        expected = os.FileMode(0700)
    }
    info, err := os.Stat(path)
    if err != nil {
        return err
    }
    if info.Mode().Perm() != expected {
        return fmt.Errorf(
            "insecure permissions %o on %s (expected %o), run: chmod %o %s",
            info.Mode().Perm(), path, expected, expected, path,
        )
    }
    return nil
}
```

#### 2. Profile name sanitization (critical)

Profile names are used in directory paths and shell variable values. Malicious names could enable path traversal or shell injection.

- Allowed characters: `[a-zA-Z0-9_-]`
- Maximum length: 64 characters
- Must not start with `-` or `.`
- Must not contain path separators (`/`, `\`)
- Must not be a reserved name (`..`, `.`, `CON`, `NUL`, etc.)
- Validate at profile creation AND at every load from YAML

#### 3. No shell expansion in configuration (critical)

Profile YAML values must NEVER be passed through a shell interpreter. Specifically:
- No `sh -c` or `bash -c` for command execution (use `exec.Command` with explicit args)
- No `os.ExpandEnv` on arbitrary config values
- Only the explicitly listed template variables (`{profile_dir}`, `{name}`, `{home}`) are expanded
- Custom provider commands are the one exception: they are executed via the shell, but this is explicitly documented and the user authors them

#### 4. Stale credential cleanup

Cached tokens that outlive their usefulness are a liability:
- `cloudmux gc` lists profiles with expired tokens and profiles past their TTL
- `cloudmux gc --dry-run` shows what would be cleaned without acting
- `cloudmux gc --force` removes stale profile directories
- `cloudmux logout <profile>` calls the native CLI logout AND removes the isolated config directory
- Optional: `ttl_days` per profile triggers a warning when exceeded

#### 5. Cross-profile confusion prevention

Operating on the wrong tenant is a significant operational risk:
- Shell prompt always shows the active profile name
- `cloudmux use` prints tenant/project identity on activation
- `cloudmux status` makes a real API call to verify the active identity (not just reading local cache)
- `confirm_on_use: true` on sensitive profiles requires typing the profile name to confirm
- `cloudmux list` shows a status column with visual indicators (checkmark, warning, cross)

#### 6. Audit trail (recommended)

- `~/.cloudmux/audit.log` records login, use, logout, gc events with timestamps
- Log format: `2026-05-26T14:30:00Z LOGIN driventic-azure azure driventic.onmicrosoft.com`
- Permissions: `0600`
- Rotation: keep last 10,000 lines or 1MB, whichever is smaller

### What cloudmux explicitly does NOT do

- Does not store, cache, or manage tokens itself (delegates to native CLIs)
- Does not intercept or proxy OAuth flows
- Does not store passwords, service principal secrets, or API keys
- Does not send any telemetry or phone home
- Does not modify `~/.azure/`, `~/.config/gcloud/`, or `~/.aws/` (only its own `~/.cloudmux/` directory tree)
- Does not require elevated privileges (no sudo, no root)

---

## Implementation plan

### Phase 1: Foundation (MVP)

The minimum viable product that solves the core problem.

1. **Project scaffolding**: Go module, CLI framework (cobra), directory structure
2. **Profile system**: YAML loading, schema validation, profile name sanitization
3. **Env isolator**: Directory creation with permission enforcement, env var mapping
4. **Azure provider**: Login, use, status, logout with `AZURE_CONFIG_DIR` isolation
5. **Shell hook**: Bash and Zsh support, PS1 integration
6. **Core commands**: `init`, `login`, `use`, `status`, `list`, `logout`
7. **Permission enforcer**: Checks on every file operation
8. **Tests**: Unit tests for profile loading, permission enforcement, name sanitization

Goal: a single user (the author) can manage 3-4 Azure tenants without browser re-authentication.

### Phase 2: Multi-provider

9. **GCP provider**: `CLOUDSDK_CONFIG` isolation + optional named-config mode
10. **AWS provider**: Named-profile mode + isolated-config mode
11. **Custom provider**: Generic env + command execution
12. **`doctor` command**: Check that required CLIs are installed and reachable
13. **Fish shell support**
14. **Shell completions** for all supported shells

### Phase 3: Quality of life

15. **`gc` command**: Stale profile detection and cleanup
16. **TTL system**: Per-profile expiry with warnings
17. **Audit logging**: Append-only log of operations
18. **Token expiry monitoring**: Background check on `use` that warns if token expires soon
19. **`confirm_on_use`**: Safety gate for production profiles
20. **Colored output**: Status indicators, expiry warnings

### Phase 4: Distribution

21. **Goreleaser config**: Cross-platform binary builds (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64)
22. **Homebrew formula**
23. **AUR package** (nice-to-have)
24. **GitHub Actions CI**: Test, lint, release pipeline
25. **Man pages / documentation site**

---

## Tech stack

| Component | Choice | Rationale |
|-----------|--------|-----------|
| Language | **Go 1.22+** | Single binary, fast startup (critical for shell hook), excellent cross-platform, standard for CLI tools in the cloud ecosystem |
| CLI framework | **cobra** | Industry standard for Go CLIs, built-in completion generation, used by kubectl/gh/docker |
| YAML parsing | **gopkg.in/yaml.v3** | Standard Go YAML library with strict mode support |
| Shell hook | **Custom (Go template → shell script)** | Minimal, no dependencies, outputs `eval`-able shell functions |
| Filesystem | **Standard library (`os`, `path/filepath`)** | No external deps needed for file operations |
| JSON parsing (CLI output) | **encoding/json** | Parsing `az account show --output json` etc. |
| Testing | **Standard library `testing` + testify** | Assertions + mocking for provider interfaces |
| Linting | **golangci-lint** | Standard Go linter aggregator |
| Build/release | **goreleaser** | Cross-compilation, checksums, changelog generation |

### External runtime dependencies (user must have installed)

- `az` CLI (for Azure profiles)
- `gcloud` CLI (for GCP profiles)
- `aws` CLI v2 (for AWS profiles)
- A supported shell (bash 4+, zsh 5+, fish 3+)

cloudmux checks for these via `cloudmux doctor` and only requires the CLIs for providers the user actually configures.

---

## Project structure

```
cloudmux/
├── cmd/
│   └── cloudmux/
│       └── main.go                 # Entry point
├── internal/
│   ├── cli/                        # Cobra command definitions
│   │   ├── root.go
│   │   ├── init.go
│   │   ├── login.go
│   │   ├── use.go
│   │   ├── status.go
│   │   ├── list.go
│   │   ├── logout.go
│   │   ├── gc.go
│   │   ├── doctor.go
│   │   └── completion.go
│   ├── config/                     # Configuration loading + validation
│   │   ├── config.go               # Global config struct + loader
│   │   ├── profile.go              # Profile struct + validation
│   │   └── config_test.go
│   ├── provider/                   # Provider plugin interface + implementations
│   │   ├── provider.go             # Interface definition
│   │   ├── registry.go             # Provider registry (name → implementation)
│   │   ├── azure/
│   │   │   ├── azure.go            # Azure provider implementation
│   │   │   └── azure_test.go
│   │   ├── gcp/
│   │   │   ├── gcp.go
│   │   │   └── gcp_test.go
│   │   ├── aws/
│   │   │   ├── aws.go
│   │   │   └── aws_test.go
│   │   └── custom/
│   │       ├── custom.go
│   │       └── custom_test.go
│   ├── session/                    # Session management + env isolation
│   │   ├── manager.go              # Core session logic
│   │   ├── envmap.go               # Env var construction
│   │   └── manager_test.go
│   ├── security/                   # Permission enforcement + sanitization
│   │   ├── permissions.go          # File permission checks + enforcement
│   │   ├── sanitize.go             # Profile name validation
│   │   ├── audit.go                # Audit logging
│   │   └── security_test.go
│   └── shell/                      # Shell hook generation
│       ├── hook.go                 # Shell-specific hook templates
│       ├── prompt.go               # PS1 modification logic
│       └── hook_test.go
├── scripts/
│   └── shell-hook/                 # Shell hook template files
│       ├── bash.tmpl
│       ├── zsh.tmpl
│       └── fish.tmpl
├── .goreleaser.yml
├── .golangci.yml
├── go.mod
├── go.sum
├── LICENSE
├── README.md
└── Makefile
```

---

## Non-goals

Things cloudmux deliberately does not do, to maintain focus and security:

- **VPN management**: Use WireGuard/Tailscale/OpenVPN directly. cloudmux switches cloud identities, not network paths.
- **SSH tunnel management**: Use native SSH or tools like `autossh`.
- **Kubernetes context switching**: Use `kubectx`/`kubens`. cloudmux may set `KUBECONFIG` as a provider env var, but it does not manage kube contexts itself.
- **Secret management**: Use Vault, 1Password CLI, or `pass`. cloudmux does not store secrets.
- **Custom token management**: cloudmux never touches tokens directly. The native CLIs handle all token lifecycle.
- **GUI / web UI**: CLI/TUI only. This runs in your terminal.
- **Centralized/team credential sharing**: cloudmux is a single-user, local-machine tool.
- **Cloud API abstraction**: cloudmux does not wrap `az`, `gcloud`, or `aws` commands. It only sets environment variables; you use the native CLIs directly.

---

## Prior art and differentiation

| Tool | What it does | How cloudmux differs |
|------|-------------|---------------------|
| **ctx** (vlebo/ctx) | Full DevOps context switcher: cloud profiles + K8s + VPN + tunnels + secrets | cloudmux is narrower: only cloud identity sessions. No VPN, tunnels, or K8s management. Smaller scope = smaller attack surface, easier to audit. |
| **aws-vault** | Secure AWS credential management with OS keychain | AWS-only. cloudmux is cloud-agnostic. |
| **kubectx/kubens** | Kubernetes context/namespace switching | K8s only, different layer. Complementary to cloudmux. |
| **granted** (common-fate) | Browser-based AWS role switching | AWS-focused with browser integration. cloudmux is CLI-only and cloud-agnostic. |
| **direnv** | Per-directory environment variables | Generic tool. Can achieve similar isolation but requires per-project `.envrc` files and no profile management, status checking, or session health. cloudmux is purpose-built for cloud auth. |
| **Manual AZURE_CONFIG_DIR** | Set env vars yourself per terminal | What cloudmux automates. Manual approach has no permission enforcement, no status checking, no prompt integration, no garbage collection. |

cloudmux's value is being the right level of abstraction: more than manual env vars, less than a full DevOps platform. It does one thing — parallel cloud identity sessions — and does it with proper security hygiene.

---

## License

MIT

---

## Contributing

Contributions welcome. Please open an issue before starting work on a feature to align on scope.

Key areas where contributions are especially welcome:
- Additional provider plugins (DigitalOcean, Hetzner, Terraform Cloud, etc.)
- Shell support (PowerShell, nushell)
- Integration tests across platforms
- Documentation and examples
