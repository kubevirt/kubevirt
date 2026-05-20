package topology

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	g "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/kubevirt/pkg/pointer"

	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"

	virtv1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Hinter", func() {

	It("should return the lowes TSC frequency in the cluster", func() {
		hinter := hinterWithNodes(
			NodeWithInvalidTSC("node0"),
			NodeWithTSC("node0", 1234, true),
			NodeWithTSC("node1", 123, true),
			NodeWithTSC("node2", 12345, false),
		)
		g.Expect(hinter.LowestTSCFrequencyOnCluster()).To(g.BeNumerically("==", 123))
	})

	It("should pick up when a minimum TSC frequency is set in the config", func() {
		hinter := hinterWithNodes(
			NodeWithInvalidTSC("node0"),
			NodeWithTSC("node0", 1234, true),
			NodeWithTSC("node1", 123, true),
			NodeWithTSC("node2", 12345, false),
		)
		g.Expect(hinter.LowestTSCFrequencyOnCluster()).To(g.BeNumerically("==", 123))
		hinter.clusterConfig = clusterConfigWithTSCFrequency(200)
		g.Expect(hinter.LowestTSCFrequencyOnCluster()).To(g.BeNumerically("==", 200))
		hinter.clusterConfig = clusterConfigWithoutTSCFrequency()
		g.Expect(hinter.LowestTSCFrequencyOnCluster()).To(g.BeNumerically("==", 123))
	})

	It("should propose a TSC frequency for the VMI", func() {
		hinter := hinterWithNodes(
			NodeWithInvalidTSC("node0"),
			NodeWithTSC("node1", 1234, true),
			NodeWithTSC("node2", 123, true),
			NodeWithTSC("node3", 12345, false),
			NodeWithTSC("node4", 12, false),
		)
		vmi := vmiWithTSCFrequencyOnNode("myvmi", 12, "oldnode")
		g.Expect(GetTscFrequencyRequirement(vmi).Type).ToNot(g.Equal(NotRequired))
		g.Expect(hinter.TopologyHintsForVMI(vmi)).To(g.Equal(
			&virtv1.TopologyHints{
				TSCFrequency: pointer.P(int64(12)),
			},
		))
	})

	It("should prefer a compatible frequency already in use over a drifted cluster minimum", func() {
		hinter := hinterWithNodes(
			NodeWithTSC("node1", 3599973000, false),
			NodeWithTSC("node2", 3599996000, false),
		)
		hinter.vmiStore = &cache.FakeCustomStore{ListFunc: func() []interface{} {
			return VMIsToObjects(
				vmiWithTSCFrequencyOnNode("existing", 3599975000, "node2"),
			)
		}}
		vmi := vmiWithTSCFrequencyOnNode("newvmi", 12, "oldnode")
		g.Expect(hinter.TopologyHintsForVMI(vmi)).To(g.Equal(
			&virtv1.TopologyHints{
				TSCFrequency: pointer.P(int64(3599975000)),
			},
		))
	})

	It("should prefer the lowest compatible measured frequency that exists on at least two nodes when no frequencies are in use", func() {
		hinter := hinterWithNodes(
			NodeWithTSC("node1", 3599973000, false),
			NodeWithTSC("node2", 3599975000, false),
			NodeWithTSC("node3", 3599975000, false),
			NodeWithTSC("node4", 3599996000, false),
			NodeWithTSC("node5", 3599998000, false),
			NodeWithTSC("node6", 3599998000, false),
		)
		vmi := vmiWithTSCFrequencyOnNode("newvmi", 12, "oldnode")
		g.Expect(hinter.TopologyHintsForVMI(vmi)).To(g.Equal(
			&virtv1.TopologyHints{
				TSCFrequency: pointer.P(int64(3599975000)),
			},
		))
	})

	It("should fall back to the raw cluster minimum when two nodes have incompatible frequency", func() {
		hinter := hinterWithNodes(
			NodeWithTSC("node1", 3599973000, false),
			NodeWithTSC("node2", 3599975000, false),
			NodeWithTSC("node3", 3700006000, false),
			NodeWithTSC("node4", 3700006000, false),
		)
		vmi := vmiWithTSCFrequencyOnNode("newvmi", 12, "oldnode")
		g.Expect(hinter.TopologyHintsForVMI(vmi)).To(g.Equal(
			&virtv1.TopologyHints{
				TSCFrequency: pointer.P(int64(3599973000)),
			},
		))
	})

	It("should fall back to the raw cluster minimum when no compatible frequency is in use", func() {
		hinter := hinterWithNodes(
			NodeWithTSC("node1", 3599973000, false),
			NodeWithTSC("node2", 3599996000, false),
		)
		hinter.vmiStore = &cache.FakeCustomStore{ListFunc: func() []interface{} {
			return VMIsToObjects(
				vmiWithTSCFrequencyOnNode("existing", 3500000000, "node2"),
			)
		}}
		vmi := vmiWithTSCFrequencyOnNode("newvmi", 12, "oldnode")
		g.Expect(hinter.TopologyHintsForVMI(vmi)).To(g.Equal(
			&virtv1.TopologyHints{
				TSCFrequency: pointer.P(int64(3599973000)),
			},
		))
	})

	It("should use frequency from VMIs ignoring lower raw cluster minimum and a frequency from two nodes", func() {
		hinter := hinterWithNodes(
			NodeWithTSC("node1", 3599973000, false),
			NodeWithTSC("node2", 3599996000, false),
			NodeWithTSC("node3", 3599996000, false),
		)
		hinter.vmiStore = &cache.FakeCustomStore{ListFunc: func() []interface{} {
			return VMIsToObjects(
				vmiWithTSCFrequencyOnNode("existing", 3600000000, "node2"),
			)
		}}
		vmi := vmiWithTSCFrequencyOnNode("newvmi", 12, "oldnode")
		g.Expect(hinter.TopologyHintsForVMI(vmi)).To(g.Equal(
			&virtv1.TopologyHints{
				TSCFrequency: pointer.P(int64(3600000000)),
			},
		))
	})

	It("should use minimal frequency from VMIs", func() {
		hinter := hinterWithNodes(
			NodeWithTSC("node1", 3599973000, false),
			NodeWithTSC("node2", 3599996000, false),
			NodeWithTSC("node3", 3599996000, false),
		)
		hinter.vmiStore = &cache.FakeCustomStore{ListFunc: func() []interface{} {
			return VMIsToObjects(
				vmiWithTSCFrequencyOnNode("existing", 3600000000, "node2"),
				vmiWithTSCFrequencyOnNode("existing", 3600005000, "node3"),
				vmiWithTSCFrequencyOnNode("existing", 3600019000, "node1"),
			)
		}}
		vmi := vmiWithTSCFrequencyOnNode("newvmi", 12, "oldnode")
		g.Expect(hinter.TopologyHintsForVMI(vmi)).To(g.Equal(
			&virtv1.TopologyHints{
				TSCFrequency: pointer.P(int64(3600000000)),
			},
		))
	})

	// если есть несколько compatible frequencies в frequenciesInUse, выбирается именно минимальная;

	It("should frequencies in use on VMIs", func() {
		hinter := hinterWithVMIs(
			vmiWithTSCFrequencyOnNode("myvm", 100, "node1"),
			vmiWithTSCFrequencyOnNode("myvm1", 90, "node1"),
			vmiWithoutTSCFrequency("irritateme"),
			vmiWithTSCFrequencyOnNode("myvm2", 123, "node1"),
			vmiWithTSCFrequencyOnNode("myvm3", 80, ""),
		)
		g.Expect(hinter.TSCFrequenciesInUse()).To(g.ConsistOf(int64(100), int64(90), int64(123), int64(80)))
	})

	DescribeTable("should not propose a TSC frequency on architectures like", func(arch string) {
		hinter := hinterWithNodes(
			NodeWithInvalidTSC("node0"),
			NodeWithTSC("node1", 1234, true),
		)
		vmi := vmiWithoutTSCFrequency("myvmi")
		g.Expect(hinter.IsTscFrequencyRequired(vmi)).To(g.BeFalse())

		hints, _, err := hinter.TopologyHintsForVMI(vmi)
		g.Expect(hints).To(g.BeNil())
		g.Expect(err).To(g.Not(g.HaveOccurred()))
	},
		Entry("arm64", "arm64"),
		Entry("ppc64le", "ppc64le"),
	)
})

