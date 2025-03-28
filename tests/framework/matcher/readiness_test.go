package matcher

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
)

var _ = Describe("Readiness", func() {
	var toNilPointer *k8sv1.Deployment = nil

	readyDeployment := &k8sv1.Deployment{
		Status: k8sv1.DeploymentStatus{
			ReadyReplicas: 2,
		},
	}

	DescribeTable("should work on a deployment", func(comparator string, count int, deployment interface{}, match bool) {
		success, err := HaveReadyReplicasNumerically(comparator, count).Match(deployment)
		Expect(err).ToNot(HaveOccurred())
		Expect(success).To(Equal(match))
		Expect(HaveReadyReplicasNumerically(comparator, count).FailureMessage(deployment)).ToNot(BeEmpty())
		Expect(HaveReadyReplicasNumerically(comparator, count).NegatedFailureMessage(deployment)).ToNot(BeEmpty())
	},
		Entry("with readyReplicas matching the expectation ", ">=", 2, readyDeployment, true),
		Entry("cope with a nil deployment", ">=", 2, nil, false),
		Entry("cope with an object pointing to nil", ">=", 2, toNilPointer, false),
		Entry("cope with an object which has no readyReplicas", ">=", 2, &v1.Service{}, false),
		Entry("cope with a non-integer object as expected readReplicas", "<=", nil, readyDeployment, false),
		Entry("with expected readyReplicas not matching the expectation", "<", 2, readyDeployment, false),
	)
})
