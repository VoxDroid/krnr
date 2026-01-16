//go:build !windows
// +build !windows

package ui

import "syscall"

// setNonblock enables non-blocking mode on a file descriptor on Unix-like systems.
func setNonblock(fd uintptr) error {
	return syscall.SetNonblock(int(fd), true)
}

// keep reference to avoid staticcheck unused warning while function is kept for platform needs
var _ func(uintptr) error = setNonblock
