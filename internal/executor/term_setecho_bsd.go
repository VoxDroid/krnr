//go:build darwin || freebsd || netbsd || openbsd || dragonfly

package executor

import "golang.org/x/sys/unix"

// setEcho toggles the ECHO bit on the terminal referenced by fd (BSD / macOS).
func setEcho(fd int, enabled bool) error {
	t, err := unix.IoctlGetTermios(fd, unix.TIOCGETA)
	if err != nil {
		return err
	}
	if enabled {
		t.Lflag |= unix.ECHO
	} else {
		t.Lflag &^= unix.ECHO
	}
	return unix.IoctlSetTermios(fd, unix.TIOCSETA, t)
}
