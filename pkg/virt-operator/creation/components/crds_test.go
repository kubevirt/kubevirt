package components

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

var _ = Describe("CRDs", func() {

	table.DescribeTable("Should patch validation", func(crd *extv1beta1.CustomResourceDefinition) {
		patchValidation(crd)
		Expect(crd.Spec.Validation).NotTo(BeNil())
	},
		table.Entry("for VM", NewVirtualMachineCrd()),
		table.Entry("for VMI", NewVirtualMachineInstanceCrd()),
		table.Entry("for VMIPRESET", NewPresetCrd()),
		table.Entry("for VMIRS", NewReplicaSetCrd()),
		table.Entry("for VMIM", NewVirtualMachineInstanceMigrationCrd()),
		table.Entry("for KV", NewKubeVirtCrd()),
		table.Entry("for VMSNAPSHOT", NewVirtualMachineSnapshotCrd()),
		table.Entry("for VMSNAPSHOTCONTENT", NewVirtualMachineSnapshotContentCrd()),
	)
})
