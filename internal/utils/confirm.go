// Package interactive provides utility functions for interactive prompts and editors.
package interactive

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// Confirm prompts the user with msg and expects y/n on stdin. Returns true for yes.
// For non-interactive environments (stdin not a terminal) it returns false.
func Confirm(msg string) bool {
	return ConfirmReader(msg, os.Stdin)
}

// ConfirmReader prompts the user with msg and reads the response from r. This is
// useful for testing where input can be provided via a buffer instead of the
// real stdin.
func ConfirmReader(msg string, r io.Reader) bool {
	fmt.Printf("%s [y/N]: ", msg)
	br := bufio.NewReader(r)
	line, _ := br.ReadString('\n')
	resp := strings.TrimSpace(strings.ToLower(line))
	return resp == "y" || resp == "yes"
}
