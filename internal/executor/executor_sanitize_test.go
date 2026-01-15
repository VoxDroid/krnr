package executor

import (
	"context"
	"io"
	"testing"
)

func TestExecuteSanitizesSmartQuotes(t *testing.T) {
	e := &Executor{DryRun: true}
	err := e.Execute(context.Background(), "echo \u201CHello\u201D", "", io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("expected sanitized command to run in dry-run, got error: %v", err)
	}
}
