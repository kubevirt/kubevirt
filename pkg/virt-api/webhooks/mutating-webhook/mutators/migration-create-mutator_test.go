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
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks/mutating-webhook/mutators"
)

var _ = Describe("VirtualMachineInstanceMigration Mutator", func() {
	It("Should mutate the VirtualMachineInstanceMigration object", func() {
		migration := newMigration()

		admissionReview, err := newAdmissionReviewForVMIMCreation(migration)
		Expect(err).ToNot(HaveOccurred())

		mutator := &mutators.MigrationCreateMutator{}
		expectedObjectMeta := expectedMigrationObjectMeta(migration.ObjectMeta, migration.Spec.VMIName)
		expectedJSONPatch, err := patch.New(patch.WithReplace("/metadata", expectedObjectMeta)).GeneratePayload()
		Expect(err).NotTo(HaveOccurred())

		Expect(mutator.Mutate(admissionReview)).To(Equal(
			&admissionv1.AdmissionResponse{
				Allowed:   true,
				PatchType: pointer.P(admissionv1.PatchTypeJSONPatch),
				Patch:     expectedJSONPatch,
			},
		))
	})
})

func newMigration() *v1.VirtualMachineInstanceMigration {
	return &v1.VirtualMachineInstanceMigration{
		ObjectMeta: k8smetav1.ObjectMeta{
			Labels: map[string]string{"test": "test"},
		},
		Spec: v1.VirtualMachineInstanceMigrationSpec{
			VMIName: "testVmi",
		},
	}
}

func newAdmissionReviewForVMIMCreation(migration *v1.VirtualMachineInstanceMigration) (*admissionv1.AdmissionReview, error) {
	migrationBytes, err := json.Marshal(migration)
	if err != nil {
		return nil, err
	}

	return &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Resource: k8smetav1.GroupVersionResource{
				Group:    v1.VirtualMachineInstanceMigrationGroupVersionKind.Group,
				Version:  v1.VirtualMachineInstanceMigrationGroupVersionKind.Version,
				Resource: "virtualmachineinstancemigrations",
			},
			Object: runtime.RawExtension{
				Raw: migrationBytes,
			},
		},
	}, nil
}

func expectedMigrationObjectMeta(currentObjectMeta k8smetav1.ObjectMeta, vmiName string) k8smetav1.ObjectMeta {
	expectedObjectMeta := currentObjectMeta

	expectedObjectMeta.Labels[v1.MigrationSelectorLabel] = vmiName
	expectedObjectMeta.Finalizers = []string{v1.VirtualMachineInstanceMigrationFinalizer}

	return expectedObjectMeta
}
