# cloudmux

**Cloud identity multiplexer — parallel, persistent cloud CLI sessions without re-authentication.**

Stop re-authenticating every time you switch between cloud tenants. cloudmux lets you login once per tenant and switch instantly between them across terminal panes.

```bash
# Login once (opens browser)
cloudmux login wbai-azure

# Switch instantly in any terminal — no browser, no waiting
cloudmux use wbai-azure
az group list  # uses WBAI credentials

# In another terminal, different tenant — simultaneously
cloudmux use driventic-azure
az group list  # uses Driventic credentials
```

Built for consultants, platform engineers, and anyone who juggles multiple cloud accounts daily.

## How it works

Cloud CLIs (`az`, `gcloud`, `aws`) store auth state globally. When you `az login` to a different tenant, it overwrites your previous session. cloudmux solves this by giving each profile its own isolated config directory:

| Provider | What cloudmux sets | Effect |
|----------|-------------------|--------|
| Azure | `AZURE_CONFIG_DIR` | Isolated token cache per tenant |
| GCP | `CLOUDSDK_CONFIG` | Isolated gcloud config per project |
| AWS | `AWS_PROFILE` | Switches named profile (no isolation needed) |
| Custom | User-defined env vars | Works with any CLI tool |

Each terminal points to a different isolated directory. No credential conflicts, no re-authentication.

## Install

### From source

```bash
go install github.com/lukassekoulidis/cloudmux/cmd/cloudmux@latest
```

### Build locally

```bash
git clone https://github.com/loukassekoulidis-eng/cloudmux.git
cd cloudmux
make build
cp bin/cloudmux ~/.local/bin/  # or anywhere on your PATH
```

### Shell hook (required)

cloudmux needs to set environment variables in your current shell. Add to your shell rc:

```bash
# ~/.zshrc
eval "$(cloudmux shell-hook zsh)"

# ~/.bashrc
eval "$(cloudmux shell-hook bash)"

# ~/.config/fish/config.fish
cloudmux shell-hook fish | source
```

### Starship prompt (optional)

If you use [Starship](https://starship.rs/), add to `~/.config/starship.toml` for an inline prompt indicator:

```toml
[env_var.CLOUDMUX_ACTIVE_PROFILE]
symbol = "☁️ "
style = "bold cyan"
format = "via [$symbol$env_value]($style) "
```

The built-in `[cloudmux: ...]` prompt prefix is automatically disabled when Starship is detected.

## Quick start

```bash
# Initialize config directory
cloudmux init

# Edit profiles
vim ~/.cloudmux/profiles.yaml
```

Add your profiles:

```yaml
profiles:
  - name: wbai-azure
    provider: azure
    description: "WE BUILD AI tenant"
    azure:
      tenant_id: "84e19cd6-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
      subscription_id: "b222ede5-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
      default_location: "westeurope"

  - name: my-gcp
    provider: gcp
    description: "GCP production"
    gcp:
      project_id: "my-project-prod"
      region: "europe-west3"

  - name: work-aws
    provider: aws
    description: "AWS work account"
    aws:
      profile_name: "work-sso"
      region: "eu-central-1"
      sso_start_url: "https://myorg.awsapps.com/start"
```

Then:

```bash
cloudmux login wbai-azure    # authenticate once (opens browser)
cloudmux use wbai-azure      # activate in current terminal (instant)
cloudmux status              # verify session
cloudmux list                # see all profiles + status
```

## Import existing sessions

Already logged in? Import your active sessions directly:

```bash
cloudmux import                    # auto-detect all active cloud sessions
cloudmux import --name my-tenant   # override the generated profile name
```

cloudmux probes Azure, GCP, and AWS for active sessions, creates profiles, and copies credentials into isolated directories — no re-authentication needed.

## Commands

| Command | Description |
|---------|-------------|
| `cloudmux init` | Create `~/.cloudmux/` directory structure |
| `cloudmux login <profile>` | Authenticate (opens browser if needed) |
| `cloudmux use <profile>` | Activate profile in current shell (instant) |
| `cloudmux status [profile]` | Show session info with real API verification |
| `cloudmux list` | List all profiles with live status |
| `cloudmux logout <profile>` | Clear credentials for a profile |
| `cloudmux gc` | Find and clean up stale/expired profiles |
| `cloudmux doctor` | Check prerequisites (CLIs installed, config health) |
| `cloudmux import` | Detect and import active cloud sessions |
| `cloudmux completion <shell>` | Generate shell completions |
| `cloudmux shell-hook <shell>` | Generate shell hook for eval |

## Custom providers

For tools not natively supported, use the `custom` provider with user-defined commands:

```yaml
  - name: hetzner-prod
    provider: custom
    description: "Hetzner Cloud"
    custom:
      env:
        HCLOUD_CONTEXT: "{name}"
      login_command: "hcloud context create {name}"
      status_command: "hcloud server list --output noheader | head -1"
      logout_command: "hcloud context delete {name}"
```

Template variables: `{profile_dir}`, `{name}`, `{home}`

## Configuration

### Global config (`~/.cloudmux/config.yaml`)

```yaml
prompt_format: "[cloudmux: {name}]"
expiry_warning_minutes: 15    # warn when token expires soon
confirm_production: true       # require confirmation for production profiles
default_ttl_days: 0           # 0 = no expiry
enforce_permissions: true      # check 0700/0600 on every operation
```

### Profile options

```yaml
  - name: prod-azure
    provider: azure
    confirm_on_use: true    # require typing profile name to activate
    ttl_days: 90            # warn after 90 days, show in `gc`
    tags: [production]      # triggers confirm_production gate
    azure:
      tenant_id: "..."
```

## Security

- All directories under `~/.cloudmux/` are `0700`, all files `0600` — enforced on every operation
- Profile names restricted to `[a-zA-Z0-9_-]` (max 64 chars) to prevent path traversal and shell injection
- Environment variable values are single-quoted in shell output to prevent injection
- cloudmux never touches tokens directly — native CLIs handle all OAuth flows and token lifecycle
- No telemetry, no network calls (except through the native CLIs you invoke)

## Requirements

- Go 1.22+ (build only)
- One or more cloud CLIs: `az`, `gcloud`, `aws` (only for providers you use)
- bash 4+, zsh 5+, or fish 3+

Run `cloudmux doctor` to check what's installed.

## License

MIT
