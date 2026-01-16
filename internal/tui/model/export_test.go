package model

import (
	"context"
	"testing"

	"github.com/VoxDroid/krnr/internal/tui/adapters"
)

// recordingImportExport records call parameters for assertions in tests
type recordingImportExport struct{ lastName, lastDest string }

func (r *recordingImportExport) Export(_ context.Context, name string, dest string) error {
	r.lastName = name
	r.lastDest = dest
	return nil
}
func (r *recordingImportExport) ImportSet(_ context.Context, src string, policy string, dedupe bool) error {
	r.lastName = "<set>"
	r.lastDest = src
	return nil
}
func (r *recordingImportExport) ImportDB(_ context.Context, src string, overwrite bool) error {
	r.lastName = "<db>"
	r.lastDest = src
	return nil
}

func TestExportDatabaseCallsExporter(t *testing.T) {
	tt := &testRegistry{items: []adapters.CommandSetSummary{{Name: "one"}}}
	rec := &recordingImportExport{}
	m := New(tt, &testExecutor{}, rec, &testInstaller{})
	if err := m.Export(context.Background(), "", "./out.db"); err != nil {
		t.Fatalf("expected export success: %v", err)
	}
	if rec.lastName != "" {
		t.Fatalf("expected exporter called with empty name for DB export, got %q", rec.lastName)
	}
	if rec.lastDest == "" {
		t.Fatalf("expected exporter destination to be set, got empty")
	}
}
