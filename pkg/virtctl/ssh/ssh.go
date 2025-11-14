/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package ssh

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
	"kubevirt.io/kubevirt/pkg/virtctl/portforward"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	portFlag, portFlagShort                         = "port", "p"
	usernameFlag, usernameFlagShort                 = "username", "l"
	IdentityFilePathFlag, identityFilePathFlagShort = "identity-file", "i"
	knownHostsFilePathFlag                          = "known-hosts"
	commandToExecute, commandToExecuteShort         = "command", "c"
	additionalOpts, additionalOptsShort             = "local-ssh-opts", "t"
)

type ssh struct {
	options *SSHOptions
	command string
}

type SSHOptions struct {
	SSHPort                   int
	SSHUsername               string
	IdentityFilePath          string
	IdentityFilePathProvided  bool
	KnownHostsFilePath        string
	KnownHostsFilePathDefault string
	AdditionalSSHLocalOptions []string
}

func NewSSH(opts *SSHOptions) *ssh {
	return &ssh{
		options: opts,
	}
}

func NewCommand() *cobra.Command {
	log.InitializeLogging("ssh")
	c := NewSSH(DefaultSSHOptions())

	cmd := &cobra.Command{
		Use:     "ssh (VM|VMI)",
		Short:   "Open a SSH connection to a virtual machine instance.",
		Example: usage(),
		Args:    cobra.ExactArgs(1),
		RunE:    c.run,
	}

	AddCommandlineArgs(cmd.Flags(), c.options)
	cmd.Flags().StringVarP(&c.command, commandToExecute, commandToExecuteShort, c.command,
		fmt.Sprintf(`--%s='ls /': Specify a command to execute in the VM`, commandToExecute))
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func AddCommandlineArgs(flagset *pflag.FlagSet, opts *SSHOptions) {
	flagset.StringVarP(&opts.SSHUsername, usernameFlag, usernameFlagShort, opts.SSHUsername,
		fmt.Sprintf("--%s=%s: Set this to the user you want to open the SSH connection as;"+
			"If unassigned, this will be empty and the SSH default will apply", usernameFlag, opts.SSHUsername))
	flagset.StringVarP(&opts.IdentityFilePath, IdentityFilePathFlag, identityFilePathFlagShort, opts.IdentityFilePath,
		fmt.Sprintf("--%s=/home/jdoe/.ssh/id_rsa: Set the path to a private key used for authenticating to the server;"+
			"If not provided, the client will try to use the local ssh-agent at $SSH_AUTH_SOCK", IdentityFilePathFlag))
	flagset.StringVar(&opts.KnownHostsFilePath, knownHostsFilePathFlag, opts.KnownHostsFilePathDefault,
		fmt.Sprintf("--%s=/home/jdoe/.ssh/kubevirt_known_hosts: Set the path to the known_hosts file.", knownHostsFilePathFlag))
	flagset.IntVarP(&opts.SSHPort, portFlag, portFlagShort, opts.SSHPort,
		fmt.Sprintf(`--%s=22: Specify a port on the VM to send SSH traffic to`, portFlag))
	flagset.StringArrayVarP(&opts.AdditionalSSHLocalOptions, additionalOpts, additionalOptsShort, opts.AdditionalSSHLocalOptions,
		fmt.Sprintf(`--%s="-o StrictHostKeyChecking=no" : Additional options to be passed to the local ssh client`, additionalOpts))
}

func DefaultSSHOptions() *SSHOptions {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Log.Warningf("failed to determine user home directory: %v", err)
	}

	options := &SSHOptions{
		SSHPort:                   22,
		SSHUsername:               DefaultUsername(),
		IdentityFilePath:          filepath.Join(homeDir, ".ssh", "id_rsa"),
		IdentityFilePathProvided:  false,
		KnownHostsFilePath:        "",
		KnownHostsFilePathDefault: "",
		AdditionalSSHLocalOptions: []string{},
	}

	if homeDir != "" {
		options.KnownHostsFilePathDefault = filepath.Join(homeDir, ".ssh", "kubevirt_known_hosts")
	}

	return options
}

