package topology

import (
	"fmt"
	"strconv"
	"strings"

	"kubevirt.io/kubevirt/pkg/util"

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
			return 0, false, fmt.Errorf("tsc frequency on node %v is invalid: expected a frequenchy bigger than 0, but got %v", node.Name, freq)
		}
		return freq, scalable, err
	}
	return 0, false, nil
}

func TSCFrequencyFromPod(pod *v1.Pod) (frequency int64, err error) {
	for key := range pod.Spec.NodeSelector {
		if strings.HasPrefix(key, TSCFrequencySchedulingLabel+"-") {
			freq, err := strconv.ParseInt(strings.TrimPrefix(key, TSCFrequencySchedulingLabel+"-"), 10, 64)
			if err != nil {
				return 0, fmt.Errorf("tsc frequency on node %v is not an int: %v", pod.Name, err)
			} else if freq <= 0 {
				return 0, fmt.Errorf("tsc frequency on node %v is invalid: expected a frequenchy bigger than 0, but got %v", pod.Name, freq)
			}
			return freq, err
		}
	}
	return 0, nil
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

func distance(freq1, freq2 int64) int64 {
	if freq1 > freq2 {
		return freq1 - freq2
	}
	return freq2 - freq1
}

func CalculateTSCLabelDiff(frequenciesInUse []int64, frequenciesOnNode []int64, nodeFrequency int64, scalable bool) (toAdd []int64, toRemove []int64) {
	frequenciesInUse = append(frequenciesInUse, nodeFrequency)
	tolerance := ToleranceForFrequency(nodeFrequency)
	requiredMap := map[int64]struct{}{}
	for _, freq := range frequenciesInUse {
		if !scalable && distance(freq, nodeFrequency) > tolerance {
			// A non-scalable node can only accept frequencies that are within Qemu's tolerance:
			// nodeFrequency*(1-0.000250) < acceptableFrequency < nodeFrequency*(1+0.000250).
			// Skip the frequencies that are outside that range
			continue
		}
		requiredMap[freq] = struct{}{}
	}

	for _, freq := range frequenciesOnNode {
		if _, exists := requiredMap[freq]; !exists {
			toRemove = append(toRemove, freq)
		}
	}

	for freq := range requiredMap {
		// For the non-scalable case, the map was already sanitized above.
		// For the scalable case, a node can accept frequencies that are either lower than its own or within the tolerance range
		if !scalable || freq <= nodeFrequency || distance(freq, nodeFrequency) <= tolerance {
			toAdd = append(toAdd, freq)
		}
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
	if util.IsVmiUsingHyperVReenlightenment(vmi) {
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
