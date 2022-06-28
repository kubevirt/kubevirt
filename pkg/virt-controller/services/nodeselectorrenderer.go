package services

type NodeSelectorRenderer struct {
	userProvidedSelectors map[string]string
}

func NewNodeSelectorRenderer(vmiNodeSelectors map[string]string) *NodeSelectorRenderer {
	podNodeSelectors := map[string]string{}
	copySelectors(vmiNodeSelectors, podNodeSelectors)
	return &NodeSelectorRenderer{userProvidedSelectors: podNodeSelectors}
}

func (nsr *NodeSelectorRenderer) Render() map[string]string {
	return nsr.userProvidedSelectors
}

func copySelectors(src map[string]string, dst map[string]string) {
	for k, v := range src {
		dst[k] = v
	}
}
