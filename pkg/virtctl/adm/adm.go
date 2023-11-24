package adm

import (
	"github.com/spf13/cobra"

	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/kubevirt/pkg/virtctl/adm/logverbosity"

	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	ADM = "adm"
)

func NewCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   ADM,
		Short: "Administrate KubeVirt configuration.",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf(cmd.UsageString())
		},
	}

	cmd.AddCommand(logverbosity.NewCommand(clientConfig))

	cmd.SetUsageTemplate(templates.UsageTemplate())

	return cmd
}
