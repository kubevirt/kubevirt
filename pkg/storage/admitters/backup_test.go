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
	"go.uber.org/mock/gomock"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"

	backupv1 "kubevirt.io/api/backup/v1alpha1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var _ = Describe("Validating VirtualMachineBackup Admitter", func() {
	var (
		config           *virtconfig.ClusterConfig
		kvStore          cache.Store
		vmBackupInformer cache.SharedIndexInformer
		admitter         *VMBackupAdmitter
		sourceRef        corev1.TypedLocalObjectReference
	)

	const (
		vmName   = "test-vm"
		apiGroup = "kubevirt.io"
	)
	BeforeEach(func() {
		config, _, kvStore = testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})
		enableFeatureGate(kvStore, "IncrementalBackup")
		vmBackupInformer = createTestVMBackupInformer()
		admitter = createTestVMBackupAdmitter(config, vmBackupInformer)
		sourceRef = corev1.TypedLocalObjectReference{
			APIGroup: pointer.P(apiGroup),
			Kind:     "VirtualMachine",
			Name:     vmName,
		}
	})
	Context("Resource validation", func() {
		It("should reject invalid resource group", func() {
			backup := &backupv1.VirtualMachineBackup{
				Spec: backupv1.VirtualMachineBackupSpec{},
			}

			ar := createBackupAdmissionReview(backup)
			ar.Request.Resource.Group = "invalid.group.io"

			resp := admitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).Should(ContainSubstring("unexpected resource"))
		})

		It("should reject invalid resource name", func() {
			backup := &backupv1.VirtualMachineBackup{
				Spec: backupv1.VirtualMachineBackupSpec{},
			}

			ar := createBackupAdmissionReview(backup)
			ar.Request.Resource.Resource = "invalidresource"

			resp := admitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).Should(ContainSubstring("unexpected resource"))
		})
	})

	It("should reject Create operation when IncrementalBackup feature gate is not enabled", func() {
		backup := &backupv1.VirtualMachineBackup{
			Spec: backupv1.VirtualMachineBackupSpec{},
		}

		ar := createBackupAdmissionReview(backup)
		disableFeatureGate(kvStore, "IncrementalBackup")

		resp := admitter.Admit(context.Background(), ar)
		Expect(resp.Allowed).To(BeFalse())
		Expect(resp.Result.Message).Should(Equal("IncrementalBackup feature gate not enabled"))
	})

	Context("Update operation validation", func() {
		It("should reject update if spec is changed", func() {
			oldBackup := &backupv1.VirtualMachineBackup{
				Spec: backupv1.VirtualMachineBackupSpec{
					Source:  sourceRef,
					PvcName: pointer.P("old-pvc"),
				},
			}

			newBackup := &backupv1.VirtualMachineBackup{
				Spec: backupv1.VirtualMachineBackupSpec{
					Source:  sourceRef,
					PvcName: pointer.P("new-pvc"),
				},
			}

			ar := createBackupUpdateAdmissionReview(oldBackup, newBackup)
			resp := admitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			Expect(resp.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			Expect(resp.Result.Details.Causes[0].Message).Should(Equal("spec is immutable after creation"))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec"))
		})
	})

	Context("Single backup validation", func() {
		It("should reject if backup with same name already exists", func() {
			existingBackup := &backupv1.VirtualMachineBackup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vm-backup",
					Namespace: "default",
				},
				Spec: backupv1.VirtualMachineBackupSpec{
					Source: corev1.TypedLocalObjectReference{
						APIGroup: pointer.P(apiGroup),
						Kind:     "VirtualMachine",
						Name:     vmName,
					},
					PvcName: pointer.P("test-pvc"),
				},
			}

			vmBackupInformer.GetStore().Add(existingBackup)

			newBackup := &backupv1.VirtualMachineBackup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vm-backup",
					Namespace: "default",
				},
				Spec: backupv1.VirtualMachineBackupSpec{
					Source: corev1.TypedLocalObjectReference{
						APIGroup: pointer.P(apiGroup),
						Kind:     "VirtualMachine",
						Name:     vmName,
					},
					PvcName: pointer.P("test-pvc"),
				},
			}

			ar := createBackupAdmissionReview(newBackup)

			resp := admitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			Expect(resp.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			Expect(resp.Result.Details.Causes[0].Message).Should(ContainSubstring("already exists"))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("metadata.name"))
		})

		It("should reject if another backup for the same source is in progress", func() {
			existingBackup := &backupv1.VirtualMachineBackup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "in-progress-backup",
					Namespace: "default",
				},
				Spec: backupv1.VirtualMachineBackupSpec{
					Source:  sourceRef,
					PvcName: pointer.P("test-pvc"),
				},
				Status: &backupv1.VirtualMachineBackupStatus{
					Conditions: []backupv1.Condition{
						{
							Type:   backupv1.ConditionProgressing,
							Status: corev1.ConditionTrue,
						},
					},
				},
			}

			vmBackupInformer.GetStore().Add(existingBackup)

			newBackup := &backupv1.VirtualMachineBackup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-backup",
					Namespace: "default",
				},
				Spec: backupv1.VirtualMachineBackupSpec{
					Source:  sourceRef,
					PvcName: pointer.P("test-pvc"),
				},
			}

			ar := createBackupAdmissionReview(newBackup)
			resp := admitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			Expect(resp.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			Expect(resp.Result.Details.Causes[0].Message).Should(ContainSubstring("in progress for source"))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.source"))
		})

		It("should allow creating backup for the same source if existing backup is done", func() {
			existingBackup := &backupv1.VirtualMachineBackup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "completed-backup",
					Namespace: "default",
				},
				Spec: backupv1.VirtualMachineBackupSpec{
					Source:  sourceRef,
					PvcName: pointer.P("test-pvc"),
				},
				Status: &backupv1.VirtualMachineBackupStatus{
					Conditions: []backupv1.Condition{
						{
							Type:   backupv1.ConditionDone,
							Status: corev1.ConditionTrue,
						},
					},
				},
			}

			vmBackupInformer.GetStore().Add(existingBackup)

			newBackup := &backupv1.VirtualMachineBackup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-backup",
					Namespace: "default",
				},
				Spec: backupv1.VirtualMachineBackupSpec{
					Source:  sourceRef,
					PvcName: pointer.P("test-pvc"),
				},
			}

			ar := createBackupAdmissionReview(newBackup)
			resp := admitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeTrue())
			Expect(resp.Result).To(BeNil())
		})
	})

	Context("Source validation", func() {
		It("should reject if source apiGroup is missing", func() {
			backup := &backupv1.VirtualMachineBackup{
				Spec: backupv1.VirtualMachineBackupSpec{
					Source: corev1.TypedLocalObjectReference{
						Kind: "VirtualMachine",
						Name: vmName,
					},
					PvcName: pointer.P("test-pvc"),
				},
			}

			ar := createBackupAdmissionReview(backup)
			resp := admitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			Expect(resp.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueNotFound))
			Expect(resp.Result.Details.Causes[0].Message).Should(Equal("missing apiGroup"))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.source.apiGroup"))
		})

		It("should reject if source apiGroup is invalid", func() {
			invalidAPIGroup := "invalid.group.io"
			backup := &backupv1.VirtualMachineBackup{
				Spec: backupv1.VirtualMachineBackupSpec{
					Source: corev1.TypedLocalObjectReference{
						APIGroup: &invalidAPIGroup,
						Kind:     "VirtualMachine",
						Name:     vmName,
					},
					PvcName: pointer.P("test-pvc"),
				},
			}

			ar := createBackupAdmissionReview(backup)
			resp := admitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			Expect(resp.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			Expect(resp.Result.Details.Causes[0].Message).Should(Equal("invalid apiGroup"))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.source.apiGroup"))
		})

		It("should reject if source kind is invalid", func() {
			backup := &backupv1.VirtualMachineBackup{
				Spec: backupv1.VirtualMachineBackupSpec{
					Source: corev1.TypedLocalObjectReference{
						APIGroup: pointer.P(apiGroup),
						Kind:     "InvalidKind",
						Name:     vmName,
					},
					PvcName: pointer.P("test-pvc"),
				},
			}

			ar := createBackupAdmissionReview(backup)
			resp := admitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			Expect(resp.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			Expect(resp.Result.Details.Causes[0].Message).Should(Equal("invalid kind"))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.source.kind"))
		})

		It("should reject if source name is missing", func() {
			backup := &backupv1.VirtualMachineBackup{
				Spec: backupv1.VirtualMachineBackupSpec{
					Source: corev1.TypedLocalObjectReference{
						APIGroup: pointer.P(apiGroup),
						Kind:     "VirtualMachine",
						Name:     "",
					},
					PvcName: pointer.P("test-pvc"),
				},
			}

			ar := createBackupAdmissionReview(backup)
			resp := admitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			Expect(resp.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			Expect(resp.Result.Details.Causes[0].Message).Should(Equal("name is required"))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.source.name"))
		})
	})

	Context("Backup mode validation", func() {
		It("should reject invalid mode", func() {
			invalidMode := backupv1.BackupMode("InvalidMode")
			backup := &backupv1.VirtualMachineBackup{
				Spec: backupv1.VirtualMachineBackupSpec{
					Source:  sourceRef,
					Mode:    pointer.P(invalidMode),
					PvcName: pointer.P("test-pvc"),
				},
			}

			ar := createBackupAdmissionReview(backup)
			resp := admitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			Expect(resp.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			Expect(resp.Result.Details.Causes[0].Message).Should(Equal("invalid mode"))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.mode"))
		})

		It("should accept valid PushMode with PVC name", func() {
			backup := &backupv1.VirtualMachineBackup{
				Spec: backupv1.VirtualMachineBackupSpec{
					Source:  sourceRef,
					Mode:    pointer.P(backupv1.PushMode),
					PvcName: pointer.P("test-pvc"),
				},
			}

			ar := createBackupAdmissionReview(backup)
			resp := admitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeTrue())
			Expect(resp.Result).To(BeNil())
		})

		It("should reject PushMode when pvcName is nil", func() {
			backup := &backupv1.VirtualMachineBackup{
				Spec: backupv1.VirtualMachineBackupSpec{
					Source:  sourceRef,
					Mode:    pointer.P(backupv1.PushMode),
					PvcName: nil,
				},
			}

			ar := createBackupAdmissionReview(backup)
			resp := admitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			Expect(resp.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			Expect(resp.Result.Details.Causes[0].Message).Should(Equal("pvcName must be provided in push mode"))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.pvcName"))
		})

		It("should reject PushMode when pvcName is empty", func() {
			backup := &backupv1.VirtualMachineBackup{
				Spec: backupv1.VirtualMachineBackupSpec{
					Source:  sourceRef,
					Mode:    pointer.P(backupv1.PushMode),
					PvcName: pointer.P(""),
				},
			}

			ar := createBackupAdmissionReview(backup)
			resp := admitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			Expect(resp.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			Expect(resp.Result.Details.Causes[0].Message).Should(Equal("pvcName must be provided in push mode"))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.pvcName"))
		})

		It("should accept empty mode (defaults to PushMode) with PVC name", func() {
			backup := &backupv1.VirtualMachineBackup{
				Spec: backupv1.VirtualMachineBackupSpec{
					Source:  sourceRef,
					PvcName: pointer.P("test-pvc"),
				},
			}

			ar := createBackupAdmissionReview(backup)
			resp := admitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeTrue())
			Expect(resp.Result).To(BeNil())
		})
	})
})

func createBackupAdmissionReview(backup *backupv1.VirtualMachineBackup) *admissionv1.AdmissionReview {
	bytes, _ := json.Marshal(backup)

	ar := &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Operation: admissionv1.Create,
			Namespace: "default",
			Resource: metav1.GroupVersionResource{
				Group:    backupv1.SchemeGroupVersion.Group,
				Resource: "virtualmachinebackups",
			},
			Object: runtime.RawExtension{
				Raw: bytes,
			},
		},
	}

	return ar
}

