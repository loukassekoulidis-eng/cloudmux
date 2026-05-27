# Phase 2: Multi-Provider — Design Spec

## Goal

Extend cloudmux beyond Azure to support GCP, AWS (named profile mode), and custom providers. Add `doctor` command, fish shell support, and shell completions.

## Scope

### In scope

- **GCP provider**: `CLOUDSDK_CONFIG` isolation, optional `CLOUDSDK_ACTIVE_CONFIG_NAME` mode via `gcp.use_named_config`
- **AWS provider**: named profile mode only — sets `AWS_PROFILE` and optionally `AWS_DEFAULT_REGION`. Login runs `aws sso login --profile <name>`
- **Custom provider**: user-defined env vars, login/status/logout shell commands with `{profile_dir}`, `{name}`, `{home}` template expansion. Commands run via `sh -c` (documented, user-authored)
- **`doctor` command**: checks that required CLIs (`az`, `gcloud`, `aws`) are installed and reachable, only for providers the user has configured
- **Fish shell hook**: fish-syntax equivalent of the bash/zsh hook
- **Shell completions**: `cloudmux completion <bash|zsh|fish>` via cobra's built-in generation
- **Config expansion**: `Profile` struct gains `GCP` and `AWS` config blocks
- **Starship hook detection**: fish hook also skips prompt when Starship is active

### Out of scope

- AWS isolated config mode (deferred)
- TTL, audit logging, confirm_on_use, gc, colored output (Phase 3)
- goreleaser, Homebrew, CI (Phase 4)

## Provider Implementations

### GCP provider

- Sets `CLOUDSDK_CONFIG` to `~/.cloudmux/profiles/<name>/.config/gcloud`
- If `gcp.use_named_config: true`, sets `CLOUDSDK_ACTIVE_CONFIG_NAME` instead (uses gcloud's native named-config system, no isolated dir)
- Login: `gcloud auth login` with `CLOUDSDK_CONFIG` set. Optionally `gcloud config set project <id>` if `project_id` is configured
- Status: `gcloud auth print-access-token` — exit code 0 means valid. Then `gcloud config get project` for display info
- Logout: `gcloud auth revoke` with `CLOUDSDK_CONFIG` set, then wipe the config dir
- Validate: requires `project_id` in profile config
- Also sets `CLOUDSDK_CORE_PROJECT` if `project_id` is set, and `CLOUDSDK_COMPUTE_REGION`/`CLOUDSDK_COMPUTE_ZONE` if region/zone are set

### AWS provider (named profile mode)

- Sets `AWS_PROFILE` to the configured `aws.profile_name`
- Optionally sets `AWS_DEFAULT_REGION` if `aws.region` is configured
- Login: `aws sso login --profile <profile_name>` (only if `sso_start_url` is configured, otherwise skip — profile may use static credentials)
- Status: `aws sts get-caller-identity --profile <profile_name>` — parses JSON for Account, Arn
- Logout: `aws sso logout --profile <profile_name>` (only if SSO), then unsets env vars
- Validate: requires `profile_name`
- Does NOT create isolated directories — relies on the user's existing `~/.aws/config`

### Custom provider

- Sets env vars from `custom.env` map, with template expansion (`{profile_dir}`, `{name}`, `{home}`)
- Login: runs `custom.login_command` via `sh -c` with template expansion
- Status: runs `custom.status_command` via `sh -c` — exit code 0 = valid, stdout captured for display
- Logout: runs `custom.logout_command` via `sh -c`
- Validate: requires at least one of `env`, `login_command`, or `status_command` to be defined
- All commands run with the profile's env vars already set in the environment

## Config Changes

`Profile` struct gains two new config blocks:

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
    Env            map[string]string `yaml:"env"`
    LoginCommand   string            `yaml:"login_command"`
    StatusCommand  string            `yaml:"status_command"`
    LogoutCommand  string            `yaml:"logout_command"`
}
```

## Doctor Command

`cloudmux doctor` checks:
1. Config directory exists with correct permissions
2. `profiles.yaml` is readable and valid
3. For each provider used in profiles, check if the CLI binary exists on PATH:
   - azure → `az`
   - gcp → `gcloud`
   - aws → `aws`
   - custom → skip (no standard binary)
4. Print a summary: checkmark for found, cross for missing, with install hints

Example output:
```
✓ Config directory: /Users/x/.cloudmux (0700)
✓ Profiles: 4 profiles loaded

Provider CLIs:
  ✓ az      (azure profiles: acme-azure, contoso-azure)
  ✗ gcloud  (gcp profiles: acme-gcp) — install: https://cloud.google.com/sdk/docs/install
  ✓ aws     (aws profiles: analytics-aws)
```

## Fish Shell Hook

Same logic as bash/zsh but in fish syntax:
- Function `cloudmux` wraps the binary
- For `use`/`logout`, captures output and evals it
- Fish uses `set -gx VAR value` instead of `export VAR=value`
- Fish uses `set -e VAR` instead of `unset VAR`
- Prompt integration skipped when Starship is active (check `$STARSHIP_SHELL`)

The `use` and `logout` CLI commands need to detect fish and output fish-syntax. Approach: the shell hook passes `--shell fish` flag when invoking the binary for `use`/`logout`, and the commands format their output accordingly.

Actually, simpler: the shell hook for fish can translate. The binary always outputs POSIX `export`/`unset` syntax. The fish hook function parses and converts:
- `export KEY='value'` → `set -gx KEY 'value'`
- `unset KEY` → `set -e KEY`

This keeps the binary simple and doesn't require the binary to know which shell it's in.

## Shell Completions

Cobra has built-in completion generation. `cloudmux completion <bash|zsh|fish>` outputs a completion script. This is a one-liner per shell using cobra's `GenBashCompletion`, `GenZshCompletion`, `GenFishCompletion`.

Additionally, profile names should complete for commands that take a `<profile>` argument (`login`, `use`, `status`, `logout`). Register a `ValidArgsFunction` on these commands that loads profiles and returns their names.

## Testing Strategy

Unit tests for:
- GCP provider: `EnvVars` (both modes), `Validate`
- AWS provider: `EnvVars`, `Validate`
- Custom provider: `EnvVars` with template expansion, `Validate`
- Doctor: mock `exec.LookPath` checks
- Fish hook: output format conversion
- Shell completions: profile name completion function
- Config: loading profiles with GCP/AWS/custom blocks

No integration tests (would require real CLI installations).
