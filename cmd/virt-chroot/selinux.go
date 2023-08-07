package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/opencontainers/selinux/go-selinux"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"

	"kubevirt.io/kubevirt/pkg/safepath"
)

const xattrNameSelinux = "security.selinux"

var root string

// NewGetEnforceCommand determines if selinux is enabled in the kernel (enforced or permissive)
func NewGetEnforceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "getenforce",
		Short: "determine if selinux is present",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			enforcing, err := os.ReadFile("/sys/fs/selinux/enforce")
			if err != nil {
				fmt.Println("disabled")
			} else if bytes.Compare(enforcing, []byte("1")) == 0 {
				fmt.Println("enforcing")
			} else {
				fmt.Println("permissive")
			}
			return nil
		},
	}
	return cmd
}

func RelabelCommand() *cobra.Command {
	relabelCommad := &cobra.Command{
		Use:       "relabel",
		Short:     "relabel a file with the given selinux label, if the path is not labeled like this already",
		Example:   "virt-chroot selinux relabel <new-label> <file-path>",
		ValidArgs: nil,
		Args:      cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			label := args[0]
			if root == "" {
				root = "/"
			}

			rootPath, err := safepath.JoinAndResolveWithRelativeRoot(root)
			if err != nil {
				return fmt.Errorf("failed to open root path %v: %v", rootPath, err)
			}
			safePath, err := safepath.JoinNoFollow(rootPath, args[1])
			if err != nil {
				return fmt.Errorf("failed to open final path %v: %v", filepath.Join(root, args[1]), err)
			}
			fd, err := safepath.OpenAtNoFollow(safePath)
			if err != nil {
				return fmt.Errorf("could not open file %v. Reason: %v", safePath, err)
			}
			defer fd.Close()
			filePath := fd.SafePath()

			if fileInfo, err := safepath.StatAtNoFollow(safePath); err != nil {
				return fmt.Errorf("could not stat file %v. Reason: %v", safePath, err)
			} else if (fileInfo.Mode() & os.ModeSocket) != 0 {
				return relabelUnixSocket(filePath, label)
			}

			writeableFD, err := os.OpenFile(filePath, os.O_APPEND|unix.S_IWRITE, os.ModePerm)
			if err != nil {
				return fmt.Errorf("error reopening file %s to write label %s. Reason: %v", filePath, label, err)
			}
			defer writeableFD.Close()

			currentFileLabel, err := getLabel(writeableFD)
			if err != nil {
				return fmt.Errorf("failed to get selinux label for file %v: %v", filePath, err)
			}

			if currentFileLabel != label {
				if err := unix.Fsetxattr(int(writeableFD.Fd()), xattrNameSelinux, []byte(label), 0); err != nil {
					return fmt.Errorf("error relabeling file %s with label %s. Reason: %v", filePath, label, err)
				}
			}

			return nil
		},
	}
	relabelCommad.Flags().StringVar(&root, "root", "/", "safe root path which will be prepended to passed in files")
	return relabelCommad
}

func getLabel(file *os.File) (string, error) {
	// let's first find out the actual buffer size
	var buffer []byte
	labelLength, err := unix.Fgetxattr(int(file.Fd()), xattrNameSelinux, buffer)
	if err != nil {
		return "", fmt.Errorf("error reading fgetxattr: %v", err)
	}
	// now ask with the needed size
	buffer = make([]byte, labelLength)
	labelLength, err = unix.Fgetxattr(int(file.Fd()), xattrNameSelinux, buffer)
	if err != nil {
		return "", fmt.Errorf("error reading fgetxattr: %v", err)
	}
	if labelLength > 0 && buffer[labelLength-1] == '\x00' {
		labelLength = labelLength - 1
	}
	return string(buffer[:labelLength]), nil
}

func relabelUnixSocket(filePath, label string) error {
	if currentLabel, err := selinux.FileLabel(filePath); err != nil {
		return fmt.Errorf("could not retrieve label of file %s. Reason: %v", filePath, err)
	} else if currentLabel != label {
		if err := unix.Setxattr(filePath, xattrNameSelinux, []byte(label), 0); err != nil {
			return fmt.Errorf("error relabeling file %s with label %s. Reason: %v", filePath, label, err)
		}
	}
	return nil
}
