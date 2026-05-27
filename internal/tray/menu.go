package tray

import (
	"fmt"
	"time"

	"fyne.io/systray"
	"github.com/lukassekoulidis/cloudmux/internal/config"
	"github.com/lukassekoulidis/cloudmux/internal/provider"
	"github.com/lukassekoulidis/cloudmux/internal/session"
)

// buildMenu rebuilds the entire systray menu from the current state.
func (a *App) buildMenu() {
	systray.ResetMenu()

	a.mu.Lock()
	pendingDetects := make([]DetectedSession, len(a.pendingDetects))
	copy(pendingDetects, a.pendingDetects)
	profiles := make([]config.Profile, len(a.profiles))
	copy(profiles, a.profiles)
	statuses := make(map[string]*provider.SessionStatus)
	for k, v := range a.statuses {
		statuses[k] = v
	}
	a.mu.Unlock()

	// Detected sessions at the top
	for _, det := range pendingDetects {
		title := fmt.Sprintf("New: %s (%s)", det.Provider, det.Info.Profile.Description)
		parent := systray.AddMenuItem(title, "")

		suggested := det.Info.SuggestedName
		addItem := parent.AddSubMenuItem(fmt.Sprintf("Add as %q", suggested), "")
		go a.handleAddDetection(addItem, det, suggested)

		variants := nameVariants(suggested, det.Provider)
		for _, v := range variants {
			item := parent.AddSubMenuItem(fmt.Sprintf("Add as %q", v), "")
			go a.handleAddDetection(item, det, v)
		}

		copyCmd := parent.AddSubMenuItem("Copy import command...", "")
		go a.handleClick(copyCmd, func() { CopyImportCommand() })

		fp := Fingerprint(det.Provider, det.Info.Profile)
		dismiss := parent.AddSubMenuItem("Dismiss", "")
		go a.handleDismiss(dismiss, fp)

		ignore := parent.AddSubMenuItem("Ignore this account", "")
		go a.handleIgnore(ignore, fp)
	}

	if len(pendingDetects) > 0 {
		systray.AddSeparator()
	}

	// Profiles header
	header := systray.AddMenuItem("Profiles", "")
	header.Disable()

	for _, p := range profiles {
		status := statuses[p.Name]
		title := formatProfileTitle(p, status)
		parent := systray.AddMenuItem(title, "")

		if status != nil && status.Valid {
			// Info submenu items (disabled, informational)
			if status.Identity != "" {
				info := parent.AddSubMenuItem(status.Identity, "")
				info.Disable()
			}
			if status.Tenant != "" {
				info := parent.AddSubMenuItem(fmt.Sprintf("Tenant: %s", status.Tenant), "")
				info.Disable()
			}
			if !status.ExpiresAt.IsZero() {
				remaining := time.Until(status.ExpiresAt)
				info := parent.AddSubMenuItem(fmt.Sprintf("Expires in %s", formatRemaining(remaining)), "")
				info.Disable()
			}

			switchItem := parent.AddSubMenuItem("Switch (copy command)", "")
			pName := p.Name
			go a.handleClick(switchItem, func() { CopyUseCommand(pName) })

			refresh := parent.AddSubMenuItem("Refresh Token", "")
			go a.handleClick(refresh, func() { OpenLoginTerminal(pName) })

			logout := parent.AddSubMenuItem("Logout", "")
			go a.handleLogout(logout, pName)
		} else {
			info := parent.AddSubMenuItem("Session expired or not logged in", "")
			info.Disable()

			reauth := parent.AddSubMenuItem("Re-authenticate", "")
			pName := p.Name
			go a.handleClick(reauth, func() { OpenLoginTerminal(pName) })

			switchItem := parent.AddSubMenuItem("Switch (copy command)", "")
			go a.handleClick(switchItem, func() { CopyUseCommand(pName) })

			remove := parent.AddSubMenuItem("Remove Profile", "")
			go a.handleRemove(remove, pName)
		}
	}

	systray.AddSeparator()

	// Global actions
	importItem := systray.AddMenuItem("Import Sessions...", "")
	go a.handleClick(importItem, func() { a.runDetection() })

	refreshAll := systray.AddMenuItem("Refresh All", "")
	go a.handleClick(refreshAll, func() {
		a.refreshStatuses()
		a.buildMenu()
	})

	doctorItem := systray.AddMenuItem("Run Doctor", "")
	go a.handleClick(doctorItem, func() {
		a.refreshProfiles()
		a.refreshStatuses()
		a.buildMenu()
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
}

// handleClick runs fn every time the menu item is clicked, until quit.
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

// handleAddDetection imports a detected session under the given name.
func (a *App) handleAddDetection(item *systray.MenuItem, det DetectedSession, name string) {
	for {
		select {
		case <-item.ClickedCh:
			detCopy := DetectedSession{
				Provider: det.Provider,
				Info: &provider.ImportInfo{
					SuggestedName: det.Info.SuggestedName,
					Profile:       det.Info.Profile,
					DefaultDir:    det.Info.DefaultDir,
				},
			}
			detCopy.Info.Profile.Name = name
			if err := ImportDetectedSession(a.configDir, detCopy); err != nil {
				a.notifier.NotifyError(fmt.Sprintf("Import failed: %s", err))
				continue
			}
			fp := Fingerprint(det.Provider, det.Info.Profile)
			a.detector.Dismiss(fp)
			a.refreshProfiles()
			a.refreshStatuses()
			a.mu.Lock()
			a.pendingDetects = removeDetection(a.pendingDetects, fp)
			a.iconState.SetNewSession(len(a.pendingDetects) > 0)
			a.mu.Unlock()
			a.updateIcon()
			a.buildMenu()
		case <-a.quit:
			return
		}
	}
}

// handleDismiss marks a detected session as dismissed.
func (a *App) handleDismiss(item *systray.MenuItem, fingerprint string) {
	for {
		select {
		case <-item.ClickedCh:
			a.detector.Dismiss(fingerprint)
			a.mu.Lock()
			a.pendingDetects = removeDetection(a.pendingDetects, fingerprint)
			a.iconState.SetNewSession(len(a.pendingDetects) > 0)
			a.mu.Unlock()
			a.updateIcon()
			a.buildMenu()
		case <-a.quit:
			return
		}
	}
}

// handleIgnore permanently ignores a detected session.
func (a *App) handleIgnore(item *systray.MenuItem, fingerprint string) {
	for {
		select {
		case <-item.ClickedCh:
			a.detector.Ignore(fingerprint)
			a.mu.Lock()
			a.pendingDetects = removeDetection(a.pendingDetects, fingerprint)
			a.iconState.SetNewSession(len(a.pendingDetects) > 0)
			a.mu.Unlock()
			a.updateIcon()
			a.buildMenu()
		case <-a.quit:
			return
		}
	}
}

// handleLogout logs out the given profile.
func (a *App) handleLogout(item *systray.MenuItem, profileName string) {
	for {
		select {
		case <-item.ClickedCh:
			mgr, err := session.NewManager(a.configDir, a.registry, a.audit)
			if err == nil {
				_, _ = mgr.Logout(profileName)
			}
			a.refreshStatuses()
			a.buildMenu()
		case <-a.quit:
			return
		}
	}
}

// handleRemove removes the profile directory.
func (a *App) handleRemove(item *systray.MenuItem, profileName string) {
	for {
		select {
		case <-item.ClickedCh:
			_ = RemoveProfileDir(a.configDir, profileName)
			a.refreshProfiles()
			a.refreshStatuses()
			a.buildMenu()
		case <-a.quit:
			return
		}
	}
}

// formatProfileTitle produces a menu title showing profile name, status, and provider.
func formatProfileTitle(p config.Profile, status *provider.SessionStatus) string {
	if status == nil || !status.Valid {
		return fmt.Sprintf("  %s  expired  %s", p.Name, p.Provider)
	}
	if !status.ExpiresAt.IsZero() {
		remaining := time.Until(status.ExpiresAt)
		return fmt.Sprintf("  %s  %s  %s", p.Name, formatRemaining(remaining), p.Provider)
	}
	return fmt.Sprintf("  %s  valid  %s", p.Name, p.Provider)
}

// formatRemaining returns a human-readable duration string.
func formatRemaining(d time.Duration) string {
	if d < 0 {
		return "expired"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh", int(d.Hours()))
}

// nameVariants generates alternative name suggestions for a detected session.
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

// removeDetection filters out a detected session by fingerprint.
func removeDetection(detections []DetectedSession, fingerprint string) []DetectedSession {
	var result []DetectedSession
	for _, d := range detections {
		if Fingerprint(d.Provider, d.Info.Profile) != fingerprint {
			result = append(result, d)
		}
	}
	return result
}
