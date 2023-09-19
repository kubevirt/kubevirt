package topology_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-controller/watch/topology"
)

var _ = Describe("Filter", func() {

	It("should filter out not schedulable nodes", func() {
		nodes := topology.NodesToObjects(
			node("node0", true),
			node("node1", false),
			node("node2", true),
			nil,
		)
		Expect(topology.FilterNodesFromCache(nodes,
			topology.IsSchedulable,
		)).To(ConsistOf(
			nodes[0],
			nodes[2],
		))
	})

	It("should inverse the search result", func() {
		nodes := topology.NodesToObjects(
			node("node0", true),
			node("node1", false),
			node("node2", true),
		)
		Expect(topology.FilterNodesFromCache(nodes,
			topology.Not(topology.IsSchedulable),
		)).To(ConsistOf(
			nodes[1],
		))
	})

	It("should concatenate the filters", func() {
		nodes := topology.NodesToObjects(
			node("node0", true),
			node("node1", false),
			node("node2", true),
			node("", false),
			node("node4", false),
			nil,
		)
		Expect(topology.FilterNodesFromCache(nodes,
			topology.Not(topology.IsSchedulable), nameIsSet,
		)).To(ConsistOf(
			nodes[1],
			nodes[4],
		))
	})

	It("should filter nodes which have a TSC frequency", func() {
		nodes := topology.NodesToObjects(
			topology.NodeWithTSC("node0", 1234, true),
			topology.NodeWithTSC("node1", 1234, true),
			node("node2", true),
			nil,
		)
		Expect(topology.FilterNodesFromCache(nodes,
			topology.HasInvTSCFrequency,
		)).To(ConsistOf(
			nodes[0],
			nodes[1],
		))
	})

	It("should filter nodes with compatible TSC frequencies with the TSCFrequencyGreaterEqual filter", func() {
		nodes := topology.NodesToObjects(
			topology.NodeWithTSC("node0", 12345, true),
			topology.NodeWithTSC("node1", 1234, false),
			topology.NodeWithTSC("node2", 1234, true),
			topology.NodeWithTSC("node3", 12345, false),
			topology.NodeWithTSC("node4", 12, false),
			topology.NodeWithTSC("node5", 12, true),
			topology.NodeWithTSC("node6", 0, true),
			topology.NodeWithInvalidTSC("node7"),
			topology.NodeWithInvalidTSC("node8"),
			nil,
		)
		Expect(topology.FilterNodesFromCache(nodes,
			topology.TSCFrequencyGreaterEqual(1234),
		)).To(ConsistOf(
			nodes[0],
			nodes[1],
			nodes[2],
		))
	})

	It("should filter nodes with compatible TSC frequencies with the NodeOfVMI filter", func() {
		nodes := topology.NodesToObjects(
			topology.NodeWithTSC("node0", 12345, true),
			topology.NodeWithTSC("node1", 1234, false),
			topology.NodeWithTSC("node2", 1234, true),
			nil,
		)
		Expect(topology.FilterNodesFromCache(nodes,
			topology.NodeOfVMI(vmiOnNode("myvmi", "node1")),
		)).To(ConsistOf(
			nodes[1],
		))
		Expect(topology.FilterNodesFromCache(nodes,
			topology.NodeOfVMI(vmiOnNode("myvmi", "")),
		)).To(BeEmpty())
	})

	It("should filter nodes running vmis", func() {
		vmiStore := cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		vmi := vmiOnNode("myvmi", "node1")
		err := vmiStore.Add(vmi)
		Expect(err).ToNot(HaveOccurred())

		nodes := topology.NodesToObjects(
			node("node0", false),
			node("node1", false),
			node("node2", false),
			nil,
		)
		Expect(topology.FilterNodesFromCache(nodes,
			topology.IsNodeRunningVmis(vmiStore),
		)).To(ConsistOf(
			nodes[1],
		))
	})

	It("should filter nodes running vmis and schedulable nodes", func() {
		vmiStore := cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		vmi := vmiOnNode("myvmi", "node1")
		err := vmiStore.Add(vmi)
		Expect(err).ToNot(HaveOccurred())

		nodes := topology.NodesToObjects(
			node("node0", false),
			node("node1", false),
			node("node2", true),
			nil,
		)
		Expect(topology.FilterNodesFromCache(nodes,
			topology.Or(
				topology.IsNodeRunningVmis(vmiStore),
				topology.IsSchedulable,
			),
		)).To(ConsistOf(
			nodes[1],
			nodes[2],
		))
	})
})

func node(name string, schedulable bool) *v1.Node {
	return &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				virtv1.NodeSchedulable: fmt.Sprintf("%v", schedulable),
			},
		},
	}
}

func nameIsSet(node *v1.Node) bool {
	if node == nil {
		return false
	}
	if node.Name == "" {
		return false
	}
	return true
}

func vmiOnNode(vmiName, nodename string) *virtv1.VirtualMachineInstance {
	return &virtv1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: vmiName,
		},
		Status: virtv1.VirtualMachineInstanceStatus{
			NodeName: nodename,
		},
	}
}
