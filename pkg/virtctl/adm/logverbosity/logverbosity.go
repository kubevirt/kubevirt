package logverbosity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/spf13/cobra"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

type Command struct{}

// for command parsing
const (
	// Verbosity must be 0-9.
	// https://kubernetes.io/docs/reference/kubectl/cheatsheet/#kubectl-output-verbosity-and-debugging
	minVerbosity = uint(0)
	maxVerbosity = uint(9)
	// noArg and NoFlag: Any number between 10 and MaxUint is fine.
	// Select less weird numbers, because these numbers will be shown in the help menu.
	// There is no option to hide the default value from the help menu.
	// See pflag.FlagUsages, which calls the FlagUsagesWrapped function:
	// https://github.com/kubevirt/kubevirt/blob/main/vendor/github.com/spf13/pflag/flag.go#L677
	noArg         = 10    // Default value if no argument specified
	NoFlag        = 11    // Default value if no flag is specified
	allComponents = "all" // Use in multiple places, so make it a constant
)

// for receiving the flag argument
var isReset bool

// Log verbosity can be set per KubeVirt component.
// https://kubevirt.io/user-guide/operations/debug/#setting-verbosity-per-kubevirt-component
// TODO: set verbosity per nodes
var virtComponents = map[string]*uint{
	"virt-api":        new(uint),
	"virt-controller": new(uint),
	"virt-handler":    new(uint),
	"virt-launcher":   new(uint),
	"virt-operator":   new(uint),
	allComponents:     new(uint),
}

// operation type of log-verbosity command
type operation int

const (
	show operation = iota
	set
	nop
)

// for patch operation
const (
	dcPath = "/spec/configuration/developerConfiguration"
	lvPath = "/spec/configuration/developerConfiguration/logVerbosity"
)

