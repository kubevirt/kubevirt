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
	"github.com/povsister/scp"

	"kubevirt.io/kubevirt/pkg/virtctl/ssh"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"

	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

func NewCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {

	c := &SCP{
		clientConfig: clientConfig,
		options:      ssh.DefaultSSHOptions(),
	}

	cmd := &cobra.Command{
		Use:     "scp (VM|VMI)",
		Short:   "SCP files from/to a virtual machine instance.",
		Example: usage(),
		Args:    templates.ExactArgs("scp", 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.Run(cmd, args)
		},
	}
	cmd.Flags().BoolVarP(&c.recursive, "recursive", "r", false, "Recursively copy entire directories")
	ssh.AddCommandlineArgs(cmd.Flags(), &c.options)
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

type SCP struct {
	clientConfig clientcmd.ClientConfig
	options      ssh.SSHOptions
	recursive    bool
}

func (o *SCP) Run(cmd *cobra.Command, args []string) error {
	local, remote, toRemote, err := PrepareCommand(cmd, o.clientConfig, &o.options, args)
	if err != nil {
		return err
	}

	sshClient := ssh.NativeSSHConnection{
		ClientConfig: o.clientConfig,
		Options:      o.options,
	}
	client, err := sshClient.PrepareSSHClient(remote.Kind, remote.Namespace, remote.Name)
	if err != nil {
		return err
	}
	scpClient, err := scp.NewClientFromExistingSSH(client, &scp.ClientOption{})
	if err != nil {
		return err
	}
	if toRemote {
		if o.recursive {
			err = scpClient.CopyDirToRemote(local.Path, remote.Path, &scp.DirTransferOption{})
			if err != nil {
				return err
			}
		} else {
			err = scpClient.CopyFileToRemote(local.Path, remote.Path, &scp.FileTransferOption{})
			if err != nil {
				return err
			}
		}
	} else {
		if o.recursive {
			err = scpClient.CopyDirFromRemote(remote.Path, local.Path, &scp.DirTransferOption{})
			if err != nil {
				return err
			}
		} else {
			err = scpClient.CopyFileFromRemote(remote.Path, local.Path, &scp.FileTransferOption{})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func usage() string {
	return `  # Copy a file to the remote home folder of user jdoe
  {{ProgramName}} scp myfile.bin jdoe@testvmi:myfile.bin

  # Copy a directory to the remote home folder of user jdoe
  {{ProgramName}} scp --recursive ~/mydir/ jdoe@testvmi:./mydir

  # Copy a file to the remote home folder of user jdoe without specifying a file name on the target
  {{ProgramName}} scp myfile.bin jdoe@testvmi:.

  # Copy a file to 'testvm' in 'mynamespace' namespace
  {{ProgramName}} scp myfile.bin jdoe@testvmi.mynamespace:myfile.bin

  # Copy a file from the remote location to a local folder
  {{ProgramName}} scp jdoe@testvmi:myfile.bin ~/myfile.bin`
}

func PrepareCommand(cmd *cobra.Command, clientConfig clientcmd.ClientConfig, opts *ssh.SSHOptions, args []string) (local templates.LocalSCPArgument, remote templates.RemoteSCPArgument, toRemote bool, err error) {
	opts.IdentityFilePathProvided = cmd.Flags().Changed(ssh.IdentityFilePathFlag)
	local, remote, toRemote, err = templates.ParseSCPArguments(args[0], args[1])
	if err != nil {
		return
	}

	if len(remote.Namespace) < 1 {
		remote.Namespace, _, err = clientConfig.Namespace()
		if err != nil {
			return
		}
	}

	if len(remote.Username) > 0 {
		opts.SshUsername = remote.Username
	}
	return
}
