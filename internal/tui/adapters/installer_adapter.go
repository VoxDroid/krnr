package adapters

import (
	"context"

	"github.com/VoxDroid/krnr/internal/install"
)

// InstallerAdapterImpl delegates install/uninstall operations to internal/install.
type InstallerAdapterImpl struct{}

// NewInstallerAdapter returns a new InstallerAdapterImpl.
func NewInstallerAdapter() *InstallerAdapterImpl { return &InstallerAdapterImpl{} }

// Install executes the install with given options and returns performed actions.
func (i *InstallerAdapterImpl) Install(_ context.Context, opts install.Options) ([]string, error) {
	return install.ExecuteInstall(opts)
}

// Uninstall performs uninstall and returns the performed actions.
func (i *InstallerAdapterImpl) Uninstall(_ context.Context) ([]string, error) {
	// CLI uninstall takes a verbose flag; choose false for non-verbose UI call.
	return install.Uninstall(false)
}