func createBackupUpdateAdmissionReview(old, current *backupv1.VirtualMachineBackup) *admissionv1.AdmissionReview {
	oldBytes, _ := json.Marshal(old)
	currentBytes, _ := json.Marshal(current)

	ar := &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Operation: admissionv1.Update,
			Namespace: "default",
			Resource: metav1.GroupVersionResource{
				Group:    backupv1.SchemeGroupVersion.Group,
				Resource: "virtualmachinebackups",
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

func createTestVMBackupInformer() cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		cache.NewListWatchFromClient(nil, "", "", nil),
		nil,
		0,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)
}

func createTestVMBackupAdmitter(config *virtconfig.ClusterConfig, vmBackupInformer cache.SharedIndexInformer) *VMBackupAdmitter {
	ctrl := gomock.NewController(GinkgoT())
	virtClient := kubecli.NewMockKubevirtClient(ctrl)
	return NewVMBackupAdmitter(config, virtClient, vmBackupInformer)
}

func enableFeatureGate(kvStore cache.Store, featureGate string) {
	testutils.UpdateFakeKubeVirtClusterConfig(kvStore, &v1.KubeVirt{
		Spec: v1.KubeVirtSpec{
			Configuration: v1.KubeVirtConfiguration{
				DeveloperConfiguration: &v1.DeveloperConfiguration{
					FeatureGates: []string{featureGate},
				},
			},
		},
	})
}

