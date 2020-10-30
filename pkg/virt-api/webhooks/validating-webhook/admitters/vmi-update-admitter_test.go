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
	"k8s.io/api/admission/v1beta1"
	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	v1 "kubevirt.io/client-go/api/v1"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-operator/creation/rbac"
)

var _ = Describe("Validating VMIUpdate Admitter", func() {
	vmiUpdateAdmitter := &VMIUpdateAdmitter{}

	table.DescribeTable("should reject documents containing unknown or missing fields for", func(data string, validationResult string, gvr metav1.GroupVersionResource, review func(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse) {
		input := map[string]interface{}{}
		json.Unmarshal([]byte(data), &input)

		ar := &v1beta1.AdmissionReview{
			Request: &v1beta1.AdmissionRequest{
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
		vmi := v1.NewMinimalVMI("testvmi")

		updateVmi := vmi.DeepCopy()
		updateVmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
			Name: "testdisk",
		})
		updateVmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: "testdisk",
			VolumeSource: v1.VolumeSource{
				ContainerDisk: &v1.ContainerDiskSource{},
			},
		})
		newVMIBytes, _ := json.Marshal(&updateVmi)
		oldVMIBytes, _ := json.Marshal(&vmi)

		ar := &v1beta1.AdmissionReview{
			Request: &v1beta1.AdmissionRequest{
				Resource: webhooks.VirtualMachineInstanceGroupVersionResource,
				Object: runtime.RawExtension{
					Raw: newVMIBytes,
				},
				OldObject: runtime.RawExtension{
					Raw: oldVMIBytes,
				},
				Operation: v1beta1.Update,
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
			vmi := v1.NewMinimalVMI("testvmi")
			updateVmi := vmi.DeepCopy() // Don't need to copy the labels
			vmi.Labels = originalVmiLabels
			updateVmi.Labels = updateVmiLabels
			newVMIBytes, _ := json.Marshal(&updateVmi)
			oldVMIBytes, _ := json.Marshal(&vmi)
			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					UserInfo: authv1.UserInfo{Username: "system:serviceaccount:someNamespace:someUser"},
					Resource: webhooks.VirtualMachineInstanceGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: newVMIBytes,
					},
					OldObject: runtime.RawExtension{
						Raw: oldVMIBytes,
					},
					Operation: v1beta1.Update,
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
			vmi := v1.NewMinimalVMI("testvmi")
			updateVmi := vmi.DeepCopy() // Don't need to copy the labels
			vmi.Labels = originalVmiLabels
			updateVmi.Labels = updateVmiLabels
			newVMIBytes, _ := json.Marshal(&updateVmi)
			oldVMIBytes, _ := json.Marshal(&vmi)
			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					UserInfo: authv1.UserInfo{Username: "system:serviceaccount:kubevirt:" + serviceAccount},
					Resource: webhooks.VirtualMachineInstanceGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: newVMIBytes,
					},
					OldObject: runtime.RawExtension{
						Raw: oldVMIBytes,
					},
					Operation: v1beta1.Update,
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
			vmi := v1.NewMinimalVMI("testvmi")
			updateVmi := vmi.DeepCopy() // Don't need to copy the labels
			vmi.Labels = originalVmiLabels
			updateVmi.Labels = updateVmiLabels
			newVMIBytes, _ := json.Marshal(&updateVmi)
			oldVMIBytes, _ := json.Marshal(&vmi)
			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					UserInfo: authv1.UserInfo{Username: "system:serviceaccount:someNamespace:someUser"},
					Resource: webhooks.VirtualMachineInstanceGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: newVMIBytes,
					},
					OldObject: runtime.RawExtension{
						Raw: oldVMIBytes,
					},
					Operation: v1beta1.Update,
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

	makeVolumes := func(volumeCount int) []v1.Volume {
		res := make([]v1.Volume, 0)
		for i := 0; i < volumeCount; i++ {
			res = append(res, v1.Volume{
				Name: fmt.Sprintf("volume-name-%d", i),
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: fmt.Sprintf("dv-name-%d", i),
					},
				},
			})
		}
		return res
	}

	makeInvalidVolumes := func(volumeCount int) []v1.Volume {
		res := make([]v1.Volume, 0)
		for i := 0; i < volumeCount; i++ {
			res = append(res, v1.Volume{
				Name: fmt.Sprintf("volume-name-%d", i),
				VolumeSource: v1.VolumeSource{
					ContainerDisk: &v1.ContainerDiskSource{},
				},
			})
		}
		return res
	}

	makeDisks := func(diskCount int) []v1.Disk {
		res := make([]v1.Disk, 0)
		for i := 0; i < diskCount; i++ {
			res = append(res, v1.Disk{
				Name: fmt.Sprintf("volume-name-%d", i),
			})
		}
		return res
	}

	makeDisksNoVolume := func(diskCount int) []v1.Disk {
		res := make([]v1.Disk, 0)
		for i := 0; i < diskCount; i++ {
			if i < diskCount-1 {
				res = append(res, v1.Disk{
					Name: fmt.Sprintf("volume-name-%d", i),
				})
			} else {
				res = append(res, v1.Disk{
					Name: fmt.Sprintf("invalid-volume-name-%d", i),
				})
			}
		}
		return res
	}

	makeStatus := func(statusCount, hotplugCount int) []v1.VolumeStatus {
		res := make([]v1.VolumeStatus, 0)
		for i := 0; i < statusCount; i++ {
			res = append(res, v1.VolumeStatus{
				Name: fmt.Sprintf("volume-name-%d", i),
			})
			if i >= statusCount-hotplugCount {
				res[i].HotplugVolume = &v1.HotplugVolumeStatus{
					AttachPodName: fmt.Sprintf("test-pod-%d", i),
					Phase:         v1.HotplugVolumeReady,
				}
			}
		}
		return res
	}

	makeExpected := func(message string) *v1beta1.AdmissionResponse {
		return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
			{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: message,
			},
		})
	}

	table.DescribeTable("Should properly calculate the hotplugvolumes", func(volumes []v1.Volume, statuses []v1.VolumeStatus, expected map[string]v1.Volume) {
		result := getHotplugVolumes(volumes, statuses)
		Expect(reflect.DeepEqual(result, expected)).To(BeTrue(), "result: %v and expected: %v do not match", result, expected)
	},
		table.Entry("Should be empty if statuses is empty", makeVolumes(0), makeStatus(0, 0), emptyResult()),
		table.Entry("Should be empty if statuses is not empty, but no hotplug", makeVolumes(0), makeStatus(2, 0), emptyResult()),
		table.Entry("Should have a single hotplug if status has one hotplug", makeVolumes(2), makeStatus(2, 1), makeResult(1)),
		table.Entry("Should have a multiple hotplug if status has multiple hotplug", makeVolumes(4), makeStatus(4, 2), makeResult(2, 3)),
	)

	table.DescribeTable("Should properly calculate the permanent volumes", func(volumes []v1.Volume, hotpluggedVolumes, expected map[string]v1.Volume) {
		result := getPermanentVolumes(volumes, hotpluggedVolumes)
		Expect(reflect.DeepEqual(result, expected)).To(BeTrue(), "result: %v and expected: %v do not match", result, expected)
	},
		table.Entry("Should be empty if volume is empty", makeVolumes(0), emptyResult(), emptyResult()),
		table.Entry("Should be empty if all volumes are hotplugged", makeVolumes(4), makeResult(0, 1, 2, 3), emptyResult()),
		table.Entry("Should return all volumes if hotplugged is empty", makeVolumes(4), emptyResult(), makeResult(0, 1, 2, 3)),
		table.Entry("Should return 3 volumes if  1 hotplugged volume", makeVolumes(4), makeResult(2), makeResult(0, 1, 3)),
	)

	table.DescribeTable("Should return proper admission response", func(newVolumes []v1.Volume, newDisks []v1.Disk, volumeStatuses []v1.VolumeStatus, expected *v1beta1.AdmissionResponse) {
		result := admitHotplug(newVolumes, newDisks, volumeStatuses)
		Expect(reflect.DeepEqual(result, expected)).To(BeTrue(), "result: %v and expected: %v do not match", result, expected)
		// hotpluggedVolumes := getHotplugVolumes(newVolumes, volumeStatuses)
		// permanent := getPermanentVolumes(newVolumes, hotpluggedVolumes)
		// Fail(fmt.Sprintf("status: %v, hp: %v, per: %v", volumeStatuses, hotpluggedVolumes, permanent))
	},
		table.Entry("Should reject if no volumes are there or added, need a minimum of 1 volume", makeVolumes(0), makeDisks(0), makeStatus(0, 0), makeExpected("cannot remove permanent volume")),
		table.Entry("Should reject if #volumes != #disks", makeVolumes(2), makeDisks(1), makeStatus(0, 0), makeExpected("number of disks does not equal the number of volumes")),
		table.Entry("Should reject if we remove a permanent volume", makeVolumes(0), makeDisks(0), makeStatus(1, 0), makeExpected("cannot remove permanent volume")),
		table.Entry("Should reject if we add a disk without a matching volume", makeVolumes(2), makeDisksNoVolume(2), makeStatus(2, 1), makeExpected("Disk invalid-volume-name-1 doesn't have a matching volume")),
		table.Entry("Should reject if we add volumes that are not PVC or DV", makeInvalidVolumes(2), makeDisks(2), makeStatus(2, 1), makeExpected("Disk volume-name-1 has a volume that is not a PVC or DataVolume")),
		table.Entry("Should accept if we add volumes and disk properly", makeVolumes(2), makeDisks(2), makeStatus(2, 1), nil),
	)

	table.DescribeTable("Admit or deny based on user", func(user string, expected types.GomegaMatcher) {
		vmi := v1.NewMinimalVMI("testvmi")
		vmi.Spec.Volumes = makeVolumes(1)
		vmi.Spec.Domain.Devices.Disks = makeDisks(1)
		vmi.Status.VolumeStatus = makeStatus(1, 0)
		updateVmi := vmi.DeepCopy()
		updateVmi.Spec.Volumes = makeVolumes(2)
		updateVmi.Spec.Domain.Devices.Disks = makeDisks(2)
		updateVmi.Status.VolumeStatus = makeStatus(2, 1)

		newVMIBytes, _ := json.Marshal(&updateVmi)
		oldVMIBytes, _ := json.Marshal(&vmi)
		ar := &v1beta1.AdmissionReview{
			Request: &v1beta1.AdmissionRequest{
				UserInfo: authv1.UserInfo{Username: user},
				Resource: webhooks.VirtualMachineInstanceGroupVersionResource,
				Object: runtime.RawExtension{
					Raw: newVMIBytes,
				},
				OldObject: runtime.RawExtension{
					Raw: oldVMIBytes,
				},
				Operation: v1beta1.Update,
			},
		}
		resp := vmiUpdateAdmitter.Admit(ar)
		Expect(resp.Allowed).To(expected)
	},
		table.Entry("Should admit internal sa", "system:serviceaccount:kubevirt:"+rbac.ApiServiceAccountName, BeTrue()),
		table.Entry("Should reject regular user", "system:serviceaccount:someNamespace:someUser", BeFalse()),
	)
})
