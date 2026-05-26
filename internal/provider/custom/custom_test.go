package custom

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lukassekoulidis/cloudmux/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomName(t *testing.T) {
	p := New()
	assert.Equal(t, "custom", p.Name())
}

func TestCustomEnvVars(t *testing.T) {
	p := New()
	profile := config.Profile{
		Name:     "hetzner-prod",
		Provider: "custom",
		Custom: config.CustomConfig{
			Env: map[string]string{
				"HCLOUD_TOKEN_FILE": "{profile_dir}/token",
				"HCLOUD_CONTEXT":    "{name}",
			},
		},
	}
	envs, err := p.EnvVars(profile, "/home/user/.cloudmux/profiles/hetzner-prod")
	require.NoError(t, err)

	assert.Equal(t, "/home/user/.cloudmux/profiles/hetzner-prod/token", envs["HCLOUD_TOKEN_FILE"])
	assert.Equal(t, "hetzner-prod", envs["HCLOUD_CONTEXT"])
}

func TestCustomEnvVarsHomeExpansion(t *testing.T) {
	p := New()
	home, _ := os.UserHomeDir()
	profile := config.Profile{
		Name:     "test",
		Provider: "custom",
		Custom: config.CustomConfig{
			Env: map[string]string{
				"MY_VAR": "{home}/.myconfig",
			},
		},
	}
	envs, err := p.EnvVars(profile, "/tmp/profiles/test")
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(home, ".myconfig"), envs["MY_VAR"])
}

func TestCustomValidate(t *testing.T) {
	p := New()

	t.Run("valid with env", func(t *testing.T) {
		profile := config.Profile{
			Provider: "custom",
			Custom: config.CustomConfig{
				Env: map[string]string{"KEY": "val"},
			},
		}
		assert.NoError(t, p.Validate(profile))
	})

	t.Run("valid with login_command only", func(t *testing.T) {
		profile := config.Profile{
			Provider: "custom",
			Custom: config.CustomConfig{
				LoginCommand: "do-something",
			},
		}
		assert.NoError(t, p.Validate(profile))
	})

	t.Run("empty custom config", func(t *testing.T) {
		profile := config.Profile{
			Provider: "custom",
			Custom:   config.CustomConfig{},
		}
		err := p.Validate(profile)
		require.Error(t, err)
	})
}

func TestExpandTemplateVars(t *testing.T) {
	home, _ := os.UserHomeDir()
	result := expandTemplateVars("path/{profile_dir}/sub/{name}/end/{home}", "/data/profiles/foo", "foo")
	assert.Equal(t, "path//data/profiles/foo/sub/foo/end/"+home, result)
}
