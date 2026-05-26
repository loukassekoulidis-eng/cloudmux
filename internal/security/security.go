package security

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

var profileNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

var reservedNames = map[string]bool{
	".": true, "..": true,
	"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true,
}

func ValidateProfileName(name string) error {
	if name == "" {
		return fmt.Errorf("profile name cannot be empty")
	}
	if len(name) > 64 {
		return fmt.Errorf("profile name too long (%d chars, max 64)", len(name))
	}
	if reservedNames[strings.ToUpper(name)] {
		return fmt.Errorf("profile name %q is reserved", name)
	}
	if !profileNameRegex.MatchString(name) {
		return fmt.Errorf("profile name %q contains invalid characters (allowed: a-z, A-Z, 0-9, _, -; must not start with - or .)", name)
	}
	return nil
}

func EnforcePermissions(path string, isDir bool) error {
	expected := os.FileMode(0600)
	if isDir {
		expected = os.FileMode(0700)
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Mode().Perm() != expected {
		return fmt.Errorf(
			"insecure permissions %04o on %s (expected %04o), run: chmod %04o %s",
			info.Mode().Perm(), path, expected, expected, path,
		)
	}
	return nil
}

func EnsureDir(path string) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return os.MkdirAll(path, 0700)
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s exists but is not a directory", path)
	}
	return EnforcePermissions(path, true)
}
