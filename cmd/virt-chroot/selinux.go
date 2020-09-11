package main

import (
	"bytes"
	"fmt"
	"io/ioutil"

	"github.com/opencontainers/selinux/go-selinux"
	"github.com/spf13/cobra"
)

// NewGetEnforceCommand determines if selinux is enabled in the kernel (enforced or permissive)
func NewGetEnforceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "getenforce",
		Short: "determine if selinux is present",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			enforcing, err := ioutil.ReadFile("/sys/fs/selinux/enforce")
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
	return &cobra.Command{
		Use:       "relabel",
		Short:     "relabel a file with the given selinux label, if the path is not labeled like this already",
		Example:   "virt-chroot selinux relabel <new-label> <file-path>",
		ValidArgs: nil,
		Args:      cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			label := args[0]
			filePath := args[1]

			currentFileLabel, err := selinux.FileLabel(filePath)
			if err != nil {
				return fmt.Errorf("could not retrieve label of file %s. Reason: %v", filePath, err)
			}

			if currentFileLabel != label {
				if err := selinux.Chcon(filePath, label, false); err != nil {
					return fmt.Errorf("error relabeling file %s with label %s. Reason: %v", filePath, label, err)
				}
			}

			return nil
		},
	}
}
