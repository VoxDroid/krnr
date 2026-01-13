package adapters

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/VoxDroid/krnr/internal/exporter"
	"github.com/VoxDroid/krnr/internal/importer"
)

// ImportExportAdapterImpl adapts exporter/importer package functions to the UI adapter.
// ImportExportAdapterImpl adapts exporter/importer package functions to the UI adapter.
type ImportExportAdapterImpl struct{ db *sql.DB }

// NewImportExportAdapter constructs a new ImportExportAdapter backed by db.
func NewImportExportAdapter(db *sql.DB) *ImportExportAdapterImpl {
	return &ImportExportAdapterImpl{db: db}
}

// Export exports either a single command set (when name is non-empty) or the
// entire database to dest.
func (i *ImportExportAdapterImpl) Export(_ context.Context, name string, dest string) error {
	if name == "" {
		// export entire database
		return exporter.ExportDatabase(dest)
	}
	if i.db == nil {
		return fmt.Errorf("no database connection for exporting command set")
	}
	return exporter.ExportCommandSet(i.db, name, dest)
}

// Import imports a command set file from src using the given policy (currently unused).
func (i *ImportExportAdapterImpl) Import(_ context.Context, src string, _ string) error {
	// Wire into internal/importer helpers; policy could be used to set options.
	// For now use ImportCommandSet with default options (no dedupe, rename on conflict).
	opts := importer.ImportOptions{Dedupe: false}
	return importer.ImportCommandSet(src, opts)
}
