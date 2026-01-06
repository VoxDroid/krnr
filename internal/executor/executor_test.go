package executor

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func TestExecuteEcho(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var out bytes.Buffer
	var errb bytes.Buffer
	e := &Executor{}
	if err := e.Execute(ctx, "echo hello", "", &out, &errb); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !strings.Contains(out.String(), "hello") {
		t.Fatalf("expected 'hello' in stdout, got: %q", out.String())
	}
}

func TestExecuteFail(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var out bytes.Buffer
	var errb bytes.Buffer
	e := &Executor{}
	// 'exit 1' should return non-zero from shell
	if err := e.Execute(ctx, "exit 1", "", &out, &errb); err == nil {
		t.Fatalf("expected error for failing command")
	}
}

func TestDryRun(t *testing.T) {
	ctx := context.Background()
	var out bytes.Buffer
	var errb bytes.Buffer
	e := &Executor{DryRun: true, Verbose: true}
	if err := e.Execute(ctx, "echo hi", "", &out, &errb); err != nil {
		t.Fatalf("dry-run should not error: %v", err)
	}
	if !strings.Contains(out.String(), "dry-run:") {
		t.Fatalf("expected dry-run message, got: %q", out.String())
	}
}
