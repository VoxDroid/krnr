// Package release contains helpers and integration tests related to the release
// process. It intentionally keeps implementation minimal and testable. The
// presence of a non-test file prevents the Go toolchain from reporting
// "no non-test Go files" when running builds that walk the module.
package release

// Placeholder to satisfy `go build` checks in CI.
func Placeholder() {}
