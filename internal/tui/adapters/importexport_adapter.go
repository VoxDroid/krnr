package adapters

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/VoxDroid/krnr/internal/exporter"
	"github.com/VoxDroid/krnr/internal/importer"
)

// ImportExportAdapterImpl adapts exporter/importer package functions to the UI adapter.
type ImportExportAdapterImpl struct{ db *sql.DB }

func NewImportExportAdapter(db *sql.DB) *ImportExportAdapterImpl { return &ImportExportAdapterImpl{db: db} }

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

func (i *ImportExportAdapterImpl) Import(_ context.Context, src string, policy string) error {
	// Wire into internal/importer helpers; policy could be used to set options.
	// For now use ImportCommandSet with default options (no dedupe, rename on conflict).
	opts := importer.ImportOptions{Dedupe: false}
	return importer.ImportCommandSet(src, opts)
}