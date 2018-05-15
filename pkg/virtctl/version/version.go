package version

import (
	"fmt"

	"github.com/spf13/cobra"

	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/version"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

func VersionCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "version",
		Short:   "Print the client and server version information",
		Example: usage(),
		Args:    cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			v := Version{clientConfig: clientConfig}
			return v.Run(cmd, args)
		},
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func usage() string {
	usage := "# Print the client and server versions for the current context \n"
	usage += "virtctl version"
	return usage
}

type Version struct {
	clientConfig clientcmd.ClientConfig
}

func (v *Version) Run(cmd *cobra.Command, args []string) error {
	virCli, err := kubecli.GetKubevirtClientFromClientConfig(v.clientConfig)
	if err != nil {
		return err
	}

	result := virCli.RestClient().Get().RequestURI("/apis/subresources.kubevirt.io/v1alpha1/version").Do()
	fmt.Println(result)
	data, _ := result.Raw()
	fmt.Println(data)
	fmt.Println(result.Error())

	fmt.Printf("Client Version: %s\n", fmt.Sprintf("%#v", version.Get()))
	return nil
}
