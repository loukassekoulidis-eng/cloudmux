package provider

import (
	"fmt"
	"time"

	"github.com/lukassekoulidis/cloudmux/internal/config"
)

type SessionStatus struct {
	Valid     bool
	Identity  string
	Tenant    string
	ExpiresAt time.Time
	Region    string
}

type Provider interface {
	Name() string
	EnvVars(profile config.Profile, profileDir string) (map[string]string, error)
	Login(profile config.Profile, profileDir string) error
	Logout(profile config.Profile, profileDir string) error
	Status(profile config.Profile, profileDir string) (*SessionStatus, error)
	Validate(profile config.Profile) error
}

type Registry struct {
	providers map[string]Provider
}

func NewRegistry() *Registry {
	return &Registry{providers: make(map[string]Provider)}
}

func (r *Registry) Register(p Provider) {
	r.providers[p.Name()] = p
}

func (r *Registry) Get(name string) (Provider, error) {
	p, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider %q", name)
	}
	return p, nil
}
