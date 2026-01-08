// Package utils provides utility functions.
package utils

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Confirm prompts the user with msg and expects y/n on stdin. Returns true for yes.
// For non-interactive environments (stdin not a terminal) it returns false.
func Confirm(msg string) bool {
	fmt.Printf("%s [y/N]: ", msg)
	r := bufio.NewReader(os.Stdin)
	line, _ := r.ReadString('\n')
	resp := strings.TrimSpace(strings.ToLower(line))
	return resp == "y" || resp == "yes"
}
