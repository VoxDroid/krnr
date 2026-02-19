//go:build linux || android

package executor

import "golang.org/x/sys/unix"

// setEcho toggles the ECHO bit on the terminal referenced by fd (Linux).
func setEcho(fd int, enabled bool) error {
	t, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return err
	}
	if enabled {
		t.Lflag |= unix.ECHO
	} else {
		t.Lflag &^= unix.ECHO
	}
	return unix.IoctlSetTermios(fd, unix.TCSETS, t)
}
