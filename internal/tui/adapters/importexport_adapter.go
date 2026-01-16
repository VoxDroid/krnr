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

// ImportSet imports a command set file from src using the given policy and
// dedupe options.
func (i *ImportExportAdapterImpl) ImportSet(_ context.Context, src string, policy string, dedupe bool) error {
	opts := importer.ImportOptions{OnConflict: policy, Dedupe: dedupe}
	return importer.ImportCommandSet(src, opts)
}

// ImportDB imports a database file into the active DB, overwriting if requested.
func (i *ImportExportAdapterImpl) ImportDB(_ context.Context, src string, overwrite bool) error {
	return importer.ImportDatabase(src, overwrite, importer.ImportOptions{})
}
