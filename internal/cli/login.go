package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "login <profile>",
		Short:             "Authenticate to a cloud profile",
		Args:              cobra.ExactArgs(1),
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
