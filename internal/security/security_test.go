package security

import "testing"

func TestCheckAllowed(t *testing.T) {
	bad := []string{
		"rm -rf /",
		"rm -rf / --no-preserve-root",
		"mkfs.ext4 /dev/sda",
		"dd if=/dev/zero of=/dev/sda bs=4096",
		":(){ :|:& };:",
		"wipefs -a /dev/sda",
	}
	for _, s := range bad {
		if err := CheckAllowed(s); err == nil {
			t.Fatalf("expected %q to be blocked", s)
		}
	}

	good := []string{
		"echo hello",
		"ls -la",
		"bash -c 'echo safe'",
	}
	for _, s := range good {
		if err := CheckAllowed(s); err != nil {
			t.Fatalf("expected %q to be allowed: %v", s, err)
		}
	}
}
