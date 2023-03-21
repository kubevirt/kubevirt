package topology

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

import (
	"fmt"

	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"

	k6tv1 "kubevirt.io/api/core/v1"
)

type Hinter interface {
	TopologyHintsForVMI(vmi *k6tv1.VirtualMachineInstance) (hints *k6tv1.TopologyHints, requirement TscFrequencyRequirementType, err error)
	IsTscFrequencyRequired(vmi *k6tv1.VirtualMachineInstance) bool
	TSCFrequenciesInUse() []int64
	LowestTSCFrequencyOnCluster() (int64, error)
}

type topologyHinter struct {
	clusterConfig *virtconfig.ClusterConfig
	nodeStore     cache.Store
	vmiStore      cache.Store
}

func (t *topologyHinter) IsTscFrequencyRequired(vmi *k6tv1.VirtualMachineInstance) bool {
	return vmi.Spec.Architecture == "amd64" && GetTscFrequencyRequirement(vmi).Type != NotRequired
}

func (t *topologyHinter) TopologyHintsForVMI(vmi *k6tv1.VirtualMachineInstance) (hints *k6tv1.TopologyHints, requirement TscFrequencyRequirementType, err error) {
	requirement = GetTscFrequencyRequirement(vmi).Type
	if requirement == NotRequired || vmi.Spec.Architecture != "amd64" {
		return
	}

	freq, err := t.LowestTSCFrequencyOnCluster()
	if err != nil {
		return nil, requirement, fmt.Errorf("failed to determine the lowest tsc frequency on the cluster: %v", err)
	}

	hints = &k6tv1.TopologyHints{TSCFrequency: pointer.Int64Ptr(freq)}
	return
}

func (t *topologyHinter) LowestTSCFrequencyOnCluster() (int64, error) {
	configTSCFrequency := t.clusterConfig.GetMinimumClusterTSCFrequency()
	if configTSCFrequency != nil {
		if *configTSCFrequency > 0 {
			return *configTSCFrequency, nil
		} else {
			return 0, fmt.Errorf("the configured minimumClusterTSCFrequency must be greater 0, but got %d", *configTSCFrequency)
		}
	}
	nodes := FilterNodesFromCache(t.nodeStore.List(),
		HasInvTSCFrequency,
	)
	freq := LowestTSCFrequency(nodes)
	return freq, nil
}

func (t *topologyHinter) TSCFrequenciesInUse() []int64 {
	frequencyMap := map[int64]struct{}{}
	for _, obj := range t.vmiStore.List() {
		vmi := obj.(*k6tv1.VirtualMachineInstance)
		if AreTSCFrequencyTopologyHintsDefined(vmi) {
			frequencyMap[*vmi.Status.TopologyHints.TSCFrequency] = struct{}{}
		}
	}
	frequencies := []int64{}
	for freq := range frequencyMap {
		frequencies = append(frequencies, freq)
	}
	return frequencies
}

func NewTopologyHinter(nodeStore cache.Store, vmiStore cache.Store, clusterConfig *virtconfig.ClusterConfig) *topologyHinter {
	return &topologyHinter{nodeStore: nodeStore, vmiStore: vmiStore, clusterConfig: clusterConfig}
}
