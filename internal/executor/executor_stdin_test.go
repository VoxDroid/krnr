package executor

import (
	"bytes"
	"context"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

func TestExecute_WithStdin(t *testing.T) {
	ctx := context.Background()
	e := &Executor{}
	// use sh for cross-unix simple read test; skip on Windows if sh missing
	if runtime.GOOS == "windows" {
		if p, err := exec.LookPath("sh"); err == nil {
			e.Shell = p
		} else {
			t.Skip("sh not available, skipping stdin test")
		}
	}

	var out bytes.Buffer
	var errb bytes.Buffer
	stdin := bytes.NewBufferString("s3cret\n")
	cmd := "read -r line; echo got:$line"
	if err := e.Execute(ctx, cmd, "", stdin, &out, &errb); err != nil {
		t.Fatalf("Execute with stdin failed: %v stderr=%q", err, errb.String())
	}
	if !strings.Contains(out.String(), "got:s3cret") {
		t.Fatalf("expected got:s3cret in stdout, got: %q", out.String())
	}
}
