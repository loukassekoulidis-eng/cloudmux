package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "use <profile>",
		Short:             "Activate a profile in the current shell",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: profileCompletionFunc,
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := newManager()
			if err != nil {
				return err
			}
			result, err := mgr.Use(args[0])
			if err != nil {
				return err
			}
			// Output export statements — the shell hook evals these.
			// Values must be single-quoted to prevent shell injection.
			for k, v := range result.EnvVars {
				fmt.Fprintf(cmd.OutOrStdout(), "export %s='%s'\n", k, v)
			}
			return nil
		},
	}
}
