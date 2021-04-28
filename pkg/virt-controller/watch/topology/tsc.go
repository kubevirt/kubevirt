package topology

import (
	"fmt"
	"strconv"

	v1 "k8s.io/api/core/v1"

	k6tv1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
)

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

func VMIHasInvTSCFeature(vmi *k6tv1.VirtualMachineInstance) bool {
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
