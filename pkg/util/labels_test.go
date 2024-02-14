package util

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Test Labels", func() {
	Context("test MergeLabels", func() {

		It("should not copy the labels if the map is nil", func() {
			src := &metav1.ObjectMeta{
				Labels: nil,
			}
			dest := &metav1.ObjectMeta{}

			MergeLabels(src, dest)

			Expect(dest.Labels).To(BeNil())
		})

		It("should create an label map if the source map is empty", func() {
			src := &metav1.ObjectMeta{
				Labels: make(map[string]string),
			}
			dest := &metav1.ObjectMeta{}

			MergeLabels(src, dest)

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

			MergeLabels(src, dest)

			Expect(dest.Labels).ToNot(BeNil())
			Expect(dest.Labels).To(HaveLen(2))
			Expect(dest.Labels).To(HaveKeyWithValue("aaa", "111"))
			Expect(dest.Labels).To(HaveKeyWithValue("bbb", "222"))
		})

		It("should merge the label map without removing dest items", func() {
			src := &metav1.ObjectMeta{
				Labels: map[string]string{
					"aaa": "111",
					"bbb": "222",
					"ccc": "333",
				},
			}
			dest := &metav1.ObjectMeta{
				Labels: map[string]string{
					"bbb": "444",
					"ccc": "444",
					"ddd": "444",
				},
			}

			MergeLabels(src, dest)

			Expect(dest.Labels).ToNot(BeNil())
			Expect(dest.Labels).To(HaveLen(4))
			Expect(dest.Labels).To(HaveKeyWithValue("aaa", "111"))
			Expect(dest.Labels).To(HaveKeyWithValue("bbb", "222"))
			Expect(dest.Labels).To(HaveKeyWithValue("ccc", "333"))
			Expect(dest.Labels).To(HaveKeyWithValue("ddd", "444"))
		})

	})

	Context("test CompareLabels", func() {

		DescribeTable("should compare source and target Labels ingnoring additional labels on target", func(src, tgt metav1.ObjectMeta, expected bool) {
			scm := &corev1.ConfigMap{
				ObjectMeta: src,
			}
			tcm := &corev1.ConfigMap{
				ObjectMeta: tgt,
			}
			c := CompareLabels(scm, tcm)
			if expected {
				Expect(c).To(BeTrue())
			} else {
				Expect(c).To(BeFalse())
			}

		},
			Entry("nil labels on source and target",
				metav1.ObjectMeta{
					Labels: nil,
				},
				metav1.ObjectMeta{
					Labels: nil,
				},
				true,
			),
			Entry("nil labels on source, empty on target",
				metav1.ObjectMeta{
					Labels: nil,
				},
				metav1.ObjectMeta{
					Labels: make(map[string]string),
				},
				true,
			),
			Entry("empty on source, nil on target",
				metav1.ObjectMeta{
					Labels: make(map[string]string),
				},
				metav1.ObjectMeta{
					Labels: nil,
				},
				true,
			),
			Entry("equal on source and target",
				metav1.ObjectMeta{
					Labels: map[string]string{
						"aaa": "111",
						"bbb": "222",
						"ccc": "333",
					},
				},
				metav1.ObjectMeta{
					Labels: map[string]string{
						"aaa": "111",
						"bbb": "222",
						"ccc": "333",
					},
				},
				true,
			),
			Entry("source is a subset of target",
				metav1.ObjectMeta{
					Labels: map[string]string{
						"aaa": "111",
						"bbb": "222",
					},
				},
				metav1.ObjectMeta{
					Labels: map[string]string{
						"aaa": "111",
						"bbb": "222",
						"ccc": "333",
					},
				},
				true,
			),
			Entry("one label with a different value",
				metav1.ObjectMeta{
					Labels: map[string]string{
						"aaa": "111",
						"bbb": "222",
						"ccc": "333",
					},
				},
				metav1.ObjectMeta{
					Labels: map[string]string{
						"aaa": "111",
						"bbb": "222",
						"ccc": "444",
					},
				},
				false,
			),
			Entry("one extra label on source",
				metav1.ObjectMeta{
					Labels: map[string]string{
						"aaa": "111",
						"bbb": "222",
						"ccc": "333",
					},
				},
				metav1.ObjectMeta{
					Labels: map[string]string{
						"aaa": "111",
						"bbb": "222",
					},
				},
				false,
			),
			Entry("labels on source, nil on target",
				metav1.ObjectMeta{
					Labels: map[string]string{
						"aaa": "111",
						"bbb": "222",
						"ccc": "333",
					},
				},
				metav1.ObjectMeta{
					Labels: nil,
				},
				false,
			),
			Entry("labels on source, empty on target",
				metav1.ObjectMeta{
					Labels: map[string]string{
						"aaa": "111",
						"bbb": "222",
						"ccc": "333",
					},
				},
				metav1.ObjectMeta{
					Labels: make(map[string]string),
				},
				false,
			),
			Entry("nil on source, labels on target",
				metav1.ObjectMeta{
					Labels: nil,
				},
				metav1.ObjectMeta{
					Labels: map[string]string{
						"aaa": "111",
						"bbb": "222",
						"ccc": "333",
					},
				},
				true,
			),
			Entry("empty on source, labels on target",
				metav1.ObjectMeta{
					Labels: make(map[string]string),
				},
				metav1.ObjectMeta{
					Labels: map[string]string{
						"aaa": "111",
						"bbb": "222",
						"ccc": "333",
					},
				},
				true,
			),
		)
	})

})
