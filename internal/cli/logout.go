package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "logout <profile>",
		Short:             "Clear credentials for a profile",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("missing profile name\n\nUsage: cloudmux logout <profile>\n\nRun 'cloudmux list' to see available profiles")
			}
			return cobra.ExactArgs(1)(cmd, args)
		},
		ValidArgsFunction: profileCompletionFunc,
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := newManager()
			if err != nil {
				return err
			}
			result, err := mgr.Logout(args[0])
			if err != nil {
				return err
			}
			// Output unset statements — the shell hook evals these
			for _, k := range result.EnvKeys {
				fmt.Fprintf(cmd.OutOrStdout(), "unset %s\n", k)
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "✓ Logged out: %s\n", args[0])
			return nil
		},
	}
}
