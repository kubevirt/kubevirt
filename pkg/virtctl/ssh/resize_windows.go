// +build windows

package ssh

import (
	"golang.org/x/crypto/ssh"
)

// resizeSessionOnWindowChange does nothing on window
func resizeSessionOnWindowChange(session *ssh.Session, fd uintptr) {
	return
}
