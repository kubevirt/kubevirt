package topology_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virt-controller/watch/topology"
)

var _ = Describe("TSC", func() {

	It("should extract TSC frequencies on nodes from labels", func() {
		n := node("mynode", true)
		n.Labels[tscFrequencyLabel(123)] = "true"
		n.Labels["random"] = "label"
		n.Labels[tscFrequencyLabel(456)] = "true"
		Expect(topology.TSCFrequenciesOnNode(n)).To(ConsistOf(int64(123), int64(456)))
	})

	It("should be able to handle invalid TSC frequency labels", func() {
		n := node("mynode", true)
		n.Labels[tscFrequencyLabel(123)] = "true"
		n.Labels[topology.TSCFrequencySchedulingLabel+"-sowrong"] = "true"
		n.Labels[tscFrequencyLabel(456)] = "true"
		Expect(topology.TSCFrequenciesOnNode(n)).To(ConsistOf(int64(123), int64(456)))
	})

	It("should convert a frequency to a proper label", func() {
		Expect(topology.ToTSCSchedulableLabels([]int64{123, 456})).To(ConsistOf(
			topology.TSCFrequencySchedulingLabel+"-123",
			topology.TSCFrequencySchedulingLabel+"-456",
		))
	})

	DescribeTable("should calculate the node label diff", func(frequenciesInUse []int64, frequenciesOnNode []int64, nodeFrequency int64, scalable bool, expectedToAdd []int64, expectedToRemove []int64) {
		toAdd, toRemove := topology.CalculateTSCLabelDiff(frequenciesInUse, frequenciesOnNode, nodeFrequency, scalable)
		Expect(toAdd).To(Equal(expectedToAdd))
		Expect(toRemove).To(Equal(expectedToRemove))
	},
		Entry(
			"on a scalable node",
			[]int64{1, 2, 3},
			[]int64{2, 4},
			int64(123),
			true,
			[]int64{1, 2, 3, 123},
			[]int64{4},
		),
		Entry(
			"on a scalable node where not all required frequencies are compatible",
			[]int64{1, 2, 3, 200},
			[]int64{2, 4},
			int64(123),
			true,
			[]int64{1, 2, 3, 123},
			[]int64{4},
		),
		Entry(
			"on a not scalable node where only the node frequency can be set",
			[]int64{1, 2, 3},
			[]int64{2, 4},
			int64(123),
			false,
			[]int64{123},
			[]int64{2, 4},
		),
	)
})

func tscFrequencyLabel(freq int64) string {
	return fmt.Sprintf("%s-%v", topology.TSCFrequencySchedulingLabel, freq)
}
