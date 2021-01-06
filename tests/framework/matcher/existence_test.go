package matcher

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
)

var _ = Describe("Existence matchers", func() {

	var toNilPointer *v1.Pod = nil

	table.DescribeTable("should detect with the positive matcher", func(obj interface{}, existence bool) {
		exists, err := Exist().Match(obj)
		Expect(err).ToNot(HaveOccurred())
		Expect(exists).To(Equal(existence))
		Expect(Exist().FailureMessage(obj)).ToNot(BeEmpty())
		Expect(Exist().NegatedFailureMessage(obj)).ToNot(BeEmpty())
	},
		table.Entry("a nil object", nil, false),
		table.Entry("a pod", &v1.Pod{}, true),
	)
	table.DescribeTable("should detect with the negative matcher", func(obj interface{}, existence bool) {
		exists, err := BeGone().Match(obj)
		Expect(err).ToNot(HaveOccurred())
		Expect(exists).To(Equal(existence))
		Expect(BeGone().FailureMessage(obj)).ToNot(BeEmpty())
		Expect(BeGone().NegatedFailureMessage(obj)).ToNot(BeEmpty())
	},
		table.Entry("the existence of a set of pods", []*v1.Pod{{}, {}}, false),
		table.Entry("the absence of a set of pods", []*v1.Pod{}, true),
		table.Entry("a nil object", nil, true),
		table.Entry("an object pointing to nil", toNilPointer, true),
		table.Entry("a pod", &v1.Pod{}, false),
	)
})
