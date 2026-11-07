//go:build linux

package tty

import "golang.org/x/sys/unix"

func isTerminal(fd uintptr) bool {
	_, err := unix.IoctlGetTermios(int(fd), unix.TCGETS)

	return err == nil
}
