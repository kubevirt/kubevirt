package services

import v1 "kubevirt.io/api/core/v1"

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
