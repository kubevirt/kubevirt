package virtctl

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virtctl/console"
	"kubevirt.io/kubevirt/pkg/virtctl/expose"
	"kubevirt.io/kubevirt/pkg/virtctl/imageupload"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
	"kubevirt.io/kubevirt/pkg/virtctl/version"
	"kubevirt.io/kubevirt/pkg/virtctl/vm"
	"kubevirt.io/kubevirt/pkg/virtctl/vnc"
)

func NewVirtctlCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "virtctl",
		Short:         "virtctl controls virtual machine related operations on your kubernetes cluster.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprint(cmd.OutOrStderr(), cmd.UsageString())
		},
	}

	optionsCmd := &cobra.Command{
		Use:    "options",
		Hidden: true,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprint(cmd.OutOrStderr(), cmd.UsageString())
		},
	}
	optionsCmd.SetUsageTemplate(templates.OptionsUsageTemplate())
	//TODO: Add a ClientConfigFactory which allows substituting the KubeVirt client with a mock for unit testing
	clientConfig := kubecli.DefaultClientConfig(rootCmd.PersistentFlags())
	AddGlogFlags(rootCmd.PersistentFlags())
	rootCmd.SetUsageTemplate(templates.MainUsageTemplate())
	rootCmd.AddCommand(
		console.NewCommand(clientConfig),
		vnc.NewCommand(clientConfig),
		vm.NewStartCommand(clientConfig),
		vm.NewStopCommand(clientConfig),
		vm.NewRestartCommand(clientConfig),
		expose.NewExposeCommand(clientConfig),
		version.VersionCommand(clientConfig),
		imageupload.NewImageUploadCommand(clientConfig),
		optionsCmd,
	)
	return rootCmd
}

func Execute() {
	log.InitializeLogging("virtctl")
	if err := NewVirtctlCommand().Execute(); err != nil {
		fmt.Println(strings.TrimSpace(err.Error()))
		os.Exit(1)
	}
}
