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
	"os"
	"path/filepath"

	"github.com/povsister/scp"
	"github.com/spf13/cobra"

	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/kubevirt/pkg/virtctl/ssh"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

func NewCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {

	c := &SCP{
		clientConfig: clientConfig,
		options:      ssh.DefaultSSHOptions(),
		recursive:    false,
		preserve:     false,
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
	cmd.Flags().BoolVarP(&c.recursive, "recursive", "r", c.recursive, "Recursively copy entire directories")
	cmd.Flags().BoolVar(&c.preserve, "preserve", c.preserve, "Preserves modification times, access times, and modes from the original file.")
	ssh.AddCommandlineArgs(cmd.Flags(), &c.options)
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

type SCP struct {
	clientConfig clientcmd.ClientConfig
	options      ssh.SSHOptions
	recursive    bool
	preserve     bool
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
	isFile, isDir, exists, err := stat(local.Path)
	if err != nil {
		return fmt.Errorf("failed reading path %q: %v", local.Path, err)
	}

	if toRemote {
		if !exists {
			return fmt.Errorf("local path %q does not exist, can't copy it", local.Path)
		}

		if o.recursive {
			if isFile {
				return fmt.Errorf("local path %q is not a direcotry but '--recursive' was provided", local.Path)
			}
			err = scpClient.CopyDirToRemote(local.Path, remote.Path, &scp.DirTransferOption{PreserveProp: o.preserve})
			if err != nil {
				return err
			}
		} else {
			if isDir {
				return fmt.Errorf("local path %q is a directory but '--recursive' was not provided", local.Path)
			}
			if err = scpClient.CopyFileToRemote(local.Path, remote.Path, &scp.FileTransferOption{PreserveProp: o.preserve}); err != nil {
				return err
			}
		}
	} else {
		if o.recursive {
			if exists {
				if !isDir {
					return fmt.Errorf("local path %q is a file but '--recursive' was provided", local.Path)
				}
				local.Path = appendRemoteBase(local.Path, remote.Path)
			}

			if err := os.MkdirAll(local.Path, os.ModePerm); err != nil {
				return fmt.Errorf("failed ensuring the existence of the local target directory %q: %v", local.Path, err)
			}

			err = scpClient.CopyDirFromRemote(remote.Path, local.Path, &scp.DirTransferOption{PreserveProp: o.preserve})
			if err != nil {
				return err
			}
		} else {
			if exists && isDir {
				local.Path = appendRemoteBase(local.Path, remote.Path)
			}
			err = scpClient.CopyFileFromRemote(remote.Path, local.Path, &scp.FileTransferOption{PreserveProp: o.preserve})
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

func stat(path string) (isFile bool, isDir bool, exists bool, err error) {
	s, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, false, false, nil
	} else if err != nil {
		return false, false, false, err
	}
	return !s.IsDir(), s.IsDir(), true, nil
}

func appendRemoteBase(localPath, remotePath string) string {
	remoteBase := filepath.Base(remotePath)
	switch remoteBase {
	case "..", ".", "/", "./", "":
		// no identifiable base name, let's go with the supplied local path
		return localPath
	default:
		// we identified a base location, let's append it to the local path
		return filepath.Join(localPath, remoteBase)
	}
}
