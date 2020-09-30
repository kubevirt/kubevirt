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
	"k8s.io/api/admission/v1beta1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var _ = Describe("Validating MigrationUpdate Admitter", func() {
	migrationUpdateAdmitter := &MigrationUpdateAdmitter{}
	_, configMapInformer, _, _ := testutils.NewFakeClusterConfig(&k8sv1.ConfigMap{})

	enableFeatureGate := func(featureGate string) {
		testutils.UpdateFakeClusterConfig(configMapInformer, &k8sv1.ConfigMap{
			Data: map[string]string{virtconfig.FeatureGatesKey: featureGate},
		})
	}
	disableFeatureGates := func() {
		testutils.UpdateFakeClusterConfig(configMapInformer, &k8sv1.ConfigMap{})
	}

	AfterEach(func() {
		disableFeatureGates()
	})

	It("should reject Migration on update if spec changes", func() {
		vmi := v1.NewMinimalVMI("testmigratevmiupdate")

		informers := webhooks.GetInformers()
		informers.VMIInformer.GetIndexer().Add(vmi)

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

		ar := &v1beta1.AdmissionReview{
			Request: &v1beta1.AdmissionRequest{
				Resource: webhooks.MigrationGroupVersionResource,
				Object: runtime.RawExtension{
					Raw: newMigrationBytes,
				},
				OldObject: runtime.RawExtension{
					Raw: oldMigrationBytes,
				},
				Operation: v1beta1.Update,
			},
		}

		resp := migrationUpdateAdmitter.Admit(ar)
		Expect(resp.Allowed).To(BeFalse())
	})

	It("should accept Migration on update if spec doesn't change", func() {
		vmi := v1.NewMinimalVMI("testmigratevmiupdate-nochange")

		informers := webhooks.GetInformers()
		informers.VMIInformer.GetIndexer().Add(vmi)

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

		ar := &v1beta1.AdmissionReview{
			Request: &v1beta1.AdmissionRequest{
				Resource: webhooks.MigrationGroupVersionResource,
				Object: runtime.RawExtension{
					Raw: migrationBytes,
				},
				OldObject: runtime.RawExtension{
					Raw: migrationBytes,
				},
				Operation: v1beta1.Update,
			},
		}

		resp := migrationUpdateAdmitter.Admit(ar)
		Expect(resp.Allowed).To(BeTrue())
	})
})
