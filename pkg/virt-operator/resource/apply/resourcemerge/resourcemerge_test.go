package resourcemerge_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virt-operator/resource/apply/resourcemerge"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("ResourceMerge", func() {
	DescribeTableSubtree("string field",
		func(getter func(metav1.Object) string, setter func(metav1.Object, string)) {
			const testValue = "test-value"

			It("should set field from required object", func() {
				var existingMeta metav1.ObjectMeta
				var requiredMeta metav1.ObjectMeta

				setter(&requiredMeta, testValue)

				Expect(resourcemerge.EnsureObjectMeta(&existingMeta, requiredMeta)).To(BeTrue())
				Expect(getter(&existingMeta)).To(Equal(testValue))
			})

			It("should not change field, if required does not specify it", func() {
				var existingMeta metav1.ObjectMeta
				var requiredMeta metav1.ObjectMeta

				setter(&existingMeta, testValue)

				Expect(resourcemerge.EnsureObjectMeta(&existingMeta, requiredMeta)).To(BeFalse())
				Expect(getter(&existingMeta)).To(Equal(testValue))
			})
		},

		Entry("Namespace", metav1.Object.GetNamespace, metav1.Object.SetNamespace),
		Entry("Name", metav1.Object.GetName, metav1.Object.SetName),
	)

	DescribeTableSubtree("map field",
		func(getter func(metav1.Object) map[string]string, setter func(metav1.Object, map[string]string)) {
			It("should merge from required object to existing one", func() {
				var existingMeta metav1.ObjectMeta
				var requiredMeta metav1.ObjectMeta
				setter(&existingMeta, map[string]string{"key1": "value1"})
				setter(&requiredMeta, map[string]string{"key2": "value2"})

				Expect(resourcemerge.EnsureObjectMeta(&existingMeta, requiredMeta)).To(BeTrue())
				Expect(getter(&existingMeta)).To(HaveKeyWithValue("key1", "value1"))
				Expect(getter(&existingMeta)).To(HaveKeyWithValue("key2", "value2"))
			})

			It("should create map, if it is nil", func() {
				var existingMeta metav1.ObjectMeta
				var requiredMeta metav1.ObjectMeta
				setter(&existingMeta, map[string]string{"key1": "value1"})
				setter(&requiredMeta, map[string]string{"key2": "value2"})

				Expect(resourcemerge.EnsureObjectMeta(&existingMeta, requiredMeta)).To(BeTrue())
				Expect(getter(&existingMeta)).To(HaveKeyWithValue("key1", "value1"))
				Expect(getter(&existingMeta)).To(HaveKeyWithValue("key2", "value2"))
			})
		},

		Entry("Labels", metav1.Object.GetLabels, metav1.Object.SetLabels),
		Entry("Annotations", metav1.Object.GetAnnotations, metav1.Object.SetAnnotations),
	)

	Context("Owner References", func() {
		It("should merge owner reference, if it exists", func() {
			existingOwnerReference := metav1.OwnerReference{
				Name:       "owner-object",
				Kind:       "Pod",
				APIVersion: "v1",
				UID:        "12345",
			}
			existingMeta := metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{existingOwnerReference},
			}
			requiredMeta := metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{{
					Name:       existingOwnerReference.Name,
					Kind:       existingOwnerReference.Kind,
					APIVersion: existingOwnerReference.APIVersion,
					UID:        "67890",
				}},
			}

			Expect(resourcemerge.EnsureObjectMeta(&existingMeta, requiredMeta)).To(BeTrue())
			Expect(existingMeta.OwnerReferences).To(HaveLen(1))
			Expect(existingMeta.OwnerReferences[0]).To(Equal(requiredMeta.OwnerReferences[0]))
		})

		It("should add owner reference, if not exists", func() {
			existingMeta := metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{{
					Name:       "owner-object",
					Kind:       "Pod",
					APIVersion: "v1",
					UID:        "12345",
				}},
			}
			requiredOwner := metav1.OwnerReference{
				Name:       "another-owner",
				Kind:       "Deployment",
				APIVersion: "apps/v1",
				UID:        "abcde",
			}
			requiredMeta := metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{requiredOwner},
			}

			Expect(resourcemerge.EnsureObjectMeta(&existingMeta, requiredMeta)).To(BeTrue())
			Expect(existingMeta.OwnerReferences).To(HaveLen(2))
			Expect(existingMeta.OwnerReferences).To(ContainElement(requiredOwner))
		})

		It("should create owner references, if nil", func() {
			existingMeta := metav1.ObjectMeta{}
			const ownerName = "owner-object"
			requiredMeta := metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{
					{Name: ownerName, Kind: "Pod", APIVersion: "v1"},
				},
			}

			Expect(resourcemerge.EnsureObjectMeta(&existingMeta, requiredMeta)).To(BeTrue())
			Expect(existingMeta.OwnerReferences).To(HaveLen(1))
			Expect(existingMeta.OwnerReferences[0]).To(Equal(requiredMeta.OwnerReferences[0]))
		})
	})
})
