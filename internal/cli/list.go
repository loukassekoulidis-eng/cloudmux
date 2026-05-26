package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
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

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "NAME\tPROVIDER\tDESCRIPTION")
			for _, p := range profiles {
				fmt.Fprintf(w, "%s\t%s\t%s\n", p.Name, p.Provider, p.Description)
			}
			w.Flush()
			return nil
		},
	}
}
