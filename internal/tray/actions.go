package tray

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/lukassekoulidis/cloudmux/internal/config"
	"github.com/lukassekoulidis/cloudmux/internal/copydir"
	"github.com/lukassekoulidis/cloudmux/internal/security"
	"golang.design/x/clipboard"
)

func init() {
	clipboard.Init()
}

func CopyToClipboard(text string) {
	clipboard.Write(clipboard.FmtText, []byte(text))
}

func CopyUseCommand(profileName string) {
	CopyToClipboard(fmt.Sprintf("cloudmux use %s", profileName))
}

func CopyImportCommand() {
	CopyToClipboard("cloudmux import --name ")
}

func OpenLoginTerminal(profileName string) error {
	binary, err := os.Executable()
	if err != nil {
		binary = "cloudmux"
	}
	// The tray binary is cloudmux-tray, but we need the CLI binary.
	// Assume cloudmux is on PATH.
	binary = "cloudmux"

	switch runtime.GOOS {
	case "darwin":
		script := fmt.Sprintf("%s login %s", binary, profileName)
		return exec.Command("osascript", "-e",
			fmt.Sprintf(`tell application "Terminal" to do script "%s"`, script),
		).Start()
	case "linux":
		for _, term := range []string{"x-terminal-emulator", "gnome-terminal", "xterm"} {
			if _, err := exec.LookPath(term); err == nil {
				return exec.Command(term, "-e", binary, "login", profileName).Start()
			}
		}
		return fmt.Errorf("no terminal emulator found")
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func RemoveProfileDir(baseDir, profileName string) error {
	profDir := filepath.Join(baseDir, "profiles", profileName)
	return os.RemoveAll(profDir)
}

func ImportDetectedSession(baseDir string, det DetectedSession) error {
	profilesPath := filepath.Join(baseDir, "profiles.yaml")
	profile := det.Info.Profile

	if det.Info.DefaultDir != "" {
		profDir := filepath.Join(baseDir, "profiles", profile.Name)
		if err := security.EnsureDir(profDir); err != nil {
			return err
		}

		var subDir string
		switch det.Provider {
		case "azure":
			subDir = ".azure"
		case "gcp":
			subDir = filepath.Join(".config", "gcloud")
		}

		if subDir != "" {
			dstPath := filepath.Join(profDir, subDir)
			if err := copydir.Copy(det.Info.DefaultDir, dstPath); err != nil {
				return fmt.Errorf("copying config: %w", err)
			}
		}

		tsPath := filepath.Join(profDir, ".cloudmux_login_ts")
		os.WriteFile(tsPath, []byte(time.Now().UTC().Format(time.RFC3339)), 0600)
	}

	return config.AppendProfile(profilesPath, profile)
}
