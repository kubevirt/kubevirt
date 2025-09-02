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
	"github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
	admissionv1 "k8s.io/api/admission/v1"
	authv1 "k8s.io/api/authentication/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"kubevirt.io/client-go/api"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
)

var _ = Describe("Validating VMIUpdate Admitter", func() {
	const kubeVirtNamespace = "kubevirt"

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
			Phase: v1.KubeVirtPhaseDeploying,
		},
	}
	config, _, kvStore := testutils.NewFakeClusterConfigUsingKV(kv)
	vmiUpdateAdmitter := NewVMIUpdateAdmitter(config, webhooks.KubeVirtServiceAccounts(kubeVirtNamespace))

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

	Context("Node restriction", func() {
		mustMarshal := func(vmi *v1.VirtualMachineInstance) []byte {
			b, err := json.Marshal(vmi)
			Expect(err).To(Not(HaveOccurred()))
			return b
		}

		admissionWithCustomUpdate := func(vmi, updatedVMI *v1.VirtualMachineInstance, handlernode string) *admissionv1.AdmissionReview {
			newVMIBytes := mustMarshal(updatedVMI)
			oldVMIBytes := mustMarshal(vmi)
			return &admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					UserInfo: authv1.UserInfo{
						Username: "system:serviceaccount:kubevirt:kubevirt-handler",
						Extra: map[string]authv1.ExtraValue{
							"authentication.kubernetes.io/node-name": {handlernode},
						},
					},
					Resource: webhooks.VirtualMachineInstanceGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: newVMIBytes,
					},
					OldObject: runtime.RawExtension{
						Raw: oldVMIBytes,
					},
					Operation: admissionv1.Update,
				},
			}
		}

		admission := func(vmi *v1.VirtualMachineInstance, handlernode string) *admissionv1.AdmissionReview {
			updatedVMI := vmi.DeepCopy()
			if updatedVMI.Labels == nil {
				updatedVMI.Labels = map[string]string{}
			}
			updatedVMI.Labels["allowed.io"] = "value"
			return admissionWithCustomUpdate(vmi, updatedVMI, handlernode)
		}

		Context("with Node Restriction feature gate enabled", func() {
			BeforeEach(func() { enableFeatureGate(featuregate.NodeRestrictionGate) })

			shouldNotAllowCrossNodeRequest := And(
				WithTransform(func(resp *admissionv1.AdmissionResponse) bool { return resp.Allowed },
					BeFalse(),
				),
				WithTransform(func(resp *admissionv1.AdmissionResponse) []metav1.StatusCause { return resp.Result.Details.Causes },
					ContainElement(
						gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
							"Message": Equal("Node restriction, virt-handler is only allowed to modify VMIs it owns"),
						}),
					),
				),
			)

			shouldBeAllowed := WithTransform(func(resp *admissionv1.AdmissionResponse) bool { return resp.Allowed },
				BeTrue(),
			)

			DescribeTable("and NodeName set", func(handlernode string, matcher types.GomegaMatcher) {
				vmi := api.NewMinimalVMI("testvmi")
				vmi.Status.NodeName = "got"

				resp := vmiUpdateAdmitter.Admit(context.Background(), admission(vmi, handlernode))
				Expect(resp).To(matcher)
			},
				Entry("should deny request if handler is on different node", "diff",
					shouldNotAllowCrossNodeRequest,
				),
				Entry("should allow request if handler is on same node", "got",
					shouldBeAllowed,
				),
			)

			DescribeTable("and TargetNode set", func(handlernode string, matcher types.GomegaMatcher) {
				vmi := api.NewMinimalVMI("testvmi")
				vmi.Status.NodeName = "got"
				vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
					TargetNode: "git",
				}

				resp := vmiUpdateAdmitter.Admit(context.Background(), admission(vmi, handlernode))
				Expect(resp).To(matcher)
			},
				Entry("should deny request if handler is on different node", "diff",
					shouldNotAllowCrossNodeRequest,
				),
				Entry("should allow request if handler is on same node", "git",
					shouldBeAllowed,
				),
			)

			DescribeTable("and both NodeName and TargetNode set", func(handlernode string, matcher types.GomegaMatcher) {
				vmi := api.NewMinimalVMI("testvmi")
				vmi.Status.NodeName = "got"
				vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
					TargetNode: "target",
				}

				resp := vmiUpdateAdmitter.Admit(context.Background(), admission(vmi, handlernode))
				Expect(resp).To(matcher)
			},
				Entry("should deny request if handler is on different node", "diff",
					shouldNotAllowCrossNodeRequest,
				),
				Entry("should allow request if handler is on source node", "got",
					shouldBeAllowed,
				),

				Entry("should allow request if handler is on target node", "target",
					shouldBeAllowed,
				),
			)

			It("should allow finalize migration", func() {
				vmi := api.NewMinimalVMI("testvmi")
				vmi.Status.NodeName = "got"
				vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
					TargetNode: "target",
				}

				updatedVMI := vmi.DeepCopy()
				updatedVMI.Status.NodeName = "target"

				resp := vmiUpdateAdmitter.Admit(context.Background(), admissionWithCustomUpdate(vmi, updatedVMI, "got"))
				Expect(resp.Allowed).To(BeTrue())
			})

			It("should not allow to set targetNode to source handler", func() {
				vmi := api.NewMinimalVMI("testvmi")
				vmi.Status.NodeName = "got"

				updatedVMI := vmi.DeepCopy()
				updatedVMI.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
					TargetNode: "target",
				}
				resp := vmiUpdateAdmitter.Admit(context.Background(), admissionWithCustomUpdate(vmi, updatedVMI, "got"))
				Expect(resp.Allowed).To(BeFalse())
			})
		})

		DescribeTable("with Node Restriction feature gate disabled should allow different handler", func(migrationState *v1.VirtualMachineInstanceMigrationState) {
			vmi := api.NewMinimalVMI("testvmi")
			vmi.Status.NodeName = "got"
			vmi.Status.MigrationState = migrationState

			resp := vmiUpdateAdmitter.Admit(context.Background(), admission(vmi, "diff"))
			Expect(resp.Allowed).To(BeTrue())
		},
			Entry("when TargetNode is not set", nil),
			Entry("when TargetNode is set", &v1.VirtualMachineInstanceMigrationState{TargetNode: "git"}),
		)

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
		Entry("VirtualMachineInstance update",
			`{"very": "unknown", "spec": { "extremely": "unknown" }}`,
			`.very in body is a forbidden property, spec.extremely in body is a forbidden property, spec.domain in body is required`,
			webhooks.VirtualMachineInstanceGroupVersionResource,
			vmiUpdateAdmitter.Admit,
		),
	)

	It("should reject valid VirtualMachineInstance spec on update", func() {
		vmi := api.NewMinimalVMI("testvmi")

		updateVmi := vmi.DeepCopy()
		updateVmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
			Name: "testdisk",
		})
		updateVmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: "testdisk",
			VolumeSource: v1.VolumeSource{
				ContainerDisk: testutils.NewFakeContainerDiskSource(),
			},
		})
		newVMIBytes, _ := json.Marshal(&updateVmi)
		oldVMIBytes, _ := json.Marshal(&vmi)

		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Resource: webhooks.VirtualMachineInstanceGroupVersionResource,
				Object: runtime.RawExtension{
					Raw: newVMIBytes,
				},
				OldObject: runtime.RawExtension{
					Raw: oldVMIBytes,
				},
				Operation: admissionv1.Update,
			},
		}

		resp := vmiUpdateAdmitter.Admit(context.Background(), ar)
		Expect(resp.Allowed).To(BeFalse())
		Expect(resp.Result.Details.Causes).To(HaveLen(1))
		Expect(resp.Result.Details.Causes[0].Message).To(Equal("update of VMI object is restricted"))
	})

	DescribeTable(
		"Should allow VMI upon modification of non kubevirt.io/ labels by non kubevirt user or service account",
		func(originalVmiLabels map[string]string, updateVmiLabels map[string]string) {
			vmi := api.NewMinimalVMI("testvmi")
			updateVmi := vmi.DeepCopy() // Don't need to copy the labels
			vmi.Labels = originalVmiLabels
			updateVmi.Labels = updateVmiLabels
			newVMIBytes, _ := json.Marshal(&updateVmi)
			oldVMIBytes, _ := json.Marshal(&vmi)
			ar := &admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					UserInfo: authv1.UserInfo{Username: "system:serviceaccount:someNamespace:someUser"},
					Resource: webhooks.VirtualMachineInstanceGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: newVMIBytes,
					},
					OldObject: runtime.RawExtension{
						Raw: oldVMIBytes,
					},
					Operation: admissionv1.Update,
				},
			}
			resp := vmiUpdateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeTrue())
		},
		Entry("Update of an existing label",
			map[string]string{"kubevirt.io/l": "someValue", "other-label/l": "value"},
			map[string]string{"kubevirt.io/l": "someValue", "other-label/l": "newValue"},
		),
		Entry("Add a new label when no labels we defined at all",
			nil,
			map[string]string{"l": "someValue"},
		),
		Entry("Delete a label",
			map[string]string{"kubevirt.io/l": "someValue", "l": "anotherValue"},
			map[string]string{"kubevirt.io/l": "someValue"},
		),
		Entry("Delete all labels",
			map[string]string{"l": "someValue", "l2": "anotherValue"},
			nil,
		),
	)

	DescribeTable(
		"Should allow VMI upon modification of kubevirt.io/ labels by kubevirt internal service account",
		func(originalVmiLabels map[string]string, updateVmiLabels map[string]string, serviceAccount string) {
			vmi := api.NewMinimalVMI("testvmi")
			updateVmi := vmi.DeepCopy() // Don't need to copy the labels
			vmi.Labels = originalVmiLabels
			updateVmi.Labels = updateVmiLabels
			newVMIBytes, _ := json.Marshal(&updateVmi)
			oldVMIBytes, _ := json.Marshal(&vmi)
			ar := &admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					UserInfo: authv1.UserInfo{Username: "system:serviceaccount:kubevirt:" + serviceAccount},
					Resource: webhooks.VirtualMachineInstanceGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: newVMIBytes,
					},
					OldObject: runtime.RawExtension{
						Raw: oldVMIBytes,
					},
					Operation: admissionv1.Update,
				},
			}

			resp := vmiUpdateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeTrue())
		},
		Entry("Update by API",
			map[string]string{v1.NodeNameLabel: "someValue"},
			map[string]string{v1.NodeNameLabel: "someNewValue"},
			components.ApiServiceAccountName,
		),
		Entry("Update by Handler",
			map[string]string{v1.NodeNameLabel: "someValue"},
			map[string]string{v1.NodeNameLabel: "someNewValue"},
			components.HandlerServiceAccountName,
		),
		Entry("Update by Controller",
			map[string]string{v1.NodeNameLabel: "someValue"},
			map[string]string{v1.NodeNameLabel: "someNewValue"},
			components.ControllerServiceAccountName,
		),
	)

	DescribeTable(
		"Should reject VMI upon modification of kubevirt.io/ reserved labels by non kubevirt user or service account",
		func(originalVmiLabels map[string]string, updateVmiLabels map[string]string) {
			vmi := api.NewMinimalVMI("testvmi")
			updateVmi := vmi.DeepCopy() // Don't need to copy the labels
			vmi.Labels = originalVmiLabels
			updateVmi.Labels = updateVmiLabels
			newVMIBytes, _ := json.Marshal(&updateVmi)
			oldVMIBytes, _ := json.Marshal(&vmi)
			ar := &admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					UserInfo: authv1.UserInfo{Username: "system:serviceaccount:someNamespace:someUser"},
					Resource: webhooks.VirtualMachineInstanceGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: newVMIBytes,
					},
					OldObject: runtime.RawExtension{
						Raw: oldVMIBytes,
					},
					Operation: admissionv1.Update,
				},
			}
			resp := vmiUpdateAdmitter.Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			Expect(resp.Result.Details.Causes[0].Message).To(Equal("modification of the following reserved kubevirt.io/ labels on a VMI object is prohibited"))
		},
		Entry("Update of an existing label",
			map[string]string{v1.CreatedByLabel: "someValue"},
			map[string]string{v1.CreatedByLabel: "someNewValue"},
		),
		Entry("Add kubevirt.io/ label when no labels we defined at all",
			nil,
			map[string]string{v1.CreatedByLabel: "someValue"},
		),
		Entry("Delete kubevirt.io/ label",
			map[string]string{"kubevirt.io/l": "someValue", v1.CreatedByLabel: "anotherValue"},
			map[string]string{"kubevirt.io/l": "someValue"},
		),
		Entry("Delete all kubevirt.io/ labels",
			map[string]string{v1.CreatedByLabel: "someValue", "kubevirt.io/l2": "anotherValue"},
			nil,
		),
	)

	DescribeTable("Admit or deny based on user", func(user string, expected types.GomegaMatcher) {
		vmi := api.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.CPU = &v1.CPU{}
		updateVmi := vmi.DeepCopy()

		// Make a spec change that is allowed only by internal service account
		updateVmi.Spec.Domain.Resources.Requests = make(k8sv1.ResourceList)
		updateVmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("128Mi")

		newVMIBytes, _ := json.Marshal(&updateVmi)
		oldVMIBytes, _ := json.Marshal(&vmi)
		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				UserInfo: authv1.UserInfo{Username: user},
				Resource: webhooks.VirtualMachineInstanceGroupVersionResource,
				Object: runtime.RawExtension{
					Raw: newVMIBytes,
				},
				OldObject: runtime.RawExtension{
					Raw: oldVMIBytes,
				},
				Operation: admissionv1.Update,
			},
		}
		resp := vmiUpdateAdmitter.Admit(context.Background(), ar)
		Expect(resp.Allowed).To(expected)
	},
		Entry("Should admit internal sa", "system:serviceaccount:kubevirt:"+components.ApiServiceAccountName, BeTrue()),
		Entry("Should reject regular user", "system:serviceaccount:someNamespace:someUser", BeFalse()),
	)

	DescribeTable("Updates in CPU topology", func(oldCPUTopology, newCPUTopology *v1.CPU, expected types.GomegaMatcher) {
		vmi := api.NewMinimalVMI("testvmi")
		updateVmi := vmi.DeepCopy()
		vmi.Spec.Domain.CPU = oldCPUTopology
		updateVmi.Spec.Domain.CPU = newCPUTopology

		newVMIBytes, _ := json.Marshal(&updateVmi)
		oldVMIBytes, _ := json.Marshal(&vmi)
		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				UserInfo: authv1.UserInfo{Username: "system:serviceaccount:kubevirt:" + components.ControllerServiceAccountName},
				Resource: webhooks.VirtualMachineInstanceGroupVersionResource,
				Object: runtime.RawExtension{
					Raw: newVMIBytes,
				},
				OldObject: runtime.RawExtension{
					Raw: oldVMIBytes,
				},
				Operation: admissionv1.Update,
			},
		}
		resp := vmiUpdateAdmitter.Admit(context.Background(), ar)
		Expect(resp.Allowed).To(expected)
	},
		Entry("deny update of maxSockets",
			&v1.CPU{
				MaxSockets: 16,
			},
			&v1.CPU{
				MaxSockets: 8,
			},
			BeFalse()))

	It("should reject updates to maxGuest", func() {
		vmi := api.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.CPU = &v1.CPU{}
		updateVmi := vmi.DeepCopy()

		maxGuest := resource.MustParse("64Mi")
		vmi.Spec.Domain.Memory = &v1.Memory{
			MaxGuest: &maxGuest,
		}
		updatedMaxGuest := resource.MustParse("128Mi")
		updateVmi.Spec.Domain.Memory = &v1.Memory{
			MaxGuest: &updatedMaxGuest,
		}

		newVMIBytes, _ := json.Marshal(&updateVmi)
		oldVMIBytes, _ := json.Marshal(&vmi)
		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				UserInfo: authv1.UserInfo{Username: "system:serviceaccount:kubevirt:" + components.ControllerServiceAccountName},
				Resource: webhooks.VirtualMachineInstanceGroupVersionResource,
				Object: runtime.RawExtension{
					Raw: newVMIBytes,
				},
				OldObject: runtime.RawExtension{
					Raw: oldVMIBytes,
				},
				Operation: admissionv1.Update,
			},
		}
		resp := vmiUpdateAdmitter.Admit(context.Background(), ar)
		Expect(resp.Allowed).To(BeFalse())
	})

})
