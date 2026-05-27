package aws

import (
	"testing"

	"github.com/lukassekoulidis/cloudmux/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAWSName(t *testing.T) {
	p := New()
	assert.Equal(t, "aws", p.Name())
}

func TestAWSEnvVars(t *testing.T) {
	p := New()
	profile := config.Profile{
		Name:     "my-aws",
		Provider: "aws",
		AWS: config.AWSConfig{
			ProfileName: "prod-account",
			Region:      "eu-central-1",
		},
	}
	envs, err := p.EnvVars(profile, "/home/user/.cloudmux/profiles/my-aws")
	require.NoError(t, err)

	assert.Equal(t, "prod-account", envs["AWS_PROFILE"])
	assert.Equal(t, "eu-central-1", envs["AWS_DEFAULT_REGION"])
}

func TestAWSEnvVarsNoRegion(t *testing.T) {
	p := New()
	profile := config.Profile{
		Name:     "minimal",
		Provider: "aws",
		AWS: config.AWSConfig{
			ProfileName: "dev",
		},
	}
	envs, err := p.EnvVars(profile, "/tmp/profiles/minimal")
	require.NoError(t, err)

	assert.Equal(t, "dev", envs["AWS_PROFILE"])
	_, hasRegion := envs["AWS_DEFAULT_REGION"]
	assert.False(t, hasRegion)
}

func TestAWSSuggestName(t *testing.T) {
	assert.Equal(t, "prod-aws", suggestName("prod", ""))
	assert.Equal(t, "123456789-aws", suggestName("", "123456789"))
	assert.Equal(t, "unknown-aws", suggestName("", ""))
}

func TestAWSValidate(t *testing.T) {
	p := New()

	t.Run("valid", func(t *testing.T) {
		profile := config.Profile{
			Provider: "aws",
			AWS:      config.AWSConfig{ProfileName: "my-profile"},
		}
		assert.NoError(t, p.Validate(profile))
	})

	t.Run("missing profile_name", func(t *testing.T) {
		profile := config.Profile{
			Provider: "aws",
			AWS:      config.AWSConfig{},
		}
		err := p.Validate(profile)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "profile_name")
	})
}
