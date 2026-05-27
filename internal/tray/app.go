package tray

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"fyne.io/systray"
	"github.com/lukassekoulidis/cloudmux/internal/audit"
	"github.com/lukassekoulidis/cloudmux/internal/config"
	"github.com/lukassekoulidis/cloudmux/internal/provider"
	"github.com/lukassekoulidis/cloudmux/internal/session"
)

// App is the main tray application that wires together the icon state,
// session detector, notifier, and systray menu.
type App struct {
	configDir string
	registry  *provider.Registry
	cfg       config.Config
	audit     *audit.Logger

	iconState *IconState
	detector  *Detector
	notifier  *Notifier

	mu             sync.Mutex
	profiles       []config.Profile
	statuses       map[string]*provider.SessionStatus
	pendingDetects []DetectedSession

	IconIdle   []byte
	IconBlue   []byte
	IconYellow []byte
	IconRed    []byte

	// Menu item references (populated once in buildMenu)
	profileSlots   []profileSlot
	importItem     *systray.MenuItem
	refreshAllItem *systray.MenuItem

	quit chan struct{}
}

// NewApp creates a new tray App wired with the given configuration.
func NewApp(configDir string, registry *provider.Registry, cfg config.Config, auditLogger *audit.Logger) *App {
	return &App{
		configDir: configDir,
		registry:  registry,
		cfg:       cfg,
		audit:     auditLogger,
		iconState: NewIconState(),
		detector:  NewDetector(cfg.IgnoredSessions),
		notifier:  NewNotifier(cfg.TrayNotifications),
		statuses:  make(map[string]*provider.SessionStatus),
		quit:      make(chan struct{}),
	}
}

// OnReady is the systray onReady callback. It sets the initial icon,
// loads profiles and statuses, builds the menu, and starts background loops.
func (a *App) OnReady() {
	systray.SetTitle("CMUX")
	systray.SetTooltip("cloudmux")

	a.refreshProfiles()
	a.buildMenu()

	go func() {
		a.refreshStatuses()
		a.updateMenuTitles()
		a.updateIcon()

		go a.detectionLoop()
		go a.expiryLoop()
	}()
}

// OnExit is the systray onExit callback.
func (a *App) OnExit() {
	close(a.quit)
}

// refreshProfiles reloads profiles from disk.
func (a *App) refreshProfiles() {
	profilesPath := filepath.Join(a.configDir, "profiles.yaml")
	profiles, err := config.LoadProfiles(profilesPath)
	if err != nil {
		return
	}
	a.mu.Lock()
	a.profiles = profiles
	a.mu.Unlock()
}

// refreshStatuses checks session status for every profile and updates
// the icon state and notifications accordingly.
func (a *App) refreshStatuses() {
	a.mu.Lock()
	profiles := make([]config.Profile, len(a.profiles))
	copy(profiles, a.profiles)
	a.mu.Unlock()

	mgr, err := session.NewManager(a.configDir, a.registry, a.audit)
	if err != nil {
		return
	}

	statuses := make(map[string]*provider.SessionStatus)
	var expiring, expired []string

	for _, p := range profiles {
		profDir := filepath.Join(a.configDir, "profiles", p.Name)
		if _, err := os.Stat(profDir); os.IsNotExist(err) {
			statuses[p.Name] = &provider.SessionStatus{Valid: false}
			continue
		}
		status, err := mgr.Status(p.Name)
		if err != nil {
			statuses[p.Name] = &provider.SessionStatus{Valid: false}
			continue
		}
		statuses[p.Name] = status

		if status.Valid && !status.ExpiresAt.IsZero() {
			remaining := time.Until(status.ExpiresAt)
			if remaining < 0 {
				expired = append(expired, p.Name)
			} else if remaining < time.Duration(a.cfg.ExpiryWarningMinutes)*time.Minute {
				expiring = append(expiring, p.Name)
			}
		}
		if !status.Valid {
			expired = append(expired, p.Name)
		}
	}

	a.mu.Lock()
	a.statuses = statuses
	a.iconState.SetExpired(len(expired) > 0)
	a.iconState.SetExpiring(len(expiring) > 0)
	a.mu.Unlock()

	a.notifier.NotifyExpiring(expiring)
	a.notifier.NotifyExpired(expired)
	a.updateIcon()
}

// runDetection scans all providers for sessions not yet tracked as profiles.
func (a *App) runDetection() {
	a.refreshProfiles()

	var detected []DetectedSession
	for _, p := range a.registry.All() {
		info, err := p.Detect()
		if err != nil || info == nil {
			continue
		}
		detected = append(detected, DetectedSession{Provider: p.Name(), Info: info})
	}

	a.mu.Lock()
	profiles := make([]config.Profile, len(a.profiles))
	copy(profiles, a.profiles)
	newSessions := a.detector.Diff(profiles, detected)
	a.pendingDetects = newSessions
	a.iconState.SetNewSession(len(newSessions) > 0)
	a.mu.Unlock()

	a.updateIcon()
	a.updateMenuTitles()
}

// updateIcon sets the systray icon based on the current icon state.
func (a *App) updateIcon() {
	a.mu.Lock()
	state := a.iconState.Current()
	a.mu.Unlock()

	switch state {
	case IconBlue:
		if a.IconBlue != nil {
			systray.SetTemplateIcon(a.IconBlue, a.IconBlue)
		}
	case IconYellow:
		if a.IconYellow != nil {
			systray.SetTemplateIcon(a.IconYellow, a.IconYellow)
		}
	case IconRed:
		if a.IconRed != nil {
			systray.SetTemplateIcon(a.IconRed, a.IconRed)
		}
	default:
		if a.IconIdle != nil {
			systray.SetTemplateIcon(a.IconIdle, a.IconIdle)
		}
	}
}