func disableFeatureGate(kvStore cache.Store, featureGate string) {
	// Get the current KubeVirt object from the store
	objects := kvStore.List()
	var currentKV *v1.KubeVirt
	for _, obj := range objects {
		if kv, ok := obj.(*v1.KubeVirt); ok {
			currentKV = kv
			break
		}
	}

	// If no KubeVirt found, create a new one with empty feature gates
	if currentKV == nil {
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, &v1.KubeVirt{
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					DeveloperConfiguration: &v1.DeveloperConfiguration{
						FeatureGates: []string{},
					},
				},
			},
		})
		return
	}

	// Remove the specified feature gate from the list
	featureGates := []string{}
	if currentKV.Spec.Configuration.DeveloperConfiguration != nil {
		for _, fg := range currentKV.Spec.Configuration.DeveloperConfiguration.FeatureGates {
			if fg != featureGate {
				featureGates = append(featureGates, fg)
			}
		}
	}

	// Update with the filtered feature gates
	kvCopy := currentKV.DeepCopy()
	if kvCopy.Spec.Configuration.DeveloperConfiguration == nil {
		kvCopy.Spec.Configuration.DeveloperConfiguration = &v1.DeveloperConfiguration{}
	}
	kvCopy.Spec.Configuration.DeveloperConfiguration.FeatureGates = featureGates
	testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kvCopy)
}
