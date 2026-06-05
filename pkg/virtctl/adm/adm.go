package adm

import (
	"github.com/spf13/cobra"

	"kubevirt.io/kubevirt/pkg/virtctl/adm/logverbosity"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	ADM = "adm"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   ADM,
		Short: "Administrate KubeVirt configuration.",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println(cmd.UsageString())
		},
	}
	cmd.AddCommand(logverbosity.NewCommand())
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}
