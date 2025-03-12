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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package admitters

import (
	"context"
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"

	v1 "kubevirt.io/api/core/v1"
	virtv1 "kubevirt.io/api/core/v1"
	poolv1 "kubevirt.io/api/pool/v1alpha1"

	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
)

var _ = Describe("Validating Pool Admitter", func() {
	config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&virtv1.KubeVirtConfiguration{})

	const kubeVirtNamespace = "kubevirt"
	poolAdmitter := &VMPoolAdmitter{
		ClusterConfig:           config,
		KubeVirtServiceAccounts: webhooks.KubeVirtServiceAccounts(kubeVirtNamespace),
	}

	always := v1.RunStrategyAlways

	DescribeTable("should reject documents containing unknown or missing fields for", func(data string, validationResult string, gvr metav1.GroupVersionResource, review func(ctx context.Context, ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse) {
		input := map[string]interface{}{}
		json.Unmarshal([]byte(data), &input)

		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Resource: gvr,
				Object: runtime.RawExtension{
					Raw: []byte(data),
				},
			},
		}
		resp := review(context.Background(), ar)
		Expect(resp.Allowed).To(BeFalse())
		Expect(resp.Result.Message).To(Equal(validationResult))
	},
		Entry("VirtualMachinePool creation and update",
			`{"very": "unknown", "spec": { "extremely": "unknown" }}`,
			`.very in body is a forbidden property`,
			webhooks.VirtualMachinePoolGroupVersionResource,
			poolAdmitter.Admit,
		),
	)
	DescribeTable("reject invalid VirtualMachineInstance spec", func(pool *poolv1.VirtualMachinePool, causes []string) {
		poolBytes, _ := json.Marshal(&pool)

		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Resource: webhooks.VirtualMachinePoolGroupVersionResource,
				Object: runtime.RawExtension{
					Raw: poolBytes,
				},
			},
		}

		resp := poolAdmitter.Admit(context.Background(), ar)
		Expect(resp.Allowed).To(BeFalse())
		Expect(resp.Result.Details.Causes).To(HaveLen(len(causes)))
		for i, cause := range causes {
			Expect(resp.Result.Details.Causes[i].Field).To(Equal(cause))
		}
	},
		Entry("with missing volume and missing labels", &poolv1.VirtualMachinePool{
			Spec: poolv1.VirtualMachinePoolSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"match": "this"},
				},
				VirtualMachineTemplate: &poolv1.VirtualMachineTemplateSpec{
					Spec: v1.VirtualMachineSpec{
						Template: newVirtualMachineBuilder().WithDisk(v1.Disk{
							Name: "testdisk",
						}).BuildTemplate(),
					},
				},
			},
		}, []string{
			"spec.virtualMachineTemplate.spec.template.spec.domain.devices.disks[0].name",
			"spec.virtualMachineTemplate.spec.running",
			"spec.selector",
		}),
		Entry("with mismatching label selectors", &poolv1.VirtualMachinePool{
			Spec: poolv1.VirtualMachinePoolSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"match": "not"},
				},
				VirtualMachineTemplate: &poolv1.VirtualMachineTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"notmatch": "val"},
					},
					Spec: v1.VirtualMachineSpec{
						Template: newVirtualMachineBuilder().
							WithDisk(v1.Disk{
								Name: "testdisk",
							}).
							WithVolume(v1.Volume{
								Name: "testdisk",
								VolumeSource: v1.VolumeSource{
									ContainerDisk: testutils.NewFakeContainerDiskSource(),
								},
							}).
							BuildTemplate(),
					},
				},
			},
		}, []string{
			"spec.virtualMachineTemplate.spec.running",
			"spec.selector",
		}),
		Entry("with invalid maxUnavailable percentage", &poolv1.VirtualMachinePool{
			Spec: poolv1.VirtualMachinePoolSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"match": "me"},
				},
				VirtualMachineTemplate: &poolv1.VirtualMachineTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"match": "me"},
					},
					Spec: v1.VirtualMachineSpec{
						RunStrategy: &always,
						Template: newVirtualMachineBuilder().
							WithDisk(v1.Disk{
								Name: "testdisk",
							}).
							WithVolume(v1.Volume{
								Name: "testdisk",
								VolumeSource: v1.VolumeSource{
									ContainerDisk: testutils.NewFakeContainerDiskSource(),
								},
							}).
							BuildTemplate(),
					},
				},
				MaxUnavailable: &intstr.IntOrString{
					Type:   intstr.String,
					StrVal: "invalid%",
				},
			},
		}, []string{
			"spec.maxUnavailable",
		}),
		Entry("with invalid maxUnavailable integer", &poolv1.VirtualMachinePool{
			Spec: poolv1.VirtualMachinePoolSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"match": "me"},
				},
				VirtualMachineTemplate: &poolv1.VirtualMachineTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"match": "me"},
					},
					Spec: v1.VirtualMachineSpec{
						RunStrategy: &always,
						Template: newVirtualMachineBuilder().
							WithDisk(v1.Disk{
								Name: "testdisk",
							}).
							WithVolume(v1.Volume{
								Name: "testdisk",
								VolumeSource: v1.VolumeSource{
									ContainerDisk: testutils.NewFakeContainerDiskSource(),
								},
							}).
							BuildTemplate(),
					},
				},
				MaxUnavailable: &intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: -1,
				},
			},
		}, []string{
			"spec.maxUnavailable",
		}),
	)
	It("should accept valid vm spec", func() {
		pool := &poolv1.VirtualMachinePool{
			Spec: poolv1.VirtualMachinePoolSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"match": "me"},
				},

				VirtualMachineTemplate: &poolv1.VirtualMachineTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"match": "me"},
					},
					Spec: v1.VirtualMachineSpec{
						RunStrategy: &always,
						Template: newVirtualMachineBuilder().
							WithDisk(v1.Disk{
								Name: "testdisk",
							}).
							WithVolume(v1.Volume{
								Name: "testdisk",
								VolumeSource: v1.VolumeSource{
									ContainerDisk: testutils.NewFakeContainerDiskSource(),
								},
							}).
							WithLabel("match", "me").
							BuildTemplate(),
					},
				},
			},
		}
		poolBytes, _ := json.Marshal(&pool)

		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Resource: webhooks.VirtualMachinePoolGroupVersionResource,
				Object: runtime.RawExtension{
					Raw: poolBytes,
				},
			},
		}

		resp := poolAdmitter.Admit(context.Background(), ar)
		Expect(resp.Allowed).To(BeTrue())
	})
})
