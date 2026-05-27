package tray

import (
	"github.com/lukassekoulidis/cloudmux/internal/config"
	"github.com/lukassekoulidis/cloudmux/internal/provider"
)

type DetectedSession struct {
	Provider string
	Info     *provider.ImportInfo
}

type Detector struct {
	ignored   map[string]bool
	dismissed map[string]bool
}

func NewDetector(ignoredSessions []string) *Detector {
	ignored := make(map[string]bool)
	for _, s := range ignoredSessions {
		ignored[s] = true
	}
	return &Detector{
		ignored:   ignored,
		dismissed: make(map[string]bool),
	}
}

func Fingerprint(providerName string, profile config.Profile) string {
	switch providerName {
	case "azure":
		return "azure:" + profile.Azure.TenantID
	case "gcp":
		return "gcp:" + profile.GCP.ProjectID
	case "aws":
		return "aws:" + profile.AWS.ProfileName
	default:
		return providerName + ":" + profile.Name
	}
}

func (d *Detector) Dismiss(fingerprint string) {
	d.dismissed[fingerprint] = true
}

func (d *Detector) Ignore(fingerprint string) {
	d.ignored[fingerprint] = true
}

func (d *Detector) Diff(existing []config.Profile, detected []DetectedSession) []DetectedSession {
	known := make(map[string]bool)
	for _, p := range existing {
		known[Fingerprint(p.Provider, p)] = true
	}

	var newSessions []DetectedSession
	for _, det := range detected {
		fp := Fingerprint(det.Provider, det.Info.Profile)
		if known[fp] || d.dismissed[fp] || d.ignored[fp] {
			continue
		}
		newSessions = append(newSessions, det)
	}
	return newSessions
}
