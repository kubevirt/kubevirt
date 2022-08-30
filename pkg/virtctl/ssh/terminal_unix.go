//go:build !windows && !excludenative

package ssh

import (
	"encoding/binary"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

func setupTerminal() (func(), error) {
	fd := int(os.Stdin.Fd())

	state, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}

	return func() { term.Restore(fd, state) }, nil
}

func requestPty(session *ssh.Session) error {
	w, h, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}

	if err := session.RequestPty(
		os.Getenv("TERM"),
		h, w,
		ssh.TerminalModes{},
	); err != nil {
		return err
	}

	go resizeSessionOnWindowChange(session, os.Stdin.Fd())

	return nil
}

// resizeSessionOnWindowChange watches for SIGWINCH and refreshes the session with the new window size
func resizeSessionOnWindowChange(session *ssh.Session, _ uintptr) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGWINCH)

	for range sigs {
		session.SendRequest("window-change", false, windowSizePayloadFor())
	}
}

func windowSizePayloadFor() []byte {
	w, h, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return buildWindowSizePayload(80, 24)
	}

	return buildWindowSizePayload(w, h)
}

func buildWindowSizePayload(width, height int) []byte {
	size := make([]byte, 16)
	binary.BigEndian.PutUint32(size, uint32(width))
	binary.BigEndian.PutUint32(size[4:], uint32(height))
	return size
}
