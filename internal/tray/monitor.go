package tray

import "time"

// detectionLoop periodically scans providers for new sessions that are
// not yet tracked as profiles.
func (a *App) detectionLoop() {
	interval := time.Duration(a.cfg.TrayDetectionIntervalMinutes) * time.Minute
	if interval <= 0 {
		interval = 5 * time.Minute
	}

	a.runDetection()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.runDetection()
		case <-a.quit:
			return
		}
	}
}

// expiryLoop periodically checks session expiry and updates the menu.
func (a *App) expiryLoop() {
	interval := time.Duration(a.cfg.TrayExpiryCheckIntervalMinutes) * time.Minute
	if interval <= 0 {
		interval = 2 * time.Minute
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.refreshStatuses()
			a.buildMenu()
		case <-a.quit:
			return
		}
	}
}
