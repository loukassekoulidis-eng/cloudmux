package cli

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/lukassekoulidis/cloudmux/internal/config"
	"github.com/lukassekoulidis/cloudmux/internal/security"
	"github.com/spf13/cobra"
)

var providerCLIs = map[string]struct {
	Binary     string
	InstallURL string
}{
	"azure": {"az", "https://learn.microsoft.com/en-us/cli/azure/install-azure-cli"},
	"gcp":   {"gcloud", "https://cloud.google.com/sdk/docs/install"},
	"aws":   {"aws", "https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html"},
}

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check prerequisites and configuration health",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			// Check config directory
			if err := security.EnforcePermissions(configDir, true); err != nil {
				fmt.Fprintf(out, "✗ Config directory: %s\n  %s\n", configDir, err)
			} else {
				fmt.Fprintf(out, "✓ Config directory: %s (0700)\n", configDir)
			}

			// Load profiles
			profilesPath := filepath.Join(configDir, "profiles.yaml")
			profiles, err := config.LoadProfiles(profilesPath)
			if err != nil {
				fmt.Fprintf(out, "✗ Profiles: %s\n", err)
				return nil
			}
			fmt.Fprintf(out, "✓ Profiles: %d profiles loaded\n", len(profiles))

			// Collect which providers are in use
			providerProfiles := make(map[string][]string)
			for _, p := range profiles {
				providerProfiles[p.Provider] = append(providerProfiles[p.Provider], p.Name)
			}

			if len(providerProfiles) == 0 {
				return nil
			}

			fmt.Fprintln(out, "\nProvider CLIs:")
			for provName, profileNames := range providerProfiles {
				info, ok := providerCLIs[provName]
				if !ok {
					continue
				}
				_, err := exec.LookPath(info.Binary)
				if err != nil {
					fmt.Fprintf(out, "  ✗ %-8s (profiles: %s) — install: %s\n",
						info.Binary, joinNames(profileNames), info.InstallURL)
				} else {
					fmt.Fprintf(out, "  ✓ %-8s (profiles: %s)\n",
						info.Binary, joinNames(profileNames))
				}
			}

			return nil
		},
	}
}

func joinNames(names []string) string {
	result := ""
	for i, n := range names {
		if i > 0 {
			result += ", "
		}
		result += n
	}
	return result
}
