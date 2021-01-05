package version

import (
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/version"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

var clientOnly bool

func VersionCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "version",
		Short:   "Print the client and server version information.",
		Example: usage(),
		Args:    templates.ExactArgs("version", 0),
		RunE: func(cmd *cobra.Command, args []string) error {
			v := Version{clientConfig: clientConfig}
			return v.Run(cmd, args)
		},
	}
	cmd.Flags().BoolVarP(&clientOnly, "client", "c", clientOnly, "Client version only (no server required).")
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func usage() string {
	usage := "  # Print the client and server versions for the current context:\n"
	usage += "  {{ProgramName}} version"
	return usage
}

type Version struct {
	clientConfig clientcmd.ClientConfig
}

func (v *Version) Run(cmd *cobra.Command, args []string) error {
	fmt.Printf("Client Version: %s\n", fmt.Sprintf("%#v", version.Get()))

	if !clientOnly {
		virCli, err := kubecli.GetKubevirtClientFromClientConfig(v.clientConfig)
		if err != nil {
			return err
		}

		serverInfo, err := virCli.ServerVersion().Get()
		if err != nil {
			return err
		}

		fmt.Printf("Server Version: %s\n", fmt.Sprintf("%#v", *serverInfo))
	}

	return nil
}
