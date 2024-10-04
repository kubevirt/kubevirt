//nolint:dupl,lll,gocyclo
package instancetype

import (
	"fmt"
	"slices"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/api/core/v1"
	apiinstancetype "kubevirt.io/api/instancetype"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/defaults"
	"kubevirt.io/kubevirt/pkg/instancetype/apply"
	"kubevirt.io/kubevirt/pkg/instancetype/find"
	"kubevirt.io/kubevirt/pkg/instancetype/infer"
	preferenceApply "kubevirt.io/kubevirt/pkg/instancetype/preference/apply"
	preferenceFind "kubevirt.io/kubevirt/pkg/instancetype/preference/find"
	"kubevirt.io/kubevirt/pkg/instancetype/revision"
	"kubevirt.io/kubevirt/pkg/instancetype/upgrade"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	utils "kubevirt.io/kubevirt/pkg/util"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

const (
	VMFieldConflictErrorFmt                         = "VM field %s conflicts with selected instance type"
	VMFieldsConflictsErrorFmt                       = "VM fields %s conflict with selected instance type"
	InsufficientInstanceTypeCPUResourcesErrorFmt    = "insufficient CPU resources of %d vCPU provided by instance type, preference requires %d vCPU"
	InsufficientVMCPUResourcesErrorFmt              = "insufficient CPU resources of %d vCPU provided by VirtualMachine, preference requires %d vCPU provided as %s"
	InsufficientInstanceTypeMemoryResourcesErrorFmt = "insufficient Memory resources of %s provided by instance type, preference requires %s"
	InsufficientVMMemoryResourcesErrorFmt           = "insufficient Memory resources of %s provided by VirtualMachine, preference requires %s"
	NoVMCPUResourcesDefinedErrorFmt                 = "no CPU resources provided by VirtualMachine, preference requires %d vCPU"
)

type Methods interface {
	Upgrade(vm *virtv1.VirtualMachine) error
	FindInstancetypeSpec(vm *virtv1.VirtualMachine) (*instancetypev1beta1.VirtualMachineInstancetypeSpec, error)
	ApplyToVmi(field *k8sfield.Path, instancetypespec *instancetypev1beta1.VirtualMachineInstancetypeSpec, preferenceSpec *instancetypev1beta1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec, vmiMetadata *metav1.ObjectMeta) Conflicts
	FindPreferenceSpec(vm *virtv1.VirtualMachine) (*instancetypev1beta1.VirtualMachinePreferenceSpec, error)
	StoreControllerRevisions(vm *virtv1.VirtualMachine) error
	InferDefaultInstancetype(vm *virtv1.VirtualMachine) error
	InferDefaultPreference(vm *virtv1.VirtualMachine) error
	CheckPreferenceRequirements(instancetypeSpec *instancetypev1beta1.VirtualMachineInstancetypeSpec, preferenceSpec *instancetypev1beta1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) (Conflicts, error)
	ApplyToVM(vm *virtv1.VirtualMachine) error
	Expand(vm *virtv1.VirtualMachine, clusterConfig *virtconfig.ClusterConfig) (*virtv1.VirtualMachine, error)
}

type Conflicts apply.Conflicts

func (c Conflicts) String() string {
	pathStrings := make([]string, 0, len(c))
	for _, path := range c {
		pathStrings = append(pathStrings, path.String())
	}
	return strings.Join(pathStrings, ", ")
}

type InstancetypeMethods struct {
	InstancetypeStore        cache.Store
	ClusterInstancetypeStore cache.Store
	PreferenceStore          cache.Store
	ClusterPreferenceStore   cache.Store
	ControllerRevisionStore  cache.Store
	Clientset                kubecli.KubevirtClient
}

var _ Methods = &InstancetypeMethods{}

func (m *InstancetypeMethods) Expand(vm *virtv1.VirtualMachine, clusterConfig *virtconfig.ClusterConfig) (*virtv1.VirtualMachine, error) {
	if vm.Spec.Instancetype == nil && vm.Spec.Preference == nil {
		return vm, nil
	}

	instancetypeSpec, err := m.FindInstancetypeSpec(vm)
	if err != nil {
		return nil, err
	}
	preferenceSpec, err := m.FindPreferenceSpec(vm)
	if err != nil {
		return nil, err
	}
	expandedVM := vm.DeepCopy()

	utils.SetDefaultVolumeDisk(&expandedVM.Spec.Template.Spec)

	if err := vmispec.SetDefaultNetworkInterface(clusterConfig, &expandedVM.Spec.Template.Spec); err != nil {
		return nil, err
	}

	conflicts := m.ApplyToVmi(
		k8sfield.NewPath("spec", "template", "spec"),
		instancetypeSpec, preferenceSpec,
		&expandedVM.Spec.Template.Spec,
		&expandedVM.Spec.Template.ObjectMeta,
	)
	if len(conflicts) > 0 {
		return nil, fmt.Errorf(VMFieldsConflictsErrorFmt, conflicts.String())
	}

	// Apply defaults to VM.Spec.Template.Spec after applying instance types to ensure we don't conflict
	if err := defaults.SetDefaultVirtualMachineInstanceSpec(clusterConfig, &expandedVM.Spec.Template.Spec); err != nil {
		return nil, err
	}

	// Remove InstancetypeMatcher and PreferenceMatcher, so the returned VM object can be used and not cause a conflict
	expandedVM.Spec.Instancetype = nil
	expandedVM.Spec.Preference = nil

	return expandedVM, nil
}

func (m *InstancetypeMethods) ApplyToVM(vm *virtv1.VirtualMachine) error {
	instancetypeFinder := find.NewSpecFinder(m.InstancetypeStore, m.ClusterInstancetypeStore, m.ControllerRevisionStore, m.Clientset)
	preferenceFinder := preferenceFind.NewSpecFinder(m.PreferenceStore, m.ClusterPreferenceStore, m.ControllerRevisionStore, m.Clientset)
	return apply.NewVMApplier(instancetypeFinder, preferenceFinder).ApplyToVM(vm)
}

func GetPreferredTopology(preferenceSpec *instancetypev1beta1.VirtualMachinePreferenceSpec) instancetypev1beta1.PreferredCPUTopology {
	return preferenceApply.GetPreferredTopology(preferenceSpec)
}

func IsPreferredTopologySupported(topology instancetypev1beta1.PreferredCPUTopology) bool {
	supportedTopologies := []instancetypev1beta1.PreferredCPUTopology{
		instancetypev1beta1.DeprecatedPreferSockets,
		instancetypev1beta1.DeprecatedPreferCores,
		instancetypev1beta1.DeprecatedPreferThreads,
		instancetypev1beta1.DeprecatedPreferSpread,
		instancetypev1beta1.DeprecatedPreferAny,
		instancetypev1beta1.Sockets,
		instancetypev1beta1.Cores,
		instancetypev1beta1.Threads,
		instancetypev1beta1.Spread,
		instancetypev1beta1.Any,
	}
	return slices.Contains(supportedTopologies, topology)
}

func GetSpreadOptions(preferenceSpec *instancetypev1beta1.VirtualMachinePreferenceSpec) (uint32, instancetypev1beta1.SpreadAcross) {
	return preferenceApply.GetSpreadOptions(preferenceSpec)
}

func GetRevisionName(vmName, resourceName, resourceVersion string, resourceUID types.UID, resourceGeneration int64) string {
	return revision.GenerateName(vmName, resourceName, resourceVersion, resourceUID, resourceGeneration)
}

func CreateControllerRevision(vm *virtv1.VirtualMachine, object runtime.Object) (*appsv1.ControllerRevision, error) {
	return revision.CreateControllerRevision(vm, object)
}

func (m *InstancetypeMethods) StoreControllerRevisions(vm *virtv1.VirtualMachine) error {
	return revision.New(
		m.InstancetypeStore,
		m.ClusterInstancetypeStore,
		m.PreferenceStore,
		m.ClusterInstancetypeStore,
		m.Clientset).Store(vm)
}

func CompareRevisions(revisionA, revisionB *appsv1.ControllerRevision) (bool, error) {
	return revision.Compare(revisionA, revisionB)
}

func checkCPUPreferenceRequirements(instancetypeSpec *instancetypev1beta1.VirtualMachineInstancetypeSpec, preferenceSpec *instancetypev1beta1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) (Conflicts, error) {
	if instancetypeSpec != nil {
		if instancetypeSpec.CPU.Guest < preferenceSpec.Requirements.CPU.Guest {
			return Conflicts{k8sfield.NewPath("spec", "instancetype")}, fmt.Errorf(InsufficientInstanceTypeCPUResourcesErrorFmt, instancetypeSpec.CPU.Guest, preferenceSpec.Requirements.CPU.Guest)
		}
		return nil, nil
	}

	cpuField := k8sfield.NewPath("spec", "template", "spec", "domain", "cpu")
	if vmiSpec.Domain.CPU == nil {
		return Conflicts{cpuField}, fmt.Errorf(NoVMCPUResourcesDefinedErrorFmt, preferenceSpec.Requirements.CPU.Guest)
	}

	switch GetPreferredTopology(preferenceSpec) {
	case instancetypev1beta1.DeprecatedPreferThreads, instancetypev1beta1.Threads:
		if vmiSpec.Domain.CPU.Threads < preferenceSpec.Requirements.CPU.Guest {
			return Conflicts{cpuField.Child("threads")}, fmt.Errorf(InsufficientVMCPUResourcesErrorFmt, vmiSpec.Domain.CPU.Threads, preferenceSpec.Requirements.CPU.Guest, "threads")
		}
	case instancetypev1beta1.DeprecatedPreferCores, instancetypev1beta1.Cores:
		if vmiSpec.Domain.CPU.Cores < preferenceSpec.Requirements.CPU.Guest {
			return Conflicts{cpuField.Child("cores")}, fmt.Errorf(InsufficientVMCPUResourcesErrorFmt, vmiSpec.Domain.CPU.Cores, preferenceSpec.Requirements.CPU.Guest, "cores")
		}
	case instancetypev1beta1.DeprecatedPreferSockets, instancetypev1beta1.Sockets:
		if vmiSpec.Domain.CPU.Sockets < preferenceSpec.Requirements.CPU.Guest {
			return Conflicts{cpuField.Child("sockets")}, fmt.Errorf(InsufficientVMCPUResourcesErrorFmt, vmiSpec.Domain.CPU.Sockets, preferenceSpec.Requirements.CPU.Guest, "sockets")
		}
	case instancetypev1beta1.DeprecatedPreferSpread, instancetypev1beta1.Spread:
		var (
			vCPUs     uint32
			conflicts Conflicts
		)
		_, across := GetSpreadOptions(preferenceSpec)
		switch across {
		case instancetypev1beta1.SpreadAcrossSocketsCores:
			vCPUs = vmiSpec.Domain.CPU.Sockets * vmiSpec.Domain.CPU.Cores
			conflicts = Conflicts{cpuField.Child("sockets"), cpuField.Child("cores")}
		case instancetypev1beta1.SpreadAcrossCoresThreads:
			vCPUs = vmiSpec.Domain.CPU.Cores * vmiSpec.Domain.CPU.Threads
			conflicts = Conflicts{cpuField.Child("cores"), cpuField.Child("threads")}
		case instancetypev1beta1.SpreadAcrossSocketsCoresThreads:
			vCPUs = vmiSpec.Domain.CPU.Sockets * vmiSpec.Domain.CPU.Cores * vmiSpec.Domain.CPU.Threads
			conflicts = Conflicts{cpuField.Child("sockets"), cpuField.Child("cores"), cpuField.Child("threads")}
		}
		if vCPUs < preferenceSpec.Requirements.CPU.Guest {
			return conflicts, fmt.Errorf(InsufficientVMCPUResourcesErrorFmt, vCPUs, preferenceSpec.Requirements.CPU.Guest, across)
		}
	case instancetypev1beta1.DeprecatedPreferAny, instancetypev1beta1.Any:
		cpuResources := vmiSpec.Domain.CPU.Cores * vmiSpec.Domain.CPU.Sockets * vmiSpec.Domain.CPU.Threads
		if cpuResources < preferenceSpec.Requirements.CPU.Guest {
			return Conflicts{cpuField.Child("cores"), cpuField.Child("sockets"), cpuField.Child("threads")}, fmt.Errorf(InsufficientVMCPUResourcesErrorFmt, cpuResources, preferenceSpec.Requirements.CPU.Guest, "cores, sockets and threads")
		}
	}

	return nil, nil
}

func checkMemoryPreferenceRequirements(instancetypeSpec *instancetypev1beta1.VirtualMachineInstancetypeSpec, preferenceSpec *instancetypev1beta1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) (Conflicts, error) {
	if instancetypeSpec != nil && instancetypeSpec.Memory.Guest.Cmp(preferenceSpec.Requirements.Memory.Guest) < 0 {
		return Conflicts{k8sfield.NewPath("spec", "instancetype")}, fmt.Errorf(InsufficientInstanceTypeMemoryResourcesErrorFmt, instancetypeSpec.Memory.Guest.String(), preferenceSpec.Requirements.Memory.Guest.String())
	}

	if instancetypeSpec == nil && vmiSpec.Domain.Memory != nil && vmiSpec.Domain.Memory.Guest.Cmp(preferenceSpec.Requirements.Memory.Guest) < 0 {
		return Conflicts{k8sfield.NewPath("spec", "template", "spec", "domain", "memory")}, fmt.Errorf(InsufficientVMMemoryResourcesErrorFmt, vmiSpec.Domain.Memory.Guest.String(), preferenceSpec.Requirements.Memory.Guest.String())
	}
	return nil, nil
}

func (m *InstancetypeMethods) CheckPreferenceRequirements(instancetypeSpec *instancetypev1beta1.VirtualMachineInstancetypeSpec, preferenceSpec *instancetypev1beta1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) (Conflicts, error) {
	if preferenceSpec == nil || preferenceSpec.Requirements == nil {
		return nil, nil
	}

	if preferenceSpec.Requirements.CPU != nil {
		if conflicts, err := checkCPUPreferenceRequirements(instancetypeSpec, preferenceSpec, vmiSpec); err != nil {
			return conflicts, err
		}
	}

	if preferenceSpec.Requirements.Memory != nil {
		if conflicts, err := checkMemoryPreferenceRequirements(instancetypeSpec, preferenceSpec, vmiSpec); err != nil {
			return conflicts, err
		}
	}

	return nil, nil
}

func (m *InstancetypeMethods) ApplyToVmi(field *k8sfield.Path, instancetypeSpec *instancetypev1beta1.VirtualMachineInstancetypeSpec, preferenceSpec *instancetypev1beta1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec, vmiMetadata *metav1.ObjectMeta) Conflicts {
	conflicts := apply.NewVMIApplier().ApplyToVMI(field, instancetypeSpec, preferenceSpec, vmiSpec, vmiMetadata)
	return Conflicts(conflicts)
}

func (m *InstancetypeMethods) FindPreferenceSpec(vm *virtv1.VirtualMachine) (*instancetypev1beta1.VirtualMachinePreferenceSpec, error) {
	return preferenceFind.NewSpecFinder(m.PreferenceStore, m.ClusterPreferenceStore, m.ControllerRevisionStore, m.Clientset).Find(vm)
}

func (m *InstancetypeMethods) FindInstancetypeSpec(vm *virtv1.VirtualMachine) (*instancetypev1beta1.VirtualMachineInstancetypeSpec, error) {
	return find.NewSpecFinder(m.InstancetypeStore, m.ClusterInstancetypeStore, m.ControllerRevisionStore, m.Clientset).Find(vm)
}

func (m *InstancetypeMethods) InferDefaultInstancetype(vm *virtv1.VirtualMachine) error {
	return infer.New(m.Clientset).Instancetype(vm)
}

func (m *InstancetypeMethods) InferDefaultPreference(vm *virtv1.VirtualMachine) error {
	return infer.New(m.Clientset).Preference(vm)
}

func AddInstancetypeNameAnnotations(vm *virtv1.VirtualMachine, target metav1.Object) {
	if vm.Spec.Instancetype == nil {
		return
	}

	if target.GetAnnotations() == nil {
		target.SetAnnotations(make(map[string]string))
	}
	switch strings.ToLower(vm.Spec.Instancetype.Kind) {
	case apiinstancetype.PluralResourceName, apiinstancetype.SingularResourceName:
		target.GetAnnotations()[virtv1.InstancetypeAnnotation] = vm.Spec.Instancetype.Name
	case "", apiinstancetype.ClusterPluralResourceName, apiinstancetype.ClusterSingularResourceName:
		target.GetAnnotations()[virtv1.ClusterInstancetypeAnnotation] = vm.Spec.Instancetype.Name
	}
}

func AddPreferenceNameAnnotations(vm *virtv1.VirtualMachine, target metav1.Object) {
	if vm.Spec.Preference == nil {
		return
	}

	if target.GetAnnotations() == nil {
		target.SetAnnotations(make(map[string]string))
	}
	switch strings.ToLower(vm.Spec.Preference.Kind) {
	case apiinstancetype.PluralPreferenceResourceName, apiinstancetype.SingularPreferenceResourceName:
		target.GetAnnotations()[virtv1.PreferenceAnnotation] = vm.Spec.Preference.Name
	case "", apiinstancetype.ClusterPluralPreferenceResourceName, apiinstancetype.ClusterSingularPreferenceResourceName:
		target.GetAnnotations()[virtv1.ClusterPreferenceAnnotation] = vm.Spec.Preference.Name
	}
}

func ApplyDevicePreferences(preferenceSpec *instancetypev1beta1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {
	preferenceApply.ApplyDevicePreferences(preferenceSpec, vmiSpec)
}

func (m *InstancetypeMethods) Upgrade(vm *virtv1.VirtualMachine) error {
	return upgrade.New(m.ControllerRevisionStore, m.Clientset).Upgrade(vm)
}

func IsObjectLatestVersion(cr *appsv1.ControllerRevision) bool {
	return upgrade.IsObjectLatestVersion(cr)
}
