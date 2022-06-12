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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package ssh

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	portFlag, portFlagShort                         = "port", "p"
	wrapLocalSSHFlag                                = "local-ssh"
	usernameFlag, usernameFlagShort                 = "username", "l"
	IdentityFilePathFlag, identityFilePathFlagShort = "identity-file", "i"
	knownHostsFilePathFlag                          = "known-hosts"
)

func NewCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {

	c := &SSH{
		clientConfig: clientConfig,
		options:      DefaultSSHOptions(),
		WrapLocalSSH: false,
	}

	cmd := &cobra.Command{
		Use:     "ssh (VM|VMI)",
		Short:   "Open a SSH connection to a virtual machine instance.",
		Example: usage(),
		Args:    templates.ExactArgs("ssh", 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.Run(cmd, args)
		},
	}

	AddCommandlineArgs(cmd.Flags(), &c.options)
	cmd.Flags().BoolVar(&c.WrapLocalSSH, wrapLocalSSHFlag, c.WrapLocalSSH,
		fmt.Sprintf("--%s=true: Set this to true to use the SSH command available on your system by using this command as ProxyCommand; If unassigned, this will establish a SSH connection with limited capabilities provided by this client", wrapLocalSSHFlag))
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func AddCommandlineArgs(flagset *pflag.FlagSet, opts *SSHOptions) {
	flagset.StringVarP(&opts.SshUsername, usernameFlag, usernameFlagShort, opts.SshUsername,
		fmt.Sprintf("--%s=%s: Set this to the user you want to open the SSH connection as; If unassigned, this will be empty and the SSH default will apply", usernameFlag, opts.SshUsername))
	flagset.StringVarP(&opts.IdentityFilePath, IdentityFilePathFlag, identityFilePathFlagShort, opts.IdentityFilePath,
		fmt.Sprintf("--%s=/home/jdoe/.ssh/id_rsa: Set the path to a private key used for authenticating to the server; If not provided, the client will try to use the local ssh-agent at $SSH_AUTH_SOCK", IdentityFilePathFlag))
	flagset.StringVar(&opts.KnownHostsFilePath, knownHostsFilePathFlag, opts.KnownHostsFilePathDefault,
		fmt.Sprintf("--%s=/home/jdoe/.ssh/kubevirt_known_hosts: Set the path to the known_hosts file.", knownHostsFilePathFlag))
	flagset.IntVarP(&opts.SshPort, portFlag, portFlagShort, opts.SshPort,
		fmt.Sprintf(`--%s=22: Specify a port on the VM to send SSH traffic to`, portFlag))
}

func DefaultSSHOptions() SSHOptions {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		glog.Warningf("failed to determine user home directory: %v", err)
	}
	options := SSHOptions{
		SshPort:                   22,
		SshUsername:               defaultUsername(),
		IdentityFilePath:          filepath.Join(homeDir, ".ssh", "id_rsa"),
		IdentityFilePathProvided:  false,
		KnownHostsFilePath:        "",
		KnownHostsFilePathDefault: "",
	}

	if len(homeDir) > 0 {
		options.KnownHostsFilePathDefault = filepath.Join(homeDir, ".ssh", "kubevirt_known_hosts")
	}
	return options
}

type SSH struct {
	clientConfig clientcmd.ClientConfig
	options      SSHOptions
	WrapLocalSSH bool
}

type SSHOptions struct {
	SshPort                   int
	SshUsername               string
	IdentityFilePath          string
	IdentityFilePathProvided  bool
	KnownHostsFilePath        string
	KnownHostsFilePathDefault string
}

func (o *SSH) Run(cmd *cobra.Command, args []string) error {
	kind, namespace, name, err := PrepareCommand(cmd, o.clientConfig, &o.options, args)
	if err != nil {
		return err
	}

	if o.WrapLocalSSH {
		return o.runLocalCommandClient(kind, namespace, name)
	}

	ssh := NativeSSHConnection{
		ClientConfig: o.clientConfig,
		Options:      o.options,
	}
	client, err := ssh.PrepareSSHClient(kind, namespace, name)
	if err != nil {
		return err
	}
	return ssh.StartSession(client)
}

func PrepareCommand(cmd *cobra.Command, clientConfig clientcmd.ClientConfig, opts *SSHOptions, args []string) (kind, namespace, name string, err error) {
	opts.IdentityFilePathProvided = cmd.Flags().Changed(IdentityFilePathFlag)
	var targetUsername string
	kind, namespace, name, targetUsername, err = templates.ParseSSHTarget(args[0])
	if err != nil {
		return
	}

	if len(namespace) < 1 {
		namespace, _, err = clientConfig.Namespace()
		if err != nil {
			return
		}
	}

	if len(targetUsername) > 0 {
		opts.SshUsername = targetUsername
	}
	return
}

func usage() string {
	return fmt.Sprintf(`  # Connect to 'testvmi' (using the built-in SSH client):
  {{ProgramName}} ssh jdoe@testvmi [--%s]

  # Connect to 'testvm' in 'mynamespace' namespace
  {{ProgramName}} ssh jdoe@vm/testvm.mynamespace [--%s]

  # Specify a username and namespace:
  {{ProgramName}} ssh --namespace=mynamespace --%s=jdoe testvmi
 
  # Connect to 'testvmi' using the local ssh binary found in $PATH:
  {{ProgramName}} ssh --%s=true jdoe@testvmi`,
		IdentityFilePathFlag,
		IdentityFilePathFlag,
		usernameFlag,
		wrapLocalSSHFlag,
	)
}

func defaultUsername() string {
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