func (o *ssh) run(cmd *cobra.Command, args []string) error {
	_, _, namespace, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return err
	}

	kind, namespace, name, err := prepareCommand(cmd, namespace, o.options, args)
	if err != nil {
		return err
	}

	clientArgs := o.BuildSSHTarget(kind, namespace, name)
	return LocalClientCmd("ssh", kind, namespace, name, o.options, clientArgs).Run()
}

func (o *ssh) BuildSSHTarget(kind, namespace, name string) []string {
	target := strings.Builder{}
	if o.options.SSHUsername != "" {
		target.WriteString(o.options.SSHUsername)
		target.WriteRune('@')
	}
	target.WriteString(kind)
	target.WriteString(".")
	target.WriteString(name)
	target.WriteString(".")
	target.WriteString(namespace)

	opts := []string{target.String()}
	if o.command != "" {
		opts = append(opts, o.command)
	}
	return opts
}

func prepareCommand(
	cmd *cobra.Command,
	fallbackNamespace string,
	opts *SSHOptions,
	args []string,
) (kind, namespace, name string, err error) {
	opts.IdentityFilePathProvided = cmd.Flags().Changed(IdentityFilePathFlag)

	targetUsername := ""
	kind, namespace, name, targetUsername, err = ParseTarget(args[0])
	if err != nil {
		return "", "", "", err
	}

	if namespace == "" {
		namespace = fallbackNamespace
	}

	if targetUsername != "" {
		opts.SSHUsername = targetUsername
	}

	return kind, namespace, name, nil
}

func usage() string {
	return fmt.Sprintf(`  # Connect to 'testvmi':
  {{ProgramName}} ssh jdoe@vmi/testvmi [--%s]

  # Connect to 'testvm' in 'mynamespace' namespace
  {{ProgramName}} ssh jdoe@vm/testvm/mynamespace [--%s]

  # Specify a username and namespace:
  {{ProgramName}} ssh --namespace=mynamespace --%s=jdoe vmi/testvmi`,
		IdentityFilePathFlag,
		IdentityFilePathFlag,
		usernameFlag,
	)
}

func DefaultUsername() string {
	vars := []string{
		"USER",     // linux
		"USERNAME", // linux, windows
		"LOGNAME",  // linux
	}
	for _, env := range vars {
		if v := os.Getenv(env); v != "" {
			return v
		}
	}

	return ""
}

// ParseTarget parse the SSH target argument supporting the form of [username@]type/name[/namespace]
func ParseTarget(arg string) (kind, namespace, name, username string, err error) {
	username = ""
	usernameAndTarget := strings.Split(arg, "@")
	if len(usernameAndTarget) > 1 {
		username = usernameAndTarget[0]
		if username == "" {
			return "", "", "", "", errors.New("expected username before '@'")
		}
		arg = usernameAndTarget[1]
	}

	if arg == "" {
		return "", "", "", "", errors.New("expected target after '@'")
	}

	kind, namespace, name, err = portforward.ParseTarget(arg)
	if err != nil {
		return "", "", "", "", err
	}

	return kind, namespace, name, username, err
}

func LocalClientCmd(command, kind, namespace, name string, options *SSHOptions, clientArgs []string) *exec.Cmd {
	args := []string{"-o", BuildProxyCommandOption(kind, namespace, name, options.SSHPort)}
	if len(options.AdditionalSSHLocalOptions) > 0 {
		args = append(args, options.AdditionalSSHLocalOptions...)
	}
	if options.IdentityFilePathProvided {
		args = append(args, "-i", options.IdentityFilePath)
	}
	args = append(args, clientArgs...)

	cmd := exec.Command(command, args...)
	const logLevel = 3
	log.Log.V(logLevel).Infof("running: %v", cmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd
}

func BuildProxyCommandOption(kind, namespace, name string, port int) string {
	proxyCommand := strings.Builder{}
	proxyCommand.WriteString("ProxyCommand=")
	proxyCommand.WriteString(os.Args[0])
	proxyCommand.WriteString(" port-forward --stdio=true ")
	proxyCommand.WriteString(kind)
	proxyCommand.WriteRune('/')
	proxyCommand.WriteString(name)
	proxyCommand.WriteRune('/')
	proxyCommand.WriteString(namespace)
	proxyCommand.WriteRune(' ')
	proxyCommand.WriteString(strconv.Itoa(port))
	return proxyCommand.String()
}
