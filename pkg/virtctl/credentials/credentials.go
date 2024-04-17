package credentials

import (
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/kubevirt/pkg/virtctl/credentials/addkey"
	"kubevirt.io/kubevirt/pkg/virtctl/credentials/password"
	"kubevirt.io/kubevirt/pkg/virtctl/credentials/removekey"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

func NewCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "credentials",
		Short: "Manipulate credentials on a virtual machine.",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Print(cmd.UsageString())
		},
	}

	cmd.AddCommand(
		addkey.NewCommand(clientConfig),
		removekey.NewCommand(clientConfig),
		password.SetPasswordCommand(clientConfig),
	)

	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}
