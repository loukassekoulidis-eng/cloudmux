package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "login <profile>",
		Short:             "Authenticate to a cloud profile",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("missing profile name\n\nUsage: cloudmux login <profile>\n\nRun 'cloudmux list' to see available profiles")
			}
			return cobra.ExactArgs(1)(cmd, args)
		},
		ValidArgsFunction: profileCompletionFunc,
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := newManager()
			if err != nil {
				return err
			}
			if err := mgr.Login(args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✓ Logged in: %s\n", args[0])
			return nil
		},
	}
}
