package ssh

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"

	"github.com/golang/glog"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/term"
)

func (o *SSH) prepareSSHClient(kind, namespace, name string, options *SSHOptions) (*ssh.Client, error) {
	streamer, err := o.prepareSSHTunnel(kind, namespace, name, options)
	if err != nil {
		return nil, err
	}

	conn := streamer.AsConn()
	addr := fmt.Sprintf("%s/%s.%s:%d", kind, name, namespace, options.SshPort)
	authMethods := o.getAuthMethods(kind, namespace, name, options)

	hostKeyCallback := ssh.InsecureIgnoreHostKey()
	if len(options.KnownHostsFilePath) > 0 {
		hostKeyCallback, err = InteractiveHostKeyCallback(options.KnownHostsFilePath)
		if err != nil {
			return nil, err
		}
	} else {
		fmt.Println("WARNING: skipping hostkey check, provide --known-hosts to fix this")
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(conn,
		addr,
		&ssh.ClientConfig{
			HostKeyCallback: hostKeyCallback,
			Auth:            authMethods,
			User:            options.SshUsername,
		},
	)
	if err != nil {
		return nil, err
	}

	return ssh.NewClient(sshConn, chans, reqs), nil
}

func (o *SSH) getAuthMethods(kind, namespace, name string, options *SSHOptions) []ssh.AuthMethod {
	var methods []ssh.AuthMethod

	methods = trySSHAgent(methods)
	methods = tryPrivateKey(methods, options)

	methods = append(methods, ssh.PasswordCallback(func() (secret string, err error) {
		password, err := readPassword(fmt.Sprintf("%s@%s/%s.%s's password: ", options.SshUsername, kind, name, namespace))
		fmt.Println()
		return string(password), err
	}))

	return methods
}

func trySSHAgent(methods []ssh.AuthMethod) []ssh.AuthMethod {
	socket := os.Getenv("SSH_AUTH_SOCK")
	if len(socket) < 1 {
		return methods
	}
	conn, err := net.Dial("unix", socket)
	if err != nil {
		glog.Error("no connection to ssh agent, skipping agent authentication:", err)
		return methods
	}
	agentClient := agent.NewClient(conn)

	return append(methods, ssh.PublicKeysCallback(agentClient.Signers))
}

func tryPrivateKey(methods []ssh.AuthMethod, options *SSHOptions) []ssh.AuthMethod {

	// If the identity file at the default does not exist but was
	// not explicitly provided, don't add the authentication mechanism.
	if !options.IdentityFilePathProvided {
		if _, err := os.Stat(options.IdentityFilePath); os.IsNotExist(err) {
			glog.V(3).Infof("No ssh key at the default location %q found, skipping RSA authentication.", options.IdentityFilePath)
			return methods
		}
	}

	callback := ssh.PublicKeysCallback(func() (signers []ssh.Signer, err error) {
		key, err := ioutil.ReadFile(options.IdentityFilePath)
		if err != nil {
			return nil, err
		}

		signer, err := ssh.ParsePrivateKey(key)
		if _, isPassErr := err.(*ssh.PassphraseMissingError); isPassErr {
			signer, err = parsePrivateKeyWithPassphrase(key, options)
			if err != nil {
				return nil, err
			}
		}
		return []ssh.Signer{signer}, nil
	})

	return append(methods, callback)
}

func parsePrivateKeyWithPassphrase(key []byte, options *SSHOptions) (ssh.Signer, error) {
	password, err := readPassword(fmt.Sprintf("Key %s requires a password: ", options.IdentityFilePath))
	fmt.Println()
	if err != nil {
		return nil, err
	}

	return ssh.ParsePrivateKeyWithPassphrase(key, password)
}

func readPassword(reason string) ([]byte, error) {
	fmt.Print(reason)
	return term.ReadPassword(int(os.Stdin.Fd()))
}

func (o *SSH) startSession(client *ssh.Client) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	session.Stdin = os.Stdin
	session.Stderr = os.Stderr
	session.Stdout = os.Stdout

	restore, err := setupTerminal(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	defer restore()

	if err := requestPty(session); err != nil {
		return err
	}

	if err := session.Shell(); err != nil {
		return err
	}

	err = session.Wait()
	if _, exited := err.(*ssh.ExitError); !exited {
		return err
	}
	return nil
}

func setupTerminal(fd int) (func(), error) {
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
