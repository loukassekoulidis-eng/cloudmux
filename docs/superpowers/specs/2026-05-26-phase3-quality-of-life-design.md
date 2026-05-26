# Phase 3: Quality of Life — Design Spec

## Goal

Add operational polish: garbage collection of stale profiles, TTL warnings, audit logging, token expiry awareness, production safety gates, and colored terminal output.

## Scope

### In scope

- **`gc` command**: list stale/expired profiles, `--dry-run` (default) and `--force` modes
- **TTL system**: per-profile `ttl_days` and global `default_ttl_days`. Profiles past TTL trigger warnings on `use` and appear in `gc`
- **Audit logging**: append-only `~/.cloudmux/audit.log` (0600). Records login/use/logout/gc events with timestamps. Max 10,000 lines, rotated on write.
- **Token expiry warnings**: `cloudmux use` checks session health and warns if token expires within `expiry_warning_minutes` (from config, default 15)
- **`confirm_on_use`**: per-profile flag. When true, `cloudmux use` requires typing the profile name to confirm before activating. Also respects global `confirm_production` for profiles tagged `production`
- **Colored output**: `list` shows colored status indicators (green checkmark, red cross, yellow warning). `status` uses color. `use` warnings in yellow. Respect `NO_COLOR` env var.

### Out of scope

- goreleaser, Homebrew, CI (Phase 4)

## Design Details

### GC Command

`cloudmux gc` scans all profiles and reports:
1. Profiles with no login directory (never used)
2. Profiles past their TTL (based on directory mtime or a `.cloudmux_login_ts` file in the profile dir)
3. Profiles where `status` returns invalid (expired tokens)

Default is `--dry-run` — lists what would be cleaned. `--force` actually removes the profile directories (but NOT the profile definition from `profiles.yaml`).

### TTL System

- Profile struct gains `TTLDays int` field (`yaml:"ttl_days"`)
- On `cloudmux use`, if the profile's TTL is exceeded, print a warning but still activate (non-blocking)
- On `cloudmux gc`, profiles past TTL appear in the stale list
- TTL is checked against a `.cloudmux_login_ts` file written by the session manager on login (ISO 8601 timestamp). If the file doesn't exist, TTL is not enforced.
- Global `default_ttl_days` applies when profile's `ttl_days` is 0

### Audit Logging

- `~/.cloudmux/audit.log`, permissions 0600
- Format: `2026-05-26T14:30:00Z <ACTION> <profile> <provider> [details]`
- Actions: `LOGIN`, `USE`, `LOGOUT`, `GC`
- Written by the session manager after successful operations
- Rotation: on each write, if file exceeds 10,000 lines, truncate to last 5,000 lines
- New `internal/audit/audit.go` package — simple, no external deps

### Token Expiry Warnings

- `cloudmux use` calls `provider.Status()` after activation
- If `status.ExpiresAt` is set and within `expiry_warning_minutes`, print a warning to stderr: `⚠ Token expires in Xm — run 'cloudmux login <profile>' to refresh`
- This is best-effort — if status check fails, silently continue (don't block activation)

### Confirm on Use

- Profile struct gains `ConfirmOnUse bool` field (`yaml:"confirm_on_use"`)
- Also triggers when global `confirm_production: true` and profile has tag `production`
- When triggered: prompt user to type the profile name. If stdin is not a terminal, skip (non-interactive use)
- Check via `term.IsTerminal(int(os.Stdin.Fd()))` from `golang.org/x/term`

### Colored Output

- New `internal/color/color.go` — minimal color helpers, no external deps
- Functions: `Green(s)`, `Red(s)`, `Yellow(s)`, `Bold(s)`, `Dim(s)`
- All functions check `NO_COLOR` env var and return plain text if set
- Also check if stdout is a terminal — no color if piped
- Apply to: `list` (status column), `status` (checkmark/cross), `use` (warnings), `doctor` (checkmark/cross), `gc` (stale items)
