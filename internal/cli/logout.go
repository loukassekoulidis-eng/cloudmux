package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout <profile>",
		Short: "Clear credentials for a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := newManager()
			if err != nil {
				return err
			}
			if err := mgr.Logout(args[0]); err != nil {
				return err
			}
			// Output unset statements — the shell hook evals these
			fmt.Fprintln(cmd.OutOrStdout(), "unset CLOUDMUX_ACTIVE_PROFILE")
			fmt.Fprintln(cmd.OutOrStdout(), "unset AZURE_CONFIG_DIR")
			fmt.Fprintln(cmd.OutOrStdout(), "unset AZURE_DEFAULTS_LOCATION")
			fmt.Fprintf(cmd.ErrOrStderr(), "✓ Logged out: %s\n", args[0])
			return nil
		},
	}
}
