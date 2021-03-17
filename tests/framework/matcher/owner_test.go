package matcher

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
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

	table.DescribeTable("should", func(pod interface{}, match bool) {
		success, err := HaveOwners().Match(pod)
		Expect(err).ToNot(HaveOccurred())
		Expect(success).To(Equal(match))
		Expect(HaveOwners().FailureMessage(pod)).ToNot(BeEmpty())
		Expect(HaveOwners().NegatedFailureMessage(pod)).ToNot(BeEmpty())
	},
		table.Entry("with an owner present report it as present", ownedPod([]metav1.OwnerReference{{}}), true),
		table.Entry("with no owner present report it as missing", ownedPod([]metav1.OwnerReference{}), false),
		table.Entry("cope with a nil pod", nil, false),
		table.Entry("cope with an object pointing to nil", toNilPointer, false),
		table.Entry("cope with an object which has nil as owners array", ownedPod(nil), false),
	)
})
