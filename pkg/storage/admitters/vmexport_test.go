/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package admitters

import (
	"context"
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	v1 "kubevirt.io/api/core/v1"
	exportv1 "kubevirt.io/api/export/v1"
	templateapi "kubevirt.io/virt-template-api/core"

	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

var _ = Describe("Validating VirtualMachineExport Admitter", func() {
	apiGroup := "v1"
	snapshotApiGroup := "snapshot.kubevirt.io"
	kubevirtApiGroup := "kubevirt.io"
	backupApiGroup := "backup.kubevirt.io"
	templateApiGroup := templateapi.GroupName

	config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})
	ociOnlyConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
		DeveloperConfiguration: &v1.DeveloperConfiguration{
			FeatureGates: []string{featuregate.OCIExport},
		},
	})
	templateOnlyConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
		DeveloperConfiguration: &v1.DeveloperConfiguration{
			FeatureGates: []string{featuregate.Template},
		},
	})
	vmTemplateExportConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
		DeveloperConfiguration: &v1.DeveloperConfiguration{
			FeatureGates: []string{featuregate.Template, featuregate.OCIExport},
		},
	})

	Context("VMExport", func() {
		It("should reject invalid request resource", func() {
			ar := &admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					Resource: webhooks.VirtualMachineGroupVersionResource,
				},
			}

			resp := createTestVMExportAdmitter(config).Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).Should(ContainSubstring("unexpected resource"))
		})

		createBlankPVCObjectRef := func() corev1.TypedLocalObjectReference {
			return corev1.TypedLocalObjectReference{
				APIGroup: &apiGroup,
				Kind:     pvc,
				Name:     "",
			}
		}

		createBlankVMSnapshotObjectRef := func() corev1.TypedLocalObjectReference {
			return corev1.TypedLocalObjectReference{
				APIGroup: &kubevirtApiGroup,
				Kind:     vmSnapshotKind,
				Name:     "",
			}
		}

		createBlankVMObjectRef := func() corev1.TypedLocalObjectReference {
			return corev1.TypedLocalObjectReference{
				APIGroup: &kubevirtApiGroup,
				Kind:     vmKind,
				Name:     "",
			}
		}

		createBlankVMBackupObjectRef := func() corev1.TypedLocalObjectReference {
			return corev1.TypedLocalObjectReference{
				APIGroup: &backupApiGroup,
				Kind:     vmBackupKind,
				Name:     "",
			}
		}

		createBlankVMTemplateObjectRef := func() corev1.TypedLocalObjectReference {
			return corev1.TypedLocalObjectReference{
				APIGroup: &templateApiGroup,
				Kind:     vmTemplateKind,
				Name:     "",
			}
		}

		DescribeTable("it should reject blank names", func(objectRefFunc func() corev1.TypedLocalObjectReference, errorString string) {
			export := &exportv1.VirtualMachineExport{
				Spec: exportv1.VirtualMachineExportSpec{
					Source: objectRefFunc(),
				},
			}
			ar := createExportAdmissionReview(export)
			resp := createTestVMExportAdmitter(config).Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).Should(ContainSubstring(errorString))
		},
			Entry("persistent volume claim", createBlankPVCObjectRef, "PVC name must not be empty"),
			Entry("virtual machine snapshot", createBlankVMSnapshotObjectRef, "VMSnapshot name must not be empty"),
			Entry("virtual machine", createBlankVMObjectRef, "Virtual Machine name must not be empty"),
			Entry("virtual machine backup", createBlankVMBackupObjectRef, "VirtualMachineBackup name must not be empty"),
		)

		It("should reject blank VMTemplate name when Template gate enabled", func() {
			export := &exportv1.VirtualMachineExport{
				Spec: exportv1.VirtualMachineExportSpec{
					Source: createBlankVMTemplateObjectRef(),
				},
			}
			ar := createExportAdmissionReview(export)
			resp := createTestVMExportAdmitter(vmTemplateExportConfig).Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).Should(ContainSubstring("VirtualMachineTemplate name must not be empty"))
		})

		DescribeTable("should reject VMTemplate source when required feature gates are missing", func(cfg *virtconfig.ClusterConfig) {
			export := &exportv1.VirtualMachineExport{
				Spec: exportv1.VirtualMachineExportSpec{
					Source: createBlankVMTemplateObjectRef(),
				},
			}
			ar := createExportAdmissionReview(export)
			resp := createTestVMExportAdmitter(cfg).Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).Should(ContainSubstring("OCIExport feature gate and virt-template deployment"))
		},
			Entry("no feature gates", config),
			Entry("only OCIExport", ociOnlyConfig),
			Entry("only Template", templateOnlyConfig),
		)

		It("should accept valid VMTemplate source when both gates enabled", func() {
			export := &exportv1.VirtualMachineExport{
				Spec: exportv1.VirtualMachineExportSpec{
					Source: corev1.TypedLocalObjectReference{
						APIGroup: &templateApiGroup,
						Kind:     vmTemplateKind,
						Name:     "test-template",
					},
				},
			}
			ar := createExportAdmissionReview(export)
			resp := createTestVMExportAdmitter(vmTemplateExportConfig).Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeTrue())
		})

		It("should reject unknown kind", func() {
			export := &exportv1.VirtualMachineExport{
				Spec: exportv1.VirtualMachineExportSpec{
					Source: corev1.TypedLocalObjectReference{
						APIGroup: &apiGroup,
						Kind:     "unknown",
						Name:     "test",
					},
				},
			}

			ar := createExportAdmissionReview(export)
			resp := createTestVMExportAdmitter(config).Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.source.kind"))
		})

		It("should reject spec update", func() {
			export := &exportv1.VirtualMachineExport{
				Spec: exportv1.VirtualMachineExportSpec{
					Source: corev1.TypedLocalObjectReference{
						APIGroup: &apiGroup,
						Kind:     pvc,
						Name:     "test",
					},
				},
			}

			oldExport := &exportv1.VirtualMachineExport{
				Spec: exportv1.VirtualMachineExportSpec{
					Source: corev1.TypedLocalObjectReference{
						APIGroup: &apiGroup,
						Kind:     pvc,
						Name:     "baz",
					},
				},
			}

			ar := createExportUpdateAdmissionReview(oldExport, export)
			resp := createTestVMExportAdmitter(config).Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec"))
		})

		It("should allow metadata update", func() {
			oldExport := &exportv1.VirtualMachineExport{
				Spec: exportv1.VirtualMachineExportSpec{
					Source: corev1.TypedLocalObjectReference{
						APIGroup: &apiGroup,
						Kind:     pvc,
						Name:     "test",
					},
				},
			}

			export := &exportv1.VirtualMachineExport{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers: []string{"finalizer"},
				},
				Spec: exportv1.VirtualMachineExportSpec{
					Source: corev1.TypedLocalObjectReference{
						APIGroup: &apiGroup,
						Kind:     pvc,
						Name:     "test",
					},
				},
			}

			ar := createExportUpdateAdmissionReview(oldExport, export)
			resp := createTestVMExportAdmitter(config).Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeTrue())
		})

		DescribeTable("it should allow", func(apiGroup, kind string) {
			export := &exportv1.VirtualMachineExport{
				Spec: exportv1.VirtualMachineExportSpec{
					Source: corev1.TypedLocalObjectReference{
						APIGroup: &apiGroup,
						Kind:     kind,
						Name:     "test",
					},
				},
			}

			ar := createExportAdmissionReview(export)
			resp := createTestVMExportAdmitter(config).Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeTrue(), "should allow APIGroup: %s, Kind: %s", apiGroup, kind)
		},
			Entry("persistent volume claim blank", "", pvc),
			Entry("virtual machine snapshot", snapshotApiGroup, vmSnapshotKind),
			Entry("virtual machine", kubevirtApiGroup, vmKind),
		)

		DescribeTable("it should reject invalid apigroups", func(apiGroup, kind string) {
			export := &exportv1.VirtualMachineExport{
				Spec: exportv1.VirtualMachineExportSpec{
					Source: corev1.TypedLocalObjectReference{
						APIGroup: &apiGroup,
						Kind:     kind,
						Name:     "test",
					},
				},
			}

			ar := createExportAdmissionReview(export)
			resp := createTestVMExportAdmitter(config).Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse(), "should reject APIGroup: %s, Kind: %s", apiGroup, kind)
		},
			Entry("persistent volume claim", "invalid", pvc),
			Entry("virtual machine snapshot", "invalid", vmSnapshotKind),
			Entry("virtual machine", "invalid", vmKind),
			Entry("virtual machine backup", "invalid", vmBackupKind),
		)

		It("should reject invalid VMTemplate apigroup", func() {
			invalidGroup := "invalid"
			export := &exportv1.VirtualMachineExport{
				Spec: exportv1.VirtualMachineExportSpec{
					Source: corev1.TypedLocalObjectReference{
						APIGroup: &invalidGroup,
						Kind:     vmTemplateKind,
						Name:     "test",
					},
				},
			}
			ar := createExportAdmissionReview(export)
			resp := createTestVMExportAdmitter(vmTemplateExportConfig).Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
		})
	})
})

func createExportAdmissionReview(export *exportv1.VirtualMachineExport) *admissionv1.AdmissionReview {
	bytes, _ := json.Marshal(export)

	ar := &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Operation: admissionv1.Create,
			Namespace: "foo",
			Resource: metav1.GroupVersionResource{
				Group:    "export.kubevirt.io",
				Resource: "virtualmachineexports",
			},
			Object: runtime.RawExtension{
				Raw: bytes,
			},
		},
	}

	return ar
}

func createExportUpdateAdmissionReview(old, current *exportv1.VirtualMachineExport) *admissionv1.AdmissionReview {
	oldBytes, _ := json.Marshal(old)
	currentBytes, _ := json.Marshal(current)

	ar := &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Operation: admissionv1.Update,
			Namespace: "foo",
			Resource: metav1.GroupVersionResource{
				Group:    "export.kubevirt.io",
				Resource: "virtualmachineexports",
			},
			Object: runtime.RawExtension{
				Raw: currentBytes,
			},
			OldObject: runtime.RawExtension{
				Raw: oldBytes,
			},
		},
	}

	return ar
}

func createTestVMExportAdmitter(config *virtconfig.ClusterConfig) *VMExportAdmitter {
	return &VMExportAdmitter{Config: config}
}
