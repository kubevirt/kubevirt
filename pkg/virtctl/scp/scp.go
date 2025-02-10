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

package scp

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
	"kubevirt.io/kubevirt/pkg/virtctl/ssh"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	recursiveFlag, recursiveFlagShort = "recursive", "r"
	preserveFlag                      = "preserve"
)

func NewCommand() *cobra.Command {
	c := &SCP{
		options: ssh.DefaultSSHOptions(),
	}
	c.options.LocalClientName = "scp"

	cmd := &cobra.Command{
		Use:     "scp (VM|VMI)",
		Short:   "SCP files from/to a virtual machine instance.",
		Example: usage(),
		Args:    cobra.ExactArgs(2),
		RunE:    c.Run,
	}

	ssh.AddCommandlineArgs(cmd.Flags(), &c.options)
	cmd.Flags().BoolVarP(&c.recursive, recursiveFlag, recursiveFlagShort, c.recursive,
		"Recursively copy entire directories")
	cmd.Flags().BoolVar(&c.preserve, preserveFlag, c.preserve,
		"Preserves modification times, access times, and modes from the original file.")
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

type SCP struct {
	options   ssh.SSHOptions
	recursive bool
	preserve  bool
}

func (o *SCP) Run(cmd *cobra.Command, args []string) error {
	client, namespace, changed, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return err
	}

	local, remote, toRemote, err := PrepareCommand(cmd, namespace, changed, &o.options, args)
	if err != nil {
		return err
	}

	if o.options.WrapLocalSSH {
		clientArgs := o.buildSCPTarget(local, remote, toRemote)
		return ssh.RunLocalClient(remote.Namespace, remote.Name, &o.options, clientArgs)
	}

	return o.nativeSCP(local, remote, toRemote, client)
}

type LocalArgument struct {
	Path string
}

type RemoteArgument struct {
	Namespace string
	Name      string
	Username  string
	Path      string
}

func PrepareCommand(cmd *cobra.Command, clientNamespace string, namespaceChanged bool, opts *ssh.SSHOptions, args []string) (*LocalArgument, *RemoteArgument, bool, error) {
	opts.IdentityFilePathProvided = cmd.Flags().Changed(ssh.IdentityFilePathFlag)

	local, remote, toRemote, err := ParseTarget(args[0], args[1])
	if err != nil {
		return nil, nil, false, err
	}

	if remote.Namespace == "" {
		remote.Namespace = clientNamespace
	} else if namespaceChanged {
		log.Log.Infof("Overriding remote namespace '%s' with namespace '%s' from commandline", remote.Namespace, clientNamespace)
		remote.Namespace = clientNamespace
	}

	if len(remote.Username) > 0 {
		opts.SSHUsername = remote.Username
	}

	return local, remote, toRemote, nil
}

func usage() string {
	return `  # Copy a file to the remote home folder of user jdoe
  {{ProgramName}} scp myfile.bin jdoe@testvmi:myfile.bin

  # Copy a directory to the remote home folder of user jdoe
  {{ProgramName}} scp --recursive ~/mydir/ jdoe@testvmi:./mydir

  # Copy a file to the remote home folder of user jdoe without specifying a file name on the target
  {{ProgramName}} scp myfile.bin jdoe@testvmi:.

  # Copy a file to 'testvm' in 'mynamespace' namespace
  {{ProgramName}} scp myfile.bin jdoe@mynamespace/testvmi:myfile.bin

  # Copy a file from the remote location to a local folder
  {{ProgramName}} scp jdoe@testvmi:myfile.bin ~/myfile.bin`
}

func ParseTarget(source, destination string) (*LocalArgument, *RemoteArgument, bool, error) {
	if strings.Contains(source, ":") && strings.Contains(destination, ":") {
		return nil, nil, false, fmt.Errorf(
			"copying from a remote location to another remote location is not supported: %q to %q",
			source, destination,
		)
	}

	if !strings.Contains(source, ":") && !strings.Contains(destination, ":") {
		return nil, nil, false, fmt.Errorf(
			"none of the two provided locations seems to be a remote location: %q to %q",
			source, destination,
		)
	}

	toRemote := false
	if strings.Contains(destination, ":") {
		source, destination = destination, source
		toRemote = true
	}

	split := strings.SplitN(source, ":", 2)
	if len(split) != 2 {
		return nil, nil, toRemote, fmt.Errorf("invalid remote argument format: %q", source)
	}

	namespace, name, username, err := ssh.ParseTarget(split[0])
	if err != nil {
		return nil, nil, false, err
	}

	local := &LocalArgument{
		Path: destination,
	}
	remote := &RemoteArgument{
		Namespace: namespace,
		Name:      name,
		Username:  username,
		Path:      split[1],
	}

	return local, remote, toRemote, nil
}
