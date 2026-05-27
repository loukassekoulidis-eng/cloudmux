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

				if _, err := os.Stat(profDir); os.IsNotExist(err) {
					fmt.Fprintf(out, "  ○ %s — never logged in\n", p.Name)
					continue
				}

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
