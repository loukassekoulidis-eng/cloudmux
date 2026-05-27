package main

import (
	"log"
	"os"
	"path/filepath"

	"fyne.io/systray"
	"github.com/lukassekoulidis/cloudmux/internal/audit"
	"github.com/lukassekoulidis/cloudmux/internal/config"
	"github.com/lukassekoulidis/cloudmux/internal/provider"
	paws "github.com/lukassekoulidis/cloudmux/internal/provider/aws"
	"github.com/lukassekoulidis/cloudmux/internal/provider/azure"
	"github.com/lukassekoulidis/cloudmux/internal/provider/custom"
	"github.com/lukassekoulidis/cloudmux/internal/provider/gcp"
	"github.com/lukassekoulidis/cloudmux/internal/tray"
)

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	configDir := filepath.Join(home, ".cloudmux")

	reg := provider.NewRegistry()
	reg.Register(azure.New())
	reg.Register(gcp.New())
	reg.Register(paws.New())
	reg.Register(custom.New())

	cfg, _ := config.LoadConfig(filepath.Join(configDir, "config.yaml"))
	auditLogger := audit.New(filepath.Join(configDir, "audit.log"))

	app := tray.NewApp(configDir, reg, cfg, auditLogger)
	app.IconIdle = iconIdle
	app.IconBlue = iconBlue
	app.IconYellow = iconYellow
	app.IconRed = iconRed

	log.Println("starting cloudmux tray...")
	systray.Run(app.OnReady, func() {
		log.Println("tray exiting")
		app.OnExit()
	})
	log.Println("systray.Run returned")
}
