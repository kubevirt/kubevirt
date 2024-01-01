package matcher

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Owner", func() {

	var toNilPointer *v1.Pod = nil

	var ownedPod = func(ownerReferences []metav1.OwnerReference) *v1.Pod {
		return &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: ownerReferences,
			},
		}
	}

	DescribeTable("should", func(pod interface{}, match bool) {
		success, err := HaveOwners().Match(pod)
		Expect(err).ToNot(HaveOccurred())
		Expect(success).To(Equal(match))
		Expect(HaveOwners().FailureMessage(pod)).ToNot(BeEmpty())
		Expect(HaveOwners().NegatedFailureMessage(pod)).ToNot(BeEmpty())
	},
		Entry("[test_cid:17983]with an owner present report it as present", ownedPod([]metav1.OwnerReference{{}}), true),
		Entry("[test_cid:41344]with no owner present report it as missing", ownedPod([]metav1.OwnerReference{}), false),
		Entry("[test_cid:31715]cope with a nil pod", nil, false),
		Entry("[test_cid:14526]cope with an object pointing to nil", toNilPointer, false),
		Entry("[test_cid:22194]cope with an object which has nil as owners array", ownedPod(nil), false),
	)
})
