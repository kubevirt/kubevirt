//go:build !excludenative

package scp

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/povsister/scp"

	"kubevirt.io/kubevirt/pkg/virtctl/ssh"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

func (o *SCP) nativeSCP(local templates.LocalSCPArgument, remote templates.RemoteSCPArgument, toRemote bool) error {
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
		return o.copyToRemote(scpClient, local.Path, remote.Path)
	}
	return o.copyFromRemote(scpClient, local.Path, remote.Path)
}

func (o *SCP) copyToRemote(client *scp.Client, localPath, remotePath string) error {
	isFile, isDir, exists, err := stat(localPath)
	if err != nil {
		return fmt.Errorf("failed reading path %q: %v", localPath, err)
	}

	if !exists {
		return fmt.Errorf("local path %q does not exist, can't copy it", localPath)
	}

	if o.recursive {
		if isFile {
			return fmt.Errorf("local path %q is not a directory but '--recursive' was provided", localPath)
		}

		return client.CopyDirToRemote(localPath, remotePath, &scp.DirTransferOption{PreserveProp: o.preserve})
	}

	if isDir {
		return fmt.Errorf("local path %q is a directory but '--recursive' was not provided", localPath)
	}

	return client.CopyFileToRemote(localPath, remotePath, &scp.FileTransferOption{PreserveProp: o.preserve})
}

func (o *SCP) copyFromRemote(client *scp.Client, localPath, remotePath string) error {
	_, isDir, exists, err := stat(localPath)
	if err != nil {
		return fmt.Errorf("failed reading path %q: %v", localPath, err)
	}

	if o.recursive {
		if exists {
			if !isDir {
				return fmt.Errorf("local path %q is a file but '--recursive' was provided", localPath)
			}
			localPath = appendRemoteBase(localPath, remotePath)
		}

		if err := os.MkdirAll(localPath, os.ModePerm); err != nil {
			return fmt.Errorf("failed ensuring the existence of the local target directory %q: %v", localPath, err)
		}

		return client.CopyDirFromRemote(remotePath, localPath, &scp.DirTransferOption{PreserveProp: o.preserve})
	}

	if exists && isDir {
		localPath = appendRemoteBase(localPath, remotePath)
	}

	return client.CopyFileFromRemote(remotePath, localPath, &scp.FileTransferOption{PreserveProp: o.preserve})
}

func stat(path string) (isFile, isDir, exists bool, err error) {
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
