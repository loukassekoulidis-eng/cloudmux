package azure

import (
	"testing"

	"github.com/lukassekoulidis/cloudmux/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAzureEnvVars(t *testing.T) {
	p := New()
	profile := config.Profile{
		Name:     "test-azure",
		Provider: "azure",
		Azure: config.AzureConfig{
			TenantID:        "tenant-123",
			SubscriptionID:  "sub-456",
			DefaultLocation: "westeurope",
		},
	}
	envs, err := p.EnvVars(profile, "/home/user/.cloudmux/profiles/test-azure")
	require.NoError(t, err)

	assert.Equal(t, "/home/user/.cloudmux/profiles/test-azure/.azure", envs["AZURE_CONFIG_DIR"])
	assert.Equal(t, "westeurope", envs["AZURE_DEFAULTS_LOCATION"])
}

func TestAzureEnvVarsNoOptionals(t *testing.T) {
	p := New()
	profile := config.Profile{
		Name:     "minimal",
		Provider: "azure",
		Azure: config.AzureConfig{
			TenantID: "tenant-123",
		},
	}
	envs, err := p.EnvVars(profile, "/tmp/profiles/minimal")
	require.NoError(t, err)

	assert.Equal(t, "/tmp/profiles/minimal/.azure", envs["AZURE_CONFIG_DIR"])
	_, hasLocation := envs["AZURE_DEFAULTS_LOCATION"]
	assert.False(t, hasLocation)
}

func TestAzureValidate(t *testing.T) {
	p := New()

	t.Run("valid", func(t *testing.T) {
		profile := config.Profile{
			Provider: "azure",
			Azure:    config.AzureConfig{TenantID: "t-123"},
		}
		assert.NoError(t, p.Validate(profile))
	})

	t.Run("missing tenant_id", func(t *testing.T) {
		profile := config.Profile{
			Provider: "azure",
			Azure:    config.AzureConfig{},
		}
		err := p.Validate(profile)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant_id")
	})
}

func TestAzureName(t *testing.T) {
	p := New()
	assert.Equal(t, "azure", p.Name())
}

func TestSuggestName(t *testing.T) {
	assert.Equal(t, "we-build-ai-azure", suggestName("we-build-ai.de"))
	assert.Equal(t, "mycompany-azure", suggestName("mycompany.onmicrosoft.com"))
	assert.Equal(t, "tenant-123-azure", suggestName("tenant-123"))
	assert.Equal(t, "unknown-azure", suggestName(""))
}
