package version

import (
	"fmt"
	"strings"

	"github.com/coreos/go-semver/semver"

	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/client-go/kubecli"
	client_version "kubevirt.io/client-go/version"

	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

type version struct {
	clientConfig clientcmd.ClientConfig
	clientOnly   bool
}

const versionsNotAlignedWarnMessage = "You are using a client virtctl version that is different from the KubeVirt version running in the cluster\nClient Version: %s\nServer Version: %s\n"

func VersionCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	v := &version{clientConfig: clientConfig}
	cmd := &cobra.Command{
		Use:     "version",
		Short:   "Print the client and server version information.",
		Example: usage(),
		Args:    cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return v.Run(cmd)
		},
	}
	cmd.Flags().BoolVarP(&v.clientOnly, "client", "c", v.clientOnly, "Client version only (no server required).")
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func usage() string {
	usage := "  # Print the client and server versions for the current context:\n"
	usage += "  {{ProgramName}} version"
	return usage
}

func (v *version) Run(cmd *cobra.Command) error {
	cmd.Printf("Client Version: %s\n", fmt.Sprintf("%#v", client_version.Get()))

	if !v.clientOnly {
		virCli, err := kubecli.GetKubevirtClientFromClientConfig(v.clientConfig)
		if err != nil {
			return err
		}

		serverInfo, err := virCli.ServerVersion().Get()
		if err != nil {
			return err
		}

		cmd.Printf("Server Version: %s\n", fmt.Sprintf("%#v", *serverInfo))
	}

	return nil
}

func CheckClientServerVersion(clientConfig *clientcmd.ClientConfig, cmd *cobra.Command) {
	clientVersion := client_version.Get()
	virCli, err := kubecli.GetKubevirtClientFromClientConfig(*clientConfig)
	if err != nil {
		cmd.Println(err)
		return
	}

	serverVersion, err := virCli.ServerVersion().Get()
	if err != nil {
		cmd.Println(err)
		return
	}

	clientGitVersion := strings.TrimPrefix(clientVersion.GitVersion, "v")
	serverGitVersion := strings.TrimPrefix(serverVersion.GitVersion, "v")
	client, err := semver.NewVersion(clientGitVersion)
	if err != nil {
		cmd.Println(err)
		return
	}

	server, err := semver.NewVersion(serverGitVersion)
	if err != nil {
		cmd.Println(err)
		return
	}

	if client.Major != server.Major || client.Minor != server.Minor {
		cmd.Printf(versionsNotAlignedWarnMessage, clientVersion, *serverVersion)
	}
}
