package matcher

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
)

var _ = Describe("Matcher", func() {

	var toNilPointer *v1.Pod = nil
	var toNilSlicePointer []*v1.Pod = nil

	var runningPod = &v1.Pod{
		Status: v1.PodStatus{
			Phase: v1.PodRunning,
		},
	}

	var stoppedPod = &v1.Pod{
		Status: v1.PodStatus{
			Phase: v1.PodSucceeded,
		},
	}

	table.DescribeTable("should work on a pod", func(exptectedPhase interface{}, pod interface{}, match bool) {
		success, err := BeInPhase(exptectedPhase).Match(pod)
		Expect(err).ToNot(HaveOccurred())
		Expect(success).To(Equal(match))
		Expect(BeInPhase(exptectedPhase).FailureMessage(pod)).ToNot(BeEmpty())
		Expect(BeInPhase(exptectedPhase).NegatedFailureMessage(pod)).ToNot(BeEmpty())
	},
		table.Entry("with expected phase as PodPhase match the pod phase", v1.PodRunning, runningPod, true),
		table.Entry("with expected phase as string match the pod phase", "Running", runningPod, true),
		table.Entry("cope with a nil pod", v1.PodRunning, nil, false),
		table.Entry("cope with an object pointing to nil", v1.PodRunning, toNilPointer, false),
		table.Entry("cope with an object which has no phase", v1.PodRunning, &v1.Service{}, false),
		table.Entry("cope with a non-stringable object as expected phase", nil, runningPod, false),
		table.Entry("with expected phase not match the pod phase", "Succeeded", runningPod, false),
	)

	table.DescribeTable("should work on a pod array", func(exptectedPhase interface{}, array interface{}, match bool) {
		success, err := BeInPhase(exptectedPhase).Match(array)
		Expect(err).ToNot(HaveOccurred())
		Expect(success).To(Equal(match))
		Expect(BeInPhase(exptectedPhase).FailureMessage(array)).ToNot(BeEmpty())
		Expect(BeInPhase(exptectedPhase).NegatedFailureMessage(array)).ToNot(BeEmpty())
	},
		table.Entry("with expected phase as PodPhase match the pod phase", v1.PodRunning, []*v1.Pod{runningPod}, true),
		table.Entry("with expected phase as PodPhase match the pod phase when not a pointer", v1.PodRunning, []v1.Pod{*runningPod}, true),
		table.Entry("with expected phase as string match the pod phase", "Running", []*v1.Pod{runningPod, runningPod}, true),
		table.Entry("with not all pods matching the expected phase", "Running", []*v1.Pod{runningPod, stoppedPod, runningPod}, false),
		table.Entry("cope with a nil array", v1.PodRunning, nil, false),
		table.Entry("cope with a nil array pointer", v1.PodRunning, toNilSlicePointer, false),
		table.Entry("cope with a nil entry", v1.PodRunning, []*v1.Pod{nil}, false),
		table.Entry("cope with an empty array", v1.PodRunning, []*v1.Pod{}, false),
		table.Entry("cope with an object which has no phase", v1.PodRunning, []*v1.Service{{}}, false),
		table.Entry("cope with a non-stringable object as expected phase", nil, []*v1.Pod{runningPod}, false),
		table.Entry("with expected phase not match the pod phase", "Succeeded", []*v1.Pod{runningPod}, false),
	)
})
