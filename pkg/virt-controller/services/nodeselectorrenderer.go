package services

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"
)

type NodeSelectorRenderer struct {
	hasDedicatedCPU       bool
	userProvidedSelectors map[string]string
}

type NodeSelectorRendererOption func(renderer *NodeSelectorRenderer)

func NewNodeSelectorRenderer(vmiNodeSelectors map[string]string, opts ...NodeSelectorRendererOption) *NodeSelectorRenderer {
	podNodeSelectors := map[string]string{}
	copySelectors(vmiNodeSelectors, podNodeSelectors)

	nodeSelectorRenderer := &NodeSelectorRenderer{userProvidedSelectors: podNodeSelectors}
	for _, opt := range opts {
		opt(nodeSelectorRenderer)
	}
	return nodeSelectorRenderer
}

func (nsr *NodeSelectorRenderer) Render() map[string]string {
	nodeSelectors := map[string]string{}
	if nsr.hasDedicatedCPU {
		nodeSelectors[v1.CPUManager] = "true"
	}
	for k, v := range nsr.userProvidedSelectors {
		nodeSelectors[k] = v
	}
	return nodeSelectors
}

func WithDedicatedCPU() NodeSelectorRendererOption {
	return func(renderer *NodeSelectorRenderer) {
		renderer.hasDedicatedCPU = true
	}
}

func copySelectors(src map[string]string, dst map[string]string) {
	for k, v := range src {
		dst[k] = v
	}
}

func CPUModelLabelFromCPUModel(vmi *v1.VirtualMachineInstance) (label string, err error) {
	if vmi.Spec.Domain.CPU == nil || vmi.Spec.Domain.CPU.Model == "" {
		err = fmt.Errorf("Cannot create CPU Model label, vmi spec is mising CPU model")
		return
	}
	label = NFD_CPU_MODEL_PREFIX + vmi.Spec.Domain.CPU.Model
	return
}

func CPUFeatureLabelsFromCPUFeatures(vmi *v1.VirtualMachineInstance) []string {
	var labels []string
	if vmi.Spec.Domain.CPU != nil && vmi.Spec.Domain.CPU.Features != nil {
		for _, feature := range vmi.Spec.Domain.CPU.Features {
			if feature.Policy == "" || feature.Policy == "require" {
				labels = append(labels, NFD_CPU_FEATURE_PREFIX+feature.Name)
			}
		}
	}
	return labels
}
