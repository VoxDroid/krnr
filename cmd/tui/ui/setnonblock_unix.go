//go:build !windows
// +build !windows

package ui

import "syscall"

// setNonblock enables non-blocking mode on a file descriptor on Unix-like systems.
func setNonblock(fd uintptr) error {
	return syscall.SetNonblock(int(fd), true)
}
