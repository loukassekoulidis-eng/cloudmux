# Import Command — Design Spec

## Goal

`cloudmux import` detects the current active session from any supported cloud CLI, creates a profile, and copies credentials into an isolated cloudmux directory. Provider-agnostic — works the same way for Azure, GCP, and AWS.

## CLI

```
cloudmux import                    # auto-detect provider from active sessions
cloudmux import --name my-tenant   # override the generated profile name
```

No provider argument needed. The command probes all registered providers and imports whichever ones have active sessions. If multiple are active, imports all of them (with confirmation).

## Provider Interface Extension

Add one new method to the `Provider` interface:

```go
type ImportInfo struct {
    SuggestedName string            // e.g. "acmecorp-azure" derived from tenant domain
    ProfileConfig map[string]string // provider-specific config to populate the Profile struct
    DefaultDir    string            // e.g. "~/.azure" — the source directory to copy
}

// Detect checks if this provider has an active session in the default (non-cloudmux)
// config location. Returns nil if no session found.
Detect() (*ImportInfo, error)
```

Each provider implements `Detect()`:

### Azure
- Runs `az account show --output json` (with NO `AZURE_CONFIG_DIR` override — uses default `~/.azure/`)
- Parses tenant ID, subscription ID, user, tenant default domain
- Suggested name: lowercase tenant domain sans `.onmicrosoft.com` + `-azure` (e.g. `acme-corp-azure`)
- ProfileConfig: `tenant_id`, `subscription_id`
- DefaultDir: `~/.azure`

### GCP
- Runs `gcloud config get project` and `gcloud config get account` (with NO `CLOUDSDK_CONFIG` override)
- Suggested name: project ID + `-gcp` (e.g. `my-project-gcp`)
- ProfileConfig: `project_id`
- DefaultDir: `~/.config/gcloud`

### AWS
- Runs `aws sts get-caller-identity --output json` (with NO custom env)
- Reads `AWS_PROFILE` env var for the current profile name
- Suggested name: profile name + `-aws`, or account ID + `-aws` if no profile set
- ProfileConfig: `profile_name`, `region` (from `AWS_DEFAULT_REGION`)
- DefaultDir: none (named profile mode — no copy needed)

## Import Flow

1. Iterate all registered providers, call `Detect()` on each
2. Collect providers that returned non-nil (have active sessions)
3. If none found → error: "no active cloud sessions detected"
4. If one found → proceed directly
5. If multiple found → print what was detected, import all (or let user pick with `--name` to do one at a time)
6. For each detected session:
   a. Generate profile name (or use `--name` override)
   b. Validate name doesn't already exist in profiles.yaml
   c. If `DefaultDir` is set: copy it to `~/.cloudmux/profiles/<name>/` with 0700/0600 permissions
   d. Append profile entry to `~/.cloudmux/profiles.yaml`
   e. Write login timestamp
   f. Print summary: `✓ Imported <name> (<provider>, <identity>)`

## Profile YAML Construction

The import command builds a `config.Profile` struct from the `ImportInfo` and appends it to the existing profiles.yaml. It reads the current file, appends the new profile to the `profiles` list, and writes it back.

## Copying Config Directories

For providers with a `DefaultDir` (Azure, GCP), the import does a recursive copy:
- Create `~/.cloudmux/profiles/<name>/` with 0700
- Copy contents of `DefaultDir` into the appropriate subdirectory (e.g. `.azure/` for Azure, `.config/gcloud/` for GCP)
- Set 0700 on all directories, 0600 on all files

For AWS (no `DefaultDir`), skip the copy — named profile mode just sets `AWS_PROFILE`.

## Edge Cases

- No active session for any provider → error with helpful message
- Profile name collision → error: `profile "X" already exists, use --name to specify a different name`
- Source config dir doesn't exist → skip copy, warn (session may be using keychain/broker auth)
- profiles.yaml doesn't exist → error: run `cloudmux init` first

## Testing

- Unit tests for each provider's `Detect()` (mocked — we can't run real CLIs in tests)
- Unit test for profile name generation from ImportInfo
- Unit test for YAML append logic
