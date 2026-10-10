package main

import "golang.org/x/sys/unix"

func terminalState(fd int) (*unix.Termios, error) { return unix.IoctlGetTermios(fd, unix.TIOCGETA) }
func setTerminalState(fd int, state *unix.Termios) error {
	return unix.IoctlSetTermios(fd, unix.TIOCSETA, state)
}

func restoreTerminalState(fd int, state *unix.Termios) error {
	return unix.IoctlSetTermios(fd, unix.TIOCSETAF, state)
}
