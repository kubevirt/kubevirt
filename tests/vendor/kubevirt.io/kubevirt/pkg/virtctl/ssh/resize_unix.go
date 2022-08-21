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
