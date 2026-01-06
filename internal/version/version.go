package version

// Version is set at build time via -ldflags "-X github.com/VoxDroid/krnr/internal/version.Version=<value>"
// The default is a development placeholder.
var Version = "v0.1.0"
