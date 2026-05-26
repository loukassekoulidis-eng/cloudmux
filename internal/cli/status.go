package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "status [profile]",
		Short:             "Show session status for a profile",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: profileCompletionFunc,
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := newManager()
			if err != nil {
				return err
			}

			var profileName string
			if len(args) > 0 {
				profileName = args[0]
			} else {
				profileName = os.Getenv("CLOUDMUX_ACTIVE_PROFILE")
				if profileName == "" {
					return fmt.Errorf("no active profile — specify a profile name or activate one with 'cloudmux use'")
				}
			}

			status, err := mgr.Status(profileName)
			if err != nil {
				return err
			}

			if !status.Valid {
				fmt.Fprintf(cmd.OutOrStdout(), "✗ %s: not authenticated\n", profileName)
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "✓ %s\n", profileName)
			fmt.Fprintf(cmd.OutOrStdout(), "  Identity: %s\n", status.Identity)
			fmt.Fprintf(cmd.OutOrStdout(), "  Tenant:   %s\n", status.Tenant)
			if !status.ExpiresAt.IsZero() {
				fmt.Fprintf(cmd.OutOrStdout(), "  Expires:  %s\n", status.ExpiresAt.Format("2006-01-02 15:04:05"))
			}
			if status.Region != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "  Region:   %s\n", status.Region)
			}
			return nil
		},
	}
}
