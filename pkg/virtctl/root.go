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
	completionCmd := &cobra.Command{
		Use:   "completion",
		Short: "Generates completion scripts for the specified shell",
		Args:  cobra.MaximumNArgs(1),
		Long: `To load completion run
. <(virtctl completion (SHELL TYPE))

To configure bash shell to load completions for each session add to bashrc

# ~/.bashrc or ~/.profile
. <(virtctl completion (SHELL TYPE))
`,
		Run: func(cmd *cobra.Command, args []string) {
			switch {
			case len(args) == 0:
				rootCmd.GenBashCompletion(os.Stdout)
			case args[0] == "bash":
				rootCmd.GenBashCompletion(os.Stdout)
			default:
				fmt.Fprintf(cmd.OutOrStderr(), "%q is not a supported shell", args[0])
			}
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
		completionCmd,
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
