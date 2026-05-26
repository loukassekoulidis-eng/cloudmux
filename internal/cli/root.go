package cli

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var configDir string

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "cloudmux",
		Short: "Cloud identity multiplexer",
		Long:  "Manage parallel cloud CLI sessions without re-authentication.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	defaultDir := filepath.Join(os.Getenv("HOME"), ".cloudmux")
	root.PersistentFlags().StringVar(&configDir, "config-dir", defaultDir, "path to cloudmux config directory")

	return root
}
