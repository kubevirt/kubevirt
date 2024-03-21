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

package mutators_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	admissionv1 "k8s.io/api/admission/v1"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks/mutating-webhook/mutators"
)

var _ = Describe("VirtualMachineInstanceMigration Mutator", func() {
	var migration *v1.VirtualMachineInstanceMigration

	getMigrationSpecMetaFromResponse := func() (*v1.VirtualMachineInstanceMigrationSpec, *k8smetav1.ObjectMeta) {
		migrationBytes, err := json.Marshal(migration)
		Expect(err).ToNot(HaveOccurred())
		By("Creating the test admissions review from the Migration object")
		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Resource: k8smetav1.GroupVersionResource{Group: v1.VirtualMachineInstanceMigrationGroupVersionKind.Group, Version: v1.VirtualMachineInstanceMigrationGroupVersionKind.Version, Resource: "virtualmachineinstancemigrations"},
				Object: runtime.RawExtension{
					Raw: migrationBytes,
				},
			},
		}

		By("Mutating the Migration")
		mutator := &mutators.MigrationCreateMutator{}
		resp := mutator.Mutate(ar)
		Expect(resp.Allowed).To(BeTrue())

		By("Getting the VMI spec from the response")
		migrationSpec := &v1.VirtualMachineInstanceMigrationSpec{}
		migrationMeta := &k8smetav1.ObjectMeta{}
		patchOps := []patch.PatchOperation{
			{Value: migrationSpec},
			{Value: migrationMeta},
		}
		err = json.Unmarshal(resp.Patch, &patchOps)
		Expect(err).ToNot(HaveOccurred())
		Expect(patchOps).NotTo(BeEmpty())

		return migrationSpec, migrationMeta
	}

	BeforeEach(func() {
		migration = &v1.VirtualMachineInstanceMigration{
			ObjectMeta: k8smetav1.ObjectMeta{
				Labels: map[string]string{"test": "test"},
			},
			Spec: v1.VirtualMachineInstanceMigrationSpec{
				VMIName: "testVmi",
			},
		}
	})

	It("should verify migration spec", func() {
		migrationSpec, _ := getMigrationSpecMetaFromResponse()
		Expect(migrationSpec.VMIName).To(Equal("testVmi"))
	})

	It("should apply finalizer on migration create", func() {
		_, migrationMeta := getMigrationSpecMetaFromResponse()
		Expect(migrationMeta.Finalizers).To(ContainElement(v1.VirtualMachineInstanceMigrationFinalizer))
	})

	It("should add the selector label", func() {
		_, migrationMeta := getMigrationSpecMetaFromResponse()
		Expect(migrationMeta.Labels).ToNot(BeNil())
		Expect(migrationMeta.Labels[v1.MigrationSelectorLabel]).To(Equal(migration.Spec.VMIName))
	})
})
