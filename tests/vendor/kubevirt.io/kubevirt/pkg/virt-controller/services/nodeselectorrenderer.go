package services

import (
	"fmt"
	"strings"

	k8sv1 "k8s.io/api/core/v1"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-controller/watch/topology"
)

type NodeSelectorRenderer struct {
	cpuFeatureLabels []string
	cpuModelLabel    string
	hasDedicatedCPU  bool
	hyperv           bool
	podNodeSelectors map[string]string
	tscFrequency     *int64
	vmiFeatures      *v1.Features
}

type NodeSelectorRendererOption func(renderer *NodeSelectorRenderer)

func NewNodeSelectorRenderer(
	vmiNodeSelectors map[string]string,
	clusterWideConfNodeSelectors map[string]string,
	architecture string,
	opts ...NodeSelectorRendererOption,
) *NodeSelectorRenderer {
	podNodeSelectors := map[string]string{v1.NodeSchedulable: "true"}
	if architecture != "" {
		podNodeSelectors[k8sv1.LabelArchStable] = strings.ToLower(architecture)
	}

	copySelectors(clusterWideConfNodeSelectors, podNodeSelectors)
	copySelectors(vmiNodeSelectors, podNodeSelectors)

	nodeSelectorRenderer := &NodeSelectorRenderer{podNodeSelectors: podNodeSelectors}
	for _, opt := range opts {
		opt(nodeSelectorRenderer)
	}
	return nodeSelectorRenderer
}

func (nsr *NodeSelectorRenderer) Render() map[string]string {
	if nsr.hasDedicatedCPU {
		nsr.enableSelectorLabel(v1.CPUManager)
	}
	if nsr.hyperv {
		copySelectors(hypervNodeSelectors(nsr.vmiFeatures), nsr.podNodeSelectors)
	}
	if nsr.cpuModelLabel != "" && nsr.cpuModelLabel != cpuModelLabel(v1.CPUModeHostModel) && nsr.cpuModelLabel != cpuModelLabel(v1.CPUModeHostPassthrough) {
		nsr.enableSelectorLabel(nsr.cpuModelLabel)
	}
	for _, cpuFeatureLabel := range nsr.cpuFeatureLabels {
		nsr.enableSelectorLabel(cpuFeatureLabel)
	}

	if nsr.isManualTSCFrequencyRequired() {
		nsr.enableSelectorLabel(topology.ToTSCSchedulableLabel(*nsr.tscFrequency))
	}

	return nsr.podNodeSelectors
}

func (nsr *NodeSelectorRenderer) enableSelectorLabel(label string) {
	nsr.podNodeSelectors[label] = "true"
}

func (nsr *NodeSelectorRenderer) isManualTSCFrequencyRequired() bool {
	return nsr.tscFrequency != nil
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

func WithHyperv(features *v1.Features) NodeSelectorRendererOption {
	return func(renderer *NodeSelectorRenderer) {
		renderer.hyperv = true
		renderer.vmiFeatures = features
	}
}

func WithModelAndFeatureLabels(modelLabel string, cpuFeatureLabels ...string) NodeSelectorRendererOption {
	return func(renderer *NodeSelectorRenderer) {
		renderer.cpuFeatureLabels = cpuFeatureLabels
		renderer.cpuModelLabel = modelLabel
	}
}

func WithTSCTimer(tscFrequency *int64) NodeSelectorRendererOption {
	return func(renderer *NodeSelectorRenderer) {
		renderer.tscFrequency = tscFrequency
	}
}

func CPUModelLabelFromCPUModel(vmi *v1.VirtualMachineInstance) (label string, err error) {
	if vmi.Spec.Domain.CPU == nil || vmi.Spec.Domain.CPU.Model == "" {
		err = fmt.Errorf("Cannot create CPU Model label, vmi spec is mising CPU model")
		return
	}
	label = cpuModelLabel(vmi.Spec.Domain.CPU.Model)
	return
}

func cpuModelLabel(cpuModel string) string {
	return NFD_CPU_MODEL_PREFIX + cpuModel
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

func hypervNodeSelectors(vmiFeatures *v1.Features) map[string]string {
	nodeSelectors := make(map[string]string)
	if vmiFeatures == nil || vmiFeatures.Hyperv == nil {
		return nodeSelectors
	}

	for _, hv := range makeHVFeatureLabelTable(vmiFeatures) {
		if isFeatureStateEnabled(hv.Feature) {
			nodeSelectors[NFD_KVM_INFO_PREFIX+hv.Label] = "true"
		}
	}

	if vmiFeatures.Hyperv.EVMCS != nil && (vmiFeatures.Hyperv.EVMCS.Enabled == nil || (*vmiFeatures.Hyperv.EVMCS.Enabled) == true) {
		nodeSelectors[v1.CPUModelVendorLabel+IntelVendorName] = "true"
	}

	return nodeSelectors
}

type hvFeatureLabel struct {
	Feature *v1.FeatureState
	Label   string
}

// makeHVFeatureLabelTable creates the mapping table between the VMI hyperv state and the label names.
// The table needs pointers to v1.FeatureHyperv struct, so it has to be generated and can't be a
// static var
func makeHVFeatureLabelTable(vmiFeatures *v1.Features) []hvFeatureLabel {
	// The following HyperV features don't require support from the host kernel, according to inspection
	// of the QEMU sources (4.0 - adb3321bfd)
	// VAPIC, Relaxed, Spinlocks, VendorID
	// VPIndex, SyNIC: depend on both MSR and capability
	// IPI, TLBFlush: depend on KVM Capabilities
	// Runtime, Reset, SyNICTimer, Frequencies, Reenlightenment: depend on KVM MSRs availability
	// EVMCS: depends on KVM capability, but the only way to know that is enable it, QEMU doesn't do
	// any check before that, so we leave it out
	//
	// see also https://schd.ws/hosted_files/devconfcz2019/cf/vkuznets_enlightening_kvm_devconf2019.pdf
	// to learn about dependencies between enlightenments

	hyperv := vmiFeatures.Hyperv // shortcut

	syNICTimer := &v1.FeatureState{}
	if hyperv.SyNICTimer != nil {
		syNICTimer.Enabled = hyperv.SyNICTimer.Enabled
	}

	return []hvFeatureLabel{
		{
			Feature: hyperv.VPIndex,
			Label:   "vpindex",
		},
		{
			Feature: hyperv.Runtime,
			Label:   "runtime",
		},
		{
			Feature: hyperv.Reset,
			Label:   "reset",
		},
		{
			// TODO: SyNIC depends on vp-index on QEMU level. We should enforce this constraint.
			Feature: hyperv.SyNIC,
			Label:   "synic",
		},
		{
			// TODO: SyNICTimer depends on SyNIC and Relaxed. We should enforce this constraint.
			Feature: syNICTimer,
			Label:   "synictimer",
		},
		{
			Feature: hyperv.Frequencies,
			Label:   "frequencies",
		},
		{
			Feature: hyperv.Reenlightenment,
			Label:   "reenlightenment",
		},
		{
			Feature: hyperv.TLBFlush,
			Label:   "tlbflush",
		},
		{
			Feature: hyperv.IPI,
			Label:   "ipi",
		},
	}
}
