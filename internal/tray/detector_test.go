package tray

import (
	"testing"

	"github.com/lukassekoulidis/cloudmux/internal/config"
	"github.com/lukassekoulidis/cloudmux/internal/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectorNewSession(t *testing.T) {
	existing := []config.Profile{
		{Name: "acme-azure", Provider: "azure", Azure: config.AzureConfig{TenantID: "t-111"}},
	}
	detected := []DetectedSession{
		{Provider: "azure", Info: &provider.ImportInfo{
			SuggestedName: "contoso-azure",
			Profile:       config.Profile{Provider: "azure", Azure: config.AzureConfig{TenantID: "t-222"}},
		}},
	}

	d := NewDetector(nil)
	newSessions := d.Diff(existing, detected)
	require.Len(t, newSessions, 1)
	assert.Equal(t, "contoso-azure", newSessions[0].Info.SuggestedName)
}

func TestDetectorKnownSession(t *testing.T) {
	existing := []config.Profile{
		{Name: "acme-azure", Provider: "azure", Azure: config.AzureConfig{TenantID: "t-111"}},
	}
	detected := []DetectedSession{
		{Provider: "azure", Info: &provider.ImportInfo{
			SuggestedName: "acme-azure",
			Profile:       config.Profile{Provider: "azure", Azure: config.AzureConfig{TenantID: "t-111"}},
		}},
	}

	d := NewDetector(nil)
	newSessions := d.Diff(existing, detected)
	assert.Empty(t, newSessions)
}

func TestDetectorDismissed(t *testing.T) {
	detected := []DetectedSession{
		{Provider: "azure", Info: &provider.ImportInfo{
			SuggestedName: "contoso-azure",
			Profile:       config.Profile{Provider: "azure", Azure: config.AzureConfig{TenantID: "t-222"}},
		}},
	}

	d := NewDetector(nil)
	d.Dismiss("azure:t-222")
	newSessions := d.Diff(nil, detected)
	assert.Empty(t, newSessions)
}

func TestDetectorIgnored(t *testing.T) {
	detected := []DetectedSession{
		{Provider: "azure", Info: &provider.ImportInfo{
			SuggestedName: "contoso-azure",
			Profile:       config.Profile{Provider: "azure", Azure: config.AzureConfig{TenantID: "t-222"}},
		}},
	}

	d := NewDetector([]string{"azure:t-222"})
	newSessions := d.Diff(nil, detected)
	assert.Empty(t, newSessions)
}

func TestFingerprint(t *testing.T) {
	p := config.Profile{Provider: "azure", Azure: config.AzureConfig{TenantID: "t-123"}}
	assert.Equal(t, "azure:t-123", Fingerprint("azure", p))

	p2 := config.Profile{Provider: "gcp", GCP: config.GCPConfig{ProjectID: "my-proj"}}
	assert.Equal(t, "gcp:my-proj", Fingerprint("gcp", p2))

	p3 := config.Profile{Provider: "aws", AWS: config.AWSConfig{ProfileName: "prod"}}
	assert.Equal(t, "aws:prod", Fingerprint("aws", p3))
}
