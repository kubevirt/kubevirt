package install

import (
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"

	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

func InstallCommand(rootCommand *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "install",
		Short:   "Install virtctl as a kubectl plugin",
		Example: usage(),
		Args:    cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			i := Install{rootCommand: rootCommand}
			return i.Run(cmd, args)
		},
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func usage() string {
	usage := "# Install virtctl as a kubectl plugin \n"
	usage += "virtctl install"
	return usage
}

type Install struct {
	rootCommand *cobra.Command
}

func (I *Install) Run(cmd *cobra.Command, args []string) error {
	pluginConfig := clientcmdapi.

	for _, command := range I.rootCommand.Commands() {

	}
	return nil
}
