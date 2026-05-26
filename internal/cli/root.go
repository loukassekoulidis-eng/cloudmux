package cli

import (
	"os"
	"path/filepath"

	"github.com/lukassekoulidis/cloudmux/internal/provider"
	"github.com/lukassekoulidis/cloudmux/internal/provider/azure"
	"github.com/lukassekoulidis/cloudmux/internal/session"
	"github.com/spf13/cobra"
)

var configDir string

func newRegistry() *provider.Registry {
	reg := provider.NewRegistry()
	reg.Register(azure.New())
	return reg
}

func newManager() (*session.Manager, error) {
	return session.NewManager(configDir, newRegistry())
}

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "cloudmux",
		Short: "Cloud identity multiplexer",
		Long:  "Manage parallel cloud CLI sessions without re-authentication.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	defaultDir := filepath.Join(home, ".cloudmux")
	root.PersistentFlags().StringVar(&configDir, "config-dir", defaultDir, "path to cloudmux config directory")

	root.AddCommand(newInitCmd())
	root.AddCommand(newShellHookCmd())
	root.AddCommand(newLoginCmd())
	root.AddCommand(newUseCmd())
	root.AddCommand(newStatusCmd())
	root.AddCommand(newListCmd())
	root.AddCommand(newLogoutCmd())

	return root
}
