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
	commandToExecute, commandToExecuteShort         = "command", "c"
	additionalOpts, additionalOptsShort             = "local-ssh-opts", "t"
)

func NewCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	c := &SSH{
		clientConfig: clientConfig,
		options:      DefaultSSHOptions(),
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
	cmd.Flags().StringVarP(&c.command, commandToExecute, commandToExecuteShort, c.command,
		fmt.Sprintf(`--%s='ls /': Specify a command to execute in the VM`, commandToExecute))
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func AddCommandlineArgs(flagset *pflag.FlagSet, opts *SSHOptions) {
	flagset.StringVarP(&opts.SSHUsername, usernameFlag, usernameFlagShort, opts.SSHUsername,
		fmt.Sprintf("--%s=%s: Set this to the user you want to open the SSH connection as; If unassigned, this will be empty and the SSH default will apply", usernameFlag, opts.SSHUsername))
	flagset.StringVarP(&opts.IdentityFilePath, IdentityFilePathFlag, identityFilePathFlagShort, opts.IdentityFilePath,
		fmt.Sprintf("--%s=/home/jdoe/.ssh/id_rsa: Set the path to a private key used for authenticating to the server; If not provided, the client will try to use the local ssh-agent at $SSH_AUTH_SOCK", IdentityFilePathFlag))
	flagset.StringVar(&opts.KnownHostsFilePath, knownHostsFilePathFlag, opts.KnownHostsFilePathDefault,
		fmt.Sprintf("--%s=/home/jdoe/.ssh/kubevirt_known_hosts: Set the path to the known_hosts file.", knownHostsFilePathFlag))
	flagset.IntVarP(&opts.SSHPort, portFlag, portFlagShort, opts.SSHPort,
		fmt.Sprintf(`--%s=22: Specify a port on the VM to send SSH traffic to`, portFlag))

	addAdditionalCommandlineArgs(flagset, opts)
}

func DefaultSSHOptions() SSHOptions {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		glog.Warningf("failed to determine user home directory: %v", err)
	}
	options := SSHOptions{
		SSHPort:                   22,
		SSHUsername:               defaultUsername(),
		IdentityFilePath:          filepath.Join(homeDir, ".ssh", "id_rsa"),
		IdentityFilePathProvided:  false,
		KnownHostsFilePath:        "",
		KnownHostsFilePathDefault: "",
		AdditionalSSHLocalOptions: []string{},
		WrapLocalSSH:              wrapLocalSSHDefault,
		LocalClientName:           "ssh",
	}

	if len(homeDir) > 0 {
		options.KnownHostsFilePathDefault = filepath.Join(homeDir, ".ssh", "kubevirt_known_hosts")
	}
	return options
}

type SSH struct {
	clientConfig clientcmd.ClientConfig
	options      SSHOptions
	command      string
}

type SSHOptions struct {
	SSHPort                   int
	SSHUsername               string
	IdentityFilePath          string
	IdentityFilePathProvided  bool
	KnownHostsFilePath        string
	KnownHostsFilePathDefault string
	AdditionalSSHLocalOptions []string
	WrapLocalSSH              bool
	LocalClientName           string
}

func (o *SSH) Run(cmd *cobra.Command, args []string) error {
	kind, namespace, name, err := PrepareCommand(cmd, o.clientConfig, &o.options, args)
	if err != nil {
		return err
	}

	if o.options.WrapLocalSSH {
		clientArgs := o.buildSSHTarget(kind, namespace, name)
		return RunLocalClient(kind, namespace, name, &o.options, clientArgs)
	}

	return o.nativeSSH(kind, namespace, name)
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
		opts.SSHUsername = targetUsername
	}
	return
}

func usage() string {
	return fmt.Sprintf(`  # Connect to 'testvmi':
  {{ProgramName}} ssh jdoe@testvmi [--%s]

  # Connect to 'testvm' in 'mynamespace' namespace
  {{ProgramName}} ssh jdoe@vm/testvm.mynamespace [--%s]

  # Specify a username and namespace:
  {{ProgramName}} ssh --namespace=mynamespace --%s=jdoe testvmi`,
		IdentityFilePathFlag,
		IdentityFilePathFlag,
		usernameFlag,
	) + additionalUsage()
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
