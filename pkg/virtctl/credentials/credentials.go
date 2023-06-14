package credentials

import (
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	add_key "kubevirt.io/kubevirt/pkg/virtctl/credentials/add-key"
	remove_key "kubevirt.io/kubevirt/pkg/virtctl/credentials/remove-key"
	set_password "kubevirt.io/kubevirt/pkg/virtctl/credentials/set-password"
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
		add_key.NewCommand(clientConfig),
		remove_key.NewCommand(clientConfig),
		set_password.SetPasswordCommand(clientConfig),
	)

	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}
