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
	TopologyHintsForVMI(vmi *k6tv1.VirtualMachineInstance) (hints *k6tv1.TopologyHints, err error)
	TopologyHintsRequiredForVMI(vmi *k6tv1.VirtualMachineInstance) bool
	TSCFrequenciesInUse() []int64
	LowestTSCFrequencyOnCluster() (int64, error)
}

type topologyHinter struct {
	clusterConfig *virtconfig.ClusterConfig
	nodeStore     cache.Store
	vmiStore      cache.Store
	arch          string
}

func (t *topologyHinter) TopologyHintsRequiredForVMI(vmi *k6tv1.VirtualMachineInstance) bool {
	return t.arch == "amd64" && VMIHasInvTSCFeature(vmi)
}

func (t *topologyHinter) TopologyHintsForVMI(vmi *k6tv1.VirtualMachineInstance) (hints *k6tv1.TopologyHints, err error) {
	if t.TopologyHintsRequiredForVMI(vmi) {
		freq, err := t.LowestTSCFrequencyOnCluster()
		if err != nil {
			return nil, fmt.Errorf("failed to determine the lowest tsc frequency on the cluster: %v", err)
		}
		return &k6tv1.TopologyHints{
			TSCFrequency: pointer.Int64Ptr(freq),
		}, nil
	}
	return nil, nil
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
	if freq == 0 {
		return 0, fmt.Errorf("no schedulable node exposes a tsc-frequency")
	}
	return freq, nil
}

func (t *topologyHinter) TSCFrequenciesInUse() []int64 {
	frequencyMap := map[int64]struct{}{}
	for _, obj := range t.vmiStore.List() {
		vmi := obj.(*k6tv1.VirtualMachineInstance)
		if vmi.Status.TopologyHints != nil && vmi.Status.TopologyHints.TSCFrequency != nil {
			frequencyMap[*vmi.Status.TopologyHints.TSCFrequency] = struct{}{}
		}
	}
	frequencies := []int64{}
	for freq := range frequencyMap {
		frequencies = append(frequencies, freq)
	}
	return frequencies
}

func NewTopologyHinter(nodeStore cache.Store, vmiStore cache.Store, arch string, clusterConfig *virtconfig.ClusterConfig) *topologyHinter {
	return &topologyHinter{nodeStore: nodeStore, vmiStore: vmiStore, arch: arch, clusterConfig: clusterConfig}
}
