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
	"context"
	"encoding/json"
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/api"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-config/deprecation"
)

var _ = Describe("Validating MigrationCreate Admitter", func() {
	config, _, kvStore := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})

	// Mock VirtualMachineInstanceMigration
	var ctrl *gomock.Controller
	var virtClient *kubecli.MockKubevirtClient
	var migrationCreateAdmitter *MigrationCreateAdmitter
	var migrationInterface *kubecli.MockVirtualMachineInstanceMigrationInterface
	var mockVMIClient *kubecli.MockVirtualMachineInstanceInterface
	//var nodeSource *framework.FakeControllerSource
	//var nodeInformer cache.SharedIndexInformer

	enableFeatureGate := func(featureGate string) {
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, &v1.KubeVirt{
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
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, &v1.KubeVirt{
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					DeveloperConfiguration: &v1.DeveloperConfiguration{
						FeatureGates: make([]string, 0),
					},
				},
			},
		})
	}

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		migrationInterface = kubecli.NewMockVirtualMachineInstanceMigrationInterface(ctrl)
		mockVMIClient = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		virtClient.EXPECT().VirtualMachineInstanceMigration("default").Return(migrationInterface).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance(gomock.Any()).Return(mockVMIClient).AnyTimes()
		migrationCreateAdmitter = &MigrationCreateAdmitter{ClusterConfig: config, VirtClient: virtClient}
	})

	It("should reject Migration spec on create when another VMI migration is in-flight", func() {
		vmi := api.NewMinimalVMI("testmigratevmi2")
		inFlightMigration := v1.VirtualMachineInstanceMigration{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: vmi.Namespace,
			},
			Spec: v1.VirtualMachineInstanceMigrationSpec{
				VMIName: vmi.Name,
			},
		}
		mockVMIClient.EXPECT().Get(context.Background(), inFlightMigration.Spec.VMIName, gomock.Any()).Return(vmi, nil)
		migrationInterface.EXPECT().List(context.Background(), gomock.Any()).Return(kubecli.NewMigrationList(inFlightMigration), nil).AnyTimes()

		migration := v1.VirtualMachineInstanceMigration{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: vmi.Namespace,
			},
			Spec: v1.VirtualMachineInstanceMigrationSpec{
				VMIName: vmi.Name,
			},
		}
		migrationBytes, _ := json.Marshal(&migration)

		enableFeatureGate(deprecation.LiveMigrationGate)

		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Resource: webhooks.MigrationGroupVersionResource,
				Object: runtime.RawExtension{
					Raw: migrationBytes,
				},
			},
		}

		resp := migrationCreateAdmitter.Admit(ar)
		Expect(resp.Allowed).To(BeFalse())
	})

	Context("with no conflicting migration", func() {

		BeforeEach(func() {
			migrationInterface.EXPECT().List(context.Background(), gomock.Any()).Return(&v1.VirtualMachineInstanceMigrationList{}, nil).MaxTimes(1)
		})

		AfterEach(func() {
			disableFeatureGates()
		})

		It("should reject invalid Migration spec on create", func() {
			migration := v1.VirtualMachineInstanceMigration{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
				},
				Spec: v1.VirtualMachineInstanceMigrationSpec{
					VMIName: "",
				},
			}
			migrationBytes, _ := json.Marshal(&migration)

			enableFeatureGate(deprecation.LiveMigrationGate)

			ar := &admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					Resource: webhooks.MigrationGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: migrationBytes,
					},
				},
			}

			resp := migrationCreateAdmitter.Admit(ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.vmiName"))
		})

		It("should accept valid Migration spec on create", func() {
			vmi := api.NewMinimalVMI("testvmimigrate1")

			mockVMIClient.EXPECT().Get(context.Background(), vmi.Name, gomock.Any()).Return(vmi, nil)

			migration := v1.VirtualMachineInstanceMigration{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: vmi.Namespace,
				},
				Spec: v1.VirtualMachineInstanceMigrationSpec{
					VMIName: "testvmimigrate1",
				},
			}
			migrationBytes, _ := json.Marshal(&migration)

			enableFeatureGate(deprecation.LiveMigrationGate)

			ar := &admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					Resource: webhooks.MigrationGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: migrationBytes,
					},
				},
			}

			resp := migrationCreateAdmitter.Admit(ar)
			Expect(resp.Allowed).To(BeTrue())
		})

		It("should accept Migration spec on create when previous VMI migration completed", func() {
			vmi := api.NewMinimalVMI("testmigratevmi4")
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID: "123",
				Completed:    true,
				Failed:       false,
			}

			mockVMIClient.EXPECT().Get(context.Background(), vmi.Name, gomock.Any()).Return(vmi, nil)

			migration := v1.VirtualMachineInstanceMigration{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: vmi.Namespace,
				},
				Spec: v1.VirtualMachineInstanceMigrationSpec{
					VMIName: "testmigratevmi4",
				},
			}
			migrationBytes, _ := json.Marshal(&migration)

			enableFeatureGate(deprecation.LiveMigrationGate)

			ar := &admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					Resource: webhooks.MigrationGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: migrationBytes,
					},
				},
			}

			resp := migrationCreateAdmitter.Admit(ar)
			Expect(resp.Allowed).To(BeTrue())
		})

		It("should reject Migration spec on create when VMI is finalized", func() {
			vmi := api.NewMinimalVMI("testmigratevmi3")
			vmi.Status.Phase = v1.Succeeded

			mockVMIClient.EXPECT().Get(context.Background(), vmi.Name, gomock.Any()).Return(vmi, nil)

			migration := v1.VirtualMachineInstanceMigration{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
				},
				Spec: v1.VirtualMachineInstanceMigrationSpec{
					VMIName: "testmigratevmi3",
				},
			}
			migrationBytes, _ := json.Marshal(&migration)

			enableFeatureGate(deprecation.LiveMigrationGate)

			ar := &admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					Resource: webhooks.MigrationGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: migrationBytes,
					},
				},
			}

			resp := migrationCreateAdmitter.Admit(ar)
			Expect(resp.Allowed).To(BeFalse())
		})

		It("should reject Migration spec for non-migratable VMIs", func() {
			vmi := api.NewMinimalVMI("testmigratevmi3")
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

			mockVMIClient.EXPECT().Get(context.Background(), vmi.Name, gomock.Any()).Return(vmi, nil)

			migration := v1.VirtualMachineInstanceMigration{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
				},
				Spec: v1.VirtualMachineInstanceMigrationSpec{
					VMIName: "testmigratevmi3",
				},
			}
			migrationBytes, _ := json.Marshal(&migration)

			enableFeatureGate(deprecation.LiveMigrationGate)

			ar := &admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					Resource: webhooks.MigrationGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: migrationBytes,
					},
				},
			}

			resp := migrationCreateAdmitter.Admit(ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).To(ContainSubstring("DisksNotLiveMigratable"))
		})

		It("accept Migration spec on create when destination node exists", func() {
			vmiName := "testmigratevmi6"
			nodeName := "destmigratenode6"
			vmi := api.NewMinimalVMI(vmiName)
			vmi.Status.Phase = v1.Running
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:   v1.VirtualMachineInstanceReady,
					Status: k8sv1.ConditionTrue,
				},
			}

			mockVMIClient.EXPECT().Get(context.Background(), vmi.Name, gomock.Any()).Return(vmi, nil)

			kubeClient := fake.NewSimpleClientset()
			virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()

			kubeClient.Fake.PrependReactor("get", "nodes", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				return true, nil, fmt.Errorf("not found")
			})

			migration := v1.VirtualMachineInstanceMigration{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
				},
				Spec: v1.VirtualMachineInstanceMigrationSpec{
					VMIName:  vmiName,
					NodeName: nodeName,
				},
			}
			migrationBytes, _ := json.Marshal(&migration)

			enableFeatureGate(deprecation.LiveMigrationGate)

			ar := &admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					Resource: webhooks.MigrationGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: migrationBytes,
					},
				},
			}

			resp := migrationCreateAdmitter.Admit(ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).To(ContainSubstring("not found"))
		})

		It("should reject Migration spec on create when the source and destination are the same node", func() {
			vmiName := "testmigratevmi7"
			nodeName := "destmigratenode7"
			vmi := api.NewMinimalVMI(vmiName)
			vmi.Status.Phase = v1.Running
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:   v1.VirtualMachineInstanceReady,
					Status: k8sv1.ConditionTrue,
				},
			}
			vmi.Status.NodeName = nodeName

			mockVMIClient.EXPECT().Get(context.Background(), vmi.Name, gomock.Any()).Return(vmi, nil)

			kubeClient := fake.NewSimpleClientset()
			virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()

			kubeClient.Fake.PrependReactor("get", "nodes", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				return true, &k8sv1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: nodeName,
					},
				}, nil
			})

			migration := v1.VirtualMachineInstanceMigration{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
				},
				Spec: v1.VirtualMachineInstanceMigrationSpec{
					VMIName:  vmiName,
					NodeName: nodeName,
				},
			}
			migrationBytes, _ := json.Marshal(&migration)

			enableFeatureGate(deprecation.LiveMigrationGate)

			ar := &admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					Resource: webhooks.MigrationGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: migrationBytes,
					},
				},
			}

			resp := migrationCreateAdmitter.Admit(ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).To(ContainSubstring("the same source and destination node"))
		})

		It("should accept Migration spec on create when the source and destination are not the same node", func() {
			vmiName := "testmigratevmi7"
			nodeName := "destmigratenode7"
			vmi := api.NewMinimalVMI(vmiName)
			vmi.Status.Phase = v1.Running
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:   v1.VirtualMachineInstanceReady,
					Status: k8sv1.ConditionTrue,
				},
			}
			vmi.Status.NodeName = "sourcemigratenode"

			mockVMIClient.EXPECT().Get(context.Background(), vmi.Name, gomock.Any()).Return(vmi, nil)

			kubeClient := fake.NewSimpleClientset()
			virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()

			kubeClient.Fake.PrependReactor("get", "nodes", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				return true, &k8sv1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: nodeName,
					},
				}, nil
			})

			migration := v1.VirtualMachineInstanceMigration{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
				},
				Spec: v1.VirtualMachineInstanceMigrationSpec{
					VMIName:  vmiName,
					NodeName: nodeName,
				},
			}
			migrationBytes, _ := json.Marshal(&migration)

			enableFeatureGate(deprecation.LiveMigrationGate)

			ar := &admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					Resource: webhooks.MigrationGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: migrationBytes,
					},
				},
			}

			resp := migrationCreateAdmitter.Admit(ar)
			Expect(resp.Allowed).To(BeTrue())
		})

		DescribeTable("should reject documents containing unknown or missing fields for", func(data string, validationResult string, gvr metav1.GroupVersionResource, review func(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse) {
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
			resp := review(ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).To(Equal(validationResult))
		},
			Entry("Migration creation ",
				`{"very": "unknown", "spec": { "extremely": "unknown" }}`,
				`.very in body is a forbidden property, spec.extremely in body is a forbidden property`,
				webhooks.MigrationGroupVersionResource,
				migrationCreateAdmitter.Admit,
			),
			Entry("Migration update",
				`{"very": "unknown", "spec": { "extremely": "unknown" }}`,
				`.very in body is a forbidden property, spec.extremely in body is a forbidden property`,
				webhooks.MigrationGroupVersionResource,
				migrationCreateAdmitter.Admit,
			),
		)
	})
})
