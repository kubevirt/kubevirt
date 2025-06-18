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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
	admissionv1 "k8s.io/api/admission/v1"
	authv1 "k8s.io/api/authentication/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"kubevirt.io/client-go/api"

	k8sv1 "k8s.io/api/core/v1"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
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

	emptyResult := func() map[string]v1.Volume {
		return make(map[string]v1.Volume, 0)
	}

	makeResult := func(indexes ...int) map[string]v1.Volume {
		res := emptyResult()
		for _, index := range indexes {
			res[fmt.Sprintf("volume-name-%d", index)] = v1.Volume{
				Name: fmt.Sprintf("volume-name-%d", index),
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: fmt.Sprintf("dv-name-%d", index),
					},
				},
			}
		}
		return res
	}

	makeVolumes := func(indexes ...int) []v1.Volume {
		res := make([]v1.Volume, 0)
		for _, index := range indexes {
			res = append(res, v1.Volume{
				Name: fmt.Sprintf("volume-name-%d", index),
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: fmt.Sprintf("dv-name-%d", index),
					},
				},
			})
		}
		return res
	}

	makeVolumesWithMemoryDumpVol := func(total int, indexes ...int) []v1.Volume {
		res := make([]v1.Volume, 0)
		for i := 0; i < total; i++ {
			memoryDump := false
			for _, index := range indexes {
				if i == index {
					memoryDump = true
					res = append(res, v1.Volume{
						Name: fmt.Sprintf("volume-name-%d", index),
						VolumeSource: v1.VolumeSource{
							MemoryDump: testutils.NewFakeMemoryDumpSource(fmt.Sprintf("volume-name-%d", index)),
						},
					})
				}
			}
			if !memoryDump {
				res = append(res, v1.Volume{
					Name: fmt.Sprintf("volume-name-%d", i),
					VolumeSource: v1.VolumeSource{
						DataVolume: &v1.DataVolumeSource{
							Name: fmt.Sprintf("dv-name-%d", i),
						},
					},
				})
			}
		}
		return res
	}

	makeInvalidVolumes := func(total int, indexes ...int) []v1.Volume {
		res := make([]v1.Volume, 0)
		for i := 0; i < total; i++ {
			foundInvalid := false
			for _, index := range indexes {
				if i == index {
					foundInvalid = true
					res = append(res, v1.Volume{
						Name: fmt.Sprintf("volume-name-%d", index),
						VolumeSource: v1.VolumeSource{
							ContainerDisk: testutils.NewFakeContainerDiskSource(),
						},
					})
				}
			}
			if !foundInvalid {
				res = append(res, v1.Volume{
					Name: fmt.Sprintf("volume-name-%d", i),
					VolumeSource: v1.VolumeSource{
						DataVolume: &v1.DataVolumeSource{
							Name: fmt.Sprintf("dv-name-%d", i),
						},
					},
				})
			}
		}
		return res
	}

	makeDisks := func(indexes ...int) []v1.Disk {
		res := make([]v1.Disk, 0)
		for _, index := range indexes {
			bootOrder := uint(index + 1)
			res = append(res, v1.Disk{
				Name: fmt.Sprintf("volume-name-%d", index),
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{
						Bus: "scsi",
					},
				},
				BootOrder: &bootOrder,
			})
		}
		return res
	}

	makeLUNDisks := func(indexes ...int) []v1.Disk {
		res := make([]v1.Disk, 0)
		for _, index := range indexes {
			bootOrder := uint(index + 1)
			res = append(res, v1.Disk{
				Name: fmt.Sprintf("volume-name-%d", index),
				DiskDevice: v1.DiskDevice{
					LUN: &v1.LunTarget{
						Bus: "scsi",
					},
				},
				BootOrder: &bootOrder,
			})
		}
		return res
	}

	makeCDRomDisks := func(indexes ...int) []v1.Disk {
		res := make([]v1.Disk, 0)
		for _, index := range indexes {
			bootOrder := uint(index + 1)
			res = append(res, v1.Disk{
				Name: fmt.Sprintf("volume-name-%d", index),
				DiskDevice: v1.DiskDevice{
					CDRom: &v1.CDRomTarget{
						Bus: "scsi",
					},
				},
				BootOrder: &bootOrder,
			})
		}
		return res
	}

	makeDisksInvalidBusLastDisk := func(indexes ...int) []v1.Disk {
		res := makeDisks(indexes...)
		for i, index := range indexes {
			if i == len(indexes)-1 {
				res[index].Disk.Bus = "invalid"
			}
		}
		return res
	}

	makeLUNDisksInvalidBusLastDisk := func(indexes ...int) []v1.Disk {
		res := makeLUNDisks(indexes...)
		for i, index := range indexes {
			if i == len(indexes)-1 {
				res[index].LUN.Bus = "invalid"
			}
		}
		return res
	}

	makeDisksWithIOThreads := func(indexes ...int) []v1.Disk {
		res := makeDisks(indexes...)
		for i, index := range indexes {
			if i == len(indexes)-1 {
				res[index].DedicatedIOThread = pointer.P(true)
			}
		}
		return res
	}

	makeDisksInvalidBootOrder := func(indexes ...int) []v1.Disk {
		res := makeDisks(indexes...)
		bootOrder := uint(0)
		for i, index := range indexes {
			if i == len(indexes)-1 {
				res[index].BootOrder = &bootOrder
			}
		}
		return res
	}

	makeDisksNoVolume := func(indexes ...int) []v1.Disk {
		res := make([]v1.Disk, 0)
		for _, index := range indexes {
			res = append(res, v1.Disk{
				Name: fmt.Sprintf("invalid-volume-name-%d", index),
			})
		}
		return res
	}

	makeFilesystems := func(indexes ...int) []v1.Filesystem {
		res := make([]v1.Filesystem, 0)
		for _, index := range indexes {
			res = append(res, v1.Filesystem{
				Name:     fmt.Sprintf("volume-name-%d", index),
				Virtiofs: &v1.FilesystemVirtiofs{},
			})
		}
		return res
	}

	makeStatus := func(statusCount, hotplugCount int) []v1.VolumeStatus {
		res := make([]v1.VolumeStatus, 0)
		for i := 0; i < statusCount; i++ {
			res = append(res, v1.VolumeStatus{
				Name:   fmt.Sprintf("volume-name-%d", i),
				Target: fmt.Sprintf("volume-target-%d", i),
			})
			if i >= statusCount-hotplugCount {
				res[i].HotplugVolume = &v1.HotplugVolumeStatus{
					AttachPodName: fmt.Sprintf("test-pod-%d", i),
				}
				res[i].Phase = v1.VolumeReady
			}
		}
		return res
	}

	makeExpected := func(message, field string) *admissionv1.AdmissionResponse {
		return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
			{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: message,
				Field:   field,
			},
		})
	}

	DescribeTable("Should properly calculate the hotplugvolumes", func(volumes []v1.Volume, statuses []v1.VolumeStatus, expected map[string]v1.Volume) {
		result := getHotplugVolumes(volumes, statuses)
		Expect(equality.Semantic.DeepEqual(result, expected)).To(BeTrue(), "result: %v and expected: %v do not match", result, expected)
	},
		Entry("Should be empty if statuses is empty", makeVolumes(), makeStatus(0, 0), emptyResult()),
		Entry("Should be empty if statuses has multiple entries, but no hotplug", makeVolumes(), makeStatus(2, 0), emptyResult()),
		Entry("Should be empty if statuses has one entry, but no hotplug", makeVolumes(), makeStatus(1, 0), emptyResult()),
		Entry("Should have a single hotplug if status has one hotplug", makeVolumes(0, 1), makeStatus(2, 1), makeResult(1)),
		Entry("Should have a multiple hotplug if status has multiple hotplug", makeVolumes(0, 1, 2, 3), makeStatus(4, 2), makeResult(2, 3)),
	)

	DescribeTable("Should properly calculate the permanent volumes", func(volumes []v1.Volume, statusVolumes []v1.VolumeStatus, expected map[string]v1.Volume) {
		result := getPermanentVolumes(volumes, statusVolumes)
		Expect(equality.Semantic.DeepEqual(result, expected)).To(BeTrue(), "result: %v and expected: %v do not match", result, expected)
	},
		Entry("Should be empty if volume is empty", makeVolumes(), makeStatus(0, 0), emptyResult()),
		Entry("Should be empty if all volumes are hotplugged", makeVolumes(0, 1, 2, 3), makeStatus(4, 4), emptyResult()),
		Entry("Should return all volumes if hotplugged is empty with multiple volumes", makeVolumes(0, 1, 2, 3), makeStatus(4, 0), makeResult(0, 1, 2, 3)),
		Entry("Should return all volumes if hotplugged is empty with a single volume", makeVolumes(0), makeStatus(1, 0), makeResult(0)),
		Entry("Should return 3 volumes if  1 hotplugged volume", makeVolumes(0, 1, 2, 3), makeStatus(4, 1), makeResult(0, 1, 2)),
	)

	testHotplugResponse := func(newVolumes, oldVolumes []v1.Volume, newDisks, oldDisks []v1.Disk, filesystems []v1.Filesystem, volumeStatuses []v1.VolumeStatus, expected *admissionv1.AdmissionResponse, featureGates ...string) {
		newVMI := api.NewMinimalVMI("testvmi")
		newVMI.Spec.Volumes = newVolumes
		newVMI.Spec.Domain.Devices.Disks = newDisks
		newVMI.Spec.Domain.Devices.Filesystems = filesystems

		for _, featureGate := range featureGates {
			enableFeatureGate(featureGate)
		}
		result := admitStorageUpdate(newVolumes, oldVolumes, newDisks, oldDisks, volumeStatuses, newVMI, vmiUpdateAdmitter.clusterConfig)
		Expect(equality.Semantic.DeepEqual(result, expected)).To(BeTrue(), "result: %v and expected: %v do not match", result, expected)
	}

	DescribeTable("Should return proper admission response", testHotplugResponse,
		Entry("Should accept if no volumes are there or added",
			makeVolumes(),
			makeVolumes(),
			makeDisks(),
			makeDisks(),
			makeFilesystems(),
			makeStatus(0, 0),
			nil),
		Entry("Should reject if #volumes != #disks",
			makeVolumes(1, 2),
			makeVolumes(1, 2),
			makeDisks(1),
			makeDisks(1),
			makeFilesystems(),
			makeStatus(0, 0),
			makeExpected("mismatch between volumes declared (2) and required (1)", "")),
		Entry("Should reject if we remove a permanent volume",
			makeVolumes(),
			makeVolumes(0),
			makeDisks(),
			makeDisks(0),
			makeFilesystems(),
			makeStatus(1, 0),
			makeExpected("Number of permanent volumes has changed", "")),
		Entry("Should reject if we add a disk without a matching volume",
			makeVolumes(0, 1),
			makeVolumes(0),
			makeDisksNoVolume(0, 1),
			makeDisksNoVolume(0),
			makeFilesystems(),
			makeStatus(1, 0),
			makeExpected("Disk volume-name-1 does not exist", "")),
		Entry("Should reject if we modify existing volume to be invalid",
			makeVolumes(0, 1),
			makeVolumes(0, 1),
			makeDisksNoVolume(0, 1),
			makeDisks(0, 1),
			makeFilesystems(),
			makeStatus(1, 0),
			makeExpected("permanent disk volume-name-0, changed", "")),
		Entry("Should reject if a hotplug volume changed",
			makeInvalidVolumes(2, 1),
			makeVolumes(0, 1),
			makeDisks(0, 1),
			makeDisks(0, 1),
			makeFilesystems(),
			makeStatus(1, 0),
			makeExpected("hotplug volume volume-name-1, changed", "")),
		Entry("Should reject if we add volumes that are not PVC or DV",
			makeInvalidVolumes(2, 1),
			makeVolumes(0),
			makeDisks(0, 1),
			makeDisks(0),
			makeFilesystems(),
			makeStatus(1, 0),
			makeExpected("volume volume-name-1 is not a PVC or DataVolume", "")),
		Entry("Should accept if we add volumes and disk properly",
			makeVolumes(0, 1),
			makeVolumes(0, 1),
			makeDisks(0, 1),
			makeDisks(0, 1),
			makeFilesystems(),
			makeStatus(2, 1),
			nil),
		Entry("Should reject if we hotplug a volume with dedicated IOThreads",
			makeVolumes(0, 1),
			makeVolumes(0),
			makeDisksWithIOThreads(0, 1),
			makeDisks(0),
			makeFilesystems(),
			makeStatus(1, 0),
			makeExpected("hotplugged Disk volume-name-1 can't use dedicated IOThread: scsi bus is unsupported.", "")),
		Entry("Should accept if we add LUN disk with valid SCSI bus",
			makeVolumes(0, 1),
			makeVolumes(0, 1),
			makeLUNDisks(0, 1),
			makeLUNDisks(0, 1),
			makeFilesystems(),
			makeStatus(2, 1),
			nil),
		Entry("Should reject if we add disk with invalid bus",
			makeVolumes(0, 1),
			makeVolumes(0),
			makeDisksInvalidBusLastDisk(0, 1),
			makeDisks(0),
			makeFilesystems(),
			makeStatus(1, 0),
			makeExpected("hotplugged Disk volume-name-1 does not use a scsi bus", "")),
		Entry("Should reject if we add LUN disk with invalid bus",
			makeVolumes(0, 1),
			makeVolumes(0),
			makeLUNDisksInvalidBusLastDisk(0, 1),
			makeLUNDisks(0),
			makeFilesystems(),
			makeStatus(1, 0),
			makeExpected("hotplugged Disk volume-name-1 does not use a scsi bus", "")),
		Entry("Should allow cd-rom",
			makeVolumes(0, 1),
			makeVolumes(0),
			makeCDRomDisks(0, 1),
			makeCDRomDisks(0),
			makeFilesystems(),
			makeStatus(1, 0),
			nil),
		Entry("Should allow cd-rom inject",
			makeVolumes(0, 1),
			makeVolumes(0),
			makeCDRomDisks(0, 1),
			makeCDRomDisks(0, 1),
			makeFilesystems(),
			makeStatus(1, 0),
			nil),
		Entry("Should allow cd-rom eject",
			makeVolumes(0),
			makeVolumes(0, 1),
			makeCDRomDisks(0, 1),
			makeCDRomDisks(0, 1),
			makeFilesystems(),
			makeStatus(1, 0),
			nil,
			featuregate.DeclarativeHotplugVolumesGate),
		Entry("Should reject cd-rom eject",
			makeVolumes(0),
			makeVolumes(0, 1),
			makeCDRomDisks(0, 1),
			makeCDRomDisks(0, 1),
			makeFilesystems(),
			makeStatus(1, 0),
			makeExpected("mismatch between volumes declared (1) and required (2)", "")),
		Entry("Should reject if we add disk with invalid boot order",
			makeVolumes(0, 1),
			makeVolumes(0),
			makeDisksInvalidBootOrder(0, 1),
			makeDisks(0),
			makeFilesystems(),
			makeStatus(1, 0),
			makeExpected("spec.domain.devices.disks[1] must have a boot order > 0, if supplied", "spec.domain.devices.disks[1].bootOrder")),
		Entry("Should accept if memory dump volume exists without matching disk",
			makeVolumesWithMemoryDumpVol(3, 2),
			makeVolumes(0, 1),
			makeDisks(0, 1),
			makeDisks(0, 1),
			makeFilesystems(),
			makeStatus(3, 1),
			nil),
		Entry("Should reject if #volumes != #disks even when there is memory dump volume",
			makeVolumesWithMemoryDumpVol(3, 2),
			makeVolumesWithMemoryDumpVol(3, 2),
			makeDisks(1),
			makeDisks(1),
			makeFilesystems(),
			makeStatus(0, 0),
			makeExpected("mismatch between volumes declared (2) and required (1)", "")),
	)

	Context("with filesystem devices", func() {
		BeforeEach(func() {
			enableFeatureGate(featuregate.VirtIOFSConfigVolumesGate)
			enableFeatureGate(featuregate.VirtIOFSStorageVolumeGate)
		})

		DescribeTable("Should return proper admission response", testHotplugResponse,
			Entry("Should accept if volume without matching disk is used by filesystem",
				makeVolumes(0, 1, 2),
				makeVolumes(0, 1),
				makeDisks(0, 2),
				makeDisks(0),
				makeFilesystems(1),
				makeStatus(3, 1),
				nil),
			Entry("Should reject if #volumes != #disks even when there are volumes used by filesystems",
				makeVolumes(0, 1, 2),
				makeVolumes(0, 1, 2),
				makeDisks(0),
				makeDisks(0),
				makeFilesystems(1),
				makeStatus(2, 0),
				makeExpected("mismatch between volumes declared (3) and required (2)", "")),
		)
	})

	DescribeTable("Admit or deny based on user", func(user string, expected types.GomegaMatcher) {
		vmi := api.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.CPU = &v1.CPU{}
		vmi.Spec.Volumes = makeVolumes(1)
		vmi.Spec.Domain.Devices.Disks = makeDisks(1)
		vmi.Status.VolumeStatus = makeStatus(1, 0)
		updateVmi := vmi.DeepCopy()
		updateVmi.Spec.Volumes = makeVolumes(2)
		updateVmi.Spec.Domain.Devices.Disks = makeDisks(2)
		updateVmi.Status.VolumeStatus = makeStatus(2, 1)

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

	DescribeTable("should allow change for a persistent volume if it is a migrated volume", func(hotpluggable bool) {
		disks := []v1.Disk{
			{
				Name:       "vol0",
				DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{Bus: v1.DiskBusVirtio}},
			},
			{
				Name:       "vol1",
				DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{Bus: v1.DiskBusVirtio}},
			},
		}
		oldVols := []v1.Volume{
			{
				Name: "vol0",
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: "pvc0"},
						Hotpluggable:                      hotpluggable,
					},
				},
			},
			{
				Name: "vol1",
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: "pvc1"},
						Hotpluggable:                      hotpluggable,
					},
				},
			},
		}
		newVols := []v1.Volume{
			{
				Name: "vol0",
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: "pvc0"},
						Hotpluggable:                      hotpluggable,
					},
				},
			},
			{
				Name: "vol1",
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: "pvc2"},
						Hotpluggable:                      hotpluggable,
					},
				},
			},
		}
		volumeStatuses := []v1.VolumeStatus{
			{Name: "vol0"},
			{Name: "vol1"},
		}
		vmi := api.NewMinimalVMI("testvmi")
		vmi.Status.MigratedVolumes = []v1.StorageMigratedVolumeInfo{
			{
				VolumeName:         "vol1",
				SourcePVCInfo:      &v1.PersistentVolumeClaimInfo{ClaimName: "pvc1"},
				DestinationPVCInfo: &v1.PersistentVolumeClaimInfo{ClaimName: "pvc1"},
			},
		}
		Expect(admitStorageUpdate(newVols, oldVols, disks, disks, volumeStatuses, vmi, vmiUpdateAdmitter.clusterConfig)).To(BeNil())
	},
		Entry("and not hotpluggable", false),
		Entry("and hotpluggable", true),
	)
})
