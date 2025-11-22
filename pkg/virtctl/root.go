package virtctl

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/coreos/go-semver/semver"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	client_version "kubevirt.io/client-go/version"

	"kubevirt.io/kubevirt/pkg/virtctl/adm"
	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
	"kubevirt.io/kubevirt/pkg/virtctl/configuration"
	"kubevirt.io/kubevirt/pkg/virtctl/console"
	"kubevirt.io/kubevirt/pkg/virtctl/create"
	"kubevirt.io/kubevirt/pkg/virtctl/credentials"
	"kubevirt.io/kubevirt/pkg/virtctl/expose"
	"kubevirt.io/kubevirt/pkg/virtctl/guestfs"
	"kubevirt.io/kubevirt/pkg/virtctl/imageupload"
	"kubevirt.io/kubevirt/pkg/virtctl/memorydump"
	"kubevirt.io/kubevirt/pkg/virtctl/objectgraph"
	"kubevirt.io/kubevirt/pkg/virtctl/pause"
	"kubevirt.io/kubevirt/pkg/virtctl/portforward"
	"kubevirt.io/kubevirt/pkg/virtctl/reset"
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

	programName = GetProgramName(filepath.Base(os.Args[0]))
)

// GetProgramName returns the command name to display in help texts.
// If `virtctl` is installed via krew to be used as a kubectl plugin, it's run via a symlink, so the basename then
// is `kubectl-virt`. In this case we want to accommodate the user by adjusting the help text (usage, examples and
// the like) by displaying `kubectl virt <command>` instead of `virtctl <command>`.
// see https://github.com/kubevirt/kubevirt/issues/2356 for more details
// see also templates.go
func GetProgramName(binary string) string {
	if strings.HasSuffix(binary, "-virt") {
		return strings.TrimSuffix(binary, "-virt") + " virt"
	}
	return "virtctl"
}

func NewOptionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "options",
		Hidden: true,
		Run: func(cmd *cobra.Command, _ []string) {
			cmd.Usage()
		},
	}
	templates.UseOptionsTemplates(cmd)
	return cmd
}

func NewVirtctlCommandFn() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           programName,
		Short:         programName + " controls virtual machine related operations on your kubernetes cluster.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Run: func(cmd *cobra.Command, _ []string) {
			cmd.Help()
		},
	}
	addVerbosityFlag(rootCmd.PersistentFlags())
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetContext(clientconfig.NewContext(
		context.Background(), kubecli.DefaultClientConfig(rootCmd.PersistentFlags()),
	))

	groups := templates.CommandGroups{
		{
			Message: "Virtual Machine Lifecycle Commands:",
			Commands: []*cobra.Command{
				vm.NewStartCommand(),
				vm.NewStopCommand(),
				vm.NewRestartCommand(),
				pause.NewCommand(),
				unpause.NewCommand(),
				softreboot.NewSoftRebootCommand(),
				reset.NewResetCommand(),
			},
		},
		{
			Message: "Virtual Machine Connectivity Commands:",
			Commands: []*cobra.Command{
				expose.NewCommand(),
				ssh.NewCommand(),
				scp.NewCommand(),
				portforward.NewCommand(),
				console.NewCommand(),
				vnc.NewCommand(),
			},
		},
		{
			Message: "Virtual Machine Volume Commands:",
			Commands: []*cobra.Command{
				vm.NewAddVolumeCommand(),
				vm.NewRemoveVolumeCommand(),
				imageupload.NewImageUploadCommand(),
			},
		},
		{
			Message: "Virtual Machine Migration Commands:",
			Commands: []*cobra.Command{
				vm.NewMigrateCommand(),
				vm.NewMigrateCancelCommand(),
				vm.NewEvacuateCancelCommand(),
			},
		},
		{
			Message: "Virtual Machine Guest Commands:",
			Commands: []*cobra.Command{
				vm.NewGuestOsInfoCommand(),
				vm.NewUserListCommand(),
				vm.NewFSListCommand(),
				credentials.NewCommand(),
			},
		},
		{
			Message: "Utility Commands:",
			Commands: []*cobra.Command{
				configuration.NewListPermittedDevices(),
				usbredir.NewCommand(),
				guestfs.NewGuestfsShellCommand(),
				vm.NewExpandCommand(),
				memorydump.NewMemoryDumpCommand(),
				vmexport.NewVirtualMachineExportCommand(),
				objectgraph.NewCommand(),
				create.NewCommand(),
				adm.NewCommand(),
			},
		},
	}

	groups.Add(rootCmd)
	filters := []string{"options"}

	templates.ActsAsRootCommand(rootCmd, filters, programName, groups...)

	rootCmd.AddCommand(NewOptionsCmd())
	rootCmd.AddCommand(version.VersionCommand())

	return rootCmd
}

func addVerbosityFlag(fs *pflag.FlagSet) {
	// The verbosity flag is added to the default flag set
	// by init() in staging/src/kubevirt.io/client-go/log/log.go.
	// We re-add it here to make it available in virtctl commands.
	if f := flag.CommandLine.Lookup("v"); f != nil {
		fs.AddFlag(pflag.PFlagFromGoFlag(f))
	} else {
		panic("failed to find verbosity flag \"v\" in default flag set")
	}
}

func Execute() int {
	log.InitializeLogging(programName)
	cmd := NewVirtctlCommand()
	if err := cmd.Execute(); err != nil {
		if versionErr := checkClientServerVersion(cmd.Context()); versionErr != nil {
			cmd.PrintErrln(versionErr)
		}
		cmd.PrintErrln(err)
		return 1
	}
	return 0
}

func checkClientServerVersion(ctx context.Context) error {
	clientSemVer, err := semver.NewVersion(strings.TrimPrefix(client_version.Get().GitVersion, "v"))
	if err != nil {
		return err
	}

	virtClient, _, _, err := clientconfig.ClientAndNamespaceFromContext(ctx)
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
