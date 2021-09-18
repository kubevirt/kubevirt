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
	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	portFlag, portFlagShort                         = "port", "p"
	wrapLocalSSHFlag                                = "local-ssh"
	usernameFlag, usernameFlagShort                 = "username", "l"
	identityFilePathFlag, identityFilePathFlagShort = "identity-file", "i"
	knownHostsFilePathFlag                          = "known-hosts"
)

var (
	wrapLocalSSH              bool = false
	sshPort                   int
	sshUsername               string
	identityFilePath          string
	knownHostsFilePath        string
	knownHostsFilePathDefault string
)

func NewCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ssh (VM|VMI)",
		Short:   "Open a SSH connection to a virtual machine instance.",
		Example: usage(),
		Args:    templates.ExactArgs("ssh", 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := SSH{clientConfig: clientConfig}
			return c.Run(cmd, args)
		},
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		glog.Warningf("failed to determine user home directory: %v", err)
	}
	if len(homeDir) > 0 {
		knownHostsFilePathDefault = filepath.Join(homeDir, ".ssh", "known_hosts")
	}

	cmd.Flags().StringVarP(&sshUsername, usernameFlag, usernameFlagShort, defaultUsername(),
		fmt.Sprintf("--%s=%s: Set this to the user you want to open the SSH connection as; If unassigned, this will be empty and the SSH default will apply", usernameFlag, defaultUsername()))
	cmd.Flags().StringVarP(&identityFilePath, identityFilePathFlag, identityFilePathFlagShort, filepath.Join(homeDir, ".ssh", "id_rsa"),
		fmt.Sprintf("--%s=/home/jdoe/.ssh/id_rsa: Set the path to a private key used for authenticating to the server; If not provided, the client will try to use the local ssh-agent at $SSH_AUTH_SOCK", identityFilePathFlag))
	cmd.Flags().StringVar(&knownHostsFilePath, knownHostsFilePathFlag, knownHostsFilePathDefault,
		fmt.Sprintf("--%s=/home/jdoe/.ssh/known_hosts: Set the path to the known_hosts file; If not provided, the client will skip host checks", knownHostsFilePathFlag))
	cmd.Flags().IntVarP(&sshPort, portFlag, portFlagShort, 22,
		fmt.Sprintf(`--%s=22: Specify a port on the VM to send SSH traffic to`, portFlag))
	cmd.Flags().BoolVar(&wrapLocalSSH, wrapLocalSSHFlag, wrapLocalSSH,
		fmt.Sprintf("--%s=true: Set this to true to use the SSH command available on your system by using this command as ProxyCommand; If unassigned, this will establish a SSH connection with limited capabilities provided by this client", wrapLocalSSHFlag))
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

type SSH struct {
	clientConfig clientcmd.ClientConfig
}

func (o *SSH) Run(cmd *cobra.Command, args []string) error {
	kind, namespace, name, err := o.prepareCommand(args)
	if err != nil {
		return err
	}

	if wrapLocalSSH {
		return runLocalCommandClient(kind, namespace, name)
	}

	client, err := o.prepareSSHClient(kind, namespace, name)
	if err != nil {
		return err
	}
	return o.startSession(client)
}

func (o *SSH) prepareCommand(args []string) (kind, namespace, name string, err error) {
	var targetUsername string
	kind, namespace, name, targetUsername, err = templates.ParseSSHTarget(args[0])
	if err != nil {
		return
	}

	if len(namespace) < 1 {
		namespace, _, err = o.clientConfig.Namespace()
		if err != nil {
			return
		}
	}

	if len(targetUsername) > 0 {
		sshUsername = targetUsername
	}

	return
}

func (o *SSH) prepareSSHTunnel(kind, namespace, name string) (kubecli.StreamInterface, error) {
	virtCli, err := kubecli.GetKubevirtClientFromClientConfig(o.clientConfig)
	if err != nil {
		return nil, err
	}

	var stream kubecli.StreamInterface
	if kind == "vmi" {
		stream, err = virtCli.VirtualMachineInstance(namespace).PortForward(name, sshPort, "tcp")
		if err != nil {
			return nil, fmt.Errorf("can't access VMI %s: %w", name, err)
		}
	} else if kind == "vm" {
		stream, err = virtCli.VirtualMachine(namespace).PortForward(name, sshPort, "tcp")
		if err != nil {
			return nil, fmt.Errorf("can't access VM %s: %w", name, err)
		}
	}

	return stream, nil
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
		identityFilePathFlag,
		identityFilePathFlag,
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
