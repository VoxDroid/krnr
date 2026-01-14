package executor

import (
	"context"
	"io"
	"testing"
)

func TestExecuteRemovesNullByte(t *testing.T) {
	e := &Executor{DryRun: true}
	err := e.Execute(context.Background(), "echo hi\x00bad", "", io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("expected NUL to be removed and command to run in dry-run, got: %v", err)
	}
}

func TestExecuteRejectsNewline(t *testing.T) {
	e := &Executor{}
	err := e.Execute(context.Background(), "echo hi\nnext", "", io.Discard, io.Discard)
	if err == nil || err.Error() == "" {
		t.Fatalf("expected error for newline, got nil")
	}
}
