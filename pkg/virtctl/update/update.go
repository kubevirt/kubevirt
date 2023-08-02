package update

import (
	"github.com/spf13/cobra"

	"k8s.io/client-go/tools/clientcmd"

	machinetype "kubevirt.io/kubevirt/pkg/virtctl/update/machine-type"
)

const (
	UPDATE = "update"
)

func NewCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   UPDATE,
		Short: "Update an attribute of one or many VMs.",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf(cmd.UsageString())
		},
	}

	cmd.AddCommand(machinetype.NewMachineTypeCommand(clientConfig))

	return cmd
}
