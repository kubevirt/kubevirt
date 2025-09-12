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
		Args:    cobra.NoArgs,
		RunE:    v.Run,
	}
	cmd.Flags().BoolVarP(&v.clientOnly, "client", "c", v.clientOnly, "VirtClient version only (no server required).")
	cmd.SetUsageTemplate(templates.UsageTemplate())

	return cmd
}

func usage() string {
	return `  # Print the client and server versions for the current context:
  {{ProgramName}} version`
}

func (v *version) Run(cmd *cobra.Command, _ []string) error {
	cmd.Printf("VirtClient Version: %#v\n", client_version.Get())

	if v.clientOnly {
		return nil
	}

	virtClient, _, _, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get virtClient config: %w", err)
	}

	serverInfo, err := virtClient.ServerVersion().Get()
	if err != nil {
		return fmt.Errorf("failed to get server version: %w", err)
	}

	cmd.Printf("Server Version: %#v\n", *serverInfo)
	return nil
}
