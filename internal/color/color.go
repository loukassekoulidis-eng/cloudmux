package color

import (
	"fmt"
	"os"
)

type Color struct {
	enabled bool
}

func New(isTTY bool) *Color {
	enabled := isTTY && os.Getenv("NO_COLOR") == ""
	return &Color{enabled: enabled}
}

func (c *Color) wrap(code string, s string) string {
	if !c.enabled {
		return s
	}
	return fmt.Sprintf("\033[%sm%s\033[0m", code, s)
}

func (c *Color) Green(s string) string  { return c.wrap("32", s) }
func (c *Color) Red(s string) string    { return c.wrap("31", s) }
func (c *Color) Yellow(s string) string { return c.wrap("33", s) }
func (c *Color) Bold(s string) string   { return c.wrap("1", s) }
func (c *Color) Dim(s string) string    { return c.wrap("2", s) }
