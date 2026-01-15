package nameutil

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ValidateName checks whether the provided name is acceptable for a command set.
// It trims and checks for empty names and non-UTF8 bytes. It does NOT mutate the
// input; use SanitizeName to remove undesirable characters first when desired.
func ValidateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("invalid name: name cannot be empty")
	}
	if !utf8.ValidString(name) {
		return fmt.Errorf("invalid name: contains invalid encoding")
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return fmt.Errorf("invalid name: contains control character U+%04X (%q)", r, r)
		}
	}
	return nil
}

// SanitizeName removes common invisible/control characters and returns the
// sanitized string and a boolean indicating whether any change was made.
// It removes control characters, NULs, and zero-width characters commonly
// introduced by copy/paste (e.g., U+200B). Trimming of leading/trailing
// whitespace is also performed.
func SanitizeName(name string) (string, bool) {
	if name == "" {
		return name, false
	}
	runes := []rune(name)
	out := make([]rune, 0, len(runes))
	changelog := false
	for _, r := range runes {
		// keep printable chars and spaces/tabs but remove control chars
		if unicode.IsControl(r) {
			changelog = true
			continue
		}
		// remove zero-width and other invisible separators
		switch r {
		case '\u200B', '\u200C', '\u200D', '\uFEFF':
			changelog = true
			continue
		}
		out = append(out, r)
	}
	res := strings.TrimSpace(string(out))
	if res != name {
		changelog = true
	}
	return res, changelog
}
