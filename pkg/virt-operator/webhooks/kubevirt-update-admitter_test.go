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

package webhooks

import (
	"encoding/json"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
)

var _ = Describe("Validating KubeVirtUpdate Admitter", func() {

	getAdmitter := func(needsVMIMock bool) *KubeVirtUpdateAdmitter {
		ctrl := gomock.NewController(GinkgoT())
		virtClient := kubecli.NewMockKubevirtClient(ctrl)

		if needsVMIMock {
			vmiInterface := kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
			virtClient.EXPECT().VirtualMachineInstance(gomock.Any()).Return(vmiInterface).AnyTimes()
		}

		return NewKubeVirtUpdateAdmitter(virtClient)
	}

	getKV := func() v1.KubeVirt {
		return v1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
			},
			Spec: v1.KubeVirtSpec{
				Workloads: nil,
			},
		}
	}

	getComponentConfig := func() v1.ComponentConfig {
		return v1.ComponentConfig{
			NodePlacement: &v1.NodePlacement{
				NodeSelector: map[string]string{
					"kubernetes.io/hostname": "node01",
				},
			},
		}
	}

	It("should accept workload update when no VMIS are running", func() {
		kvAdmitter := getAdmitter(true)
		kv := getKV()
		kvBytes, _ := json.Marshal(&kv)

		cc := getComponentConfig()
		kv.Spec.Workloads = &cc
		kvUpdateBytes, _ := json.Marshal(&kv)

		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Resource: webhooks.KubeVirtGroupVersionResource,
				Object: runtime.RawExtension{
					Raw: kvUpdateBytes,
				},
				OldObject: runtime.RawExtension{
					Raw: kvBytes,
				},
				Operation: admissionv1.Update,
			},
		}

		resp := kvAdmitter.Admit(ar)
		Expect(resp.Allowed).To(BeTrue())
	})

	It("should accept KV update with VMIS running if workloads object is not changed", func() {
		kvAdmitter := getAdmitter(false)
		kv := getKV()
		cc := getComponentConfig()
		kv.Spec.Workloads = &cc

		kvBytes, _ := json.Marshal(&kv)

		kv.ObjectMeta.Labels = map[string]string{
			"kubevirt.io": "new-label",
		}
		kvUpdateBytes, _ := json.Marshal(&kv)

		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Resource: webhooks.KubeVirtGroupVersionResource,
				Object: runtime.RawExtension{
					Raw: kvUpdateBytes,
				},
				OldObject: runtime.RawExtension{
					Raw: kvBytes,
				},
				Operation: admissionv1.Update,
			},
		}

		resp := kvAdmitter.Admit(ar)
		Expect(resp.Allowed).To(BeTrue())
	})

	table.DescribeTable("test validateCustomizeComponents", func(cc v1.CustomizeComponents, expectedCauses int) {
		causes := validateCustomizeComponents(cc)
		Expect(len(causes)).To(Equal(expectedCauses))
	},
		table.Entry("valid values accepted", v1.CustomizeComponents{
			Patches: []v1.CustomizeComponentsPatch{
				{
					ResourceName: "virt-api",
					ResourceType: "Deployment",
					Type:         v1.StrategicMergePatchType,
					Patch:        `{"json: "not valid"}`,
				},
			},
		}, 1),
		table.Entry("valid values accepted", v1.CustomizeComponents{
			Patches: []v1.CustomizeComponentsPatch{
				{
					ResourceName: "virt-api",
					ResourceType: "Deployment",
					Type:         v1.StrategicMergePatchType,
					Patch:        `{}`,
				},
			},
		}, 0),
	)

	It("should reject with JSON patch validation invalid", func() {
		kvAdmitter := getAdmitter(true)
		kv := getKV()
		kvBytes, _ := json.Marshal(&kv)

		cc := getComponentConfig()
		kv.Spec.Workloads = &cc
		kv.Spec.CustomizeComponents = v1.CustomizeComponents{
			Patches: []v1.CustomizeComponentsPatch{
				{
					ResourceName: "virt-api",
					ResourceType: "Deployment",
					Type:         v1.StrategicMergePatchType,
					Patch:        ``,
				},
			},
		}
		kvUpdateBytes, _ := json.Marshal(&kv)

		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Resource: webhooks.KubeVirtGroupVersionResource,
				Object: runtime.RawExtension{
					Raw: kvUpdateBytes,
				},
				OldObject: runtime.RawExtension{
					Raw: kvBytes,
				},
				Operation: admissionv1.Update,
			},
		}

		resp := kvAdmitter.Admit(ar)
		Expect(resp.Allowed).To(BeFalse())
		Expect(len(resp.Result.Details.Causes)).To(Equal(1))
	})
})
