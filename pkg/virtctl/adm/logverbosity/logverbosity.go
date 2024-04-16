package logverbosity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/spf13/cobra"

	"k8s.io/client-go/tools/clientcmd"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

type Command struct {
	clientConfig clientcmd.ClientConfig
	command      string
}

// constants and variables for command parsing
const (
	// Verbosity must be 0-9.
	// https://kubernetes.io/docs/reference/kubectl/cheatsheet/#kubectl-output-verbosity-and-debugging
	minVerbosity  = uint(0)
	maxVerbosity  = uint(9)
	noArg         = "show" // default value if no argument specified
	NoFlag        = ""     // default value if no flag is specified (exposed to the test file)
	allComponents = "all"  // all KubeVirt components
)

var virtComponents = map[string]*string{
	"virt-api":        new(string),
	"virt-controller": new(string),
	"virt-handler":    new(string),
	"virt-launcher":   new(string),
	"virt-operator":   new(string),
	allComponents:     new(string),
}

var resetNames = []string{}
var vmNames = []string{}
var vmLevels = []string{}

type vmProperties struct {
	name  string
	level string
	obj   *v1.VirtualMachine
}

// operation type of log-verbosity command
type operation int

const (
	show operation = iota
	set
	nop
)

// constants for patch operation
// for the details of the patch,
// see https://www.rfc-editor.org/rfc/rfc6902
const (
	patchAdd    = patch.PatchAddOp
	patchRemove = patch.PatchRemoveOp
	dcPath      = "/spec/configuration/developerConfiguration"
	lvPath      = "/spec/configuration/developerConfiguration/logVerbosity"
	labelPath   = "/spec/template/metadata/labels/logVerbosity"
)

func NewCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "log-verbosity",
		Short: "Show, Set or Reset the log verbosity. The verbosity value must be 0-9. The default cluster config is normally 2.\n",
		Long: `- To show the log verbosity of one or more KubeVirt components and/or one or more VMs
- To set the log verbosity of one or more KubeVirt components and/or one or more VMs
- To reset the log verbosity of all KubeVirt components and/or one or more VMs
  - For KubeVirt components, empty the log verbosity field, i.e. reset to the default verbosity
  - For VMs, remove the logVerbosity label from the VMs

- The KubeVirt components are <virt-api | virt-controller | virt-handler | virt-launcher | virt-operator>.
- Show and Set/Reset cannot coexist.
- The verbosity value must be 0-9. The default cluster config is normally 2.
- For the KubeVirt components, flag syntax must be "flag=arg" ("flag arg" not supported).
- For the VM, the new verbosity is applied when the VM is (re)started.
  - When the VM is not running, show the verbosity when the VM starts running.
  - When the VM is running, show the verbosity of the currently running VMI, even if the new verbosity was set after the VMI started.`,
		Example: usage(),
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c := Command{command: "log-verbosity", clientConfig: clientConfig}
			return c.RunE(cmd)
		},
	}

	// A flag without an argument should only be used for boolean flags.
	// However, we want to use a flag without an argument (e.g. --virt-api) to show virt component verbosity.
	// To do this, we set the default value (NoOptDefVal=noArg) when the flag has no argument.
	// Otherwise, the pflag package will return an error due to a missing argument.
	cmd.Flags().StringVar(virtComponents["virt-api"], "virt-api", NoFlag, "show/set the log verbosity (0-9) of virt-api")
	cmd.Flags().Lookup("virt-api").NoOptDefVal = noArg

	cmd.Flags().StringVar(virtComponents["virt-controller"], "virt-controller", NoFlag, "show/set the log verbosity (0-9) of virt-controller")
	cmd.Flags().Lookup("virt-controller").NoOptDefVal = noArg

	cmd.Flags().StringVar(virtComponents["virt-handler"], "virt-handler", NoFlag, "show/set the log verbosity (0-9) of virt-handler")
	cmd.Flags().Lookup("virt-handler").NoOptDefVal = noArg

	cmd.Flags().StringVar(virtComponents["virt-launcher"], "virt-launcher", NoFlag, "show/set the log verbosity (0-9) of virt-launcher")
	cmd.Flags().Lookup("virt-launcher").NoOptDefVal = noArg

	cmd.Flags().StringVar(virtComponents["virt-operator"], "virt-operator", NoFlag, "show/set the log verbosity (0-9) of virt-operator")
	cmd.Flags().Lookup("virt-operator").NoOptDefVal = noArg

	cmd.Flags().StringVar(virtComponents[allComponents], allComponents, NoFlag, "show/set the log verbosity (0-9) of all KubeVirt components")
	cmd.Flags().Lookup(allComponents).NoOptDefVal = noArg

	cmd.Flags().StringSliceVar(&resetNames, "reset", []string{}, "reset the log verbosity of all KubeVirt components to the default verbosity (2) and remove the logVerbosity label in the VM object")
	cmd.Flags().Lookup("reset").NoOptDefVal = allComponents

	// VM related flags always have an argument.
	cmd.Flags().StringSliceVar(&vmNames, "vm", []string{}, "VM name")

	cmd.Flags().StringSliceVar(&vmLevels, "level", []string{}, "the log verbosity (0-9) for a VM")

	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

