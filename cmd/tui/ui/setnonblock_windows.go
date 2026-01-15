//go:build windows
// +build windows

package ui

import "syscall"

// setNonblock enables non-blocking mode on a handle on Windows.
func setNonblock(fd uintptr) error {
	return syscall.SetNonblock(syscall.Handle(fd), true)
}