func NewCommand() *cobra.Command {
	c := Command{}
	cmd := &cobra.Command{
		Use:   "log-verbosity",
		Short: "Show, Set or Reset log verbosity. The verbosity value must be 0-9. The default cluster config is normally 2.\n",
		Long: `- To show the log verbosity of one or more components
  (when the log verbosity is unattended in the KubeVirt CR, show the default verbosity).
- To set the log verbosity of one or more components.
- To reset the log verbosity of all components
  (empty the log verbosity field, which means reset to the default verbosity).

- The components are <virt-api | virt-controller | virt-handler | virt-launcher | virt-operator>.
- Show and Set/Reset cannot coexist.
- The verbosity value must be 0-9. The default cluster config is normally 2.
- The verbosity value 10 is accepted but the operation is "show" instead of "set" (e.g. "--virt-api=10" = "--virt-api").
- Flag syntax must be "flag=arg" ("flag arg" not supported).`,
		Example: usage(),
		Args:    cobra.NoArgs,
		RunE:    c.RunE,
	}

	cmd.Flags().UintVar(virtComponents["virt-api"], "virt-api", NoFlag, "show/set virt-api log verbosity (0-9)")
	// A flag without an argument should only be used for boolean flags.
	// However, we want to use a flag without an argument (e.g. --virt-api) to show verbosity.
	// To do this, we set the default value (NoOptDefVal=noArg) when the flag has no argument.
	// Otherwise, the pflag package will return an error due to a missing argument.
	// The caveat is that there is no way to distinguish between user-specified 10 and NoOptDefVal=noArg after the following point.
	// https://github.com/kubevirt/kubevirt/blob/main/vendor/github.com/spf13/pflag/flag.go#L989
	// So, we accept "flag=10" but the operation is "show" instead of "set".
	cmd.Flags().Lookup("virt-api").NoOptDefVal = strconv.FormatUint(noArg, 10)

	cmd.Flags().UintVar(virtComponents["virt-controller"], "virt-controller", NoFlag, "show/set virt-controller log verbosity (0-9)")
	cmd.Flags().Lookup("virt-controller").NoOptDefVal = strconv.FormatUint(noArg, 10)

	cmd.Flags().UintVar(virtComponents["virt-handler"], "virt-handler", NoFlag, "show/set virt-handler log verbosity (0-9)")
	cmd.Flags().Lookup("virt-handler").NoOptDefVal = strconv.FormatUint(noArg, 10)

	cmd.Flags().UintVar(virtComponents["virt-launcher"], "virt-launcher", NoFlag, "show/set virt-launcher log verbosity (0-9)")
	cmd.Flags().Lookup("virt-launcher").NoOptDefVal = strconv.FormatUint(noArg, 10)

	cmd.Flags().UintVar(virtComponents["virt-operator"], "virt-operator", NoFlag, "show/set virt-operator log verbosity (0-9)")
	cmd.Flags().Lookup("virt-operator").NoOptDefVal = strconv.FormatUint(noArg, 10)

	cmd.Flags().UintVar(virtComponents[allComponents], allComponents, NoFlag, "show/set all component log verbosity (0-9)")
	cmd.Flags().Lookup(allComponents).NoOptDefVal = strconv.FormatUint(noArg, 10)

	cmd.Flags().BoolVar(&isReset, "reset", false, "reset log verbosity to the default verbosity (2) (empty the log verbosity)")

	// cannot specify "reset" and "all" flag at the same time
	cmd.MarkFlagsMutuallyExclusive("reset", allComponents)

	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

// Command line flag syntax:
//
//	OK: --flag=x
//	NG: --flag x
//
// Another caveat of NoOptDefVal is that
// we cannot use the "--flag x" syntax, because "--flag x" only applies to flags without a default value.
// See vendor/github.com/spf13/pflag/flag.go, especially note the order of the if clause below.
// https://github.com/kubevirt/kubevirt/blob/main/vendor/github.com/spf13/pflag/flag.go#L983C1-L998C3
func usage() string {
	return `  # reset (to default) log-verbosity for all components:
  {{ProgramName}} adm logVerbosity --reset

  # show log-verbosity for all components:
  {{ProgramName}} adm log-verbosity --all
  # set log-verbosity to 3 for all components:
  {{ProgramName}} adm log-verbosity --all=3

  # show log-verbosity for virt-handler:
  {{ProgramName}} adm log-verbosity --virt-handler
  # set log-verbosity to 7 for virt-handler:
  {{ProgramName}} adm log-verbosity --virt-handler=7

  # show log-verbosity for virt-handler and virt-launcher:
  {{ProgramName}} adm log-verbosity --virt-handler --virt-launcher
  # set log-verbosity for virt-handler to 7 and virt-launcher to 3:
  {{ProgramName}} adm log-verbosity --virt-handler=7 --virt-launcher=3

  # reset all components to default besides virt-handler which is 7:
  {{ProgramName}} adm log-verbosity --reset --virt-handler=7
  # set all components to 3 besides virt-handler which is 7:
  {{ProgramName}} adm log-verbosity --all=3 --virt-handler=7`
}

// component name to JSON name
func getJSONNameByComponentName(componentName string) string {
	componentNameToJSONName := map[string]string{
		"virt-api":        "virtAPI",
		"virt-controller": "virtController",
		"virt-handler":    "virtHandler",
		"virt-launcher":   "virtLauncher",
		"virt-operator":   "virtOperator",
		allComponents:     allComponents,
	}
	return componentNameToJSONName[componentName]
}

func detectInstallNamespaceAndName(virtClient kubecli.KubevirtClient) (namespace, name string, err error) {
	kvs, err := virtClient.KubeVirt(k8smetav1.NamespaceAll).List(context.Background(), k8smetav1.ListOptions{})
	if err != nil {
		return "", "", fmt.Errorf("could not list KubeVirt CRs across all namespaces: %v", err)
	}
	if len(kvs.Items) == 0 {
		return "", "", errors.New("could not detect a KubeVirt installation")
	}
	if len(kvs.Items) > 1 {
		return "", "", errors.New("invalid kubevirt installation, more than one KubeVirt resource found")
	}
	namespace = kvs.Items[0].Namespace
	name = kvs.Items[0].Name
	return namespace, name, nil
}

func hasVerbosityInKV(kv *v1.KubeVirt) (verbosityMap map[string]uint, hasDeveloperConfiguration bool, err error) {
	verbosityMap = map[string]uint{} // key: component name, value: verbosity
	hasDeveloperConfiguration = true

	if kv.Spec.Configuration.DeveloperConfiguration == nil {
		// If DeveloperConfiguration is absent in the KubeVirt CR, need to add it before adding LogVerbosity.
		// So set the hasDeveloperConfiguration flag to false.
		hasDeveloperConfiguration = false
	} else if kv.Spec.Configuration.DeveloperConfiguration.LogVerbosity != nil {
		// If LogVerbosity is present in the KubeVirt CR,
		// get the logVerbosity field, and put it to verbosityMap.
		lvJSON, err := json.Marshal(kv.Spec.Configuration.DeveloperConfiguration.LogVerbosity)
		if err != nil {
			return nil, hasDeveloperConfiguration, err
		}
		if err := json.Unmarshal(lvJSON, &verbosityMap); err != nil {
			return nil, hasDeveloperConfiguration, err
		}
	}

	return verbosityMap, hasDeveloperConfiguration, nil
}

func createOutputLines(verbosityVal map[string]uint) []string {
	var lines []string

	allIsSet := *virtComponents[allComponents] != NoFlag

	for componentName, verbosity := range virtComponents {
		if componentName == allComponents {
			continue
		}
		JSONName := getJSONNameByComponentName(componentName)
		if *verbosity != NoFlag || allIsSet {
			line := fmt.Sprintf("%s=%d", componentName, verbosityVal[JSONName])
			lines = append(lines, line)
		}
	}

	// output message sorted by lexicographical order of component name
	sort.Strings(lines)

	return lines
}

func createShowMessage(currentLv map[string]uint) []string {
	// fill the unattended verbosity with default verbosity
	// key: JSONName, value: verbosity
	verbosityVal := map[string]uint{
		"virtAPI":        virtconfig.DefaultVirtAPILogVerbosity,
		"virtController": virtconfig.DefaultVirtControllerLogVerbosity,
		"virtHandler":    virtconfig.DefaultVirtHandlerLogVerbosity,
		"virtLauncher":   virtconfig.DefaultVirtLauncherLogVerbosity,
		"virtOperator":   virtconfig.DefaultVirtOperatorLogVerbosity,
	}

	// update the verbosity based on the existing verbosity in the KubeVirt CR
	for key, value := range currentLv {
		verbosityVal[key] = value
	}

	lines := createOutputLines(verbosityVal)

	return lines
}

func setVerbosity(currentLv map[string]uint) {
	// update currentLv based on the user-specified verbosity for all components
	if *virtComponents[allComponents] != NoFlag {
		for componentName := range virtComponents {
			if componentName == allComponents {
				continue
			}
			JSONName := getJSONNameByComponentName(componentName)
			currentLv[JSONName] = *virtComponents[allComponents]
		}
	}

	// update currentLv based on the user-specified verbosity for each component
	for componentName, verbosity := range virtComponents {
		if componentName == allComponents || *verbosity == NoFlag {
			continue
		}
		JSONName := getJSONNameByComponentName(componentName)
		currentLv[JSONName] = *verbosity
	}
}

func createPatch(currentLv map[string]uint, hasDeveloperConfiguration bool) ([]byte, error) {
	patchSet := patch.New()

	// reset only if verbosity exists, otherwise do nothing
	if isReset && len(currentLv) != 0 {
		if !hasDeveloperConfiguration {
			// if DeveloperConfiguration is absent, add DeveloperConfiguration first
			patchSet.AddOption(patch.WithAdd(dcPath, v1.DeveloperConfiguration{}))
			hasDeveloperConfiguration = true
		}
		// add an empty object
		currentLv = map[string]uint{}
		patchSet.AddOption(patch.WithAdd(lvPath, currentLv))
	}

	setVerbosity(currentLv)

	// in case of just reset (no set operation after the reset), don't need to add another patch
	if len(currentLv) != 0 {
		if !hasDeveloperConfiguration {
			// if DeveloperConfiguration is absent, add DeveloperConfiguration first
			patchSet.AddOption(patch.WithAdd(dcPath, &v1.DeveloperConfiguration{}))
		}
		patchSet.AddOption(patch.WithAdd(lvPath, currentLv))
	}
	if patchSet.IsEmpty() {
		return nil, nil
	}

	return patchSet.GeneratePayload()
}

func findOperation(cmd *cobra.Command) (operation, error) {
	isShow, isSet := false, false

	for componentName, verbosity := range virtComponents {
		// check if the flag for the component is specified
		// cannot use NoFlag to check, because user can accidentally specify the same number as NoFlag for the verbosity
		if !cmd.Flags().Changed(componentName) {
			continue
		}

		// if flag is specified, it means either set or show
		// if the value = noArg, it means show
		// if the value != noArg, it means set
		isShow = isShow || *verbosity == noArg
		isSet = isSet || *verbosity != noArg

		// check whether the verbosity is in the range
		// Note that noArg is acceptable but operation is show instead of set.
		if *verbosity != noArg && *verbosity > maxVerbosity {
			return nop, fmt.Errorf("%s: log verbosity must be %d-%d", componentName, minVerbosity, maxVerbosity)
		}
	}

	// do not distinguish between set and reset at this point, because set and reset can coexist
	if isReset {
		isSet = true
	}

	switch {
	case isShow && isSet:
		return nop, errors.New("only show or set is allowed")
	case isShow:
		return show, nil
	case isSet:
		return set, nil
	default:
		return nop, nil
	}
}

func (c *Command) RunE(cmd *cobra.Command, _ []string) error {
	virtClient, _, _, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return err
	}
	namespace, name, err := detectInstallNamespaceAndName(virtClient)
	if err != nil {
		return err
	}
	kv, err := virtClient.KubeVirt(namespace).Get(context.Background(), name, k8smetav1.GetOptions{})
	if err != nil {
		return err
	}

	// check the operation type (nop/show/set)
	op, err := findOperation(cmd)
	if err != nil {
		return err
	}

	switch op {
	case nop:
		if err := cmd.Help(); err != nil {
			return err
		}
		return errors.New("no flag specified - expecting at least one flag")
	case show:
		// if verbosity has been set in the KubeVirt CR, use the verbosity
		currentLv, _, err := hasVerbosityInKV(kv)
		if err != nil {
			return err
		}
		lines := createShowMessage(currentLv)
		for _, line := range lines {
			cmd.Println(line)
		}
	case set: // set and/or reset
		// "Add" patch removes the value if we do not specify the value, even if we do not change the existing value.
		// So, we need to get the existing verbosity in the KubeVirt CR.
		// Also, "Add" patch needs a DeveloperConfiguration entry before adding a LogVerbosity entry.
		// So, we need to know if DeveloperConfiguration is present or absent.
		currentLv, hasDeveloperConfiguration, err := hasVerbosityInKV(kv)
		if err != nil {
			return err
		}
		patchData, err := createPatch(currentLv, hasDeveloperConfiguration)
		if err != nil {
			return err
		}
		if patchData == nil {
			return nil
		}
		_, err = virtClient.KubeVirt(namespace).Patch(context.Background(), name, types.JSONPatchType, patchData, k8smetav1.PatchOptions{})
		if err != nil {
			return err
		}
		cmd.Println("successfully set/reset the log verbosity")
	default:
		return fmt.Errorf("op: an unknown operation: %v", op)
	}

	return nil
}