func hinterWithNodes(nodes ...*v1.Node) *topologyHinter {

	return &topologyHinter{
		clusterConfig: clusterConfigWithoutTSCFrequency(),
		nodeStore: &cache.FakeCustomStore{
			ListFunc: func() []interface{} {
				return NodesToObjects(nodes...)
			},
		},
		vmiStore: &cache.FakeCustomStore{
			ListFunc: func() []interface{} {
				return []interface{}{}
			},
		},
	}
}

func hinterWithVMIs(vmis ...*virtv1.VirtualMachineInstance) *topologyHinter {
	return &topologyHinter{
		vmiStore: &cache.FakeCustomStore{
			ListFunc: func() []interface{} {
				return VMIsToObjects(vmis...)
			},
		},
	}
}

func NodesToObjects(nodes ...*v1.Node) (objs []interface{}) {
	for i := range nodes {
		objs = append(objs, nodes[i])
	}
	return
}

func VMIsToObjects(vmis ...*virtv1.VirtualMachineInstance) (objs []interface{}) {
	for i := range vmis {
		objs = append(objs, vmis[i])
	}
	return
}

func NodeWithTSC(name string, frequency int64, scalable bool, schedulable ...int64) *v1.Node {
	labels := map[string]string{
		TSCFrequencyLabel:      fmt.Sprintf("%d", frequency),
		TSCScalableLabel:       fmt.Sprintf("%v", scalable),
		virtv1.NodeSchedulable: "true",
	}

	for _, freq := range schedulable {
		labels[ToTSCSchedulableLabel(freq)] = "true"
	}

	return &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}
}

