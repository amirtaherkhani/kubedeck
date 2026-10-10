package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"time"
)

func readHiddenBrowserToken(ctx context.Context, in *os.File, out io.Writer) (string, error) {
	old, e := terminalState(int(in.Fd()))
	if e != nil {
		return "", errors.New("interactive_terminal_required")
	}
	hidden := *old
	hidden.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON
	hidden.Cc[unix.VMIN] = 1
	hidden.Cc[unix.VTIME] = 0
	if setTerminalState(int(in.Fd()), &hidden) != nil {
		return "", errors.New("hidden_input_unavailable")
	}
	defer restoreTerminalState(int(in.Fd()), old)
	if _, e = fmt.Fprint(out, "Paste the official browser Copy-to-clipboard value here, then press Enter (input hidden): "); e != nil {
		return "", errors.New("hidden_input_unavailable")
	}
	defer fmt.Fprintln(out)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	return readHiddenLine(ctx, bufio.NewReaderSize(terminalInput{ctx: ctx, fd: int(in.Fd())}, 4096))
}

// Poll in this goroutine so cancellation leaves no reader consuming future shell
// input. Restoration flushes queued paste bytes before echo is re-enabled.
type terminalInput struct {
	ctx context.Context
	fd  int
}

func (r terminalInput) Read(b []byte) (int, error) {
	for r.ctx.Err() == nil {
		if r.fd >= unix.FD_SETSIZE {
			return 0, errors.New("hidden_input_unavailable")
		}
		var fds unix.FdSet
		fds.Set(r.fd)
		timeout := unix.NsecToTimeval((100 * time.Millisecond).Nanoseconds())
		n, e := unix.Select(r.fd+1, &fds, nil, nil, &timeout)
		if e == unix.EINTR {
			continue
		}
		if e != nil {
			return 0, e
		}
		if n > 0 {
			return unix.Read(r.fd, b)
		}
	}
	return 0, r.ctx.Err()
}
func readHiddenLine(ctx context.Context, in io.Reader) (string, error) {
	b := make([]byte, 0, 2048)
	one := make([]byte, 1)
	for len(b) <= 64<<10 {
		if ctx.Err() != nil {
			return "", errors.New("hidden_input_cancelled")
		}
		n, e := in.Read(one)
		if e != nil || n != 1 {
			return "", errors.New("hidden_input_unavailable")
		}
		switch one[0] {
		case '\r', '\n':
			return string(b), nil
		case 8, 127:
			if len(b) > 0 {
				b = b[:len(b)-1]
			}
		case 4:
			return "", errors.New("hidden_input_cancelled")
		default:
			c := one[0]
			if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '+' || c == '/' || c == '=') {
				return "", errors.New("invalid_browser_fallback")
			}
			b = append(b, c)
		}
	}
	return "", errors.New("invalid_browser_fallback")
}
