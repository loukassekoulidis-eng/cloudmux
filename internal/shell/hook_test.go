package shell

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateHook(t *testing.T) {
	t.Run("bash hook", func(t *testing.T) {
		out, err := GenerateHook("bash", "cloudmux")
		require.NoError(t, err)
		assert.Contains(t, out, "cloudmux()")
		assert.Contains(t, out, "eval")
		assert.Contains(t, out, "use")
		assert.Contains(t, out, "logout")
		assert.Contains(t, out, "CLOUDMUX_ACTIVE_PROFILE")
	})

	t.Run("zsh hook", func(t *testing.T) {
		out, err := GenerateHook("zsh", "cloudmux")
		require.NoError(t, err)
		assert.Contains(t, out, "cloudmux()")
		assert.Contains(t, out, "eval")
		assert.Contains(t, out, "CLOUDMUX_ACTIVE_PROFILE")
	})

	t.Run("unsupported shell", func(t *testing.T) {
		_, err := GenerateHook("powershell", "cloudmux")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported")
	})
}

func TestGenerateHookFish(t *testing.T) {
	out, err := GenerateHook("fish", "cloudmux")
	require.NoError(t, err)
	assert.Contains(t, out, "function cloudmux")
	assert.Contains(t, out, "set -gx")
	assert.Contains(t, out, "set -e")
	assert.Contains(t, out, "CLOUDMUX_ACTIVE_PROFILE")
}
