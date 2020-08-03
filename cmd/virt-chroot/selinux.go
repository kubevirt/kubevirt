package main

import (
	"fmt"

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
			if selinux.GetEnabled() {
				mode := selinux.EnforceMode()
				if mode == selinux.Enforcing {
					fmt.Println("enforcing")
				} else if mode == selinux.Permissive {
					fmt.Println("permissive")
				} else {
					fmt.Println("disabled")
				}
			} else {
				fmt.Println("disabled")
			}
			return nil
		},
	}
	return cmd
}

func RelabelFile() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "relabel",
		Short:   "relabel a file with the given selinux label",
		Example: "virt-chroot selinux relabel <file-path> <new-label>",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]
			label := args[1]

			if err := selinux.Chcon(filePath, label, false); err != nil {
				return fmt.Errorf("error relabeling file %s with label %s. Reason: %v", filePath, label, err)
			}
			return nil
		},
	}
	return cmd
}
