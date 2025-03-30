package matcher

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Existence matchers", func() {
	var toNilPointer *v1.Pod = nil

	DescribeTable("should detect with the positive matcher", func(obj interface{}, existence bool) {
		exists, err := Exist().Match(obj)
		Expect(err).ToNot(HaveOccurred())
		Expect(exists).To(Equal(existence))
		Expect(Exist().FailureMessage(obj)).ToNot(BeEmpty())
		Expect(Exist().NegatedFailureMessage(obj)).ToNot(BeEmpty())
	},
		Entry("a nil object", nil, false),
		Entry("a pod", &v1.Pod{}, true),
	)
	DescribeTable("should detect with the negative matcher", func(obj interface{}, existence bool) {
		exists, err := BeGone().Match(obj)
		Expect(err).ToNot(HaveOccurred())
		Expect(exists).To(Equal(existence))
		Expect(BeGone().FailureMessage(obj)).ToNot(BeEmpty())
		Expect(BeGone().NegatedFailureMessage(obj)).ToNot(BeEmpty())
	},
		Entry("the existence of a set of pods", []*v1.Pod{{}, {}}, false),
		Entry("the absence of a set of pods", []*v1.Pod{}, true),
		Entry("a nil object", nil, true),
		Entry("an object pointing to nil", toNilPointer, true),
		Entry("a pod", &v1.Pod{}, false),
	)

	It("formating", func() {
		obj := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "HI",
				ManagedFields: []metav1.ManagedFieldsEntry{
					{
						Manager: "something",
					},
				},
			},
		}
		Expect(BeGone().FailureMessage(obj.DeepCopy())).To(
			SatisfyAll(
				ContainSubstring("metadata"),
				ContainSubstring("status"),
				Not(ContainSubstring("something")),
				Not(ContainSubstring("Spec")),
			),
		)
		Expect(BeGone().NegatedFailureMessage(obj.DeepCopy())).To(
			SatisfyAll(
				ContainSubstring("metadata"),
				ContainSubstring("status"),
				Not(ContainSubstring("something")),
				Not(ContainSubstring("Spec")),
			),
		)
	})
})