// Command line flag syntax (for KubeVirt components):
//
//	OK: --flag=x
//	NG: --flag x
//
// A caveat of NoOptDefVal is that
// we cannot use the "--flag x" syntax, because "--flag x" only applies to flags without a default value.
// See https://github.com/kubevirt/kubevirt/blob/main/vendor/github.com/spf13/pflag/flag.go#L983C1-L998C3
func usage() string {
	return `  1. KubeVirt component
  # show the log verbosity of virt-handler and virt-launcher:
  {{ProgramName}} adm log-verbosity --virt-handler --virt-launcher
  # show the log verbosity of all components:
  {{ProgramName}} adm log-verbosity --all  

  # set the log verbosity for virt-handler to 7 and virt-launcher to 3:
  {{ProgramName}} adm log-verbosity --virt-handler=7 --virt-launcher=3
  # set the log verbosity for all components to 3:
  {{ProgramName}} adm log-verbosity --all=3
  # set all components to 3 besides virt-handler which is 7:
  {{ProgramName}} adm log-verbosity --all=3 --virt-handler=7

  # reset the log verbosity for all components (to default):
  {{ProgramName}} adm logVerbosity --reset
  # reset all components to default besides virt-handler which is 7:
  {{ProgramName}} adm log-verbosity --reset --virt-handler=7

  2. VM
  # show the log verbosity of testvm1 and testvm2
  {{ProgramName}} adm log-verbosity --vm=testvm1 --vm=testvm2
  
  # set the log verbosity for testvm1 to 5 and for testvm2 to 6
  {{ProgramName}} adm log-verbosity --vm=testvm1 --level=5 --vm=testvm2 --level=6
  
  # reset (remove the logVerbosity label from) testvm1 and testvm2
  {{ProgramName}} adm log-verbosity --reset=testvm1 --reset=testvm2
  # reset (remove the logVerbosity label from) testvm1 and set log-verbosity for testvm2 to 5
  {{ProgramName}} adm log-verbosity --reset=testvm1 --vm=testvm2 --level=5
  
  3. both KubeVirt component and VM
  # show the log verbosity of virt-api and testvm
  {{ProgramName}} adm log-verbosity --virt-api --vm=testvm
  # show the log verbosity of all virt components and testvm
  {{ProgramName}} adm log-verbosity --all --vm=testvm
  
  # set the log verbosity for virt-api to 3 and testvm to 5
  {{ProgramName}} adm log-verbosity --virt-api=3 --vm=testvm --level=5
  # set the log verbosity for all components to 3 and testvm to 5
  {{ProgramName}} adm log-verbosity --all=3 --vm=testvm --level=5
  # reset the log verbosity of all components (to default) and (remove the logVerbosity label from) testvm
  {{ProgramName}} adm log-verbosity --reset --reset=testvm
  # reset the log verbosity of all components (to default) and set the log verbosity for testvm to 5
  {{ProgramName}} adm log-verbosity --reset --vm=testvm --level=5`
}

