package credentials

import (
	"github.com/spf13/cobra"

	"kubevirt.io/kubevirt/pkg/virtctl/credentials/addkey"
	"kubevirt.io/kubevirt/pkg/virtctl/credentials/password"
	"kubevirt.io/kubevirt/pkg/virtctl/credentials/removekey"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "credentials",
		Short: "Manipulate credentials on a virtual machine.",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Print(cmd.UsageString())
		},
	}

	cmd.AddCommand(
		addkey.NewCommand(),
		removekey.NewCommand(),
		password.SetPasswordCommand(),
	)

	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}
