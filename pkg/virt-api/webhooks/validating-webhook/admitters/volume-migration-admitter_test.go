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
 * Copyright 2024 The KubeVirt Authors
 *
 */

package admitters

import (
	"context"
	"encoding/json"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	v1 "kubevirt.io/api/core/v1"
	virtstorage "kubevirt.io/api/storage/v1alpha1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var _ = Describe("Validating VolumeMigration Admitter", func() {
	config, _, kvInformer := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})
	var volAdmitter *VolumeMigrationAdmitter

	Context("Without feature gate enabled", func() {
		BeforeEach(func() {
			volAdmitter = createTestVolumeMigrationAdmitter(config, nil)
		})
		It("should reject anything", func() {
			volMig := &virtstorage.VolumeMigration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vol-mig",
					Namespace: "default",
				},
				Spec: virtstorage.VolumeMigrationSpec{},
			}

			ar := createVolueMigrationAdmissionReview(volMig)
			resp := volAdmitter.Admit(ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).Should(Equal("VolumeMigration feature gate not enabled"))
		})
	})

	Context("With feature gate enabled", func() {
		const (
			srcName = "src"
			dstName = "dst"
			volName = "disk"
		)
		validVolMig := &virtstorage.VolumeMigration{
			Spec: virtstorage.VolumeMigrationSpec{
				MigratedVolume: []virtstorage.MigratedVolume{{SourceClaim: srcName, DestinationClaim: dstName}},
			},
		}
		validVMI := &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Volumes: []v1.Volume{
					{
						Name: volName,
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
								PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
									ClaimName: srcName,
								},
							},
						},
					},
				},
			},
		}
		enableFeatureGate := func(featureGate string) {
			testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, &v1.KubeVirt{
				Spec: v1.KubeVirtSpec{
					Configuration: v1.KubeVirtConfiguration{
						DeveloperConfiguration: &v1.DeveloperConfiguration{
							FeatureGates: []string{featureGate},
						},
					},
				},
			})
		}
		disableFeatureGates := func() {
			testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, &v1.KubeVirt{
				Spec: v1.KubeVirtSpec{
					Configuration: v1.KubeVirtConfiguration{
						DeveloperConfiguration: &v1.DeveloperConfiguration{
							FeatureGates: make([]string, 0),
						},
					},
				},
			})
		}

		BeforeEach(func() {
			enableFeatureGate(string(virtconfig.VolumeMigration))
		})

		AfterEach(func() {
			disableFeatureGates()
		})

		It("should reject invalid request resource", func() {
			volAdmitter = createTestVolumeMigrationAdmitter(config, nil)
			ar := &admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					Resource: webhooks.VirtualMachineGroupVersionResource,
				},
			}
			resp := volAdmitter.Admit(ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).Should(ContainSubstring("unexpected resource"))
		})

		DescribeTable("validate volume migration", func(volMig *virtstorage.VolumeMigration, vmi *v1.VirtualMachineInstance, err string) {
			volAdmitter = createTestVolumeMigrationAdmitter(config, vmi)
			ar := createVolueMigrationAdmissionReview(volMig)

			resp := volAdmitter.Admit(ar)
			if err != "" {
				Expect(resp.Allowed).To(BeFalse())
				Expect(resp.Result.Message).Should(ContainSubstring(err))
			} else {
				Expect(resp.Allowed).To(BeTrue())
			}
		},
			Entry("with valid request", validVolMig, validVMI, ""),
			Entry("with empty volume migration", &virtstorage.VolumeMigration{}, nil, "Empty migrated volumes"),
			Entry("with empty source", &virtstorage.VolumeMigration{
				Spec: virtstorage.VolumeMigrationSpec{
					MigratedVolume: []virtstorage.MigratedVolume{{SourceClaim: "", DestinationClaim: dstName}},
				},
			}, nil, "has an empty source PVC name"),
			Entry("with empty destinatio", &virtstorage.VolumeMigration{
				Spec: virtstorage.VolumeMigrationSpec{
					MigratedVolume: []virtstorage.MigratedVolume{{SourceClaim: srcName, DestinationClaim: ""}},
				},
			}, nil, "has an empty destination PVC name"),
		)

	})

	It("should reject spec update", func() {
		oldVolMig := &virtstorage.VolumeMigration{
			Spec: virtstorage.VolumeMigrationSpec{
				MigratedVolume: []virtstorage.MigratedVolume{{SourceClaim: "src1", DestinationClaim: "dst1"}},
			},
		}
		volMig := &virtstorage.VolumeMigration{
			Spec: virtstorage.VolumeMigrationSpec{
				MigratedVolume: []virtstorage.MigratedVolume{{SourceClaim: "src2", DestinationClaim: "dst2"}},
			},
		}

		ar := createVolueMigrationUpdateAdmissionReview(oldVolMig, volMig)
		volAdmitter = createTestVolumeMigrationAdmitter(config, nil)
		resp := volAdmitter.Admit(ar)
		Expect(resp.Allowed).To(BeFalse())
		Expect(resp.Result.Details.Causes).To(HaveLen(1))
		Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec"))
	})

})

func createTestVolumeMigrationAdmitter(
	config *virtconfig.ClusterConfig,
	vmi *v1.VirtualMachineInstance,
) *VolumeMigrationAdmitter {
	var vmiList *v1.VirtualMachineInstanceList
	if vmi != nil {
		vmiList = &v1.VirtualMachineInstanceList{Items: []v1.VirtualMachineInstance{*vmi}}
	}
	ctrl := gomock.NewController(GinkgoT())
	virtClient := kubecli.NewMockKubevirtClient(ctrl)
	vmiInterface := kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
	virtClient.EXPECT().VirtualMachineInstance(gomock.Any()).Return(vmiInterface).AnyTimes()
	vmiInterface.EXPECT().List(context.Background(), gomock.Any()).Return(vmiList, nil).AnyTimes()

	return &VolumeMigrationAdmitter{Config: config, Client: virtClient}
}

func createVolueMigrationAdmissionReview(volMig *virtstorage.VolumeMigration) *admissionv1.AdmissionReview {
	bytes, _ := json.Marshal(volMig)

	ar := &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Operation: admissionv1.Create,
			Namespace: "default",
			Resource: metav1.GroupVersionResource{
				Group:    "storage.kubevirt.io",
				Resource: "volumemigrations",
			},
			Object: runtime.RawExtension{
				Raw: bytes,
			},
		},
	}

	return ar
}

func createVolueMigrationUpdateAdmissionReview(old, current *virtstorage.VolumeMigration) *admissionv1.AdmissionReview {
	oldBytes, _ := json.Marshal(old)
	currentBytes, _ := json.Marshal(current)

	ar := &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Operation: admissionv1.Update,
			Namespace: "default",
			Resource: metav1.GroupVersionResource{
				Group:    "storage.kubevirt.io",
				Resource: "volumemigrations",
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
