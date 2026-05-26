package cli

import (
	"fmt"

	"github.com/lukassekoulidis/cloudmux/internal/shell"
	"github.com/spf13/cobra"
)

func newShellHookCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "shell-hook <bash|zsh>",
		Short: "Generate shell hook for eval",
		Long:  "Output a shell hook script. Add to your shell rc:\n  eval \"$(cloudmux shell-hook zsh)\"",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			hook, err := shell.GenerateHook(args[0], "cloudmux")
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), hook)
			return nil
		},
	}
}
