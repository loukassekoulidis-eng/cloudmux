# Phase 1: cloudmux MVP — Design Spec

## Goal

A single user can manage 3-4 Azure tenants across terminal panes without browser re-authentication. Login once per tenant, switch instantly.

## Scope

Phase 1 delivers Azure-only support with the core session isolation mechanism. Other providers (GCP, AWS, custom) are Phase 2.

### In scope

- Project scaffolding: Go module, cobra CLI, Makefile, golangci-lint config
- Profile system: YAML loading from `~/.cloudmux/profiles.yaml`, schema validation, profile name sanitization
- Global config: YAML loading from `~/.cloudmux/config.yaml` with defaults
- Security: filesystem permission enforcement (0700 dirs, 0600 files), profile name validation (`[a-zA-Z0-9_-]`, max 64 chars)
- Azure provider: implements Provider interface — `EnvVars`, `Login`, `Logout`, `Status`, `Validate`
- Session manager: orchestrates profile loading → provider resolution → env var construction → directory validation
- Shell hooks: bash and zsh. Defines a `cloudmux()` shell function wrapping the binary for `use`/`logout`. PS1 prompt integration showing active profile.
- CLI commands: `init`, `login`, `use`, `status`, `list`, `logout`, `shell-hook`
- Unit tests: profile loading, name sanitization, permission enforcement, env var construction

### Out of scope (Phase 2+)

- GCP, AWS, custom providers
- `doctor`, `gc`, `completion` commands
- Fish shell support
- TTL system, audit logging, confirm_on_use, colored output
- goreleaser, Homebrew, CI

## Architecture

Follows the README spec exactly. Key packages:

### `internal/config`

- `Config` struct: global settings loaded from `~/.cloudmux/config.yaml`
- `Profile` struct: name, provider, description, tags, provider-specific config (azure block)
- `ProfileStore`: loads/validates `~/.cloudmux/profiles.yaml`
- Strict YAML parsing (disallow unknown fields)
- Default config values when file doesn't exist

### `internal/provider`

- `Provider` interface: `Name()`, `EnvVars(Profile)`, `Login(Profile)`, `Logout(Profile)`, `Status(Profile)`, `Validate(Profile)`
- `SessionStatus` struct: Valid, Identity, Tenant, ExpiresAt, Region
- `Registry`: maps provider name strings to Provider implementations
- `azure/`: sets `AZURE_CONFIG_DIR` to `~/.cloudmux/profiles/<name>/.azure`. Login runs `az login --tenant <id>`. Status runs `az account show --output json`. Logout runs `az logout` then wipes the `.azure` dir.

### `internal/session`

- `Manager`: the orchestrator. Takes a config store, provider registry, and security enforcer.
- `Use(profileName)` → returns env var map + display info (for shell hook to export)
- `Login(profileName)` → creates dirs, sets permissions, delegates to provider
- `Status(profileName)` → delegates to provider, formats result
- `Logout(profileName)` → delegates to provider, cleans up

### `internal/security`

- `ValidateProfileName(name)` → error if invalid
- `EnforcePermissions(path, isDir)` → checks 0700/0600, returns error with fix command
- `EnsureDir(path)` → creates with 0700 if missing, verifies permissions if exists

### `internal/shell`

- `GenerateHook(shellType)` → outputs eval-able shell script
- Hook defines `cloudmux()` function: for `use`/`logout`, captures binary stdout (env var exports) and evals them. For other commands, passes through to binary directly.
- Prompt integration: sets `CLOUDMUX_ACTIVE_PROFILE` env var, PS1 shows `[cloudmux: <name>]`

### `cmd/cloudmux/main.go`

Entry point. Initializes cobra root command, registers subcommands.

### `internal/cli`

Thin cobra command definitions. Each command: parse args → call session manager → format output.

- `root.go`: root command, persistent flags (`--config-dir`)
- `init.go`: creates `~/.cloudmux/` structure with correct permissions
- `login.go`: `cloudmux login <profile>`
- `use.go`: `cloudmux use <profile>` — outputs `export` statements (shell hook evals them)
- `status.go`: `cloudmux status [profile]` — shows identity, tenant, expiry
- `list.go`: `cloudmux list` — table of all profiles with status
- `logout.go`: `cloudmux logout <profile>`
- `shell_hook.go`: `cloudmux shell-hook <bash|zsh>`

## Shell Hook Design

The binary's `use` command outputs lines like:
```
export AZURE_CONFIG_DIR=/Users/x/.cloudmux/profiles/foo/.azure
export CLOUDMUX_ACTIVE_PROFILE=foo
```

The shell hook function intercepts `cloudmux use` and `cloudmux logout`, captures this output, and evals it. All other subcommands pass through to the binary directly.

For zsh, the hook also prepends `[cloudmux: $CLOUDMUX_ACTIVE_PROFILE]` to `PROMPT` when the var is set.

## Data on disk

```
~/.cloudmux/
├── config.yaml          # global settings
├── profiles.yaml        # profile definitions
└── profiles/
    └── <name>/
        └── .azure/      # isolated Azure config dir (created on login)
```

## Security

Per README spec:
- All dirs 0700, all files 0600, enforced on every touch
- Profile names: `[a-zA-Z0-9_-]`, 1-64 chars, no leading `-`/`.`, no reserved names
- No shell expansion except `{profile_dir}`, `{name}`, `{home}` template vars
- Native CLI commands via `exec.Command` with explicit args, never `sh -c`

## Testing Strategy

Unit tests for:
- Profile YAML parsing (valid, invalid, missing fields, unknown fields)
- Profile name validation (valid names, injection attempts, edge cases)
- Permission checking logic (correct perms, wrong perms, missing paths)
- Azure provider `EnvVars()` construction
- Shell hook output (correct export statements)
- Session manager orchestration (mocked provider)

No integration tests in Phase 1 (would require real `az` CLI + tenant).
