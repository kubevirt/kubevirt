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
 * Copyright 2019 Red Hat, Inc.
 */

package mutators

import (
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	v1 "kubevirt.io/api/core/v1"
	virtv1 "kubevirt.io/api/core/v1"
	poolv1 "kubevirt.io/api/pool/v1alpha1"
	utiltypes "kubevirt.io/kubevirt/pkg/util/types"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
)

var _ = Describe("VirtualMachinePool Mutator", func() {
	var pool *poolv1.VirtualMachinePool
	var oldPool *poolv1.VirtualMachinePool
	var mutator *VMPoolsMutator

	getPoolMetaFromResponse := func(op admissionv1.Operation, expectNoPatch bool) *k8smetav1.ObjectMeta {
		poolBytes, err := json.Marshal(pool)
		Expect(err).ToNot(HaveOccurred())
		By("Creating the test admissions review from the Pool")
		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Resource: k8smetav1.GroupVersionResource{Group: webhooks.VirtualMachinePoolGroupVersionResource.Group, Version: webhooks.VirtualMachinePoolGroupVersionResource.Version, Resource: webhooks.VirtualMachinePoolGroupVersionResource.Resource},
				Object: runtime.RawExtension{
					Raw: poolBytes,
				},
				Operation: op,
			},
		}

		if oldPool != nil {
			oldPoolBytes, err := json.Marshal(oldPool)
			Expect(err).ToNot(HaveOccurred())
			ar.Request.OldObject.Raw = oldPoolBytes
		}

		By("Mutating the VMPool")
		resp := mutator.Mutate(ar)
		Expect(resp.Allowed).To(BeTrue())

		if expectNoPatch {
			Expect(len(resp.Patch)).To(Equal(0))
			return nil
		}

		By("Getting the VMPool from the response")
		poolMeta := &k8smetav1.ObjectMeta{}
		patch := []utiltypes.PatchOperation{
			{Value: poolMeta},
		}
		err = json.Unmarshal(resp.Patch, &patch)
		Expect(err).ToNot(HaveOccurred())
		Expect(patch).NotTo(BeEmpty())

		return poolMeta
	}

	BeforeEach(func() {
		oldPool = nil
		pool = &poolv1.VirtualMachinePool{
			ObjectMeta: k8smetav1.ObjectMeta{
				Labels: map[string]string{"test": "test"},
			},
		}
		pool.Spec.VirtualMachineTemplate = &poolv1.VirtualMachineTemplateSpec{}
		pool.Spec.VirtualMachineTemplate.Spec.Template = &v1.VirtualMachineInstanceTemplateSpec{}

		mutator = &VMPoolsMutator{}
	})

	It("should inject hash labels create", func() {
		poolMeta := getPoolMetaFromResponse(admissionv1.Create, false)

		vmHash, exists := poolMeta.Labels[virtv1.VirtualMachineTemplateHash]
		Expect(exists).To(BeTrue())
		Expect(vmHash).ToNot(Equal(""))

		vmiHash, exists := poolMeta.Labels[virtv1.VirtualMachineInstanceTemplateHash]
		Expect(exists).To(BeTrue())
		Expect(vmiHash).ToNot(Equal(""))
	})

	It("should update hash labels if they get removed", func() {
		oldPool = pool.DeepCopy()
		poolMeta := getPoolMetaFromResponse(admissionv1.Update, false)

		vmHash, exists := poolMeta.Labels[virtv1.VirtualMachineTemplateHash]
		Expect(exists).To(BeTrue())
		Expect(vmHash).ToNot(Equal(""))

		vmiHash, exists := poolMeta.Labels[virtv1.VirtualMachineInstanceTemplateHash]
		Expect(exists).To(BeTrue())
		Expect(vmiHash).ToNot(Equal(""))
	})

	It("should do nothing if hash exists and old and new match", func() {

		pool.Labels[virtv1.VirtualMachineInstanceTemplateHash] = "some-vmi-hash"
		pool.Labels[virtv1.VirtualMachineTemplateHash] = "some-vm-hash"

		oldPool = pool.DeepCopy()
		poolMeta := getPoolMetaFromResponse(admissionv1.Update, true)
		Expect(poolMeta).To(BeNil())

	})

	It("should update hash labels when vm template changes on update", func() {
		oldPool = pool.DeepCopy()
		pool.Spec.VirtualMachineTemplate.Spec.Template.ObjectMeta.Labels = map[string]string{"somelabel": "someval"}

		poolMeta := getPoolMetaFromResponse(admissionv1.Update, false)

		vmHash, exists := poolMeta.Labels[virtv1.VirtualMachineTemplateHash]
		Expect(exists).To(BeTrue())
		Expect(vmHash).ToNot(Equal(""))

		vmiHash, exists := poolMeta.Labels[virtv1.VirtualMachineInstanceTemplateHash]
		Expect(exists).To(BeTrue())
		Expect(vmiHash).ToNot(Equal(""))
	})

	It("should only update hash labels vm if vm template changes but vmi template does not", func() {
		oldPool = pool.DeepCopy()
		pool.Labels[virtv1.VirtualMachineInstanceTemplateHash] = "some-vmi-hash"
		pool.Labels[virtv1.VirtualMachineTemplateHash] = "some-vm-hash"
		pool.Spec.VirtualMachineTemplate.ObjectMeta.Labels = map[string]string{"somelabel": "someval"}

		poolMeta := getPoolMetaFromResponse(admissionv1.Update, false)

		vmHash, exists := poolMeta.Labels[virtv1.VirtualMachineTemplateHash]
		Expect(exists).To(BeTrue())
		Expect(vmHash).ToNot(Equal(""))
		Expect(vmHash).ToNot(Equal("some-vm-hash"))

		vmiHash, exists := poolMeta.Labels[virtv1.VirtualMachineInstanceTemplateHash]
		Expect(exists).To(BeTrue())
		Expect(vmiHash).To(Equal("some-vmi-hash"))
	})

	It("should update both hash labels both vm and vmi template change", func() {
		oldPool = pool.DeepCopy()
		pool.Labels[virtv1.VirtualMachineInstanceTemplateHash] = "some-vmi-hash"
		pool.Labels[virtv1.VirtualMachineTemplateHash] = "some-vm-hash"
		pool.Spec.VirtualMachineTemplate.Spec.Template.ObjectMeta.Labels = map[string]string{"somelabel": "someval"}

		poolMeta := getPoolMetaFromResponse(admissionv1.Update, false)

		vmHash, exists := poolMeta.Labels[virtv1.VirtualMachineTemplateHash]
		Expect(exists).To(BeTrue())
		Expect(vmHash).ToNot(Equal(""))
		Expect(vmHash).ToNot(Equal("some-vm-hash"))

		vmiHash, exists := poolMeta.Labels[virtv1.VirtualMachineInstanceTemplateHash]
		Expect(exists).To(BeTrue())
		Expect(vmiHash).ToNot(Equal("some-vmi-hash"))
	})
})
