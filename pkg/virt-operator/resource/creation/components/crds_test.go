package components

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

var _ = Describe("CRDs", func() {

	table.DescribeTable("Should patch validation", func(crdFunc func() (*extv1beta1.CustomResourceDefinition, error)) {
		crd, err := crdFunc()
		Expect(err).NotTo(HaveOccurred())
		patchValidation(crd)
		Expect(crd.Spec.Validation).NotTo(BeNil())
	},
		table.Entry("for VM", NewVirtualMachineCrd),
		table.Entry("for VMI", NewVirtualMachineInstanceCrd),
		table.Entry("for VMIPRESET", NewPresetCrd),
		table.Entry("for VMIRS", NewReplicaSetCrd),
		table.Entry("for VMIM", NewVirtualMachineInstanceMigrationCrd),
		table.Entry("for KV", NewKubeVirtCrd),
		table.Entry("for VMSNAPSHOT", NewVirtualMachineSnapshotCrd),
		table.Entry("for VMSNAPSHOTCONTENT", NewVirtualMachineSnapshotContentCrd),
	)

	It("DataVolumeTemplates should have nullable a XPreserveUnknownFields on metadata", func() {
		crd, err := NewVirtualMachineCrd()
		Expect(err).NotTo(HaveOccurred())
		patchValidation(crd)
		spec := crd.Spec.Validation.OpenAPIV3Schema.Properties["spec"]
		dataVolumeTemplates := spec.Properties["dataVolumeTemplates"]
		items := dataVolumeTemplates.Items
		metadata := items.Schema.Properties["metadata"]
		Expect(metadata.Nullable).To(BeTrue())
		Expect(metadata.XPreserveUnknownFields).NotTo(BeNil())
		Expect(*metadata.XPreserveUnknownFields).To(BeTrue())

	})

	It("Template in VM should have nullable a XPreserveUnknownFields on metadata", func() {
		crd, err := NewVirtualMachineCrd()
		Expect(err).NotTo(HaveOccurred())
		patchValidation(crd)
		spec := crd.Spec.Validation.OpenAPIV3Schema.Properties["spec"]
		template := spec.Properties["template"]
		metadata := template.Properties["metadata"]
		Expect(metadata.Nullable).To(BeTrue())
		Expect(metadata.XPreserveUnknownFields).NotTo(BeNil())
		Expect(*metadata.XPreserveUnknownFields).To(BeTrue())
	})

	It("Template in VMRS should have nullable a XPreserveUnknownFields on metadata", func() {
		crd, err := NewReplicaSetCrd()
		Expect(err).NotTo(HaveOccurred())
		patchValidation(crd)
		spec := crd.Spec.Validation.OpenAPIV3Schema.Properties["spec"]
		template := spec.Properties["template"]
		metadata := template.Properties["metadata"]
		Expect(metadata.Nullable).To(BeTrue())
		Expect(metadata.XPreserveUnknownFields).NotTo(BeNil())
		Expect(*metadata.XPreserveUnknownFields).To(BeTrue())
	})

	It("Template in VMSnapshotContent should have nullable a XPreserveUnknownFields on metadata", func() {
		crd, err := NewVirtualMachineSnapshotContentCrd()
		Expect(err).NotTo(HaveOccurred())
		patchValidation(crd)
		spec := crd.Spec.Validation.OpenAPIV3Schema.Properties["spec"]
		source := spec.Properties["source"]
		vm := source.Properties["virtualMachine"]
		vmspec := vm.Properties["spec"]
		template := vmspec.Properties["template"]
		metadata := template.Properties["metadata"]

		Expect(metadata.Nullable).To(BeTrue())
		Expect(metadata.XPreserveUnknownFields).NotTo(BeNil())
		Expect(*metadata.XPreserveUnknownFields).To(BeTrue())
	})
})