// component name to JSON name
func getJSONNameByComponentName(componentName string) string {
	var componentNameToJSONName = map[string]string{
		"virt-api":        "virtAPI",
		"virt-controller": "virtController",
		"virt-handler":    "virtHandler",
		"virt-launcher":   "virtLauncher",
		"virt-operator":   "virtOperator",
		allComponents:     allComponents,
	}
	return componentNameToJSONName[componentName]
}

func detectInstallNamespaceAndName(virtClient kubecli.KubevirtClient) (string, string, error) {
	kvs, err := virtClient.KubeVirt(k8smetav1.NamespaceAll).List(&k8smetav1.ListOptions{})
	if err != nil {
		return "", "", fmt.Errorf("could not list KubeVirt CRs across all namespaces: %v", err)
	}
	if len(kvs.Items) == 0 {
		return "", "", errors.New("could not detect a KubeVirt installation")
	}
	if len(kvs.Items) > 1 {
		return "", "", errors.New("invalid kubevirt installation, more than one KubeVirt resource found")
	}
	namespace := kvs.Items[0].Namespace
	name := kvs.Items[0].Name
	return namespace, name, nil
}

func hasVerbosityInKV(kv *v1.KubeVirt) (map[string]uint, bool, error) {
	verbosityMap := map[string]uint{} // key: component name, value: verbosity
	hasDeveloperConfiguration := true

	if kv.Spec.Configuration.DeveloperConfiguration == nil {
		// If DeveloperConfiguration is absent in the KubeVirt CR, need to add it before adding LogVerbosity.
		// Set the hasDeveloperConfiguration flag to false.
		hasDeveloperConfiguration = false
	} else if kv.Spec.Configuration.DeveloperConfiguration.LogVerbosity != nil {
		// If LogVerbosity is present in the KubeVirt CR, get the logVerbosity field, and put it to verbosityMap.
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

func createKvOutputLines(lines *[]string, verbosityVal map[string]uint) {
	allIsSet := (*virtComponents[allComponents] != NoFlag)
	for componentName, verbosity := range virtComponents {
		if componentName == allComponents {
			continue
		}
		JSONName := getJSONNameByComponentName(componentName)
		if *verbosity != NoFlag || allIsSet {
			line := fmt.Sprintf("%s = %d", componentName, verbosityVal[JSONName])
			*lines = append(*lines, line)
		}
	}
	// virt component output message sorted by lexicographical order of component name
	sort.Strings(*lines)
}

func createVMOutputLines(lines *[]string, verbosityVal map[string]uint, vms []vmProperties) error {
	for _, vm := range vms {
		line := fmt.Sprintf("%s = %d", vm.name, verbosityVal[vm.name])
		*lines = append(*lines, line)

		// When the VM is running, return the verbosity set on the running VM.
		// Get the verbosity set on the running VM, and if it is different from the verbosity of the VM object, add a warning message
		obj := vm.obj
		var val uint
		var err error
		if verbosity, exist := obj.Spec.Template.ObjectMeta.Labels["logVerbosity"]; exist {
			// if label is specified, use the label in the vm object
			if val, err = atou(verbosity); err != nil {
				return err
			}
		} else {
			// if label is not specified, use the virt-launcher log verbosity
			// Note: verbosityVal[] always has the "virt-launcher" key
			val = verbosityVal[getJSONNameByComponentName("virt-launcher")]
		}
		if verbosityVal[vm.name] != val {
			line := fmt.Sprintf("%s: updated verbosity %d will be applied after the VM is restarted", vm.name, val)
			*lines = append(*lines, line)
		}
	}
	return nil
}

func createOutputLines(verbosityVal map[string]uint, vms []vmProperties) ([]string, error) {
	var lines []string

	// for virt component output message
	createKvOutputLines(&lines, verbosityVal)

	// for vm output message
	if err := createVMOutputLines(&lines, verbosityVal, vms); err != nil {
		return nil, err
	}

	return lines, nil
}

func createShowMessage(currentLv map[string]uint, vms []vmProperties) ([]string, error) {
	// fill the unattended verbosity with default verbosity
	// key: JSONName, value: verbosity
	var verbosityVal = map[string]uint{
		"virtAPI":        virtconfig.DefaultVirtAPILogVerbosity,
		"virtController": virtconfig.DefaultVirtControllerLogVerbosity,
		"virtHandler":    virtconfig.DefaultVirtHandlerLogVerbosity,
		"virtLauncher":   virtconfig.DefaultVirtLauncherLogVerbosity,
		"virtOperator":   virtconfig.DefaultVirtOperatorLogVerbosity,
	}

	for _, vm := range vms {
		verbosityVal[vm.name] = virtconfig.DefaultVirtLauncherLogVerbosity
	}

	// update the verbosity based on the existing verbosity in the KubeVirt CR and the logVerbosity label of VM
	for key, value := range currentLv {
		verbosityVal[key] = value
	}

	lines, err := createOutputLines(verbosityVal, vms)
	if err != nil {
		return nil, err
	}

	return lines, nil
}

func addPatch(patchData *[]patch.PatchOperation, op string, path string, value interface{}) {
	*patchData = append(*patchData, patch.PatchOperation{
		Op:    op,
		Path:  path,
		Value: value,
	})
}

func createKvComponentPatch(currentLv map[string]uint, hasDeveloperConfiguration *bool, patchData *[]patch.PatchOperation) {
	// update currentLv based on the user-specified verbosity for all components
	if *virtComponents[allComponents] != NoFlag {
		for componentName := range virtComponents {
			if componentName == allComponents {
				continue
			}
			JSONName := getJSONNameByComponentName(componentName)
			currentLv[JSONName], _ = atou(*virtComponents[allComponents])
		}
	}

	// update currentLv based on the user-specified verbosity for each component
	for componentName, verbosity := range virtComponents {
		if componentName == allComponents || *verbosity == NoFlag {
			continue
		}
		JSONName := getJSONNameByComponentName(componentName)
		currentLv[JSONName], _ = atou(*verbosity)
	}

	// in case of just reset (no set operation after the reset), don't need to add another patch
	if !*hasDeveloperConfiguration {
		// if DeveloperConfiguration is absent, add DeveloperConfiguration first
		addPatch(patchData, patchAdd, dcPath, &v1.DeveloperConfiguration{})
	}
	addPatch(patchData, patchAdd, lvPath, currentLv)
}

func createKvResetPatch(currentLv map[string]uint, hasDeveloperConfiguration *bool, patchData *[]patch.PatchOperation) {
	// reset only if verbosity exists, otherwise do nothing
	if len(currentLv) != 0 {
		if !*hasDeveloperConfiguration {
			// if DeveloperConfiguration is absent, add DeveloperConfiguration first
			addPatch(patchData, patchAdd, dcPath, &v1.DeveloperConfiguration{})
			*hasDeveloperConfiguration = true
		}
		// add an empty object
		currentLv = map[string]uint{}
		addPatch(patchData, patchAdd, lvPath, currentLv)
	}
}

func createKvPatch(isReset, isComponent bool, kv *v1.KubeVirt, lines *[]string) ([]byte, error) {
	// "Add" patch removes the value if we do not specify the value, even if we do not change the existing value.
	// So, we need to get the existing verbosity in the KubeVirt CR.
	// Also, "Add" patch needs a DeveloperConfiguration entry before adding a LogVerbosity entry.
	// So, we need to know if DeveloperConfiguration is present or absent.
	currentLv, hasDeveloperConfiguration, err := hasVerbosityInKV(kv)
	if err != nil {
		return nil, err
	}
	patchData := []patch.PatchOperation{}
	if isReset {
		createKvResetPatch(currentLv, &hasDeveloperConfiguration, &patchData)
		currentLv = map[string]uint{}
		line := "successfully reset the verbosity of all KubeVirt components to default (2)"
		*lines = append(*lines, line)
	}
	if isComponent {
		createKvComponentPatch(currentLv, &hasDeveloperConfiguration, &patchData)
		line := "successfully set the verbosity of the KubeVirt component(s)"
		*lines = append(*lines, line)
	}
	return json.Marshal(patchData)
}

func processVMFlags(isShow, isSet *bool, vms *[]vmProperties) error {
	vmNamesLen := len(vmNames)
	vmLevelsLen := len(vmLevels)
	switch {
	case vmNamesLen > 0 && vmLevelsLen == 0:
		*isShow = true
	case vmNamesLen > 0 && vmLevelsLen > 0:
		if vmNamesLen != vmLevelsLen {
			return fmt.Errorf("number of vm flags %d not equal to number of level flags %d", vmNamesLen, vmLevelsLen)
		}
		// set the specified verbosity level in the vm struct
		for i, level := range vmLevels {
			val, err := strconv.Atoi(level)
			if val > int(maxVerbosity) || val < int(minVerbosity) || err != nil {
				return fmt.Errorf("%s: log verbosity must be %d-%d", (*vms)[i].name, minVerbosity, maxVerbosity)
			}
			(*vms)[i].level = level
		}
		*isSet = true
	}
	return nil
}

func processVirtComponents(cmd *cobra.Command, isShow, isSet, isComponent *bool) error {
	for componentName, verbosity := range virtComponents {
		// check if the flag for the component is specified
		// check NoFlag is not enough, because user can accidentally specify the same number as NoFlag for the verbosity
		if !cmd.Flags().Changed(componentName) || *verbosity == NoFlag {
			continue
		}

		*isComponent = true // the operation for virt components

		// if flag is specified, it means either set or show
		// if the value = noArg, it means show
		// if the value != noArg, it means set
		*isShow = *isShow || *verbosity == noArg
		*isSet = *isSet || *verbosity != noArg

		// check whether the verbosity is in the range
		if *verbosity != noArg {
			val, err := strconv.Atoi(*verbosity)
			if val > int(maxVerbosity) || val < int(minVerbosity) || err != nil {
				return fmt.Errorf("%s: log verbosity must be %d-%d", componentName, minVerbosity, maxVerbosity)
			}
		}
	}
	return nil
}

func findOperation(cmd *cobra.Command, isReset, isVMReset bool, isComponent *bool, vms *[]vmProperties) (operation, error) {
	isShow, isSet := false, false
	// for vm
	if err := processVMFlags(&isShow, &isSet, vms); err != nil {
		return nop, err
	}
	// for virt components
	if err := processVirtComponents(cmd, &isShow, &isSet, isComponent); err != nil {
		return nop, err
	}
	// do not distinguish between set and reset at this point, because set and reset can coexist
	if isReset || isVMReset {
		isSet = true
	}
	// determine operation
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

func processRunningVM(virtClient kubecli.KubevirtClient, currentLv map[string]uint, vmName, vmNamespace string) error {
	podList, err := virtClient.CoreV1().Pods(vmNamespace).List(context.Background(), k8smetav1.ListOptions{})
	if err != nil {
		return err
	}
	for i := range podList.Items {
		pod := &podList.Items[i]
		if pod.GenerateName != "virt-launcher-"+vmName+"-" {
			continue
		}
		for _, container := range pod.Spec.Containers {
			if container.Name != "compute" {
				continue
			}
			for _, env := range container.Env {
				if env.Name == "VIRT_LAUNCHER_LOG_VERBOSITY" {
					level, err := atou(env.Value)
					if err != nil {
						return err
					}
					currentLv[vmName] = level
					return nil
				}
				// if there is no VIRT_LAUNCHER_LOG_VERBOSITY, the virt-launcher verbosity was set as default
			}
		}
	}
	return nil
}

func processStoppedVM(currentLv map[string]uint, vmName string, obj *v1.VirtualMachine) error {
	if verbosity, exist := obj.Spec.Template.ObjectMeta.Labels["logVerbosity"]; exist {
		// if label is specified, use the label in the vm object
		level, err := atou(verbosity)
		if err != nil {
			return err
		}
		currentLv[vmName] = level
	} else if val, exist := currentLv["virtLauncher"]; exist {
		// if label is not specified, use the virt-launcher log verbosity
		currentLv[vmName] = val
	}
	return nil
}

func getVMVerbosity(virtClient kubecli.KubevirtClient, currentLv map[string]uint, vmNamespace string, vms []vmProperties) error {
	for _, vm := range vms {
		obj := vm.obj
		if *obj.Spec.Running {
			// check pod object's environmental variable VIRT_LAUNCHER_LOG_VERBOSITY
			if err := processRunningVM(virtClient, currentLv, vm.name, vmNamespace); err != nil {
				return err
			}
		} else {
			if err := processStoppedVM(currentLv, vm.name, obj); err != nil {
				return err
			}
		}
	}
	return nil
}

func atou(s string) (uint, error) {
	val, err := strconv.Atoi(s)
	if err != nil {
		return uint(val), fmt.Errorf("verbosity %s cannot cast to int: %v", s, err)
	}
	if val < 0 {
		return uint(val), fmt.Errorf("verbosity %s is negative", s)
	}
	return uint(val), err
}

func createVMPatch(cmd *cobra.Command, vm vmProperties) ([]byte, error) {
	patchData := []patch.PatchOperation{}
	obj := vm.obj
	if cmd.Flags().Changed("vm") {
		if obj.Spec.Template.ObjectMeta.Labels == nil || len(obj.Spec.Template.ObjectMeta.Labels) == 0 {
			// if vm.Spec.Template.ObjectMeta.Labels == map[string]string{}, add patch operation returns an error
			// (missing path: "/spec/template/metadata/labels/logVerbosity": missing value)
			addPatch(&patchData, patchAdd, "/spec/template/metadata/labels", map[string]string{"logVerbosity": ""})
		}
		verbosity := vm.level
		addPatch(&patchData, patchAdd, labelPath, verbosity)
	}
	return json.Marshal(patchData)
}

func createResetVMPatch(cmd *cobra.Command, vm vmProperties) ([]byte, error) {
	patchData := []patch.PatchOperation{}
	obj := vm.obj
	if _, exist := obj.Spec.Template.ObjectMeta.Labels["logVerbosity"]; exist {
		patchData = append(patchData, patch.PatchOperation{
			Op:   patchRemove,
			Path: labelPath,
		})
	}
	return json.Marshal(patchData)
}

func setVMProperties(virtClient kubecli.KubevirtClient, name string, vmNamespace string, item *vmProperties) error {
	var err error
	item.name = name
	// at this point, we do not know whether to show or set, so enter noArg (in case of set, enter the verbosity level later)
	item.level = noArg
	item.obj, err = virtClient.VirtualMachine(vmNamespace).Get(context.Background(), name, &k8smetav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error fetching Virtual Machine: %v", err)
	}
	return nil
}

func parseResetFlag(virtClient kubecli.KubevirtClient, cmd *cobra.Command, vmNamespace string, isReset, isVMReset *bool, resetVms *[]vmProperties) error {
	if cmd.Flags().Changed("reset") {
		for _, name := range resetNames {
			*isReset = *isReset || name == allComponents
			*isVMReset = *isVMReset || name != allComponents
			if name != allComponents {
				var item vmProperties
				if err := setVMProperties(virtClient, name, vmNamespace, &item); err != nil {
					return err
				}
				*resetVms = append(*resetVms, item)
			}
		}
	}
	return nil
}

func parseVMFlag(virtClient kubecli.KubevirtClient, vmNamespace string, vms *[]vmProperties) error {
	vmNamesLen := len(vmNames)
	vmLevelsLen := len(vmLevels)
	if vmNamesLen == 0 && vmLevelsLen != 0 {
		return fmt.Errorf("level: need vm flag")
	} else if vmNamesLen != 0 {
		for _, name := range vmNames {
			item := vmProperties{}
			if err := setVMProperties(virtClient, name, vmNamespace, &item); err != nil {
				return err
			}
			*vms = append(*vms, item)
		}
	}
	return nil
}

func (c *Command) RunE(cmd *cobra.Command) error {
	isReset, isVMReset, isComponent := false, false, false
	vms := []vmProperties{}
	resetVms := []vmProperties{}

	virtClient, err := kubecli.GetKubevirtClientFromClientConfig(c.clientConfig)
	if err != nil {
		return err
	}

	kvNamespace, kvName, err := detectInstallNamespaceAndName(virtClient)
	if err != nil {
		return err
	}
	kv, err := virtClient.KubeVirt(kvNamespace).Get(kvName, &k8smetav1.GetOptions{})
	if err != nil {
		return err
	}

	vmNamespace, _, err := c.clientConfig.Namespace()
	if err != nil {
		return err
	}

	// parse vm flag (set the specified vm name and vm object in the vm struct)
	err = parseVMFlag(virtClient, vmNamespace, &vms)
	if err != nil {
		return err
	}

	// parse reset flag (set the specified vm name and vm object in the vm struct)
	err = parseResetFlag(virtClient, cmd, vmNamespace, &isReset, &isVMReset, &resetVms)
	if err != nil {
		return err
	}

	// check the operation type (nop/show/set)
	op, err := findOperation(cmd, isReset, isVMReset, &isComponent, &vms)
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
		// if vm flag is specified, put the verbosity of vm in currentLv
		if len(vmNames) > 0 {
			err = getVMVerbosity(virtClient, currentLv, vmNamespace, vms)
			if err != nil {
				return err
			}
		}
		lines, err := createShowMessage(currentLv, vms)
		if err != nil {
			return err
		}
		for _, line := range lines {
			cmd.Println(line)
		}
	case set: // set and/or reset
		// patch KubeVirt CR
		if isComponent || isReset {
			lines := []string{}
			patchByte, err := createKvPatch(isReset, isComponent, kv, &lines)
			if err != nil {
				return err
			}
			_, err = virtClient.KubeVirt(kvNamespace).Patch(kvName, types.JSONPatchType, patchByte, &k8smetav1.PatchOptions{})
			if err != nil {
				return err
			}
			for _, line := range lines {
				cmd.Println(line)
			}
		}
		// reset VM object
		if isVMReset {
			for _, vm := range resetVms {
				patchByte, err := createResetVMPatch(cmd, vm)
				if err != nil {
					return err
				}
				_, err = virtClient.VirtualMachine(vmNamespace).Patch(
					context.Background(), vm.name, types.JSONPatchType, patchByte, &k8smetav1.PatchOptions{},
				)
				if err != nil {
					return err
				}
				cmd.Println(fmt.Sprintf("%s: successfully removed the logVerbosity label - need to (re)start vm to apply the reset", vm.name))
			}
		}
		// patch VM object
		for _, vm := range vms {
			patchByte, err := createVMPatch(cmd, vm)
			if err != nil {
				return err
			}
			_, err = virtClient.VirtualMachine(vmNamespace).Patch(
				context.Background(), vm.name, types.JSONPatchType, patchByte, &k8smetav1.PatchOptions{},
			)
			if err != nil {
				return err
			}
			cmd.Println(fmt.Sprintf("%s: successfully set the logVerbosity label - need to (re)start vm to apply the new verbosity", vm.name))
		}
	default:
		return fmt.Errorf("op: an unknown operation: %v", op)
	}

	return nil
}
