package components

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

var _ = Describe("CRDs", func() {
	DescribeTable("Should patch validation", func(crdFunc func() (*extv1.CustomResourceDefinition, error)) {
		crd, err := crdFunc()
		Expect(err).NotTo(HaveOccurred())
		for i := range crd.Spec.Versions {
			patchValidation(crd, &crd.Spec.Versions[i])
			Expect(crd.Spec.Versions[i].Schema).NotTo(BeNil())
		}
	},
		Entry("for VM", NewVirtualMachineCrd),
		Entry("for VMI", NewVirtualMachineInstanceCrd),
		Entry("for VMIPRESET", NewPresetCrd),
		Entry("for VMIRS", NewReplicaSetCrd),
		Entry("for VMIM", NewVirtualMachineInstanceMigrationCrd),
		Entry("for KV", NewKubeVirtCrd),
		Entry("for VMSNAPSHOT", NewVirtualMachineSnapshotCrd),
		Entry("for VMSNAPSHOTCONTENT", NewVirtualMachineSnapshotContentCrd),
		Entry("for VMPOOL", NewVirtualMachinePoolCrd),
	)

	It("DataVolumeTemplates should have nullable a XPreserveUnknownFields on metadata", func() {
		crd, err := NewVirtualMachineCrd()
		Expect(err).NotTo(HaveOccurred())
		for i := range crd.Spec.Versions {
			patchValidation(crd, &crd.Spec.Versions[i])
			spec := crd.Spec.Versions[i].Schema.OpenAPIV3Schema.Properties["spec"]
			dataVolumeTemplates := spec.Properties["dataVolumeTemplates"]
			items := dataVolumeTemplates.Items
			metadata := items.Schema.Properties["metadata"]
			Expect(metadata.Nullable).To(BeTrue())
			Expect(metadata.XPreserveUnknownFields).NotTo(BeNil())
			Expect(*metadata.XPreserveUnknownFields).To(BeTrue())
		}
	})

	It("Template in VM should have nullable a XPreserveUnknownFields on metadata", func() {
		crd, err := NewVirtualMachineCrd()
		Expect(err).NotTo(HaveOccurred())
		for i := range crd.Spec.Versions {
			patchValidation(crd, &crd.Spec.Versions[i])
			spec := crd.Spec.Versions[i].Schema.OpenAPIV3Schema.Properties["spec"]
			template := spec.Properties["template"]
			metadata := template.Properties["metadata"]
			Expect(metadata.Nullable).To(BeTrue())
			Expect(metadata.XPreserveUnknownFields).NotTo(BeNil())
			Expect(*metadata.XPreserveUnknownFields).To(BeTrue())
		}
	})

	It("Template in VMRS should have nullable a XPreserveUnknownFields on metadata", func() {
		crd, err := NewReplicaSetCrd()
		Expect(err).NotTo(HaveOccurred())
		for i := range crd.Spec.Versions {
			patchValidation(crd, &crd.Spec.Versions[i])
			spec := crd.Spec.Versions[i].Schema.OpenAPIV3Schema.Properties["spec"]
			template := spec.Properties["template"]
			metadata := template.Properties["metadata"]
			Expect(metadata.Nullable).To(BeTrue())
			Expect(metadata.XPreserveUnknownFields).NotTo(BeNil())
			Expect(*metadata.XPreserveUnknownFields).To(BeTrue())
		}
	})

	It("Template in VMSnapshotContent should have nullable a XPreserveUnknownFields on metadata", func() {
		crd, err := NewVirtualMachineSnapshotContentCrd()
		Expect(err).NotTo(HaveOccurred())
		for i := range crd.Spec.Versions {
			patchValidation(crd, &crd.Spec.Versions[i])
			spec := crd.Spec.Versions[i].Schema.OpenAPIV3Schema.Properties["spec"]
			source := spec.Properties["source"]
			vm := source.Properties["virtualMachine"]
			vmspec := vm.Properties["spec"]
			template := vmspec.Properties["template"]
			metadata := template.Properties["metadata"]

			Expect(metadata.Nullable).To(BeTrue())
			Expect(metadata.XPreserveUnknownFields).NotTo(BeNil())
			Expect(*metadata.XPreserveUnknownFields).To(BeTrue())
		}
	})
})
