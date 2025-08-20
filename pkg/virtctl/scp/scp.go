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

package scp

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

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
		Options: ssh.DefaultSSHOptions(),
	}
	c.Options.LocalClientName = "scp"

	cmd := &cobra.Command{
		Use:     "scp (VM|VMI)",
		Short:   "SCP files from/to a virtual machine instance.",
		Example: usage(),
		Args:    cobra.ExactArgs(2),
		RunE:    c.run,
	}

	ssh.AddCommandlineArgs(cmd.Flags(), &c.Options)
	cmd.Flags().BoolVarP(&c.Recursive, recursiveFlag, recursiveFlagShort, c.Recursive,
		"Recursively copy entire directories")
	cmd.Flags().BoolVar(&c.Preserve, preserveFlag, c.Preserve,
		"Preserves modification times, access times, and modes from the original file.")
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

type SCP struct {
	Options   ssh.SSHOptions
	Recursive bool
	Preserve  bool
}

func (o *SCP) run(cmd *cobra.Command, args []string) error {
	_, namespace, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return err
	}

	local, remote, toRemote, err := prepareCommand(cmd, namespace, &o.Options, args)
	if err != nil {
		return err
	}

	clientArgs := o.BuildSCPTarget(local, remote, toRemote)
	return ssh.LocalClientCmd(remote.Kind, remote.Namespace, remote.Name, &o.Options, clientArgs).Run()
}

func (o *SCP) BuildSCPTarget(local *LocalArgument, remote *RemoteArgument, toRemote bool) (opts []string) {
	if o.Recursive {
		opts = append(opts, "-r")
	}
	if o.Preserve {
		opts = append(opts, "-p")
	}

	target := strings.Builder{}
	if len(o.Options.SSHUsername) > 0 {
		target.WriteString(o.Options.SSHUsername)
		target.WriteRune('@')
	}
	target.WriteString(remote.Kind)
	target.WriteString(".")
	target.WriteString(remote.Name)
	target.WriteString(".")
	target.WriteString(remote.Namespace)
	target.WriteRune(':')
	target.WriteString(remote.Path)

	if toRemote {
		opts = append(opts, local.Path, target.String())
	} else {
		opts = append(opts, target.String(), local.Path)
	}
	return
}

type LocalArgument struct {
	Path string
}

type RemoteArgument struct {
	Kind      string
	Namespace string
	Name      string
	Username  string
	Path      string
}

func prepareCommand(cmd *cobra.Command, fallbackNamespace string, opts *ssh.SSHOptions, args []string) (*LocalArgument, *RemoteArgument, bool, error) {
	opts.IdentityFilePathProvided = cmd.Flags().Changed(ssh.IdentityFilePathFlag)

	local, remote, toRemote, err := ParseTarget(args[0], args[1])
	if err != nil {
		return nil, nil, false, err
	}

	if len(remote.Namespace) < 1 {
		remote.Namespace = fallbackNamespace
	}

	if len(remote.Username) > 0 {
		opts.SSHUsername = remote.Username
	}

	return local, remote, toRemote, nil
}

func usage() string {
	return `  # Copy a file to the remote home folder of user jdoe
  {{ProgramName}} scp myfile.bin jdoe@vmi/testvmi:myfile.bin

  # Copy a directory to the remote home folder of user jdoe
  {{ProgramName}} scp --Recursive ~/mydir/ jdoe@vmi/testvmi:./mydir

  # Copy a file to the remote home folder of user jdoe without specifying a file name on the target
  {{ProgramName}} scp myfile.bin jdoe@vmi/testvmi:.

  # Copy a file to 'testvm' in 'mynamespace' namespace
  {{ProgramName}} scp myfile.bin jdoe@vmi/testvmi.mynamespace:myfile.bin

  # Copy a file from the remote location to a local folder
  {{ProgramName}} scp jdoe@vmi/testvmi:myfile.bin ~/myfile.bin`
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

	var toRemote bool
	if strings.Contains(destination, ":") {
		source, destination = destination, source
		toRemote = true
	}

	split := strings.SplitN(source, ":", 2)
	if len(split) != 2 {
		return nil, nil, toRemote, fmt.Errorf("invalid remote argument format: %q", source)
	}

	remote := &RemoteArgument{
		Path: split[1],
	}
	local := &LocalArgument{
		Path: destination,
	}
	var err error
	remote.Kind, remote.Namespace, remote.Name, remote.Username, err = ssh.ParseTarget(split[0])

	if err != nil {
		return nil, nil, false, err
	}

	return local, remote, toRemote, nil
}
