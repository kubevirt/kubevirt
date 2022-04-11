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

	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/client-go/kubecli"
)

type NativeSSHConnection struct {
	ClientConfig clientcmd.ClientConfig
	Options      SSHOptions
}

func (o *NativeSSHConnection) PrepareSSHClient(kind, namespace, name string) (*ssh.Client, error) {
	streamer, err := o.prepareSSHTunnel(kind, namespace, name)
	if err != nil {
		return nil, err
	}

	conn := streamer.AsConn()
	addr := fmt.Sprintf("%s/%s.%s:%d", kind, name, namespace, o.Options.SshPort)
	authMethods := o.getAuthMethods(kind, namespace, name)

	hostKeyCallback := ssh.InsecureIgnoreHostKey()
	if len(o.Options.KnownHostsFilePath) > 0 {
		hostKeyCallback, err = InteractiveHostKeyCallback(o.Options.KnownHostsFilePath)
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
			User:            o.Options.SshUsername,
		},
	)
	if err != nil {
		return nil, err
	}

	return ssh.NewClient(sshConn, chans, reqs), nil
}

func (o *NativeSSHConnection) getAuthMethods(kind, namespace, name string) []ssh.AuthMethod {
	var methods []ssh.AuthMethod

	methods = o.trySSHAgent(methods)
	methods = o.tryPrivateKey(methods)

	methods = append(methods, ssh.PasswordCallback(func() (secret string, err error) {
		password, err := readPassword(fmt.Sprintf("%s@%s/%s.%s's password: ", o.Options.SshUsername, kind, name, namespace))
		fmt.Println()
		return string(password), err
	}))

	return methods
}

func (o *NativeSSHConnection) trySSHAgent(methods []ssh.AuthMethod) []ssh.AuthMethod {
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

func (o *NativeSSHConnection) tryPrivateKey(methods []ssh.AuthMethod) []ssh.AuthMethod {

	// If the identity file at the default does not exist but was
	// not explicitly provided, don't add the authentication mechanism.
	if !o.Options.IdentityFilePathProvided {
		if _, err := os.Stat(o.Options.IdentityFilePath); os.IsNotExist(err) {
			glog.V(3).Infof("No ssh key at the default location %q found, skipping RSA authentication.", o.Options.IdentityFilePath)
			return methods
		}
	}

	callback := ssh.PublicKeysCallback(func() (signers []ssh.Signer, err error) {
		key, err := ioutil.ReadFile(o.Options.IdentityFilePath)
		if err != nil {
			return nil, err
		}

		signer, err := ssh.ParsePrivateKey(key)
		if _, isPassErr := err.(*ssh.PassphraseMissingError); isPassErr {
			signer, err = o.parsePrivateKeyWithPassphrase(key)
		}
		if err != nil {
			return nil, err
		}

		return []ssh.Signer{signer}, nil
	})

	return append(methods, callback)
}

func (o *NativeSSHConnection) parsePrivateKeyWithPassphrase(key []byte) (ssh.Signer, error) {
	password, err := readPassword(fmt.Sprintf("Key %s requires a password: ", o.Options.IdentityFilePath))
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

func (o *NativeSSHConnection) StartSession(client *ssh.Client) error {
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

func (o *NativeSSHConnection) prepareSSHTunnel(kind, namespace, name string) (kubecli.StreamInterface, error) {
	virtCli, err := kubecli.GetKubevirtClientFromClientConfig(o.ClientConfig)
	if err != nil {
		return nil, err
	}

	var stream kubecli.StreamInterface
	if kind == "vmi" {
		stream, err = virtCli.VirtualMachineInstance(namespace).PortForward(name, o.Options.SshPort, "tcp")
		if err != nil {
			return nil, fmt.Errorf("can't access VMI %s: %w", name, err)
		}
	} else if kind == "vm" {
		stream, err = virtCli.VirtualMachine(namespace).PortForward(name, o.Options.SshPort, "tcp")
		if err != nil {
			return nil, fmt.Errorf("can't access VM %s: %w", name, err)
		}
	}

	return stream, nil
}
