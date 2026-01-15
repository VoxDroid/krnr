//go:build windows
// +build windows

package ui

import "syscall"

// setNonblock enables non-blocking mode on a handle on Windows.
//
//nolint:unused // used conditionally by tests on Windows when enabling PTY non-blocking
func setNonblock(fd uintptr) error {
	return syscall.SetNonblock(syscall.Handle(fd), true)
}
