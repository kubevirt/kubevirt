//go:build !excludenative

package ssh

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// InteractiveHostKeyCallback verifying the host key against known_hosts and adding the key if
// the user replies accordingly.
func InteractiveHostKeyCallback(knownHostsFilePath string) (ssh.HostKeyCallback, error) {
	if _, err := os.Stat(knownHostsFilePath); os.IsNotExist(err) {
		f, err := os.Create(knownHostsFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed creating known hosts file %q: %v", knownHostsFilePath, err)
		}
		_ = f.Close()
	} else if err != nil {
		return nil, fmt.Errorf("failed reading known host file %q: %v", knownHostsFilePath, err)
	}
	validator, err := knownhosts.New(knownHostsFilePath)
	if err != nil {
		return nil, err
	}

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := validator(hostname, remote, key)
		if err == nil {
			return nil
		}

		var keyErr *knownhosts.KeyError
		if errors.As(err, &keyErr) && len(keyErr.Want) == 0 {
			shouldAdd, err := askToAddHostKey(hostname, remote, key)
			if err != nil || !shouldAdd {
				return err
			}
			if err := addHostKey(knownHostsFilePath, hostname, key); err != nil {
				return err
			}
			return nil
		}

		return err
	}, nil
}

func askToAddHostKey(hostname string, remote net.Addr, key ssh.PublicKey) (bool, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf(
		`The authenticity of host '%s (%s)' can't be established.
ECDSA key fingerprint is %s.
Are you sure you want to continue connecting (yes/no)? `,
		hostname, remote, ssh.FingerprintSHA256(key),
	)
	confirmation, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	confirmation = strings.TrimSpace(confirmation)

	if confirmation == "yes" {
		return true, nil
	}
	if confirmation == "no" {
		return false, nil
	}

	fmt.Println("Please reply with either yes or no.")
	return askToAddHostKey(hostname, remote, key)
}

func addHostKey(knownHostsFilePath, hostname string, key ssh.PublicKey) error {
	f, err := os.OpenFile(knownHostsFilePath, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	addresses := []string{hostname}
	_, err = fmt.Fprintln(f, knownhosts.Line(addresses, key))
	return err
}
