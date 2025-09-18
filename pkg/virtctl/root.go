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

func NewVirtctlCommandFn() *cobra.Command {
	// used in cobra templates to display either `kubectl virt` or `virtctl`
	cobra.AddTemplateFunc(
		"ProgramName", func() string {
			return programName
		},
	)

	// used to enable replacement of `ProgramName` placeholder for cobra.Example, which has no template support
	cobra.AddTemplateFunc(
		"prepare", func(s string) string {
			return strings.Replace(s, "{{ProgramName}}", programName, -1)
		},
	)

	optionsCmd := &cobra.Command{
		Use:    "options",
		Hidden: true,
		Run: func(cmd *cobra.Command, _ []string) {
			cmd.Printf(cmd.UsageString())
		},
	}
	optionsCmd.SetUsageTemplate(templates.OptionsUsageTemplate())

	rootCmd := &cobra.Command{
		Use:           programName,
		Short:         programName + " controls virtual machine related operations on your kubernetes cluster.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Run: func(cmd *cobra.Command, _ []string) {
			cmd.Printf(cmd.UsageString())
		},
	}
	addVerbosityFlag(rootCmd.PersistentFlags())
	rootCmd.SetUsageTemplate(templates.MainUsageTemplate())
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetContext(clientconfig.NewContext(
		context.Background(), kubecli.DefaultClientConfig(rootCmd.PersistentFlags()),
	))

	rootCmd.AddCommand(
		configuration.NewListPermittedDevices(),
		console.NewCommand(),
		usbredir.NewCommand(),
		vnc.NewCommand(),
		scp.NewCommand(),
		ssh.NewCommand(),
		portforward.NewCommand(),
		vm.NewStartCommand(),
		vm.NewStopCommand(),
		vm.NewRestartCommand(),
		vm.NewMigrateCommand(),
		vm.NewMigrateCancelCommand(),
		vm.NewGuestOsInfoCommand(),
		vm.NewUserListCommand(),
		vm.NewFSListCommand(),
		vm.NewAddVolumeCommand(),
		vm.NewRemoveVolumeCommand(),
		vm.NewExpandCommand(),
		memorydump.NewMemoryDumpCommand(),
		pause.NewCommand(),
		unpause.NewCommand(),
		softreboot.NewSoftRebootCommand(),
		reset.NewResetCommand(),
		expose.NewCommand(),
		version.VersionCommand(),
		imageupload.NewImageUploadCommand(),
		guestfs.NewGuestfsShellCommand(),
		vmexport.NewVirtualMachineExportCommand(),
		create.NewCommand(),
		credentials.NewCommand(),
		adm.NewCommand(),
		objectgraph.NewCommand(),
		optionsCmd,
	)

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

// Unit Test fix. To be addressed at the upstream Kubevirt repo
func checkClientServerVersion(ctx context.Context) error {
	raw_version := client_version.Get().GitVersion
	var clientSemVer *semver.Version
	if raw_version != "" {
		var err error
		raw_version = strings.TrimPrefix(raw_version, "v")
		clientSemVer, err = semver.NewVersion(raw_version)
		if err != nil {
			return err
		}
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

	if clientSemVer == nil || clientSemVer.Major != serverSemVer.Major || clientSemVer.Minor != serverSemVer.Minor {
		return fmt.Errorf("You are using a client virtctl version that is different from the KubeVirt version running in the cluster\nClient Version: %s\nServer Version: %s\n", client_version.Get(), *serverVersion)
	}

	return nil
}
