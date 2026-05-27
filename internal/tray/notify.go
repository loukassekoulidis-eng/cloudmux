package tray

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gen2brain/beeep"
)

type Notifier struct {
	enabled  bool
	cooldown time.Duration
	mu       sync.Mutex
	lastSent map[string]time.Time
}

func NewNotifier(enabled bool) *Notifier {
	return &Notifier{
		enabled:  enabled,
		cooldown: 1 * time.Hour,
		lastSent: make(map[string]time.Time),
	}
}

func (n *Notifier) NotifyExpiring(profiles []string) {
	if !n.enabled || len(profiles) == 0 {
		return
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	var pending []string
	now := time.Now()
	for _, name := range profiles {
		key := "expiring:" + name
		if last, ok := n.lastSent[key]; ok && now.Sub(last) < n.cooldown {
			continue
		}
		pending = append(pending, name)
		n.lastSent[key] = now
	}

	if len(pending) == 0 {
		return
	}

	var msg string
	if len(pending) == 1 {
		msg = fmt.Sprintf("%s: token expires soon", pending[0])
	} else {
		msg = fmt.Sprintf("%d sessions expiring: %s", len(pending), strings.Join(pending, ", "))
	}
	beeep.Notify("cloudmux", msg, "")
}

func (n *Notifier) NotifyExpired(profiles []string) {
	if !n.enabled || len(profiles) == 0 {
		return
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	var pending []string
	now := time.Now()
	for _, name := range profiles {
		key := "expired:" + name
		if last, ok := n.lastSent[key]; ok && now.Sub(last) < n.cooldown {
			continue
		}
		pending = append(pending, name)
		n.lastSent[key] = now
	}

	if len(pending) == 0 {
		return
	}

	var msg string
	if len(pending) == 1 {
		msg = fmt.Sprintf("%s: session expired", pending[0])
	} else {
		msg = fmt.Sprintf("%d sessions expired: %s", len(pending), strings.Join(pending, ", "))
	}
	beeep.Notify("cloudmux", msg, "")
}

func (n *Notifier) NotifyError(msg string) {
	beeep.Notify("cloudmux", msg, "")
}
