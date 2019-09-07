package virtctl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/virtctl/console"
	"kubevirt.io/kubevirt/pkg/virtctl/expose"
	"kubevirt.io/kubevirt/pkg/virtctl/imageupload"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
	"kubevirt.io/kubevirt/pkg/virtctl/version"
	"kubevirt.io/kubevirt/pkg/virtctl/vm"
	"kubevirt.io/kubevirt/pkg/virtctl/vnc"
)

var programName string

const symlinkAsKrewPlugin = "kubectl-virt"

func NewVirtctlCommand() *cobra.Command {

	// if `virtctl` is installed via krew to be used as a kubectl plugin, it's run via a symlink, so the basename then
	// is `kubectl-virt`. In this case we want to accomodate the user by adjusting the help text (usage, examples and
	// the like) by displaying `kubectl virt <command>` instead of `virtctl <command>`.
	// see https://github.com/kubevirt/kubevirt/issues/2356 for more details
	// see also templates.go
	programName = "virtctl"
	if filepath.Base(os.Args[0]) == symlinkAsKrewPlugin {
		programName = "kubectl virt"
	}

	// used in cobra templates to display either `kubectl virt` or `virtctl`
	cobra.AddTemplateFunc("ProgramName", func() string { return programName })

	// used to enable replacement of `ProgramName` placeholder for cobra.Example, which has no template support
	cobra.AddTemplateFunc(
		"prepare", func(s string) string { return strings.Replace(s, "{{ProgramName}}", programName, -1) })

	rootCmd := &cobra.Command{
		Use:           programName,
		Short:         programName + " controls virtual machine related operations on your kubernetes cluster.",
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
		vm.NewMigrateCommand(clientConfig),
		expose.NewExposeCommand(clientConfig),
		version.VersionCommand(clientConfig),
		imageupload.NewImageUploadCommand(clientConfig),
		optionsCmd,
	)
	return rootCmd
}

func Execute() {
	log.InitializeLogging(programName)
	if err := NewVirtctlCommand().Execute(); err != nil {
		fmt.Println(strings.TrimSpace(err.Error()))
		os.Exit(1)
	}
}
