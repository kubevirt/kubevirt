package version

import (
	"fmt"
	"strings"

	"github.com/blang/semver"

	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/version"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

var (
	cmd        *cobra.Command
	clientOnly bool
)

const versionsNotAlignedWarnMessage = "You are using a client virtctl version that is different from the KubeVirt version running in the cluster\nClient Version: %s\nServer Version: %s\n"

func VersionCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd = &cobra.Command{
		Use:     "version",
		Short:   "Print the client and server version information.",
		Example: usage(),
		Args:    templates.ExactArgs("version", 0),
		RunE: func(cmd *cobra.Command, args []string) error {
			v := Version{clientConfig: clientConfig}
			return v.Run()
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

func (v *Version) Run() error {
	cmd.Printf("Client Version: %s\n", fmt.Sprintf("%#v", version.Get()))

	if !clientOnly {
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

func CheckClientServerVersion(clientConfig *clientcmd.ClientConfig) {
	clientVersion := version.Get()
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
	client, err := semver.Make(clientGitVersion)
	if err != nil {
		cmd.Println(err)
		return
	}

	server, err := semver.Make(serverGitVersion)
	if err != nil {
		cmd.Println(err)
		return
	}

	if client.Major != server.Major || client.Minor != server.Minor {
		cmd.Printf(versionsNotAlignedWarnMessage, clientVersion, *serverVersion)
	}
}
