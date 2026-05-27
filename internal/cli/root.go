package cli

import (
	"os"
	"path/filepath"

	"github.com/lukassekoulidis/cloudmux/internal/audit"
	"github.com/lukassekoulidis/cloudmux/internal/provider"
	paws "github.com/lukassekoulidis/cloudmux/internal/provider/aws"
	"github.com/lukassekoulidis/cloudmux/internal/provider/azure"
	"github.com/lukassekoulidis/cloudmux/internal/provider/custom"
	"github.com/lukassekoulidis/cloudmux/internal/provider/gcp"
	"github.com/lukassekoulidis/cloudmux/internal/session"
	"github.com/spf13/cobra"
)

var configDir string

func newRegistry() *provider.Registry {
	reg := provider.NewRegistry()
	reg.Register(azure.New())
	reg.Register(gcp.New())
	reg.Register(paws.New())
	reg.Register(custom.New())
	return reg
}

func newManager() (*session.Manager, error) {
	auditPath := filepath.Join(configDir, "audit.log")
	auditLogger := audit.New(auditPath)
	return session.NewManager(configDir, newRegistry(), auditLogger)
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
	root.AddCommand(newDoctorCmd())
	root.AddCommand(newCompletionCmd())
	root.AddCommand(newGCCmd())

	return root
}
