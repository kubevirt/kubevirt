package templates

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/controller"
)

// UsageTemplate returns the usage template for all subcommands
func UsageTemplate() string {
	return `Usage:{{if .Runnable}}
  {{prepare .UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{prepare .CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{prepare .Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{prepare .Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Use "{{ProgramName}} options" for a list of global command-line options (applies to all commands).{{end}}
`
}

// MainUsageTemplate returns the usage template for the root command
func MainUsageTemplate() string {
	return `Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{prepare .Short}}{{end}}{{end}}

Use "{{ProgramName}} <command> --help" for more information about a given command.
Use "{{ProgramName}} options" for a list of global command-line options (applies to all commands).
`
}

// OptionsUsageTemplate returns a template which prints all global available commands
func OptionsUsageTemplate() string {
	return `The following options can be passed to any command:{{if .HasAvailableInheritedFlags}}

{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}
`
}

// ExactArgs validate the number of input parameters
func ExactArgs(nameOfCommand string, n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) != n {
			fmt.Fprintf(os.Stderr, "fatal: Number of input parameters is incorrect, %s accepts %d arg(s), received %d\n\n", nameOfCommand, n, len(args))
			cmd.Help()
			return errors.New("argument validation failed")
		}
		return nil
	}
}

// PrintWarningForPausedVMI prints warning message if VMI is paused
func PrintWarningForPausedVMI(virtCli kubecli.KubevirtClient, vmiName string, namespace string) {
	vmi, err := virtCli.VirtualMachineInstance(namespace).Get(vmiName, &k8smetav1.GetOptions{})
	if err != nil {
		return
	}
	condManager := controller.NewVirtualMachineInstanceConditionManager()
	if condManager.HasCondition(vmi, v1.VirtualMachineInstancePaused) {
		fmt.Fprintf(os.Stderr, "\rWarning: %s is paused. Console will be active after unpause.\n", vmiName)
	}
}
