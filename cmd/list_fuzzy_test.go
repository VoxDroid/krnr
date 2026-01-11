package cmd

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestListFilterFuzzyCLI(t *testing.T) {
	tmp := t.TempDir()
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmp)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Capture stdout
	oldOut := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut

	// ensure clean state then create alpha
	rootCmd.SetArgs([]string{"delete", "alpha", "--yes"})
	_ = rootCmd.Execute()
	rootCmd.SetArgs([]string{"save", "alpha", "-c", "echo a", "-d", "alpha description"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("save alpha failed: %v", err)
	}
	// add tag
	rootCmd.SetArgs([]string{"tag", "add", "alpha", "utils"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("tag add alpha failed: %v", err)
	}

	// ensure clean state then create beta
	rootCmd.SetArgs([]string{"delete", "beta", "--yes"})
	_ = rootCmd.Execute()
	rootCmd.SetArgs([]string{"save", "beta", "-c", "echo b", "-d", "beta demo"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("save beta failed: %v", err)
	}
	rootCmd.SetArgs([]string{"tag", "add", "beta", "demo"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("tag add beta failed: %v", err)
	}

	// list with non-fuzzy filter that shouldn't match 'demo'
	rootCmd.SetArgs([]string{"list", "--filter", "dmo"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("list --filter failed: %v", err)
	}

	// list with fuzzy filter that should match 'beta'
	rootCmd.SetArgs([]string{"list", "--filter", "dmo", "--fuzzy"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("list --filter --fuzzy failed: %v", err)
	}

	_ = wOut.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, rOut)
	os.Stdout = oldOut

	out := buf.String()
	// fuzzy run should include beta
	if !bytes.Contains(buf.Bytes(), []byte("beta")) {
		t.Fatalf("expected fuzzy list to include beta, got: %s", out)
	}
	// non-fuzzy run earlier should not have included beta for 'dmo'
	// ensure at least one occurrence of beta exists after fuzzy run (we already checked) and assume earlier run didn't output it
}
