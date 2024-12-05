package virtctl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/coreos/go-semver/semver"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	client_version "kubevirt.io/client-go/version"

	"kubevirt.io/kubevirt/pkg/virtctl/adm"
	"kubevirt.io/kubevirt/pkg/virtctl/configuration"
	"kubevirt.io/kubevirt/pkg/virtctl/console"
	"kubevirt.io/kubevirt/pkg/virtctl/create"
	"kubevirt.io/kubevirt/pkg/virtctl/credentials"
	"kubevirt.io/kubevirt/pkg/virtctl/expose"
	"kubevirt.io/kubevirt/pkg/virtctl/guestfs"
	"kubevirt.io/kubevirt/pkg/virtctl/imageupload"
	"kubevirt.io/kubevirt/pkg/virtctl/memorydump"
	"kubevirt.io/kubevirt/pkg/virtctl/pause"
	"kubevirt.io/kubevirt/pkg/virtctl/portforward"
	"kubevirt.io/kubevirt/pkg/virtctl/scp"
	"kubevirt.io/kubevirt/pkg/virtctl/softreboot"
	"kubevirt.io/kubevirt/pkg/virtctl/ssh"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
	"kubevirt.io/kubevirt/pkg/virtctl/unpause"
	"kubevirt.io/kubevirt/pkg/virtctl/usbredir"
	"kubevirt.io/kubevirt/pkg/virtctl/version"
	"kubevirt.io/kubevirt/pkg/virtctl/vm"
	"kubevirt.io/kubevirt/pkg/virtctl/vmexport"
	"kubevirt.io/kubevirt/pkg/virtctl/vnc"
)

var (
	NewVirtctlCommand = NewVirtctlCommandFn

	programName string
)

func NewVirtctlCommandFn() (*cobra.Command, clientcmd.ClientConfig) {

	programName := GetProgramName(filepath.Base(os.Args[0]))

	// used in cobra templates to display either `kubectl virt` or `virtctl`
	cobra.AddTemplateFunc(
		"ProgramName", func() string {
			return programName
		},
	)

	// used to enable replacement of `ProgramName` placeholder for cobra.Example, which has no template support
	cobra.AddTemplateFunc(
		"prepare", func(s string) string {
			// order matters!
			result := strings.Replace(s, "{{ProgramName}}", programName, -1)
			return result
		},
	)

	rootCmd := &cobra.Command{
		Use:           programName,
		Short:         programName + " controls virtual machine related operations on your kubernetes cluster.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf(cmd.UsageString())
		},
	}

	optionsCmd := &cobra.Command{
		Use:    "options",
		Hidden: true,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf(cmd.UsageString())
		},
	}
	optionsCmd.SetUsageTemplate(templates.OptionsUsageTemplate())
	//TODO: Add a ClientConfigFactory which allows substituting the KubeVirt client with a mock for unit testing
	clientConfig := kubecli.DefaultClientConfig(rootCmd.PersistentFlags())
	rootCmd.SetUsageTemplate(templates.MainUsageTemplate())
	rootCmd.SetOut(os.Stdout)
	rootCmd.AddCommand(
		configuration.NewListPermittedDevices(clientConfig),
		console.NewCommand(clientConfig),
		usbredir.NewCommand(clientConfig),
		vnc.NewCommand(clientConfig),
		scp.NewCommand(clientConfig),
		ssh.NewCommand(clientConfig),
		portforward.NewCommand(clientConfig),
		vm.NewStartCommand(clientConfig),
		vm.NewStopCommand(clientConfig),
		vm.NewRestartCommand(clientConfig),
		vm.NewMigrateCommand(clientConfig),
		vm.NewMigrateCancelCommand(clientConfig),
		vm.NewGuestOsInfoCommand(clientConfig),
		vm.NewUserListCommand(clientConfig),
		vm.NewFSListCommand(clientConfig),
		vm.NewAddVolumeCommand(clientConfig),
		vm.NewRemoveVolumeCommand(clientConfig),
		vm.NewExpandCommand(clientConfig),
		memorydump.NewMemoryDumpCommand(clientConfig),
		pause.NewCommand(clientConfig),
		unpause.NewCommand(clientConfig),
		softreboot.NewSoftRebootCommand(clientConfig),
		expose.NewCommand(clientConfig),
		version.VersionCommand(clientConfig),
		imageupload.NewImageUploadCommand(clientConfig),
		guestfs.NewGuestfsShellCommand(clientConfig),
		vmexport.NewVirtualMachineExportCommand(clientConfig),
		create.NewCommand(clientConfig),
		credentials.NewCommand(clientConfig),
		adm.NewCommand(clientConfig),
		optionsCmd,
	)
	return rootCmd, clientConfig
}

// GetProgramName returns the command name to display in help texts.
// If `virtctl` is installed via krew to be used as a kubectl plugin, it's run via a symlink, so the basename then
// is `kubectl-virt`. In this case we want to accommodate the user by adjusting the help text (usage, examples and
// the like) by displaying `kubectl virt <command>` instead of `virtctl <command>`.
// see https://github.com/kubevirt/kubevirt/issues/2356 for more details
// see also templates.go
func GetProgramName(binary string) string {
	if strings.HasSuffix(binary, "-virt") {
		return fmt.Sprintf("%s virt", strings.TrimSuffix(binary, "-virt"))
	}
	return "virtctl"
}

func Execute() int {
	log.InitializeLogging(programName)
	cmd, clientConfig := NewVirtctlCommand()
	if err := cmd.Execute(); err != nil {
		if versionErr := checkClientServerVersion(clientConfig); versionErr != nil {
			cmd.PrintErrln(versionErr)
		}
		cmd.PrintErrln(err)
		return 1
	}
	return 0
}

func checkClientServerVersion(clientConfig clientcmd.ClientConfig) error {
	clientSemVer, err := semver.NewVersion(strings.TrimPrefix(client_version.Get().GitVersion, "v"))
	if err != nil {
		return err
	}

	virtClient, err := kubecli.GetKubevirtClientFromClientConfig(clientConfig)
	if err != nil {
		return err
	}
	serverVersion, err := virtClient.ServerVersion().Get()
	if err != nil {
		return err
	}
	serverSemVer, err := semver.NewVersion(strings.TrimPrefix(serverVersion.GitVersion, "v"))
	if err != nil {
		return err
	}

	if clientSemVer.Major != serverSemVer.Major || clientSemVer.Minor != serverSemVer.Minor {
		return fmt.Errorf("You are using a client virtctl version that is different from the KubeVirt version running in the cluster\nClient Version: %s\nServer Version: %s\n", client_version.Get(), *serverVersion)
	}

	return nil
}
