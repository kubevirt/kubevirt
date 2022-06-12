package scp

import (
	"time"

	"golang.org/x/crypto/ssh"
)

// defaultConnTimeout is the default timeout for establishing a TCP connection to server
const defaultConnTimeout = 3 * time.Second

// NewSSHConfigFromPassword returns a *ssh.ClientConfig with ssh.Password AuthMethod
// and 3 seconds timeout for connecting the server.
//
// It *insecurely* ignores server's host key validation.
func NewSSHConfigFromPassword(username, password string) *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         defaultConnTimeout,
	}
}

// NewSSHConfigFromPrivateKey returns a *ssh.ClientConfig with ssh.PublicKey AuthMethod
// and 3 seconds timeout for connecting the server.
//
// The passphrase is optional.
// If multiple passphrase are provided, only the first will be used.
//
// If the private key is encrypted, it will return a ssh.PassphraseMissingError.
//
// It *insecurely* ignores server's host key validation.
func NewSSHConfigFromPrivateKey(username string, privPEM []byte, passphrase ...string) (cfg *ssh.ClientConfig, err error) {
	var priv ssh.Signer
	if len(passphrase) > 0 && len(passphrase[0]) > 0 {
		pw := passphrase[0]
		priv, err = ssh.ParsePrivateKeyWithPassphrase(privPEM, []byte(pw))
	} else {
		priv, err = ssh.ParsePrivateKey(privPEM)
	}
	if err != nil {
		return
	}

	cfg = &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(priv),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         defaultConnTimeout,
	}
	return
}
