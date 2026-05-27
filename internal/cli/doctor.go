package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/lukassekoulidis/cloudmux/internal/color"
	"github.com/lukassekoulidis/cloudmux/internal/config"
	"github.com/lukassekoulidis/cloudmux/internal/security"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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
			c := color.New(term.IsTerminal(int(os.Stdout.Fd())))

			if err := security.EnforcePermissions(configDir, true); err != nil {
				fmt.Fprintf(out, "%s Config directory: %s\n  %s\n", c.Red("✗"), configDir, err)
			} else {
				fmt.Fprintf(out, "%s Config directory: %s (0700)\n", c.Green("✓"), configDir)
			}

			profilesPath := filepath.Join(configDir, "profiles.yaml")
			profiles, err := config.LoadProfiles(profilesPath)
			if err != nil {
				fmt.Fprintf(out, "%s Profiles: %s\n", c.Red("✗"), err)
				return nil
			}
			fmt.Fprintf(out, "%s Profiles: %d profiles loaded\n", c.Green("✓"), len(profiles))

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
					fmt.Fprintf(out, "  %s %-8s (profiles: %s) — install: %s\n",
						c.Red("✗"), info.Binary, joinNames(profileNames), info.InstallURL)
				} else {
					fmt.Fprintf(out, "  %s %-8s (profiles: %s)\n",
						c.Green("✓"), info.Binary, joinNames(profileNames))
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
