package tray

import (
	"fmt"
	"time"

	"fyne.io/systray"
	"github.com/lukassekoulidis/cloudmux/internal/config"
	"github.com/lukassekoulidis/cloudmux/internal/provider"
	"github.com/lukassekoulidis/cloudmux/internal/session"
)

const maxProfileSlots = 10

type profileSlot struct {
	menuItem    *systray.MenuItem
	switchItem  *systray.MenuItem
	refreshItem *systray.MenuItem
	logoutItem  *systray.MenuItem
	name        string
}

// buildMenu creates the static menu structure once. Called from OnReady on main thread.
func (a *App) buildMenu() {
	a.mu.Lock()
	profiles := make([]config.Profile, len(a.profiles))
	copy(profiles, a.profiles)
	a.mu.Unlock()

	// Profiles header
	header := systray.AddMenuItem("Profiles", "")
	header.Disable()

	// Pre-allocate profile slots
	a.profileSlots = make([]profileSlot, maxProfileSlots)
	for i := 0; i < maxProfileSlots; i++ {
		slot := &a.profileSlots[i]
		slot.menuItem = systray.AddMenuItem("", "")
		slot.switchItem = slot.menuItem.AddSubMenuItem("Switch (copy command)", "")
		slot.refreshItem = slot.menuItem.AddSubMenuItem("Re-authenticate", "")
		slot.logoutItem = slot.menuItem.AddSubMenuItem("Logout", "")
		slot.menuItem.Hide()

		idx := i
		go a.handleProfileSlot(idx)
	}

	systray.AddSeparator()

	// Actions
	a.importItem = systray.AddMenuItem("Import Sessions...", "")
	go a.handleClick(a.importItem, func() {
		go a.runDetection()
	})

	a.refreshAllItem = systray.AddMenuItem("Refresh All", "")
	go a.handleClick(a.refreshAllItem, func() {
		go func() {
			a.refreshStatuses()
			a.updateMenuTitles()
			a.updateIcon()
		}()
	})

	systray.AddSeparator()
	quitItem := systray.AddMenuItem("Quit", "")
	go func() {
		select {
		case <-quitItem.ClickedCh:
			systray.Quit()
		case <-a.quit:
		}
	}()

	// Populate slots with current profiles
	a.updateMenuTitles()
}

// handleProfileSlot handles clicks on a pre-allocated profile menu slot.
func (a *App) handleProfileSlot(idx int) {
	slot := &a.profileSlots[idx]
	for {
		select {
		case <-slot.switchItem.ClickedCh:
			a.mu.Lock()
			name := slot.name
			a.mu.Unlock()
			if name != "" {
				CopyUseCommand(name)
			}
		case <-slot.refreshItem.ClickedCh:
			a.mu.Lock()
			name := slot.name
			a.mu.Unlock()
			if name != "" {
				OpenLoginTerminal(name)
			}
		case <-slot.logoutItem.ClickedCh:
			a.mu.Lock()
			name := slot.name
			a.mu.Unlock()
			if name != "" {
				go func() {
					mgr, err := session.NewManager(a.configDir, a.registry, a.audit)
					if err == nil {
						_, _ = mgr.Logout(name)
					}
					a.refreshStatuses()
					a.updateMenuTitles()
					a.updateIcon()
				}()
			}
		case <-a.quit:
			return
		}
	}
}

// updateMenuTitles refreshes profile slot titles from current state. Safe from any goroutine.
func (a *App) updateMenuTitles() {
	a.mu.Lock()
	profiles := make([]config.Profile, len(a.profiles))
	copy(profiles, a.profiles)
	statuses := make(map[string]*provider.SessionStatus)
	for k, v := range a.statuses {
		statuses[k] = v
	}
	a.mu.Unlock()

	for i := 0; i < maxProfileSlots; i++ {
		slot := &a.profileSlots[i]
		if i < len(profiles) {
			p := profiles[i]
			status := statuses[p.Name]
			title := formatProfileTitle(p, status)
			slot.menuItem.SetTitle(title)
			slot.menuItem.Show()

			a.mu.Lock()
			slot.name = p.Name
			a.mu.Unlock()

			if status != nil && status.Valid {
				slot.refreshItem.SetTitle("Refresh Token")
				slot.logoutItem.SetTitle("Logout")
			} else {
				slot.refreshItem.SetTitle("Re-authenticate")
				slot.logoutItem.SetTitle("Remove Profile")
			}
		} else {
			slot.menuItem.Hide()
			a.mu.Lock()
			slot.name = ""
			a.mu.Unlock()
		}
	}
}

func (a *App) handleClick(item *systray.MenuItem, fn func()) {
	for {
		select {
		case <-item.ClickedCh:
			fn()
		case <-a.quit:
			return
		}
	}
}

func formatProfileTitle(p config.Profile, status *provider.SessionStatus) string {
	if status == nil || !status.Valid {
		return fmt.Sprintf("  %s — expired  [%s]", p.Name, p.Provider)
	}
	if !status.ExpiresAt.IsZero() {
		remaining := time.Until(status.ExpiresAt)
		return fmt.Sprintf("  %s — %s  [%s]", p.Name, formatRemaining(remaining), p.Provider)
	}
	return fmt.Sprintf("  %s — valid  [%s]", p.Name, p.Provider)
}

func formatRemaining(d time.Duration) string {
	if d < 0 {
		return "expired"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh", int(d.Hours()))
}

func nameVariants(suggested, providerName string) []string {
	var variants []string
	base := suggested
	suffixes := []string{"-azure", "-gcp", "-aws"}
	for _, s := range suffixes {
		if len(base) > len(s) && base[len(base)-len(s):] == s {
			base = base[:len(base)-len(s)]
			break
		}
	}
	if base != suggested {
		variants = append(variants, base)
		variants = append(variants, providerName+"-"+base)
	}
	return variants
}

func removeDetection(detections []DetectedSession, fingerprint string) []DetectedSession {
	var result []DetectedSession
	for _, d := range detections {
		if Fingerprint(d.Provider, d.Info.Profile) != fingerprint {
			result = append(result, d)
		}
	}
	return result
}
