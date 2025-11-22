package templates

import (
	"context"
	"fmt"
	"os"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/controller"
)

const (
	// SectionVars is the help template section that declares variables to be used in the template.
	SectionVars = `{{$rootCmd := rootCmd .}}` +
		`{{$visibleFlags := visibleFlags (flagsNotIntersected .LocalFlags .PersistentFlags)}}` +
		`{{$optionsCmdFor := optionsCmdFor .}}` +
		`{{$usageLine := usageLine .}}` +
		`{{$reverseParentsNames := reverseParentsNames .}}`

	// SectionUsage is the help template section that displays the command's usage.
	SectionUsage = `{{if and .Runnable (ne .UseLine "") (ne .UseLine $rootCmd)}}Usage:
  {{trimRight $usageLine}}

{{end}}`

	// SectionExamples is the help template section that displays command examples.
	SectionExamples = `{{if .HasExample}}Examples:
{{prepare (trimRight .Example)}}

{{end}}`

	// SectionsFlags is the help template section that displays command flags.
	SectionFlags = `{{ if $visibleFlags.HasFlags }}Flags:
{{ trimRight (flagsUsages $visibleFlags) }}

{{ end }}`

	// SectionSubcommands is the help template section that displays the command's subcommands.
	SectionSubcommands = `{{if .HasAvailableSubCommands}}{{cmdGroupsString .}}

{{end}}`

	// SectionTipsHelp is the help template section that displays the '--help' hint.
	SectionTipsHelp = `{{if .HasSubCommands}}Use "{{range $reverseParentsNames}}{{.}} {{end}}<command> --help" for more information about a given command.
{{end}}`

	// SectionTipsGlobalOptions is the help template section that displays the 'options' hint for displaying global flags.
	SectionTipsGlobalOptions = `{{if $optionsCmdFor}}Use "{{$optionsCmdFor}}" for a list of global command-line options (applies to all commands).
{{end}}`

	// MainUsageTemplate is the usage template for the root command.
	MainUsageTemplate = "\n\n" +
		SectionVars +
		SectionExamples +
		SectionSubcommands +
		SectionFlags +
		SectionUsage +
		SectionTipsHelp +
		SectionTipsGlobalOptions

	// MainHelpTemplate is the help template for the root command.
	MainHelpTemplate = `{{with or .Long .Short }}{{. | trim}}{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`

	// OptionsUsageTemplate is the usage template for the options command.
	OptionsUsageTemplate = `{{ if .HasInheritedFlags}}The following options can be passed to any command:
{{flagsUsages .InheritedFlags}}

{{end}}`
)

// PrintWarningForPausedVMI prints warning message if VMI is paused
func PrintWarningForPausedVMI(virtCli kubecli.KubevirtClient, vmiName string, namespace string) {
	vmi, err := virtCli.VirtualMachineInstance(namespace).Get(context.Background(), vmiName, k8smetav1.GetOptions{})
	if err != nil {
		return
	}
	condManager := controller.NewVirtualMachineInstanceConditionManager()
	if condManager.HasCondition(vmi, v1.VirtualMachineInstancePaused) {
		fmt.Fprintf(os.Stderr, "\rWarning: %s is paused. Console will be active after unpause.\n", vmiName)
	}
}
