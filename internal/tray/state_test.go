package tray

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIconStateIdle(t *testing.T) {
	s := NewIconState()
	assert.Equal(t, IconIdle, s.Current())
}

func TestIconStatePriority(t *testing.T) {
	s := NewIconState()

	s.SetNewSession(true)
	assert.Equal(t, IconBlue, s.Current())

	s.SetExpiring(true)
	assert.Equal(t, IconYellow, s.Current())

	s.SetExpired(true)
	assert.Equal(t, IconRed, s.Current())

	s.SetExpired(false)
	assert.Equal(t, IconYellow, s.Current())

	s.SetExpiring(false)
	assert.Equal(t, IconBlue, s.Current())

	s.SetNewSession(false)
	assert.Equal(t, IconIdle, s.Current())
}

func TestIconStateReset(t *testing.T) {
	s := NewIconState()
	s.SetNewSession(true)
	s.SetExpired(true)
	s.Reset()
	assert.Equal(t, IconIdle, s.Current())
}
