package operands

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	cdiv1beta1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1beta1"
)

var _ = Describe("Test operator.go", func() {
	Context("Test applyAnnotationPatch", func() {
		It("Should fail for bad json", func() {
			obj := &cdiv1beta1.CDI{}

			err := applyAnnotationPatch(obj, `{]`)
			Expect(err).To(HaveOccurred())
			fmt.Fprintf(GinkgoWriter, "Expected error: %v\n", err)
		})

		It("Should fail for single patch object (instead of an array)", func() {
			obj := &cdiv1beta1.CDI{}

			err := applyAnnotationPatch(obj, `{"op": "add", "path": "/spec/config/featureGates/-", "value": "fg1"}`)
			Expect(err).To(HaveOccurred())
			fmt.Fprintf(GinkgoWriter, "Expected error: %v\n", err)
		})

		It("Should fail for unknown op in a patch object", func() {
			obj := &cdiv1beta1.CDI{}

			err := applyAnnotationPatch(obj, `[{"op": "unknown", "path": "/spec/config/featureGates/-", "value": "fg1"}]`)
			Expect(err).To(HaveOccurred())
			fmt.Fprintf(GinkgoWriter, "Expected error: %v\n", err)
		})

		It("Should fail for wrong path - not starts with '/spec/' - patch object", func() {
			obj := &cdiv1beta1.CDI{}

			err := applyAnnotationPatch(obj, `[{"op": "add", "path": "/config/featureGates/-", "value": "fg1"}]`)
			Expect(err).To(HaveOccurred())
			fmt.Fprintf(GinkgoWriter, "Expected error: %v\n", err)
		})

		It("Should fail for adding to a not exist object", func() {
			obj := &cdiv1beta1.CDI{}

			err := applyAnnotationPatch(obj, `[{"op": "add", "path": "/spec/config/filesystemOverhead/global", "value": "65"}]`)
			Expect(err).To(HaveOccurred())
			fmt.Fprintf(GinkgoWriter, "Expected error: %v\n", err)
		})

		It("Should fail for removing non-exist field", func() {
			obj := &cdiv1beta1.CDI{
				Spec: cdiv1beta1.CDISpec{
					Config: &cdiv1beta1.CDIConfigSpec{
						FilesystemOverhead: &cdiv1beta1.FilesystemOverhead{},
					},
				},
			}

			err := applyAnnotationPatch(obj, `[{"op": "remove", "path": "/spec/config/filesystemOverhead/global"}]`)
			Expect(err).To(HaveOccurred())
			fmt.Fprintf(GinkgoWriter, "Expected error: %v\n", err)
		})

		It("Should apply annotation if everything is corrct", func() {
			obj := &cdiv1beta1.CDI{
				Spec: cdiv1beta1.CDISpec{
					Config: &cdiv1beta1.CDIConfigSpec{
						FilesystemOverhead: &cdiv1beta1.FilesystemOverhead{},
					},
				},
			}

			err := applyAnnotationPatch(obj, `[{"op": "add", "path": "/spec/config/filesystemOverhead/global", "value": "55"}]`)
			Expect(err).ToNot(HaveOccurred())
			Expect(obj.Spec.Config).NotTo(BeNil())
			Expect(obj.Spec.Config.FilesystemOverhead).NotTo(BeNil())
			Expect(obj.Spec.Config.FilesystemOverhead.Global).Should(BeEquivalentTo("55"))
		})
	})
})
