package topology

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

import (
	"fmt"

	"k8s.io/client-go/tools/cache"

	"kubevirt.io/kubevirt/pkg/pointer"

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

	frequenciesFromNodes := TSCFrequenciesFromNodes(FilterNodesFromCache(t.nodeStore.List(),
		HasInvTSCFrequency,
		IsSchedulable,
	))

	stableFreq := pickStableBaselineTSCFrequency(freq, t.TSCFrequenciesInUse(), frequenciesFromNodes)
	hints = &k6tv1.TopologyHints{TSCFrequency: pointer.P(stableFreq)}
	return
}

func pickStableBaselineTSCFrequency(clusterMin int64, frequenciesInUse []int64, frequenciesOnNodes []int64) int64 {
	var selected int64

	// First, try to pick minimal frequency that already in use by VMIs
	// and compatible with cluster wide minimal frequency.
	for _, freq := range frequenciesInUse {
		if !IsTSCFrequencyCompatible(clusterMin, false, freq) {
			continue
		}
		if selected == 0 || freq < selected {
			selected = freq
		}
	}
	if selected > 0 {
		return selected
	}

	// Next, get frequencies from all nodes and count that compatible with cluster wide minimal frequency.
	compatibleCounts := map[int64]int{}
	for _, freq := range frequenciesOnNodes {
		if !IsTSCFrequencyCompatible(clusterMin, false, freq) {
			continue
		}
		compatibleCounts[freq]++
	}

	// Try to pick frequency that present on at least 2 nodes.
	// This is a nice trick to overcome frequency drifting noise
	// and increase chances for VM to be able to live migrate in the future.
	for freq, count := range compatibleCounts {
		if count < 2 {
			continue
		}
		if selected == 0 || freq < selected {
			selected = freq
		}
	}
	if selected > 0 {
		return selected
	}

	// Fallback to cluster wide minimal if more stable baseline frequency is not found.
	return clusterMin
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
		Or(
			IsSchedulable,
			IsNodeRunningVmis(t.vmiStore),
		),
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
