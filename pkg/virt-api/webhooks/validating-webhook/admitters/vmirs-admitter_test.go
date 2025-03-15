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
 * Copyright 2018 Red Hat, Inc.
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

	"kubevirt.io/kubevirt/pkg/libvmi"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
)

// withDisk adds a disk to the VMI
func withDisk(disk v1.Disk) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, disk)
	}
}

// withVolume adds a volume to the VMI
func withVolume(volume v1.Volume) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, volume)
	}
}

// buildVMITemplate converts a VMI to a VMI template specification
func buildVMITemplate(vmi *v1.VirtualMachineInstance) *v1.VirtualMachineInstanceTemplateSpec {
	return &v1.VirtualMachineInstanceTemplateSpec{
		ObjectMeta: vmi.ObjectMeta,
		Spec:       vmi.Spec,
	}
}

var _ = Describe("Validating VMIRS Admitter", func() {
	config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})
	vmirsAdmitter := &VMIRSAdmitter{ClusterConfig: config}

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
		Entry("VirtualMachineInstanceReplicaSet creation and update",
			`{"very": "unknown", "spec": { "extremely": "unknown" }}`,
			`.very in body is a forbidden property, spec.extremely in body is a forbidden property, spec.selector in body is required, spec.template in body is required`,
			webhooks.VirtualMachineInstanceReplicaSetGroupVersionResource,
			vmirsAdmitter.Admit,
		),
	)
	DescribeTable("reject invalid VirtualMachineInstance spec", func(vmirs *v1.VirtualMachineInstanceReplicaSet, causes []string) {
		vmirsBytes, _ := json.Marshal(&vmirs)

		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Resource: webhooks.VirtualMachineInstanceReplicaSetGroupVersionResource,
				Object: runtime.RawExtension{
					Raw: vmirsBytes,
				},
			},
		}

		resp := vmirsAdmitter.Admit(context.Background(), ar)
		Expect(resp.Allowed).To(BeFalse())
		Expect(resp.Result.Details.Causes).To(HaveLen(len(causes)))
		for i, cause := range causes {
			Expect(resp.Result.Details.Causes[i].Field).To(Equal(cause))
		}
	},
		Entry("with missing volume and missing labels", &v1.VirtualMachineInstanceReplicaSet{
			Spec: v1.VirtualMachineInstanceReplicaSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"match": "this"},
				},
				Template: buildVMITemplate(
					libvmi.New(
						libvmi.WithName("testvmi"),
						withDisk(v1.Disk{
							Name: "testdisk",
						}),
					),
				),
			},
		}, []string{
			"spec.template.spec.domain.devices.disks[0].name",
			"spec.selector",
		}),
		Entry("with mismatching label selectors", &v1.VirtualMachineInstanceReplicaSet{
			Spec: v1.VirtualMachineInstanceReplicaSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"match": "not"},
				},
				Template: buildVMITemplate(
					libvmi.New(
						libvmi.WithName("testvmi"),
						libvmi.WithLabel("match", "this"),
					),
				),
			},
		}, []string{
			"spec.selector",
		}),
	)
	It("should accept valid vmi spec", func() {
		vmirs := &v1.VirtualMachineInstanceReplicaSet{
			Spec: v1.VirtualMachineInstanceReplicaSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"match": "me"},
				},
				Template: buildVMITemplate(
					libvmi.New(
						libvmi.WithName("testvmi"),
						withDisk(v1.Disk{
							Name: "testdisk",
						}),
						withVolume(v1.Volume{
							Name: "testdisk",
							VolumeSource: v1.VolumeSource{
								ContainerDisk: testutils.NewFakeContainerDiskSource(),
							},
						}),
						libvmi.WithLabel("match", "me"),
					),
				),
			},
		}
		vmirsBytes, _ := json.Marshal(&vmirs)

		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Resource: webhooks.VirtualMachineInstanceReplicaSetGroupVersionResource,
				Object: runtime.RawExtension{
					Raw: vmirsBytes,
				},
			},
		}

		resp := vmirsAdmitter.Admit(context.Background(), ar)
		Expect(resp.Allowed).To(BeTrue())
	})
})