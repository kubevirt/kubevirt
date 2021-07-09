package matcher

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
)

var _ = Describe("Matcher", func() {

	var toNilPointer *v1.Pod = nil

	var runningPod = &v1.Pod{
		Status: v1.PodStatus{
			Phase: v1.PodRunning,
		},
	}

	table.DescribeTable("should", func(exptectedPhase interface{}, pod interface{}, match bool) {
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
})
