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

package admitters_test

import (
	"context"
	"encoding/json"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	admissionv1 "k8s.io/api/admission/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks/validating-webhook/admitters"
)

var _ = Describe("Validating MigrationUpdate Admitter", func() {
	It("should reject Migration on update if spec changes", func() {
		migration := &v1.VirtualMachineInstanceMigration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "somemigrationthatchanged",
				Namespace: "default",
				UID:       "abc",
			},
			Spec: v1.VirtualMachineInstanceMigrationSpec{
				VMIName: "testmigratevmiupdate",
			},
		}

		newMigration := migration.DeepCopy()
		newMigration.Spec.VMIName = "somethingelse"

		ar, err := newAdmissionReviewForVMIMUpdate(migration, newMigration)
		Expect(err).ToNot(HaveOccurred())

		admitter := &admitters.MigrationUpdateAdmitter{}
		resp := admitter.Admit(context.Background(), ar)

		expectedResponse := &admissionv1.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Code:    http.StatusUnprocessableEntity,
				Message: "update of Migration object's spec is restricted",
				Reason:  metav1.StatusReasonInvalid,
				Details: &metav1.StatusDetails{
					Causes: []metav1.StatusCause{
						{
							Type:    metav1.CauseTypeFieldValueNotSupported,
							Message: "update of Migration object's spec is restricted",
						},
					},
				},
			},
		}

		Expect(resp).To(Equal(expectedResponse))
	})

	It("should accept Migration on update if spec doesn't change", func() {
		migration := &v1.VirtualMachineInstanceMigration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "somemigration",
				Namespace: "default",
				UID:       "1234",
			},
			Spec: v1.VirtualMachineInstanceMigrationSpec{
				VMIName: "testmigratevmiupdate-nochange",
			},
		}

		newMigration := migration.DeepCopy()

		ar, err := newAdmissionReviewForVMIMUpdate(migration, newMigration)
		Expect(err).ToNot(HaveOccurred())

		admitter := &admitters.MigrationUpdateAdmitter{}
		resp := admitter.Admit(context.Background(), ar)
		Expect(resp).To(Equal(allowedAdmissionResponse()))
	})

	It("should reject Migration on update if labels include our selector and are removed", func() {
		vmi := libvmi.New(libvmi.WithName("testmigratevmiupdate-labelsremoved"))

		migration := &v1.VirtualMachineInstanceMigration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "somemigration",
				Namespace: "default",
				UID:       "1234",
				Labels: map[string]string{
					v1.MigrationSelectorLabel: vmi.Name,
					"someOtherLabel":          vmi.Name,
				},
			},
			Spec: v1.VirtualMachineInstanceMigrationSpec{
				VMIName: vmi.Name,
			},
		}

		newMigration := migration.DeepCopy()
		newMigration.Labels = nil

		ar, err := newAdmissionReviewForVMIMUpdate(migration, newMigration)
		Expect(err).ToNot(HaveOccurred())

		admitter := &admitters.MigrationUpdateAdmitter{}
		resp := admitter.Admit(context.Background(), ar)

		expectedResponse := &admissionv1.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Code:    http.StatusUnprocessableEntity,
				Message: "selector label can't be removed from an in-flight migration",
				Reason:  metav1.StatusReasonInvalid,
				Details: &metav1.StatusDetails{
					Causes: []metav1.StatusCause{
						{
							Type:    metav1.CauseTypeFieldValueNotSupported,
							Message: "selector label can't be removed from an in-flight migration",
						},
					},
				},
			},
		}

		Expect(resp).To(Equal(expectedResponse))
	})

	It("should reject Migration on update if our selector label is removed", func() {
		vmi := libvmi.New(libvmi.WithName("testmigratevmiupdate-selectorremoved"))

		migration := &v1.VirtualMachineInstanceMigration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "somemigration",
				Namespace: "default",
				UID:       "1234",
				Labels: map[string]string{
					v1.MigrationSelectorLabel: vmi.Name,
					"someOtherLabel":          vmi.Name,
				},
			},
			Spec: v1.VirtualMachineInstanceMigrationSpec{
				VMIName: vmi.Name,
			},
		}

		newMigration := migration.DeepCopy()
		delete(newMigration.Labels, v1.MigrationSelectorLabel)

		ar, err := newAdmissionReviewForVMIMUpdate(migration, newMigration)
		Expect(err).ToNot(HaveOccurred())

		admitter := &admitters.MigrationUpdateAdmitter{}
		resp := admitter.Admit(context.Background(), ar)

		expectedResponse := &admissionv1.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Code:    http.StatusUnprocessableEntity,
				Message: "selector label can't be modified on an in-flight migration",
				Reason:  metav1.StatusReasonInvalid,
				Details: &metav1.StatusDetails{
					Causes: []metav1.StatusCause{
						{
							Type:    metav1.CauseTypeFieldValueNotSupported,
							Message: "selector label can't be modified on an in-flight migration",
						},
					},
				},
			},
		}

		Expect(resp).To(Equal(expectedResponse))
	})

	It("should accept Migration on update if non-selector label is removed", func() {
		vmi := libvmi.New(libvmi.WithName("testmigratevmiupdate-otherremoved"))

		migration := &v1.VirtualMachineInstanceMigration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "somemigration",
				Namespace: "default",
				UID:       "1234",
				Labels: map[string]string{
					v1.MigrationSelectorLabel: vmi.Name,
					"someOtherLabel":          vmi.Name,
				},
			},
			Spec: v1.VirtualMachineInstanceMigrationSpec{
				VMIName: vmi.Name,
			},
		}

		newMigration := migration.DeepCopy()
		delete(newMigration.Labels, "someOtherLabel")

		ar, err := newAdmissionReviewForVMIMUpdate(migration, newMigration)
		Expect(err).ToNot(HaveOccurred())

		admitter := &admitters.MigrationUpdateAdmitter{}
		resp := admitter.Admit(context.Background(), ar)

		Expect(resp).To(Equal(allowedAdmissionResponse()))
	})
})

func newAdmissionReviewForVMIMUpdate(oldMigration, newMigration *v1.VirtualMachineInstanceMigration) (*admissionv1.AdmissionReview, error) {
	oldMigrationBytes, err := json.Marshal(oldMigration)
	if err != nil {
		return nil, err
	}

	newMigrationBytes, err := json.Marshal(newMigration)
	if err != nil {
		return nil, err
	}

	return &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Resource: webhooks.MigrationGroupVersionResource,
			Object: runtime.RawExtension{
				Raw: newMigrationBytes,
			},
			OldObject: runtime.RawExtension{
				Raw: oldMigrationBytes,
			},
			Operation: admissionv1.Update,
		},
	}, nil
}
