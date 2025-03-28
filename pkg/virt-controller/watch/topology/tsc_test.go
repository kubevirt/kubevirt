package topology_test

import (
	"fmt"

	"kubevirt.io/kubevirt/pkg/pointer"

	v1 "kubevirt.io/api/core/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/libvmi"
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
		Expect(toAdd).To(ConsistOf(expectedToAdd))
		Expect(toRemove).To(ConsistOf(expectedToRemove))
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
			[]int64{1, 2, 3, 123130, 200000}, // 123130 is above but within 250 PPM
			[]int64{2, 4},
			int64(123123),
			true,
			[]int64{1, 2, 3, 123123, 123130},
			[]int64{4},
		),
		Entry(
			"on a non-scalable node where only the node frequency can be set",
			[]int64{1, 2, 3},
			[]int64{2, 4},
			int64(123),
			false,
			[]int64{123},
			[]int64{2, 4},
		),
		Entry(
			"on a non-scalable node where other node frequencies are close-enough",
			[]int64{1, 2, 123120, 123130}, // 250 PPM of 123123 is 30
			[]int64{2, 4},
			int64(123123),
			false,
			[]int64{123123, 123120, 123130},
			[]int64{2, 4},
		),
	)

	Context("needs to be set when", func() {
		newVmi := func(options ...libvmi.Option) *v1.VirtualMachineInstance {
			vmi := libvmi.New(options...)
			vmi.Status.TopologyHints = &v1.TopologyHints{TSCFrequency: pointer.P(int64(12345))}

			return vmi
		}

		It("invtsc feature exists", func() {
			vmi := newVmi(
				libvmi.WithCPUFeature("invtsc", "require"),
			)

			Expect(topology.IsManualTSCFrequencyRequired(vmi)).To(BeTrue())
		})

		It("HyperV reenlightenment is enabled", func() {
			vmi := newVmi()
			vmi.Spec.Domain.Features = &v1.Features{
				Hyperv: &v1.FeatureHyperv{
					Reenlightenment: &v1.FeatureState{Enabled: pointer.P(true)},
				},
			}

			Expect(topology.IsManualTSCFrequencyRequired(vmi)).To(BeTrue())
		})
	})
})

func tscFrequencyLabel(freq int64) string {
	return fmt.Sprintf("%s-%v", topology.TSCFrequencySchedulingLabel, freq)
}
