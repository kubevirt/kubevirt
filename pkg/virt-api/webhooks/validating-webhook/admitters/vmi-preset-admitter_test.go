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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
)

var _ = Describe("Validating VMIPreset Admitter", func() {
	vmiPresetAdmitter := &VMIPresetAdmitter{}

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
		Entry("VirtualMachineInstancePreset creation and update",
			`{"very": "unknown", "spec": { "extremely": "unknown" }}`,
			`.very in body is a forbidden property, spec.extremely in body is a forbidden property, spec.selector in body is required`,
			webhooks.VirtualMachineInstancePresetGroupVersionResource,
			vmiPresetAdmitter.Admit,
		),
	)
	It("reject invalid VirtualMachineInstance spec", func() {
		vmi := libvmi.New(
			libvmi.WithName("testvmi"),
		)
		vmiPDomain := &v1.DomainSpec{}
		vmiDomainByte, _ := json.Marshal(vmi.Spec.Domain)
		Expect(json.Unmarshal(vmiDomainByte, &vmiPDomain)).To(Succeed())

		vmiPDomain.Devices.Disks = append(vmiPDomain.Devices.Disks, v1.Disk{
			Name: "testdisk",
			DiskDevice: v1.DiskDevice{
				Disk:  &v1.DiskTarget{},
				CDRom: &v1.CDRomTarget{},
			},
		})
		vmiPreset := &v1.VirtualMachineInstancePreset{
			Spec: v1.VirtualMachineInstancePresetSpec{
				Domain: vmiPDomain,
			},
		}
		vmiPresetBytes, _ := json.Marshal(vmiPreset)

		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Resource: webhooks.VirtualMachineInstancePresetGroupVersionResource,
				Object: runtime.RawExtension{
					Raw: vmiPresetBytes,
				},
			},
		}

		resp := vmiPresetAdmitter.Admit(context.Background(), ar)
		Expect(resp.Allowed).To(BeFalse())
		Expect(resp.Result.Details.Causes).To(HaveLen(1))
		Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.domain.devices.disks[0]"))
	})
	It("should accept valid vmi spec", func() {
		vmiPreset := &v1.VirtualMachineInstancePreset{
			Spec: v1.VirtualMachineInstancePresetSpec{
				Domain: &v1.DomainSpec{},
			},
		}
		vmiPresetBytes, _ := json.Marshal(&vmiPreset)

		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Resource: webhooks.VirtualMachineInstancePresetGroupVersionResource,
				Object: runtime.RawExtension{
					Raw: vmiPresetBytes,
				},
			},
		}

		resp := vmiPresetAdmitter.Admit(context.Background(), ar)
		Expect(resp.Allowed).To(BeTrue())
	})
})
