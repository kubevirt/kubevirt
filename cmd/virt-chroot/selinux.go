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
					fmt.Println("enabled")
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
