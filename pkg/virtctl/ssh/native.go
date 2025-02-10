//go:build !excludenative

package ssh

import (
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/spf13/pflag"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/term"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
)

const (
	wrapLocalSSHDefault = false
)

func additionalUsage() string {
	return fmt.Sprintf(`
	# Connect to 'testvmi' using the local ssh binary found in $PATH:
	{{ProgramName}} ssh --%s=true jdoe@testvmi`,
		wrapLocalSSHFlag,
	)
}

func addAdditionalCommandlineArgs(flagset *pflag.FlagSet, opts *SSHOptions) {
	flagset.StringArrayVarP(&opts.AdditionalSSHLocalOptions, additionalOpts, additionalOptsShort, opts.AdditionalSSHLocalOptions,
		fmt.Sprintf(`--%s="-o StrictHostKeyChecking=no" : Additional options to be passed to the local ssh. This is applied only if local-ssh=true`, additionalOpts))
	flagset.BoolVar(&opts.WrapLocalSSH, wrapLocalSSHFlag, opts.WrapLocalSSH,
		fmt.Sprintf("--%s=true: Set this to true to use the SSH/SCP client available on your system by using this command as ProxyCommand; If set to false, this will establish a SSH/SCP connection with limited capabilities provided by this client", wrapLocalSSHFlag))
}

type NativeSSHConnection struct {
	Client  kubecli.KubevirtClient
	Options SSHOptions
}

func (o *SSH) nativeSSH(namespace, name string, client kubecli.KubevirtClient) error {
	conn := NativeSSHConnection{
		Client:  client,
		Options: o.options,
	}
	sshClient, err := conn.PrepareSSHClient(namespace, name)
	if err != nil {
		return err
	}
	return conn.StartSession(sshClient, o.command)
}

func (o *NativeSSHConnection) PrepareSSHClient(namespace, name string) (*ssh.Client, error) {
	streamer, err := o.Client.VirtualMachineInstance(namespace).PortForward(name, o.Options.SSHPort, "tcp")
	if err != nil {
		return nil, fmt.Errorf("can't access VMI %s: %w", name, err)
	}

	conn := streamer.AsConn()
	addr := fmt.Sprintf("%s/%s:%d", namespace, name, o.Options.SSHPort)
	authMethods := o.getAuthMethods(namespace, name)

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
			User:            o.Options.SSHUsername,
		},
	)
	if err != nil {
		return nil, err
	}

	return ssh.NewClient(sshConn, chans, reqs), nil
}

func (o *NativeSSHConnection) getAuthMethods(namespace, name string) []ssh.AuthMethod {
	var methods []ssh.AuthMethod

	methods = o.trySSHAgent(methods)
	methods = o.tryPrivateKey(methods)

	methods = append(methods, ssh.PasswordCallback(func() (secret string, err error) {
		password, err := readPassword(fmt.Sprintf("%s@%s/%s's password: ", o.Options.SSHUsername, namespace, name))
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
		log.Log.Errorf("no connection to ssh agent, skipping agent authentication: %v", err)
		return methods
	}
	agentClient := agent.NewClient(conn)

	return append(methods, ssh.PublicKeysCallback(agentClient.Signers))
}

func (o *NativeSSHConnection) tryPrivateKey(methods []ssh.AuthMethod) []ssh.AuthMethod {
	// If the identity file at the default does not exist but was
	// not explicitly provided, don't add the authentication mechanism.
	if !o.Options.IdentityFilePathProvided {
		if _, err := os.Stat(o.Options.IdentityFilePath); errors.Is(err, os.ErrNotExist) {
			log.Log.V(3).Infof("No ssh key at the default location %q found, skipping RSA authentication.", o.Options.IdentityFilePath)
			return methods
		}
	}

	callback := ssh.PublicKeysCallback(func() (signers []ssh.Signer, err error) {
		key, err := os.ReadFile(o.Options.IdentityFilePath)
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

func (o *NativeSSHConnection) StartSession(client *ssh.Client, command string) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	session.Stdin = os.Stdin
	session.Stderr = os.Stderr
	session.Stdout = os.Stdout

	if command != "" {
		if err := session.Run(command); err != nil {
			return err
		}
		return nil
	}

	restore, err := setupTerminal()
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
