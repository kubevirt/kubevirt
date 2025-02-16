package version

import (
	"fmt"

	"github.com/spf13/cobra"

	client_version "kubevirt.io/client-go/version"

	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

type version struct {
	clientOnly bool
}

func VersionCommand() *cobra.Command {
	v := &version{}
	cmd := &cobra.Command{
		Use:     "version",
		Short:   "Print the client and server version information.",
		Example: usage(),
		Args:    cobra.ExactArgs(0),
		RunE:    v.Run,
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

func (v *version) Run(cmd *cobra.Command, _ []string) error {
	cmd.Printf("Client Version: %s\n", fmt.Sprintf("%#v", client_version.Get()))

	if !v.clientOnly {
		virCli, _, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
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
