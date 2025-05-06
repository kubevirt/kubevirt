package components

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/jsonpath"

	clonev1beta1 "kubevirt.io/api/clone/v1beta1"
	v1 "kubevirt.io/api/core/v1"
	exportv1beta1 "kubevirt.io/api/export/v1beta1"
	poolv1 "kubevirt.io/api/pool/v1alpha1"
	snapshotv1beta1 "kubevirt.io/api/snapshot/v1beta1"

	"kubevirt.io/kubevirt/pkg/pointer"
)

const timestamp = "2025-01-01T12:34:56Z"

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
		Entry("for VirtualMachineInstancePreset", NewPresetCrd),
		Entry("for VirtualMachineInstanceReplicaSet", NewReplicaSetCrd, "Desired", "Current", "Ready", "Age"),
		Entry("for VirtualMachineInstanceMigration", NewVirtualMachineInstanceMigrationCrd, "Phase", "VMI"),
		Entry("for KubeVirt", NewKubeVirtCrd, "Age", "Phase"),
		Entry("for VirtualMachinePool", NewVirtualMachinePoolCrd, "Desired", "Current", "Ready", "Age"),
		Entry("for VirtualMachineSnapshot", NewVirtualMachineSnapshotCrd, "SourceKind", "SourceName", "Phase", "ReadyToUse", "CreationTime", "Error"),
		Entry("for VirtualMachineSnapshotContent", NewVirtualMachineSnapshotContentCrd, "ReadyToUse", "CreationTime", "Error"),
		Entry("for VirtualMachineRestore", NewVirtualMachineRestoreCrd, "TargetKind", "TargetName", "Complete", "RestoreTime"),
		Entry("for VirtualMachineExport", NewVirtualMachineExportCrd, "SourceKind", "SourceName", "Phase"),
		Entry("for VirtualMachineInstancetype", NewVirtualMachineInstancetypeCrd),
		Entry("for VirtualMachineClusterInstancetype", NewVirtualMachineClusterInstancetypeCrd),
		Entry("for VirtualMachinePreference", NewVirtualMachinePreferenceCrd),
		Entry("for VirtualMachineClusterPreference", NewVirtualMachineClusterPreferenceCrd),
		Entry("for VirtualMachineClone", NewVirtualMachineCloneCrd, "Phase", "SourceVirtualMachine", "TargetVirtualMachine"),
		Entry("for MigrationPolicy", NewMigrationPolicyCrd),
	)

	DescribeTable("Additional printer columns map to expected value", func(crdFunc func() (*extv1.CustomResourceDefinition, error), obj any, expected ...string) {
		crd, err := crdFunc()
		Expect(err).ToNot(HaveOccurred())
		s := serialize(obj)
		for _, version := range crd.Spec.Versions {
			Expect(version.AdditionalPrinterColumns).To(HaveLen(len(expected)))
			for i := range expected {
				Expect(extract(version.AdditionalPrinterColumns[i].JSONPath, s)).To(Equal(expected[i]))
			}
		}
	},
		Entry("for VirtualMachineInstance", NewVirtualMachineInstanceCrd,
			v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: createTime(),
				},
				Status: v1.VirtualMachineInstanceStatus{
					Phase: v1.Running,
					Interfaces: []v1.VirtualMachineInstanceNetworkInterface{
						{
							IP: "1.2.3.4",
						},
					},
					NodeName: "test-node",
					Conditions: []v1.VirtualMachineInstanceCondition{
						{
							Type:   "Ready",
							Status: "True",
						},
						{
							Type:   "LiveMigratable",
							Status: "False",
						},
						{
							Type:   "Paused",
							Status: "False",
						},
					},
				},
			},
			timestamp, "Running", "1.2.3.4", "test-node", "True", "False", "False",
		),
		Entry("for VirtualMachine", NewVirtualMachineCrd,
			v1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: createTime(),
				},
				Status: v1.VirtualMachineStatus{
					PrintableStatus: v1.VirtualMachineStatusRunning,
					Conditions: []v1.VirtualMachineCondition{
						{
							Type:   "Ready",
							Status: "True",
						},
					},
				},
			},
			timestamp, "Running", "True",
		),
		Entry("for VirtualMachineInstanceReplicaSet", NewReplicaSetCrd,
			v1.VirtualMachineInstanceReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: createTime(),
				},
				Spec: v1.VirtualMachineInstanceReplicaSetSpec{
					Replicas: pointer.P(int32(2)),
				},
				Status: v1.VirtualMachineInstanceReplicaSetStatus{
					Replicas:      int32(4),
					ReadyReplicas: int32(5),
				},
			},
			"2", "4", "5", timestamp,
		),
		Entry("for VirtualMachineInstanceMigration", NewVirtualMachineInstanceMigrationCrd,
			v1.VirtualMachineInstanceMigration{
				Spec: v1.VirtualMachineInstanceMigrationSpec{
					VMIName: "test-vmi",
				},
				Status: v1.VirtualMachineInstanceMigrationStatus{
					Phase: v1.MigrationRunning,
				},
			},
			"Running", "test-vmi",
		),
		Entry("for KubeVirt", NewKubeVirtCrd,
			v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: createTime(),
				},
				Status: v1.KubeVirtStatus{
					Phase: v1.KubeVirtPhaseDeployed,
				},
			},
			timestamp, "Deployed",
		),
		Entry("for VirtualMachinePool", NewVirtualMachinePoolCrd,
			poolv1.VirtualMachinePool{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: createTime(),
				},
				Spec: poolv1.VirtualMachinePoolSpec{
					Replicas: pointer.P(int32(2)),
				},
				Status: poolv1.VirtualMachinePoolStatus{
					Replicas:      int32(4),
					ReadyReplicas: int32(5),
				},
			},
			"2", "4", "5", timestamp,
		),
		Entry("for VirtualMachineSnapshot", NewVirtualMachineSnapshotCrd,
			snapshotv1beta1.VirtualMachineSnapshot{
				Spec: snapshotv1beta1.VirtualMachineSnapshotSpec{
					Source: k8sv1.TypedLocalObjectReference{
						Kind: "VirtualMachine",
						Name: "test-vm",
					},
				},
				Status: &snapshotv1beta1.VirtualMachineSnapshotStatus{
					Phase:        snapshotv1beta1.InProgress,
					ReadyToUse:   pointer.P(false),
					CreationTime: pointer.P(createTime()),
					Error: &snapshotv1beta1.Error{
						Message: pointer.P("test-error"),
					},
				},
			},
			"VirtualMachine", "test-vm", "InProgress", "false", timestamp, "test-error",
		),
		Entry("for VirtualMachineSnapshotContent", NewVirtualMachineSnapshotContentCrd,
			snapshotv1beta1.VirtualMachineSnapshotContent{
				Status: &snapshotv1beta1.VirtualMachineSnapshotContentStatus{
					ReadyToUse:   pointer.P(false),
					CreationTime: pointer.P(createTime()),
					Error: &snapshotv1beta1.Error{
						Message: pointer.P("test-error"),
					},
				},
			},
			"false", timestamp, "test-error",
		),
		Entry("for VirtualMachineRestore", NewVirtualMachineRestoreCrd,
			snapshotv1beta1.VirtualMachineRestore{
				Spec: snapshotv1beta1.VirtualMachineRestoreSpec{
					Target: k8sv1.TypedLocalObjectReference{
						Kind: "VirtualMachine",
						Name: "test-vm",
					},
				},
				Status: &snapshotv1beta1.VirtualMachineRestoreStatus{
					Complete:    pointer.P(false),
					RestoreTime: pointer.P(createTime()),
				},
			},
			"VirtualMachine", "test-vm", "false", timestamp,
		),
		Entry("for VirtualMachineExport", NewVirtualMachineExportCrd,
			exportv1beta1.VirtualMachineExport{
				Spec: exportv1beta1.VirtualMachineExportSpec{
					Source: k8sv1.TypedLocalObjectReference{
						Kind: "VirtualMachine",
						Name: "test-vm",
					},
				},
				Status: &exportv1beta1.VirtualMachineExportStatus{
					Phase: exportv1beta1.Ready,
				},
			},
			"VirtualMachine", "test-vm", "Ready",
		),
		Entry("for VirtualMachineClone", NewVirtualMachineCloneCrd,
			clonev1beta1.VirtualMachineClone{
				Spec: clonev1beta1.VirtualMachineCloneSpec{
					Source: &k8sv1.TypedLocalObjectReference{
						Name: "test-source",
					},
					Target: &k8sv1.TypedLocalObjectReference{
						Name: "test-target",
					},
				},
				Status: clonev1beta1.VirtualMachineCloneStatus{
					Phase: clonev1beta1.RestoreInProgress,
				},
			},
			"RestoreInProgress", "test-source", "test-target",
		),
	)
})

func createTime() metav1.Time {
	p, err := time.Parse(time.RFC3339, timestamp)
	Expect(err).ToNot(HaveOccurred())
	return metav1.Time{Time: p}
}

func serialize(data any) map[string]any {
	m, err := json.Marshal(data)
	Expect(err).ToNot(HaveOccurred())
	var obj map[string]any
	Expect(json.Unmarshal(m, &obj)).To(Succeed())
	return obj
}

func extract(path string, data any) string {
	jp := jsonpath.New(path)
	Expect(jp.Parse(fmt.Sprintf("{ %s }", path))).To(Succeed())
	var buf bytes.Buffer
	Expect(jp.Execute(&buf, data)).To(Succeed())
	return buf.String()
}
