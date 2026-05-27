# Menu Bar Tray App — Design Spec

## Goal

A macOS/Linux/Windows menu bar app that provides a status dashboard for cloudmux profiles, proactive session detection, and token expiry notifications. Built in Go using fyne-io/systray, living in the same repo.

## Scope

### In scope

- Menu bar tray icon with notification dot states (idle, new session, expiring, expired)
- Profile list with live status (green/red dots, expiry time inline)
- Profile submenus with contextual actions (switch/copy, refresh, re-authenticate, logout/remove)
- Background session detection loop (every 5 min, configurable) using existing `Provider.Detect()`
- Detection banner: suggest profile name, offer pre-computed name variants, dismiss/ignore
- Token expiry monitoring with macOS notifications (single notification per expiry cycle)
- Actions: Import Sessions, Refresh All, Run Doctor
- `cloudmux tray` command to launch the app
- Reads existing `~/.cloudmux/config.yaml` and `profiles.yaml`

### Out of scope

- Custom rendered popup windows (systray limitation — dropdown menus only)
- Text input fields (systray limitation — use pre-computed name variants)
- Modifying terminal environment (tray can't set env vars in user's shell)
- Auto-start on login (user can configure via macOS Login Items manually)

## Library

**fyne-io/systray** — actively maintained (v1.12.0, Dec 2025), cross-platform, supports dynamic menu updates, submenus, checkmarks, enable/disable. Requires CGO.

## Menu Structure

### Normal state (no detections)

```
[Profiles]                           (section header, disabled)
  ● acme-azure          47m  Azure  ▸  (submenu)
  ● contoso-azure       2h   Azure  ▸
  ● staging-gcp     expired  GCP    ▸
  ● work-aws            3h   AWS    ▸
─────────────────────────────────────
Import Sessions...
Refresh All
Run Doctor
─────────────────────────────────────
Quit
```

### Detection state

```
● New: Azure (contoso.onmicrosoft.com)  ▸  (submenu with name options)
─────────────────────────────────────
[Profiles]
  ● acme-azure          47m  Azure  ▸
  ● work-aws            3h   AWS    ▸
─────────────────────────────────────
Import Sessions...
Refresh All
Run Doctor
─────────────────────────────────────
Quit
```

### Detection submenu (name variants)

```
Add as "contoso-azure"
Add as "contoso"
Add as "azure-contoso"
Copy import command...
─────────────────────────────────────
Dismiss
Ignore this account
```

### Profile submenu (valid)

```
user@acme-corp.com
Tenant: xxxxxxxx-xxxx                (disabled, info only)
Valid -- expires in 47m              (disabled, info only)
─────────────────────────────────────
Switch (copy command)
Refresh Token
─────────────────────────────────────
Logout
```

### Profile submenu (expired)

```
dev@staging.iam
Project: staging-123                 (disabled, info only)
Session expired                      (disabled, info only)
─────────────────────────────────────
Re-authenticate
Switch (copy command)                (dimmed)
─────────────────────────────────────
Remove Profile
```

## Menu Bar Icon

Text-based icon: `MUX` in the menu bar (systray supports setting title text).

Notification states via icon swap (4 icon variants, embedded as PNG):
- **Idle**: `MUX` — no dot
- **New session detected**: `MUX` with blue dot overlay
- **Token expiring** (under warning threshold): `MUX` with yellow dot overlay
- **Token expired**: `MUX` with red dot overlay

Priority: expired (red) > expiring (yellow) > new session (blue) > idle.

## Background Loops

### Session detection (every 5 minutes, configurable)

1. Call `Detect()` on all registered providers
2. Compare detected sessions against existing profiles (match by provider + tenant/project/account)
3. New sessions: add to pending detections list, set blue dot, rebuild menu
4. Already-known sessions: skip
5. Dismissed sessions: skip (stored in memory, reset on app restart)
6. Ignored sessions: skip (stored in `~/.cloudmux/config.yaml` under `ignored_sessions`)

### Token expiry check (every 2 minutes)

1. Call `Status()` on all profiles that have a profile directory
2. Track expiry times
3. If any token is within `expiry_warning_minutes` of config: set yellow dot
4. If any token is expired: set red dot
5. Send macOS notification once per expiry transition (not every cycle)

### Notification rules

- New session detected: blue dot only, no OS notification (not urgent)
- Token expiring (< 15 min): yellow dot + one macOS notification per profile
- Token expired: red dot + one macOS notification per profile
- Multiple expiring: bundle into one notification
- Notification cooldown: suppress re-notification for same profile for 1 hour after dismissal
- Config option: `tray_notifications: true|false` (default true)

## Actions

### Switch (copy command)
Copies `cloudmux use <profile-name>` to the system clipboard. User pastes into terminal.

### Refresh Token / Re-authenticate
Opens a new terminal window and runs `cloudmux login <profile>`. On macOS: `open -a Terminal "cloudmux login <name>"`. Cross-platform: use `exec.Command` with platform-appropriate terminal emulator.

### Import Sessions...
Opens a new terminal window and runs `cloudmux import`.

### Run Doctor
Opens a new terminal window and runs `cloudmux doctor`.

### Refresh All
Re-runs the status check loop immediately (in background, rebuilds menu when done).

### Add detected session (name variant)
Runs `cloudmux import --name <selected-name>` in background. On success, rebuilds menu. On failure (name collision), shows the profile with an error indicator.

### Copy import command...
Copies `cloudmux import --name ` (with trailing space) to clipboard. User pastes into terminal and types their custom name.

### Dismiss
Removes detection from pending list. Stores session fingerprint (provider + tenant hash) in memory. Same session won't reappear until app restart.

### Ignore this account
Stores session fingerprint in `~/.cloudmux/config.yaml` under `ignored_sessions`. Persists across restarts.

## Project Structure

```
cmd/
  cloudmux-tray/
    main.go              # Tray app entry point
    icons/               # Embedded PNG icons (idle, blue, yellow, red)
internal/
  tray/
    tray.go              # Main tray app logic, menu building
    detector.go          # Background detection loop
    monitor.go           # Background expiry monitoring
    actions.go           # Menu action handlers
    notify.go            # OS notification helpers
```

The tray app imports the existing `internal/config`, `internal/provider`, `internal/session`, and `internal/security` packages. It does NOT duplicate any logic — it's a GUI frontend to the same core.

## CLI Integration

```
cloudmux tray              # Launch the tray app (foreground)
cloudmux tray --detach     # Launch and detach (background)
```

Or as a separate binary: `cloudmux-tray` (built via `make build-tray`).

## Config Extensions

Add to `~/.cloudmux/config.yaml`:

```yaml
tray_notifications: true          # enable macOS notifications
tray_detection_interval_minutes: 5
tray_expiry_check_interval_minutes: 2
ignored_sessions: []              # list of provider:fingerprint strings
```

## Testing

- Unit tests for detection diffing (new vs known vs dismissed vs ignored)
- Unit tests for notification state machine (idle → blue → yellow → red priority)
- Unit tests for menu structure building
- Manual testing for tray integration (requires display)
