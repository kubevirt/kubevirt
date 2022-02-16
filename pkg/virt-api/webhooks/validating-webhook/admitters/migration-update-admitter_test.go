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
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"kubevirt.io/client-go/api"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var _ = Describe("Validating MigrationUpdate Admitter", func() {
	migrationUpdateAdmitter := &MigrationUpdateAdmitter{}
	_, _, kvInformer := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})

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

	AfterEach(func() {
		disableFeatureGates()
	})

	It("should reject Migration on update if spec changes", func() {
		migration := v1.VirtualMachineInstanceMigration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "somemigrationthatchanged",
				Namespace: "default",
				UID:       "abc",
			},
			Spec: v1.VirtualMachineInstanceMigrationSpec{
				VMIName: "testmigratevmiupdate",
			},
		}
		oldMigrationBytes, _ := json.Marshal(&migration)

		newMigration := migration.DeepCopy()
		newMigration.Spec.VMIName = "somethingelse"
		newMigrationBytes, _ := json.Marshal(&newMigration)

		enableFeatureGate(virtconfig.LiveMigrationGate)

		ar := &admissionv1.AdmissionReview{
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
		}

		resp := migrationUpdateAdmitter.Admit(ar)
		Expect(resp.Allowed).To(BeFalse())
	})

	It("should accept Migration on update if spec doesn't change", func() {
		migration := v1.VirtualMachineInstanceMigration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "somemigration",
				Namespace: "default",
				UID:       "1234",
			},
			Spec: v1.VirtualMachineInstanceMigrationSpec{
				VMIName: "testmigratevmiupdate-nochange",
			},
		}

		migrationBytes, _ := json.Marshal(&migration)

		enableFeatureGate(virtconfig.LiveMigrationGate)

		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Resource: webhooks.MigrationGroupVersionResource,
				Object: runtime.RawExtension{
					Raw: migrationBytes,
				},
				OldObject: runtime.RawExtension{
					Raw: migrationBytes,
				},
				Operation: admissionv1.Update,
			},
		}

		resp := migrationUpdateAdmitter.Admit(ar)
		Expect(resp.Allowed).To(BeTrue())
	})

	It("should reject Migration on update if labels include our selector and are removed", func() {
		vmi := api.NewMinimalVMI("testmigratevmiupdate-labelsremoved")

		migration := v1.VirtualMachineInstanceMigration{
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

		oldMigrationBytes, _ := json.Marshal(&migration)

		newMigration := migration.DeepCopy()
		newMigration.Labels = nil
		newMigrationBytes, _ := json.Marshal(&newMigration)

		enableFeatureGate(virtconfig.LiveMigrationGate)

		ar := &admissionv1.AdmissionReview{
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
		}

		resp := migrationUpdateAdmitter.Admit(ar)
		Expect(resp.Allowed).To(BeFalse())
	})

	It("should reject Migration on update if our selector label is removed", func() {
		vmi := api.NewMinimalVMI("testmigratevmiupdate-selectorremoved")

		migration := v1.VirtualMachineInstanceMigration{
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

		oldMigrationBytes, _ := json.Marshal(&migration)

		newMigration := migration.DeepCopy()
		delete(newMigration.Labels, v1.MigrationSelectorLabel)
		newMigrationBytes, _ := json.Marshal(&newMigration)

		enableFeatureGate(virtconfig.LiveMigrationGate)

		ar := &admissionv1.AdmissionReview{
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
		}

		resp := migrationUpdateAdmitter.Admit(ar)
		Expect(resp.Allowed).To(BeFalse())
	})

	It("should accept Migration on update if non-selector label is removed", func() {
		vmi := api.NewMinimalVMI("testmigratevmiupdate-otherremoved")

		migration := v1.VirtualMachineInstanceMigration{
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

		oldMigrationBytes, _ := json.Marshal(&migration)

		newMigration := migration.DeepCopy()
		delete(newMigration.Labels, "someOtherLabel")
		newMigrationBytes, _ := json.Marshal(&newMigration)

		enableFeatureGate(virtconfig.LiveMigrationGate)

		ar := &admissionv1.AdmissionReview{
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
		}

		resp := migrationUpdateAdmitter.Admit(ar)
		Expect(resp.Allowed).To(BeTrue())
	})
})
