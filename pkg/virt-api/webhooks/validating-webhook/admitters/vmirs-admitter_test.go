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

	"kubevirt.io/client-go/api"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
)

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
				Template: newVirtualMachineBuilder().WithDisk(v1.Disk{
					Name: "testdisk",
				}).BuildTemplate(),
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
				Template: newVirtualMachineBuilder().WithLabel("match", "this").BuildTemplate(),
			},
		}, []string{
			"spec.selector",
		}),
	)
	// The hugepage validator reads the memory sources in turn and used to dereference
	// an unset guest on the way to the limit. Both operations reach it through Admit.
	DescribeTable("validates hugepages in the template with no guest memory set", func(operation admissionv1.Operation, pageSize string, expectedField string) {
		template := newVirtualMachineBuilder().
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
			BuildTemplate()
		template.Spec.Domain.Memory = &v1.Memory{Hugepages: &v1.Hugepages{PageSize: pageSize}}
		template.Spec.Domain.Resources.Requests = nil

		vmirs := &v1.VirtualMachineInstanceReplicaSet{
			Spec: v1.VirtualMachineInstanceReplicaSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"match": "me"},
				},
				Template: template,
			},
		}
		vmirsBytes, _ := json.Marshal(&vmirs)

		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Operation: operation,
				Resource:  webhooks.VirtualMachineInstanceReplicaSetGroupVersionResource,
				Object: runtime.RawExtension{
					Raw: vmirsBytes,
				},
			},
		}

		resp := vmirsAdmitter.Admit(context.Background(), ar)

		if expectedField == "" {
			Expect(resp.Allowed).To(BeTrue())
			return
		}
		Expect(resp.Allowed).To(BeFalse())
		Expect(resp.Result.Details.Causes).To(HaveLen(1))
		Expect(resp.Result.Details.Causes[0].Field).To(Equal(expectedField))
	},
		Entry("accepting a valid page size on create", admissionv1.Create, "2Mi", ""),
		Entry("accepting a valid page size on update", admissionv1.Update, "2Mi", ""),
		Entry("rejecting a zero page size on create", admissionv1.Create, "0", "spec.template.spec.domain.memory.hugepages.pageSize"),
		Entry("rejecting a zero page size on update", admissionv1.Update, "0", "spec.template.spec.domain.memory.hugepages.pageSize"),
	)

	It("should accept valid vmi spec", func() {
		vmirs := &v1.VirtualMachineInstanceReplicaSet{
			Spec: v1.VirtualMachineInstanceReplicaSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"match": "me"},
				},
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

	It("should reject a template with more than one unset-enabled ramFB display", func() {
		template := newVirtualMachineBuilder().
			WithDisk(v1.Disk{Name: "testdisk"}).
			WithVolume(v1.Volume{
				Name: "testdisk",
				VolumeSource: v1.VolumeSource{
					ContainerDisk: testutils.NewFakeContainerDiskSource(),
				},
			}).
			WithLabel("match", "me").
			BuildTemplate()
		template.Spec.Domain.Devices.GPUs = []v1.GPU{
			{Name: "gpu0", DeviceName: "vendor.com/gpu0", VirtualGPUOptions: &v1.VGPUOptions{Display: &v1.VGPUDisplayOptions{Enabled: pointer.P(true), RamFB: &v1.FeatureState{}}}},
			{Name: "gpu1", DeviceName: "vendor.com/gpu1", VirtualGPUOptions: &v1.VGPUOptions{Display: &v1.VGPUDisplayOptions{Enabled: pointer.P(true), RamFB: &v1.FeatureState{}}}},
		}
		vmirs := &v1.VirtualMachineInstanceReplicaSet{
			Spec: v1.VirtualMachineInstanceReplicaSetSpec{
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"match": "me"}},
				Template: template,
			},
		}
		vmirsBytes, _ := json.Marshal(&vmirs)

		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Resource: webhooks.VirtualMachineInstanceReplicaSetGroupVersionResource,
				Object:   runtime.RawExtension{Raw: vmirsBytes},
			},
		}

		resp := vmirsAdmitter.Admit(context.Background(), ar)
		Expect(resp.Allowed).To(BeFalse())
		Expect(resp.Result.Details.Causes).To(ContainElement(WithTransform(func(c metav1.StatusCause) string { return c.Field }, Equal("spec.template.spec.GPUs"))))
	})
})

type virtualMachineBuilder struct {
	disks   []v1.Disk
	volumes []v1.Volume
	labels  map[string]string
}

func (b *virtualMachineBuilder) WithDisk(disk v1.Disk) *virtualMachineBuilder {
	b.disks = append(b.disks, disk)
	return b
}

func (b *virtualMachineBuilder) WithLabel(key string, value string) *virtualMachineBuilder {
	b.labels[key] = value
	return b
}

func (b *virtualMachineBuilder) WithVolume(volume v1.Volume) *virtualMachineBuilder {
	b.volumes = append(b.volumes, volume)
	return b
}

func (b *virtualMachineBuilder) Build() *v1.VirtualMachineInstance {

	vmi := api.NewMinimalVMI("testvmi")
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, b.disks...)
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, b.volumes...)
	vmi.Labels = b.labels

	return vmi
}

func (b *virtualMachineBuilder) BuildTemplate() *v1.VirtualMachineInstanceTemplateSpec {
	vmi := b.Build()

	return &v1.VirtualMachineInstanceTemplateSpec{
		ObjectMeta: vmi.ObjectMeta,
		Spec:       vmi.Spec,
	}

}

func newVirtualMachineBuilder() *virtualMachineBuilder {
	return &virtualMachineBuilder{
		labels: map[string]string{},
	}
}
