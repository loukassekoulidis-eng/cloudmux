package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/lukassekoulidis/cloudmux/internal/color"
	"github.com/lukassekoulidis/cloudmux/internal/config"
	"github.com/lukassekoulidis/cloudmux/internal/copydir"
	"github.com/lukassekoulidis/cloudmux/internal/provider"
	"github.com/lukassekoulidis/cloudmux/internal/security"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newImportCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "import",
		Short: "Detect active cloud sessions and import them as profiles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			c := color.New(term.IsTerminal(int(os.Stdout.Fd())))

			reg := newRegistry()
			profilesPath := filepath.Join(configDir, "profiles.yaml")

			type detection struct {
				providerName string
				info         *provider.ImportInfo
			}
			var detected []detection

			fmt.Fprintln(out, "Scanning for active cloud sessions...")
			for _, p := range reg.All() {
				info, err := p.Detect()
				if err != nil {
					continue
				}
				if info != nil {
					fmt.Fprintf(out, "  %s %s: %s\n", c.Green("✓"), p.Name(), info.Profile.Description)
					detected = append(detected, detection{providerName: p.Name(), info: info})
				}
			}

			if len(detected) == 0 {
				return fmt.Errorf("no active cloud sessions detected — login with your cloud CLI first (az login, gcloud auth login, aws sso login)")
			}

			for _, d := range detected {
				profileName := d.info.SuggestedName
				if name != "" && len(detected) == 1 {
					profileName = name
				}

				if err := security.ValidateProfileName(profileName); err != nil {
					return fmt.Errorf("generated profile name %q is invalid: %w — use --name to override", profileName, err)
				}

				profile := d.info.Profile
				profile.Name = profileName

				// Copy config directory if needed
				if d.info.DefaultDir != "" {
					srcInfo, err := os.Stat(d.info.DefaultDir)
					if err != nil || !srcInfo.IsDir() {
						fmt.Fprintf(out, "  %s skipping config copy for %s (source dir not found)\n", c.Yellow("⚠"), profileName)
					} else {
						profDir := filepath.Join(configDir, "profiles", profileName)
						if err := security.EnsureDir(profDir); err != nil {
							return err
						}

						var subDir string
						switch d.providerName {
						case "azure":
							subDir = ".azure"
						case "gcp":
							subDir = filepath.Join(".config", "gcloud")
						}

						if subDir != "" {
							dstPath := filepath.Join(profDir, subDir)
							fmt.Fprintf(out, "  Copying %s → %s...\n", d.info.DefaultDir, dstPath)
							if err := copydir.Copy(d.info.DefaultDir, dstPath); err != nil {
								return fmt.Errorf("copying config directory: %w", err)
							}
						}

						tsPath := filepath.Join(profDir, ".cloudmux_login_ts")
						os.WriteFile(tsPath, []byte(time.Now().UTC().Format(time.RFC3339)), 0600)
					}
				}

				if err := config.AppendProfile(profilesPath, profile); err != nil {
					return err
				}

				fmt.Fprintf(out, "  %s Imported as %s\n", c.Green("✓"), c.Bold(profileName))
			}

			fmt.Fprintf(out, "\nDone. Use %s to activate a profile.\n", c.Bold("cloudmux use <profile>"))
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "override the generated profile name")
	return cmd
}
