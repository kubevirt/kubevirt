package config

import (
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage virtctl configuration",
	}

	var vncViewer string

	setupCmd := &cobra.Command{
		Use:   "setup",
		Short: "Run setup wizard",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunSetup(vncViewer)
		},
	}
	setupCmd.Flags().StringVar(&vncViewer, "vnc-viewer", "", "Set VNC viewer directly (skip interactive prompt). Supported: remote-viewer, virt-viewer, vncviewer")

	cmd.AddCommand(setupCmd)
	return cmd
}
