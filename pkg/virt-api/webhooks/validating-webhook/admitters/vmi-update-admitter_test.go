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
	"fmt"
	"reflect"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	admissionv1 "k8s.io/api/admission/v1"
	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"kubevirt.io/client-go/api"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/pkg/testutils"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/rbac"
)

var _ = Describe("Validating VMIUpdate Admitter", func() {
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
	config, _, _ := testutils.NewFakeClusterConfigUsingKV(kv)
	vmiUpdateAdmitter := &VMIUpdateAdmitter{config}

	table.DescribeTable("should reject documents containing unknown or missing fields for", func(data string, validationResult string, gvr metav1.GroupVersionResource, review func(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse) {
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
		table.Entry("VirtualMachineInstance update",
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

		resp := vmiUpdateAdmitter.Admit(ar)
		Expect(resp.Allowed).To(BeFalse())
		Expect(len(resp.Result.Details.Causes)).To(Equal(1))
		Expect(resp.Result.Details.Causes[0].Message).To(Equal("update of VMI object is restricted"))
	})

	table.DescribeTable(
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
			resp := admitVMILabelsUpdate(updateVmi, vmi, ar)
			Expect(resp).To(BeNil())
		},
		table.Entry("Update of an existing label",
			map[string]string{"kubevirt.io/l": "someValue", "other-label/l": "value"},
			map[string]string{"kubevirt.io/l": "someValue", "other-label/l": "newValue"},
		),
		table.Entry("Add a new label when no labels we defined at all",
			nil,
			map[string]string{"l": "someValue"},
		),
		table.Entry("Delete a label",
			map[string]string{"kubevirt.io/l": "someValue", "l": "anotherValue"},
			map[string]string{"kubevirt.io/l": "someValue"},
		),
		table.Entry("Delete all labels",
			map[string]string{"l": "someValue", "l2": "anotherValue"},
			nil,
		),
	)

	table.DescribeTable(
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
			resp := admitVMILabelsUpdate(updateVmi, vmi, ar)
			Expect(resp).To(BeNil())
		},
		table.Entry("Update by API",
			map[string]string{v1.NodeNameLabel: "someValue"},
			map[string]string{v1.NodeNameLabel: "someNewValue"},
			rbac.ApiServiceAccountName,
		),
		table.Entry("Update by Handler",
			map[string]string{v1.NodeNameLabel: "someValue"},
			map[string]string{v1.NodeNameLabel: "someNewValue"},
			rbac.HandlerServiceAccountName,
		),
		table.Entry("Update by Controller",
			map[string]string{v1.NodeNameLabel: "someValue"},
			map[string]string{v1.NodeNameLabel: "someNewValue"},
			rbac.ControllerServiceAccountName,
		),
	)

	table.DescribeTable(
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
			resp := admitVMILabelsUpdate(updateVmi, vmi, ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(len(resp.Result.Details.Causes)).To(Equal(1))
			Expect(resp.Result.Details.Causes[0].Message).To(Equal("modification of the following reserved kubevirt.io/ labels on a VMI object is prohibited"))
		},
		table.Entry("Update of an existing label",
			map[string]string{v1.CreatedByLabel: "someValue"},
			map[string]string{v1.CreatedByLabel: "someNewValue"},
		),
		table.Entry("Add kubevirt.io/ label when no labels we defined at all",
			nil,
			map[string]string{v1.CreatedByLabel: "someValue"},
		),
		table.Entry("Delete kubevirt.io/ label",
			map[string]string{"kubevirt.io/l": "someValue", v1.CreatedByLabel: "anotherValue"},
			map[string]string{"kubevirt.io/l": "someValue"},
		),
		table.Entry("Delete all kubevirt.io/ labels",
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

	makeDisksInvalidBusLastDisk := func(indexes ...int) []v1.Disk {
		res := makeDisks(indexes...)
		for i, index := range indexes {
			if i == len(indexes)-1 {
				res[index].Disk.Bus = "invalid"
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

	table.DescribeTable("Should properly calculate the hotplugvolumes", func(volumes []v1.Volume, statuses []v1.VolumeStatus, expected map[string]v1.Volume) {
		result := getHotplugVolumes(volumes, statuses)
		Expect(reflect.DeepEqual(result, expected)).To(BeTrue(), "result: %v and expected: %v do not match", result, expected)
	},
		table.Entry("Should be empty if statuses is empty", makeVolumes(), makeStatus(0, 0), emptyResult()),
		table.Entry("Should be empty if statuses has multiple entries, but no hotplug", makeVolumes(), makeStatus(2, 0), emptyResult()),
		table.Entry("Should be empty if statuses has one entry, but no hotplug", makeVolumes(), makeStatus(1, 0), emptyResult()),
		table.Entry("Should have a single hotplug if status has one hotplug", makeVolumes(0, 1), makeStatus(2, 1), makeResult(1)),
		table.Entry("Should have a multiple hotplug if status has multiple hotplug", makeVolumes(0, 1, 2, 3), makeStatus(4, 2), makeResult(2, 3)),
	)

	table.DescribeTable("Should properly calculate the permanent volumes", func(volumes []v1.Volume, statusVolumes []v1.VolumeStatus, expected map[string]v1.Volume) {
		result := getPermanentVolumes(volumes, statusVolumes)
		Expect(reflect.DeepEqual(result, expected)).To(BeTrue(), "result: %v and expected: %v do not match", result, expected)
	},
		table.Entry("Should be empty if volume is empty", makeVolumes(), makeStatus(0, 0), emptyResult()),
		table.Entry("Should be empty if all volumes are hotplugged", makeVolumes(0, 1, 2, 3), makeStatus(4, 4), emptyResult()),
		table.Entry("Should return all volumes if hotplugged is empty with multiple volumes", makeVolumes(0, 1, 2, 3), makeStatus(4, 0), makeResult(0, 1, 2, 3)),
		table.Entry("Should return all volumes if hotplugged is empty with a single volume", makeVolumes(0), makeStatus(1, 0), makeResult(0)),
		table.Entry("Should return 3 volumes if  1 hotplugged volume", makeVolumes(0, 1, 2, 3), makeStatus(4, 1), makeResult(0, 1, 2)),
	)

	table.DescribeTable("Should return proper admission response", func(newVolumes, oldVolumes []v1.Volume, newDisks, oldDisks []v1.Disk, volumeStatuses []v1.VolumeStatus, expected *admissionv1.AdmissionResponse) {
		newVMI := api.NewMinimalVMI("testvmi")
		newVMI.Spec.Volumes = newVolumes
		newVMI.Spec.Domain.Devices.Disks = newDisks

		result := admitHotplug(newVolumes, oldVolumes, newDisks, oldDisks, volumeStatuses, newVMI, vmiUpdateAdmitter.ClusterConfig)
		Expect(reflect.DeepEqual(result, expected)).To(BeTrue(), "result: %v and expected: %v do not match", result, expected)
	},
		table.Entry("Should accept if no volumes are there or added",
			makeVolumes(),
			makeVolumes(),
			makeDisks(),
			makeDisks(),
			makeStatus(0, 0),
			nil),
		table.Entry("Should reject if #volumes != #disks",
			makeVolumes(1, 2),
			makeVolumes(1, 2),
			makeDisks(1),
			makeDisks(1),
			makeStatus(0, 0),
			makeExpected("number of disks (1) does not equal the number of volumes (2)", "")),
		table.Entry("Should reject if we remove a permanent volume",
			makeVolumes(),
			makeVolumes(0),
			makeDisks(),
			makeDisks(0),
			makeStatus(1, 0),
			makeExpected("Number of permanent volumes has changed", "")),
		table.Entry("Should reject if we add a disk without a matching volume",
			makeVolumes(0, 1),
			makeVolumes(0),
			makeDisksNoVolume(0, 1),
			makeDisksNoVolume(0),
			makeStatus(1, 0),
			makeExpected("Disk volume-name-1 does not exist", "")),
		table.Entry("Should reject if we modify existing volume to be invalid",
			makeVolumes(0, 1),
			makeVolumes(0, 1),
			makeDisksNoVolume(0, 1),
			makeDisks(0, 1),
			makeStatus(1, 0),
			makeExpected("permanent disk volume-name-0, changed", "")),
		table.Entry("Should reject if a hotplug volume changed",
			makeInvalidVolumes(2, 1),
			makeVolumes(0, 1),
			makeDisks(0, 1),
			makeDisks(0, 1),
			makeStatus(1, 0),
			makeExpected("hotplug volume volume-name-1, changed", "")),
		table.Entry("Should reject if we add volumes that are not PVC or DV",
			makeInvalidVolumes(2, 1),
			makeVolumes(0),
			makeDisks(0, 1),
			makeDisks(0),
			makeStatus(1, 0),
			makeExpected("volume volume-name-1 is not a PVC or DataVolume", "")),
		table.Entry("Should accept if we add volumes and disk properly",
			makeVolumes(0, 1),
			makeVolumes(0, 1),
			makeDisks(0, 1),
			makeDisks(0, 1),
			makeStatus(2, 1),
			nil),
		table.Entry("Should reject if we add disk with invalid bus",
			makeVolumes(0, 1),
			makeVolumes(0),
			makeDisksInvalidBusLastDisk(0, 1),
			makeDisks(0),
			makeStatus(1, 0),
			makeExpected("hotplugged Disk volume-name-1 does not use a scsi bus", "")),
		table.Entry("Should reject if we add disk with invalid boot order",
			makeVolumes(0, 1),
			makeVolumes(0),
			makeDisksInvalidBootOrder(0, 1),
			makeDisks(0),
			makeStatus(1, 0),
			makeExpected("spec.domain.devices.disks[1] must have a boot order > 0, if supplied", "spec.domain.devices.disks[1].bootOrder")),
	)

	table.DescribeTable("Admit or deny based on user", func(user string, expected types.GomegaMatcher) {
		vmi := api.NewMinimalVMI("testvmi")
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
		resp := vmiUpdateAdmitter.Admit(ar)
		Expect(resp.Allowed).To(expected)
	},
		table.Entry("Should admit internal sa", "system:serviceaccount:kubevirt:"+rbac.ApiServiceAccountName, BeTrue()),
		table.Entry("Should reject regular user", "system:serviceaccount:someNamespace:someUser", BeFalse()),
	)
})
