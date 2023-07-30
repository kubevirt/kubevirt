package topology

import (
	"math"

	"k8s.io/client-go/tools/cache"

	v1 "k8s.io/api/core/v1"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
)

const TSCFrequencyLabel = virtv1.CPUTimerLabel + "tsc-frequency"
const TSCFrequencySchedulingLabel = "scheduling.node.kubevirt.io/tsc-frequency"
const TSCScalableLabel = virtv1.CPUTimerLabel + "tsc-scalable"
const TSCTolerancePPM float64 = 250

type FilterPredicateFunc func(node *v1.Node) bool

func IsSchedulable(node *v1.Node) bool {
	if node == nil {
		return false
	}

	return node.Labels[virtv1.NodeSchedulable] == "true"
}

func HasInvTSCFrequency(node *v1.Node) bool {
	if node == nil {
		return false
	}
	freq, _, err := TSCFrequencyFromNode(node)
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("Excluding node %s with invalid tsc-frequency", node.Name)
		return false
	} else if freq == 0 {
		return false
	}
	return true
}

func TSCFrequencyGreaterEqual(frequency int64) FilterPredicateFunc {
	return func(node *v1.Node) bool {
		if node == nil {
			return false
		}
		freq, scalable, err := TSCFrequencyFromNode(node)
		if err != nil {
			log.DefaultLogger().Reason(err).Errorf("Excluding node %s with invalid tsc-frequency", node.Name)
			return false
		} else if freq == 0 {
			return false
		}
		return (scalable && freq >= frequency) || (freq == frequency && !scalable)
	}
}

func NodeOfVMI(vmi *virtv1.VirtualMachineInstance) FilterPredicateFunc {
	return func(node *v1.Node) bool {
		if vmi.Status.NodeName == "" {
			return false
		}
		if node == nil {
			return false
		}
		if node.Name == vmi.Status.NodeName {
			return true
		}
		return false
	}
}

func Not(f FilterPredicateFunc) FilterPredicateFunc {
	return func(node *v1.Node) bool {
		return !f(node)
	}
}

func Or(predicates ...FilterPredicateFunc) FilterPredicateFunc {
	return func(node *v1.Node) bool {
		for _, p := range predicates {
			if p(node) {
				return true
			}
		}
		return false
	}
}

func FilterNodesFromCache(objs []interface{}, predicates ...FilterPredicateFunc) []*v1.Node {
	match := []*v1.Node{}
	for _, obj := range objs {
		node := obj.(*v1.Node)
		passes := true
		for _, p := range predicates {
			if !p(node) {
				passes = false
				break
			}
		}
		if passes {
			match = append(match, node)
		}
	}
	return match
}

func IsNodeRunningVmis(vmiStore cache.Store) FilterPredicateFunc {
	return func(node *v1.Node) bool {
		if node == nil {
			return false
		}

		for _, vmi := range vmiStore.List() {
			vmi := vmi.(*virtv1.VirtualMachineInstance)
			if vmi.Status.NodeName == node.Name {
				return true
			}
		}
		return false
	}
}

// ToleranceForFrequency returns TSCTolerancePPM parts per million of freq, rounded down to the nearest Hz
func ToleranceForFrequency(freq int64) int64 {
	return int64(math.Floor(float64(freq) * (TSCTolerancePPM / 1000000)))
}
