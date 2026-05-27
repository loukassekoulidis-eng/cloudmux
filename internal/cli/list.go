package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/lukassekoulidis/cloudmux/internal/color"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all configured profiles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := newManager()
			if err != nil {
				return err
			}

			profiles := mgr.Profiles()
			if len(profiles) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No profiles configured. Add profiles to ~/.cloudmux/profiles.yaml")
				return nil
			}

			c := color.New(term.IsTerminal(int(os.Stdout.Fd())))

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "NAME\tPROVIDER\tSTATUS\tDESCRIPTION")
			for _, p := range profiles {
				status, err := mgr.Status(p.Name)
				var statusStr string
				if err != nil || !status.Valid {
					statusStr = c.Red("✗ expired")
				} else {
					statusStr = c.Green("✓ valid")
					if status.Identity != "" {
						statusStr += " " + c.Dim(status.Identity)
					}
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.Name, p.Provider, statusStr, p.Description)
			}
			w.Flush()
			return nil
		},
	}
}
