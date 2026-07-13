package framework

import (
	"context"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"

	virtv1 "kubevirt.io/api/core/v1"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/topology"
)

type noopTopologyHinter struct{}

func (n *noopTopologyHinter) TopologyHintsForVMI(_ *virtv1.VirtualMachineInstance) (*virtv1.TopologyHints, topology.TscFrequencyRequirementType, error) {
	return nil, topology.NotRequired, nil
}

func (n *noopTopologyHinter) IsTscFrequencyRequired(_ *virtv1.VirtualMachineInstance) bool {
	return false
}

func (n *noopTopologyHinter) TSCFrequenciesInUse() []int64 {
	return nil
}

func (n *noopTopologyHinter) LowestTSCFrequencyOnCluster() (int64, error) {
	return 0, nil
}

type noopAnnotationsGenerator struct{}

func (n *noopAnnotationsGenerator) GenerateFromActivePod(_ *virtv1.VirtualMachineInstance, _ *k8sv1.Pod) map[string]string {
	return map[string]string{}
}

type noopStorageAnnotationsGenerator struct{}

func (n *noopStorageAnnotationsGenerator) Generate(_ *virtv1.VirtualMachineInstance) (map[string]string, error) {
	return map[string]string{}, nil
}

func (n *noopStorageAnnotationsGenerator) ManagedAnnotationKeys() []string {
	return nil
}

type noopMigrationEvaluator struct{}

func (n *noopMigrationEvaluator) Evaluate(_ *virtv1.VirtualMachineInstance, _ *k8sv1.Pod) k8sv1.ConditionStatus {
	return k8sv1.ConditionUnknown
}

type noopSynchronizer struct{}

func (n *noopSynchronizer) Sync(vm *virtv1.VirtualMachine, _ *virtv1.VirtualMachineInstance) (*virtv1.VirtualMachine, error) {
	return vm, nil
}

func noopStatusUpdater(_ *virtv1.VirtualMachineInstance, _ *k8sv1.Pod) error {
	return nil
}

func noopSpecValidator(_ *k8sfield.Path, _ *virtv1.VirtualMachineInstanceSpec, _ *virtconfig.ClusterConfig) []metav1.StatusCause {
	return nil
}

type noopAPIServiceClient struct{}

func (n *noopAPIServiceClient) Get(_ context.Context, _ string, _ metav1.GetOptions) (*apiregv1.APIService, error) {
	return nil, errors.NewNotFound(schema.GroupResource{Group: "apiregistration.k8s.io", Resource: "apiservices"}, "")
}

func (n *noopAPIServiceClient) Create(_ context.Context, _ *apiregv1.APIService, _ metav1.CreateOptions) (*apiregv1.APIService, error) {
	return nil, nil
}

func (n *noopAPIServiceClient) Delete(_ context.Context, _ string, _ metav1.DeleteOptions) error {
	return nil
}

func (n *noopAPIServiceClient) Patch(_ context.Context, _ string, _ types.PatchType, _ []byte, _ metav1.PatchOptions, _ ...string) (*apiregv1.APIService, error) {
	return nil, nil
}
