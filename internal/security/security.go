// Package security provides security-related utilities.
package security

import (
	"errors"
	"regexp"
	"strings"
)

var dangerousPatterns = []*regexp.Regexp{
	// Destructive filesystem ops
	regexp.MustCompile(`(?i)\brm\s+-rf\s+/?$`),
	regexp.MustCompile(`(?i)\brm\s+-rf\s+/`),
	regexp.MustCompile(`(?i)\bmkfs\b`),
	regexp.MustCompile(`(?i)\bdd\s+if=`),
	// fork bombs (e.g. :(){ :|:& };:)
	regexp.MustCompile(`:\(\)\s*\{`),
	// package managers removing all packages
	regexp.MustCompile(`(?i)\bapt\-get\s+remove\s+`),
	regexp.MustCompile(`(?i)\byum\s+remove\s+`),
	// wipe disk
	regexp.MustCompile(`(?i)\bwipefs\b`),
	// dangerous use of sudo combined with other dangerous patterns
}

// CheckAllowed returns nil if the command is allowed to run, or an error
// describing why it's blocked. Checking is conservative and not exhaustive.
func CheckAllowed(command string) error {
	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return errors.New("empty command")
	}
	for _, re := range dangerousPatterns {
		if re.MatchString(cmd) {
			return errors.New("command appears destructive or unsafe")
		}
	}
	return nil
}
