package topology

import (
	"fmt"
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"

	k6tv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
)

type TscFrequencyRequirementType int

const (
	RequiredForBoot TscFrequencyRequirementType = iota
	RequiredForMigration
	NotRequired
)

type TscFrequencyRequirement struct {
	Type   TscFrequencyRequirementType
	Reason string
}

func LowestTSCFrequency(nodes []*v1.Node) int64 {
	var lowest int64
	for i, node := range nodes {
		freq, _, err := TSCFrequencyFromNode(node)
		if err != nil {
			log.DefaultLogger().Reason(err).Errorf("Excluding node %s with invalid tsc-frequency", node.Name)
		}
		if freq > 0 && (i == 0 || freq < lowest) {
			lowest = freq
		}
	}
	return lowest
}

func TSCFrequencyFromNode(node *v1.Node) (frequency int64, scalable bool, err error) {
	if val, exists := node.Labels[TSCScalableLabel]; exists {
		scalable = val == "true"
	}
	if val, exists := node.Labels[TSCFrequencyLabel]; exists {
		freq, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return 0, false, fmt.Errorf("tsc frequency on node %v is not an int: %v", node.Name, err)
		} else if freq <= 0 {
			return 0, false, fmt.Errorf("tsc frequency on node %v is invalid: expected a frequenсy bigger than 0, but got %v", node.Name, freq)
		}
		return freq, scalable, err
	}
	return 0, false, nil
}

func TSCFrequenciesOnNode(node *v1.Node) (frequencies []int64) {
	for key := range node.Labels {
		if strings.HasPrefix(key, TSCFrequencySchedulingLabel+"-") {
			freq, err := strconv.ParseInt(strings.TrimPrefix(key, TSCFrequencySchedulingLabel+"-"), 10, 64)
			if err != nil {
				log.DefaultLogger().Object(node).Reason(err).Errorf("Label %s is invalid", key)
				continue
			}
			frequencies = append(frequencies, freq)
		}
	}
	return
}

func TSCFrequenciesFromNodes(nodes []*v1.Node) (frequencies []int64) {
	for _, node := range nodes {
		freq, _, err := TSCFrequencyFromNode(node)
		if err != nil {
			log.DefaultLogger().Reason(err).Errorf("Excluding node %s with invalid tsc-frequency", node.Name)
			continue
		}
		if freq <= 0 {
			continue
		}
		frequencies = append(frequencies, freq)
	}
	return
}

func distance(freq1, freq2 int64) int64 {
	if freq1 > freq2 {
		return freq1 - freq2
	}
	return freq2 - freq1
}

func IsTSCFrequencyCompatible(nodeFrequency int64, scalable bool, freq int64) bool {
	tolerance := ToleranceForFrequency(nodeFrequency)
	if !scalable {
		// A non-scalable node can only accept frequencies that are within Qemu's tolerance:
		// nodeFrequency*(1-0.000250) < acceptableFrequency < nodeFrequency*(1+0.000250).
		return distance(freq, nodeFrequency) <= tolerance
	}

	// A scalable node can accept frequencies that are either lower than its own or within the tolerance range.
	return freq <= nodeFrequency || distance(freq, nodeFrequency) <= tolerance
}

func CalculateTSCLabelDiff(frequenciesInUse []int64, frequenciesOnNode []int64, frequenciesFromNodes []int64, nodeFrequency int64, scalable bool) (toAdd []int64, toRemove []int64) {
	requiredMap := map[int64]struct{}{}
	// Always preserve the node's own frequency label.
	requiredMap[nodeFrequency] = struct{}{}

	// Preserve all frequencies currently in use that are compatible with node.
	for _, freq := range frequenciesInUse {
		if IsTSCFrequencyCompatible(nodeFrequency, scalable, freq) {
			requiredMap[freq] = struct{}{}
		}
	}

	// Make index of compatible measured frequencies still present on nodes in the cluster.
	nodesOwnFrequencies := map[int64]struct{}{}
	for _, freq := range frequenciesFromNodes {
		if IsTSCFrequencyCompatible(nodeFrequency, scalable, freq) {
			nodesOwnFrequencies[freq] = struct{}{}
		}
	}

	// Keep compatible frequencies that already on node:
	// 1. If already in requiredMap (own and in use)
	// 2. If present as "own" on other nodes.
	// Remove non-compatible and unused frequencies.
	for _, freq := range frequenciesOnNode {
		if !IsTSCFrequencyCompatible(nodeFrequency, scalable, freq) {
			toRemove = append(toRemove, freq)
			continue
		}
		if _, exists := requiredMap[freq]; exists {
			continue
		}
		if _, exists := nodesOwnFrequencies[freq]; exists {
			requiredMap[freq] = struct{}{}
			continue
		}
		toRemove = append(toRemove, freq)
	}

	for freq := range requiredMap {
		toAdd = append(toAdd, freq)
	}

	return
}

func ToTSCSchedulableLabels(frequencies []int64) (labels []string) {
	for _, freq := range frequencies {
		labels = append(labels, ToTSCSchedulableLabel(freq))
	}
	return
}

func ToTSCSchedulableLabel(frequency int64) string {
	return fmt.Sprintf("%s-%d", TSCFrequencySchedulingLabel, frequency)
}

func AreTSCFrequencyTopologyHintsDefined(vmi *k6tv1.VirtualMachineInstance) bool {
	if vmi == nil {
		return false
	}

	topologyHints := vmi.Status.TopologyHints
	return topologyHints != nil && topologyHints.TSCFrequency != nil && *topologyHints.TSCFrequency > 0
}

func IsManualTSCFrequencyRequired(vmi *k6tv1.VirtualMachineInstance) bool {
	return vmi != nil &&
		GetTscFrequencyRequirement(vmi).Type != NotRequired &&
		AreTSCFrequencyTopologyHintsDefined(vmi)
}

func GetTscFrequencyRequirement(vmi *k6tv1.VirtualMachineInstance) TscFrequencyRequirement {
	newRequirement := func(reqType TscFrequencyRequirementType, reason string) TscFrequencyRequirement {
		return TscFrequencyRequirement{Type: reqType, Reason: reason}
	}

	if vmiHasInvTSCFeature(vmi) {
		return newRequirement(RequiredForBoot, "VMI with invtsc CPU feature must have tsc frequency defined in order to boot")
	}
	if isVmiUsingHyperVReenlightenment(vmi) {
		return newRequirement(RequiredForMigration, "HyperV Reenlightenment VMIs cannot migrate when TSC Frequency is not exposed on the cluster: guest timers might be inconsistent")
	}

	return newRequirement(NotRequired, "")
}

func vmiHasInvTSCFeature(vmi *k6tv1.VirtualMachineInstance) bool {
	if cpu := vmi.Spec.Domain.CPU; cpu != nil {
		for _, f := range cpu.Features {
			if f.Name != "invtsc" {
				continue
			}
			switch f.Policy {
			case "require", "force":
				return true
			}
		}
	}
	return false
}

func isVmiUsingHyperVReenlightenment(vmi *k6tv1.VirtualMachineInstance) bool {
	if vmi == nil {
		return false
	}

	domainFeatures := vmi.Spec.Domain.Features

	return domainFeatures != nil && domainFeatures.Hyperv != nil && domainFeatures.Hyperv.Reenlightenment != nil &&
		domainFeatures.Hyperv.Reenlightenment.Enabled != nil && *domainFeatures.Hyperv.Reenlightenment.Enabled
}
