package gcp

import (
	"testing"

	"github.com/lukassekoulidis/cloudmux/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGCPName(t *testing.T) {
	p := New()
	assert.Equal(t, "gcp", p.Name())
}

func TestGCPEnvVarsIsolated(t *testing.T) {
	p := New()
	profile := config.Profile{
		Name:     "my-gcp",
		Provider: "gcp",
		GCP: config.GCPConfig{
			ProjectID: "my-project",
			Region:    "europe-west3",
			Zone:      "europe-west3-a",
		},
	}
	envs, err := p.EnvVars(profile, "/home/user/.cloudmux/profiles/my-gcp")
	require.NoError(t, err)

	assert.Equal(t, "/home/user/.cloudmux/profiles/my-gcp/.config/gcloud", envs["CLOUDSDK_CONFIG"])
	assert.Equal(t, "my-project", envs["CLOUDSDK_CORE_PROJECT"])
	assert.Equal(t, "europe-west3", envs["CLOUDSDK_COMPUTE_REGION"])
	assert.Equal(t, "europe-west3-a", envs["CLOUDSDK_COMPUTE_ZONE"])
	_, hasCloudsdk := envs["CLOUDSDK_ACTIVE_CONFIG_NAME"]
	assert.False(t, hasCloudsdk)
}

func TestGCPEnvVarsNamedConfig(t *testing.T) {
	p := New()
	profile := config.Profile{
		Name:     "my-gcp",
		Provider: "gcp",
		GCP: config.GCPConfig{
			ProjectID:      "my-project",
			UseNamedConfig: true,
		},
	}
	envs, err := p.EnvVars(profile, "/home/user/.cloudmux/profiles/my-gcp")
	require.NoError(t, err)

	assert.Equal(t, "my-gcp", envs["CLOUDSDK_ACTIVE_CONFIG_NAME"])
	_, hasCloudsdk := envs["CLOUDSDK_CONFIG"]
	assert.False(t, hasCloudsdk)
}

func TestGCPEnvVarsMinimal(t *testing.T) {
	p := New()
	profile := config.Profile{
		Name:     "minimal",
		Provider: "gcp",
		GCP: config.GCPConfig{
			ProjectID: "proj",
		},
	}
	envs, err := p.EnvVars(profile, "/tmp/profiles/minimal")
	require.NoError(t, err)

	assert.Equal(t, "/tmp/profiles/minimal/.config/gcloud", envs["CLOUDSDK_CONFIG"])
	assert.Equal(t, "proj", envs["CLOUDSDK_CORE_PROJECT"])
	_, hasRegion := envs["CLOUDSDK_COMPUTE_REGION"]
	assert.False(t, hasRegion)
	_, hasZone := envs["CLOUDSDK_COMPUTE_ZONE"]
	assert.False(t, hasZone)
}

func TestGCPValidate(t *testing.T) {
	p := New()

	t.Run("valid", func(t *testing.T) {
		profile := config.Profile{
			Provider: "gcp",
			GCP:      config.GCPConfig{ProjectID: "my-proj"},
		}
		assert.NoError(t, p.Validate(profile))
	})

	t.Run("missing project_id", func(t *testing.T) {
		profile := config.Profile{
			Provider: "gcp",
			GCP:      config.GCPConfig{},
		}
		err := p.Validate(profile)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project_id")
	})
}
