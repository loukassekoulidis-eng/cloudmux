package color

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGreen(t *testing.T) {
	os.Unsetenv("NO_COLOR")
	c := New(true)
	assert.Equal(t, "\033[32mhello\033[0m", c.Green("hello"))
}

func TestRed(t *testing.T) {
	os.Unsetenv("NO_COLOR")
	c := New(true)
	assert.Equal(t, "\033[31merror\033[0m", c.Red("error"))
}

func TestYellow(t *testing.T) {
	os.Unsetenv("NO_COLOR")
	c := New(true)
	assert.Equal(t, "\033[33mwarn\033[0m", c.Yellow("warn"))
}

func TestBold(t *testing.T) {
	os.Unsetenv("NO_COLOR")
	c := New(true)
	assert.Equal(t, "\033[1mtitle\033[0m", c.Bold("title"))
}

func TestDim(t *testing.T) {
	os.Unsetenv("NO_COLOR")
	c := New(true)
	assert.Equal(t, "\033[2mfaded\033[0m", c.Dim("faded"))
}

func TestNoColor(t *testing.T) {
	os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")
	c := New(true)
	assert.Equal(t, "hello", c.Green("hello"))
	assert.Equal(t, "error", c.Red("error"))
}

func TestNotTTY(t *testing.T) {
	os.Unsetenv("NO_COLOR")
	c := New(false)
	assert.Equal(t, "hello", c.Green("hello"))
}
