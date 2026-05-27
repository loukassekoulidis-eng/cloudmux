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
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("missing profile name\n\nUsage: cloudmux use <profile>\n\nRun 'cloudmux list' to see available profiles")
			}
			return cobra.ExactArgs(1)(cmd, args)
		},
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
