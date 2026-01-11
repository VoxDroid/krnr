package interactive

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// Prompt prompts the user and reads a single-line response from stdin.
func Prompt(msg string) string {
	return PromptReader(msg, os.Stdin)
}

// PromptReader prompts the user using the provided reader (useful for tests).
func PromptReader(msg string, r io.Reader) string {
	fmt.Printf("%s: ", msg)
	br := bufio.NewReader(r)
	line, _ := br.ReadString('\n')
	return strings.TrimSpace(line)
}