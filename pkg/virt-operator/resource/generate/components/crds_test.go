package components

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

var _ = Describe("CRDs", func() {
	DescribeTable("Schema should be present on CRD", func(crdFunc func() (*extv1.CustomResourceDefinition, error)) {
		crd, err := crdFunc()
		Expect(err).ToNot(HaveOccurred())
		for _, version := range crd.Spec.Versions {
			Expect(version.Schema).ToNot(BeNil())
		}
	},
		Entry("for VirtualMachineInstance", NewVirtualMachineInstanceCrd),
		Entry("for VirtualMachine", NewVirtualMachineCrd),
		Entry("for VirtualMachineIPreset", NewPresetCrd),
		Entry("for VirtualMachineIReplicaSet", NewReplicaSetCrd),
		Entry("for VirtualMachineInstanceMigration", NewVirtualMachineInstanceMigrationCrd),
		Entry("for KubeVirt", NewKubeVirtCrd),
		Entry("for VirtualMachinePool", NewVirtualMachinePoolCrd),
		Entry("for VirtualMachineSnapshot", NewVirtualMachineSnapshotCrd),
		Entry("for VirtualMachineSnapshotContent", NewVirtualMachineSnapshotContentCrd),
		Entry("for VirtualMachineRestore", NewVirtualMachineRestoreCrd),
		Entry("for VirtualMachineExport", NewVirtualMachineExportCrd),
		Entry("for VirtualMachineInstancetype", NewVirtualMachineInstancetypeCrd),
		Entry("for VirtualMachineClusterInstancetype", NewVirtualMachineClusterInstancetypeCrd),
		Entry("for VirtualMachinePreference", NewVirtualMachinePreferenceCrd),
		Entry("for VirtualMachineClusterPreference", NewVirtualMachineClusterPreferenceCrd),
		Entry("for VirtualMachineClone", NewVirtualMachineCloneCrd),
		Entry("for MigrationPolicy", NewMigrationPolicyCrd),
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

	DescribeTable("Expected additional printer columns should be present on CRD", func(crdFunc func() (*extv1.CustomResourceDefinition, error), expected ...string) {
		crd, err := crdFunc()
		Expect(err).ToNot(HaveOccurred())
		for _, version := range crd.Spec.Versions {
			Expect(version.AdditionalPrinterColumns).To(WithTransform(func(definitions []extv1.CustomResourceColumnDefinition) []string {
				names := make([]string, 0)
				for _, definition := range definitions {
					names = append(names, definition.Name)
				}
				return names
			}, Equal(expected)))
		}
	},
		Entry("for VirtualMachineInstance", NewVirtualMachineInstanceCrd, "Age", "Phase", "IP", "NodeName", "Ready", "Live-Migratable", "Paused"),
		Entry("for VirtualMachine", NewVirtualMachineCrd, "Age", "Status", "Ready"),
		Entry("for VirtualMachineIPreset", NewPresetCrd),
		Entry("for VirtualMachineIReplicaSet", NewReplicaSetCrd, "Desired", "Current", "Ready", "Age"),
		Entry("for VirtualMachineInstanceMigration", NewVirtualMachineInstanceMigrationCrd, "Phase", "VMI"),
		Entry("for KubeVirt", NewKubeVirtCrd, "Age", "Phase"),
		Entry("for VirtualMachinePool", NewVirtualMachinePoolCrd, "Desired", "Current", "Ready", "Age"),
		Entry("for VirtualMachineSnapshot", NewVirtualMachineSnapshotCrd, "SourceKind", "SourceName", "Phase", "ReadyToUse", "CreationTime", "Error"),
		Entry("for VirtualMachineSnapshotContent", NewVirtualMachineSnapshotContentCrd, "ReadyToUse", "CreationTime", "Error"),
		Entry("for VirtualMachineRestore", NewVirtualMachineRestoreCrd, "TargetKind", "TargetName", "Complete", "RestoreTime", "Error"),
		Entry("for VirtualMachineExport", NewVirtualMachineExportCrd, "SourceKind", "SourceName", "Phase"),
		Entry("for VirtualMachineInstancetype", NewVirtualMachineInstancetypeCrd),
		Entry("for VirtualMachineClusterInstancetype", NewVirtualMachineClusterInstancetypeCrd),
		Entry("for VirtualMachinePreference", NewVirtualMachinePreferenceCrd),
		Entry("for VirtualMachineClusterPreference", NewVirtualMachineClusterPreferenceCrd),
		Entry("for VirtualMachineClone", NewVirtualMachineCloneCrd, "Phase", "SourceVirtualMachine", "TargetVirtualMachine"),
		Entry("for MigrationPolicy", NewMigrationPolicyCrd),
	)
})
