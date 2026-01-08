package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/VoxDroid/krnr/internal/user"
)

func TestWhoamiSetShowClear(t *testing.T) {
	// ensure clean
	_ = user.ClearProfile()

	var out bytes.Buffer
	whoamiSetCmd.SetOut(&out)
	if err := whoamiSetCmd.RunE(whoamiSetCmd, []string{}); err == nil {
		t.Fatalf("expected error when missing --name")
	}
	out.Reset()

	whoamiSetCmd.SetOut(&out)
	whoamiSetCmd.Flags().Set("name", "Bob")
	whoamiSetCmd.Flags().Set("email", "bob@example.com")
	if err := whoamiSetCmd.RunE(whoamiSetCmd, []string{}); err != nil {
		t.Fatalf("whoami set failed: %v", err)
	}

	whoamiShowCmd.SetOut(&out)
	if err := whoamiShowCmd.RunE(whoamiShowCmd, []string{}); err != nil {
		t.Fatalf("whoami show failed: %v", err)
	}
	if !strings.Contains(out.String(), "Bob <bob@example.com>") {
		t.Fatalf("unexpected show output: %s", out.String())
	}
	out.Reset()

	whoamiClearCmd.SetOut(&out)
	if err := whoamiClearCmd.RunE(whoamiClearCmd, []string{}); err != nil {
		t.Fatalf("whoami clear failed: %v", err)
	}
	if err := whoamiShowCmd.RunE(whoamiShowCmd, []string{}); err != nil {
		t.Fatalf("whoami show after clear failed: %v", err)
	}
}
