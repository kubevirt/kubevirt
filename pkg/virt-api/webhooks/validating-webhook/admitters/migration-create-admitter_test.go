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

package admitters_test

import (
	"context"
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	v1 "kubevirt.io/api/core/v1"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks/validating-webhook/admitters"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

var _ = Describe("Validating MigrationCreate Admitter", func() {
	const (
		vmiRefName        = "vmiName"
		testMigrationName = "testMigration"
	)

	kv := &v1.KubeVirt{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubevirt",
			Namespace: "kubevirt",
		},
		Spec: v1.KubeVirtSpec{
			Configuration: v1.KubeVirtConfiguration{
				DeveloperConfiguration: &v1.DeveloperConfiguration{},
			},
		},
		Status: v1.KubeVirtStatus{
			Phase:               v1.KubeVirtPhaseDeploying,
			DefaultArchitecture: "amd64",
		},
	}
	config, _, kvStore := testutils.NewFakeClusterConfigUsingKV(kv)

	enableFeatureGate := func(featureGate string) {
		kvConfig := kv.DeepCopy()
		kvConfig.Spec.Configuration.DeveloperConfiguration.FeatureGates = []string{featureGate}
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kvConfig)
	}

	disableFeatureGates := func() {
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kv)
	}

	AfterEach(func() {
		disableFeatureGates()
	})

	It("should reject Migration spec on create when another VMI migration is in-flight", func() {
		vmi := libvmi.New(libvmi.WithNamespace(k8sv1.NamespaceDefault))
		inFlightMigration := &v1.VirtualMachineInstanceMigration{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: vmi.Namespace,
				Labels:    map[string]string{v1.MigrationSelectorLabel: vmi.Name},
			},
			Spec: v1.VirtualMachineInstanceMigrationSpec{
				VMIName: vmi.Name,
			},
		}

		migration := createMigration(vmi.Namespace, testMigrationName, vmi.Name)
		virtClient := kubevirtfake.NewSimpleClientset(vmi, inFlightMigration)
		migrationCreateAdmitter := admitters.NewMigrationCreateAdmitter(virtClient, config)
		ar, err := newAdmissionReviewForVMIMCreation(migration)
		Expect(err).ToNot(HaveOccurred())

		resp := migrationCreateAdmitter.Admit(context.Background(), ar)
		Expect(resp.Allowed).To(BeFalse())
	})

	Context("with no conflicting migration", func() {
		It("should reject invalid Migration spec on create", func() {
			migration := createMigration("default", testMigrationName, "")

			virtClient := kubevirtfake.NewSimpleClientset()
			migrationCreateAdmitter := admitters.NewMigrationCreateAdmitter(virtClient, config)
			ar, err := newAdmissionReviewForVMIMCreation(migration)
			Expect(err).ToNot(HaveOccurred())

			resp := migrationCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.vmiName"))
		})

		It("should accept valid Migration spec on create", func() {
			vmi := libvmi.New(libvmi.WithNamespace(k8sv1.NamespaceDefault))

			migration := createMigration(vmi.Namespace, testMigrationName, vmi.Name)
			virtClient := kubevirtfake.NewSimpleClientset(vmi)
			migrationCreateAdmitter := admitters.NewMigrationCreateAdmitter(virtClient, config)
			ar, err := newAdmissionReviewForVMIMCreation(migration)
			Expect(err).ToNot(HaveOccurred())

			resp := migrationCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeTrue())
		})

		It("should accept Migration spec on create when previous VMI migration completed", func() {
			vmi := libvmi.New(libvmi.WithNamespace(k8sv1.NamespaceDefault))
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID: "123",
				Completed:    true,
				Failed:       false,
			}

			migration := createMigration(vmi.Namespace, testMigrationName, vmi.Name)
			virtClient := kubevirtfake.NewSimpleClientset(vmi)
			migrationCreateAdmitter := admitters.NewMigrationCreateAdmitter(virtClient, config)
			ar, err := newAdmissionReviewForVMIMCreation(migration)
			Expect(err).ToNot(HaveOccurred())

			resp := migrationCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeTrue())
		})

		It("should reject Migration spec on create when VMI is finalized", func() {
			vmi := libvmi.New(libvmi.WithNamespace(k8sv1.NamespaceDefault))
			vmi.Status.Phase = v1.Succeeded

			migration := createMigration(vmi.Namespace, testMigrationName, vmi.Name)
			virtClient := kubevirtfake.NewSimpleClientset(vmi)
			migrationCreateAdmitter := admitters.NewMigrationCreateAdmitter(virtClient, config)
			ar, err := newAdmissionReviewForVMIMCreation(migration)
			Expect(err).ToNot(HaveOccurred())

			resp := migrationCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
		})

		It("should reject Migration spec for non-migratable VMIs", func() {
			vmi := libvmi.New(libvmi.WithNamespace(k8sv1.NamespaceDefault))
			vmi.Status.Phase = v1.Running
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:    v1.VirtualMachineInstanceIsMigratable,
					Status:  k8sv1.ConditionFalse,
					Reason:  v1.VirtualMachineInstanceReasonDisksNotMigratable,
					Message: "cannot migrate VMI with mixes shared and non-shared volumes",
				},
				{
					Type:   v1.VirtualMachineInstanceReady,
					Status: k8sv1.ConditionTrue,
				},
			}

			migration := createMigration(vmi.Namespace, testMigrationName, vmi.Name)
			virtClient := kubevirtfake.NewSimpleClientset(vmi)
			migrationCreateAdmitter := admitters.NewMigrationCreateAdmitter(virtClient, config)

			ar, err := newAdmissionReviewForVMIMCreation(migration)
			Expect(err).ToNot(HaveOccurred())

			resp := migrationCreateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).To(ContainSubstring("DisksNotLiveMigratable"))
		})

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
			Entry("Migration creation ",
				`{"very": "unknown", "spec": { "extremely": "unknown" }}`,
				`.very in body is a forbidden property, spec.extremely in body is a forbidden property`,
				webhooks.MigrationGroupVersionResource,
				admitters.NewMigrationCreateAdmitter(kubevirtfake.NewSimpleClientset(), config).Admit,
			),
			Entry("Migration update",
				`{"very": "unknown", "spec": { "extremely": "unknown" }}`,
				`.very in body is a forbidden property, spec.extremely in body is a forbidden property`,
				webhooks.MigrationGroupVersionResource,
				admitters.NewMigrationCreateAdmitter(kubevirtfake.NewSimpleClientset(), config).Admit,
			),
		)
	})

	Context("feature gate", func() {
		It("should reject vmim with sendTo if featuregate is disabled", func() {
			vmi := libvmi.New(libvmi.WithNamespace(k8sv1.NamespaceDefault))
			vmi.Status.Phase = v1.Running
			virtClient := kubevirtfake.NewSimpleClientset(vmi)
			migration := createMigration(vmi.Namespace, testMigrationName, vmi.Name)
			migration.Spec.SendTo = &v1.VirtualMachineInstanceMigrationSource{
				MigrationID: "migrationID",
				ConnectURL:  "1.1.1.1:12345",
			}
			ar, err := newAdmissionReviewForVMIMCreation(migration)
			Expect(err).ToNot(HaveOccurred())
			admitter := admitters.NewMigrationCreateAdmitter(virtClient, config)
			resp := admitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).To(ContainSubstring("DecentralizedLiveMigration feature gate is not enabled in kubevirt resource"))
		})

		It("should reject vmim with receive if featuregate is disabled", func() {
			vmi := libvmi.New(libvmi.WithNamespace(k8sv1.NamespaceDefault))
			vmi.Status.Phase = v1.Running
			virtClient := kubevirtfake.NewSimpleClientset(vmi)
			migration := createMigration(vmi.Namespace, testMigrationName, vmi.Name)
			migration.Spec.Receive = &v1.VirtualMachineInstanceMigrationTarget{
				MigrationID: "migrationID",
			}
			ar, err := newAdmissionReviewForVMIMCreation(migration)
			Expect(err).ToNot(HaveOccurred())
			admitter := admitters.NewMigrationCreateAdmitter(virtClient, config)
			resp := admitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).To(ContainSubstring("DecentralizedLiveMigration feature gate is not enabled in kubevirt resource"))
		})

		It("should allow vmim with sendTo if featuregate is enabled", func() {
			vmi := libvmi.New(libvmi.WithNamespace(k8sv1.NamespaceDefault))
			vmi.Status.Phase = v1.Running
			virtClient := kubevirtfake.NewSimpleClientset(vmi)
			migration := createMigration(vmi.Namespace, testMigrationName, vmi.Name)
			migration.Spec.SendTo = &v1.VirtualMachineInstanceMigrationSource{
				MigrationID: "migrationID",
				ConnectURL:  "1.1.1.1:12345",
			}
			ar, err := newAdmissionReviewForVMIMCreation(migration)
			Expect(err).ToNot(HaveOccurred())
			enableFeatureGate(featuregate.DecentralizedLiveMigration)
			admitter := admitters.NewMigrationCreateAdmitter(virtClient, config)
			resp := admitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeTrue())
		})

		It("should allow vmim with receive if featuregate is enabled", func() {
			vmi := libvmi.New(libvmi.WithNamespace(k8sv1.NamespaceDefault))
			vmi.Status.Phase = v1.Running
			virtClient := kubevirtfake.NewSimpleClientset(vmi)
			migration := createMigration(vmi.Namespace, testMigrationName, vmi.Name)
			migration.Spec.Receive = &v1.VirtualMachineInstanceMigrationTarget{
				MigrationID: "migrationID",
			}
			ar, err := newAdmissionReviewForVMIMCreation(migration)
			Expect(err).ToNot(HaveOccurred())
			enableFeatureGate(featuregate.DecentralizedLiveMigration)
			admitter := admitters.NewMigrationCreateAdmitter(virtClient, config)
			resp := admitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeTrue())
		})

		It("should allow vmim with sendTo and receiver if featuregate is enabled", func() {
			vmi := libvmi.New(libvmi.WithNamespace(k8sv1.NamespaceDefault))
			vmi.Status.Phase = v1.Running
			virtClient := kubevirtfake.NewSimpleClientset(vmi)
			migration := createMigration(vmi.Namespace, testMigrationName, vmi.Name)
			migration.Spec.SendTo = &v1.VirtualMachineInstanceMigrationSource{
				MigrationID: "migrationID",
				ConnectURL:  "1.1.1.1:12345",
			}
			migration.Spec.Receive = &v1.VirtualMachineInstanceMigrationTarget{
				MigrationID: "migrationID",
			}
			ar, err := newAdmissionReviewForVMIMCreation(migration)
			Expect(err).ToNot(HaveOccurred())
			enableFeatureGate(featuregate.DecentralizedLiveMigration)
			admitter := admitters.NewMigrationCreateAdmitter(virtClient, config)
			resp := admitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeTrue())
		})
	})
})

func newAdmissionReviewForVMIMCreation(migration *v1.VirtualMachineInstanceMigration) (*admissionv1.AdmissionReview, error) {
	migrationBytes, err := json.Marshal(migration)
	if err != nil {
		return nil, err
	}

	return &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Resource: webhooks.MigrationGroupVersionResource,
			Object: runtime.RawExtension{
				Raw: migrationBytes,
			},
			Operation: admissionv1.Create,
		},
	}, nil
}

func createMigration(namespace, name, vmiRef string) *v1.VirtualMachineInstanceMigration {
	return &v1.VirtualMachineInstanceMigration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.VirtualMachineInstanceMigrationSpec{
			VMIName: vmiRef,
		},
	}
}
