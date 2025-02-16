//nolint:lll
package instancetype

import (
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/instancetype/apply"
	"kubevirt.io/kubevirt/pkg/instancetype/conflict"
	"kubevirt.io/kubevirt/pkg/instancetype/find"
	"kubevirt.io/kubevirt/pkg/instancetype/infer"
	preferenceApply "kubevirt.io/kubevirt/pkg/instancetype/preference/apply"
	preferenceFind "kubevirt.io/kubevirt/pkg/instancetype/preference/find"
	"kubevirt.io/kubevirt/pkg/instancetype/preference/validation"
	"kubevirt.io/kubevirt/pkg/instancetype/revision"
	"kubevirt.io/kubevirt/pkg/instancetype/upgrade"
)

type Methods interface {
	Upgrade(vm *virtv1.VirtualMachine) error
	FindInstancetypeSpec(vm *virtv1.VirtualMachine) (*instancetypev1beta1.VirtualMachineInstancetypeSpec, error)
	ApplyToVmi(field *k8sfield.Path, instancetypespec *instancetypev1beta1.VirtualMachineInstancetypeSpec, preferenceSpec *instancetypev1beta1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec, vmiMetadata *metav1.ObjectMeta) conflict.Conflicts
	FindPreferenceSpec(vm *virtv1.VirtualMachine) (*instancetypev1beta1.VirtualMachinePreferenceSpec, error)
	InferDefaultInstancetype(vm *virtv1.VirtualMachine) error
	InferDefaultPreference(vm *virtv1.VirtualMachine) error
	ApplyToVM(vm *virtv1.VirtualMachine) error
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

func (m *InstancetypeMethods) ApplyToVM(vm *virtv1.VirtualMachine) error {
	instancetypeFinder := find.NewSpecFinder(m.InstancetypeStore, m.ClusterInstancetypeStore, m.ControllerRevisionStore, m.Clientset)
	preferenceFinder := preferenceFind.NewSpecFinder(m.PreferenceStore, m.ClusterPreferenceStore, m.ControllerRevisionStore, m.Clientset)
	return apply.NewVMApplier(instancetypeFinder, preferenceFinder).ApplyToVM(vm)
}

func GetPreferredTopology(preferenceSpec *instancetypev1beta1.VirtualMachinePreferenceSpec) instancetypev1beta1.PreferredCPUTopology {
	return preferenceApply.GetPreferredTopology(preferenceSpec)
}

func IsPreferredTopologySupported(topology instancetypev1beta1.PreferredCPUTopology) bool {
	return validation.IsPreferredTopologySupported(topology)
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

func CompareRevisions(revisionA, revisionB *appsv1.ControllerRevision) (bool, error) {
	return revision.Compare(revisionA, revisionB)
}

func (m *InstancetypeMethods) ApplyToVmi(field *k8sfield.Path, instancetypeSpec *instancetypev1beta1.VirtualMachineInstancetypeSpec, preferenceSpec *instancetypev1beta1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec, vmiMetadata *metav1.ObjectMeta) conflict.Conflicts {
	conflicts := apply.NewVMIApplier().ApplyToVMI(field, instancetypeSpec, preferenceSpec, vmiSpec, vmiMetadata)
	return conflicts
}

func (m *InstancetypeMethods) FindPreferenceSpec(vm *virtv1.VirtualMachine) (*instancetypev1beta1.VirtualMachinePreferenceSpec, error) {
	return preferenceFind.NewSpecFinder(m.PreferenceStore, m.ClusterPreferenceStore, m.ControllerRevisionStore, m.Clientset).FindPreference(vm)
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

func ApplyDevicePreferences(preferenceSpec *instancetypev1beta1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {
	preferenceApply.ApplyDevicePreferences(preferenceSpec, vmiSpec)
}

func (m *InstancetypeMethods) Upgrade(vm *virtv1.VirtualMachine) error {
	return upgrade.New(m.ControllerRevisionStore, m.Clientset).Upgrade(vm)
}

func IsObjectLatestVersion(cr *appsv1.ControllerRevision) bool {
	return upgrade.IsObjectLatestVersion(cr)
}
