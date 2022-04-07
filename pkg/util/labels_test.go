package util

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Test Labels", func() {
	Context("test DeepCopyLabels", func() {

		It("should not copy the labels if the map is nil", func() {
			src := &metav1.ObjectMeta{
				Labels: nil,
			}
			dest := &metav1.ObjectMeta{}

			DeepCopyLabels(src, dest)

			Expect(dest.Labels).To(BeNil())
		})

		It("should create an label map if the source map is empty", func() {
			src := &metav1.ObjectMeta{
				Labels: make(map[string]string),
			}
			dest := &metav1.ObjectMeta{}

			DeepCopyLabels(src, dest)

			Expect(dest.Labels).ToNot(BeNil())
			Expect(dest.Labels).To(BeEmpty())
		})

		It("should copy the label map if the source map is not empty", func() {
			src := &metav1.ObjectMeta{
				Labels: map[string]string{
					"aaa": "111",
					"bbb": "222",
				},
			}
			dest := &metav1.ObjectMeta{}

			DeepCopyLabels(src, dest)

			Expect(dest.Labels).ToNot(BeNil())
			Expect(dest.Labels).To(HaveLen(2))
			Expect(dest.Labels).To(HaveKeyWithValue("aaa", "111"))
			Expect(dest.Labels).To(HaveKeyWithValue("bbb", "222"))
		})
	})
})