func NodeWithInvalidTSC(name string) *v1.Node {
	return &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				TSCFrequencyLabel: fmt.Sprintf("%v", "a"+rand.String(5)),
				TSCScalableLabel:  fmt.Sprintf("%v", rand.String(10)),
			},
		},
	}
}
func vmiWithoutTSCFrequency(vmiName string) *virtv1.VirtualMachineInstance {
	return &virtv1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: vmiName,
		},
		Spec: virtv1.VirtualMachineInstanceSpec{
			Domain: virtv1.DomainSpec{
				CPU: &virtv1.CPU{
					Features: []virtv1.CPUFeature{
						{
							Name:   "invtsc",
							Policy: "require",
						},
					},
				},
			},
		},
	}
}

func vmiWithTSCFrequencyOnNode(vmiName string, frequency int64, nodename string) *virtv1.VirtualMachineInstance {
	return &virtv1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: vmiName,
		},
		Spec: virtv1.VirtualMachineInstanceSpec{
			Domain: virtv1.DomainSpec{
				CPU: &virtv1.CPU{
					Features: []virtv1.CPUFeature{
						{
							Name:   "invtsc",
							Policy: "require",
						},
					},
				},
			},
			Architecture: "amd64",
		},
		Status: virtv1.VirtualMachineInstanceStatus{
			NodeName:      nodename,
			TopologyHints: &virtv1.TopologyHints{TSCFrequency: &frequency},
		},
	}
}

func clusterConfigWithTSCFrequency(freq int64) *virtconfig.ClusterConfig {
	config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&virtv1.KubeVirtConfiguration{
		DeveloperConfiguration: &virtv1.DeveloperConfiguration{
			MinimumClusterTSCFrequency: &freq,
		},
	})
	return config
}

func clusterConfigWithoutTSCFrequency() *virtconfig.ClusterConfig {
	config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&virtv1.KubeVirtConfiguration{
		DeveloperConfiguration: &virtv1.DeveloperConfiguration{
			MinimumClusterTSCFrequency: nil,
		},
	})
	return config
}
