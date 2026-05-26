package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lukassekoulidis/cloudmux/internal/security"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize cloudmux directory structure",
		RunE: func(cmd *cobra.Command, args []string) error {
			dirs := []string{
				configDir,
				filepath.Join(configDir, "profiles"),
			}
			for _, dir := range dirs {
				if err := security.EnsureDir(dir); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "✓ %s\n", dir)
			}

			configPath := filepath.Join(configDir, "config.yaml")
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				defaultConfig := `# cloudmux global configuration
prompt_format: "[cloudmux: {name}]"
prompt_show_expiry: true
expiry_warning_minutes: 15
confirm_production: true
default_ttl_days: 0
enforce_permissions: true
`
				if err := os.WriteFile(configPath, []byte(defaultConfig), 0600); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "✓ %s (created)\n", configPath)
			}

			profilesPath := filepath.Join(configDir, "profiles.yaml")
			if _, err := os.Stat(profilesPath); os.IsNotExist(err) {
				defaultProfiles := `# cloudmux profile definitions
# See README.md for configuration format
profiles: []
`
				if err := os.WriteFile(profilesPath, []byte(defaultProfiles), 0600); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "✓ %s (created)\n", profilesPath)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "\ncloudmux initialized. Add profiles to profiles.yaml to get started.")
			return nil
		},
	}
}
