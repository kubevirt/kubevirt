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

package tests_test

import (
	"flag"
	"fmt"
	"net/http"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("[rfe_id:609][crit:medium][vendor:cnv-qe@redhat.com][level:component]VMIPreset", func() {
	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	var vmi *v1.VirtualMachineInstance
	var memoryPreset *v1.VirtualMachineInstancePreset
	var cpuPreset *v1.VirtualMachineInstancePreset

	flavorKey := fmt.Sprintf("%s/flavor", v1.GroupName)
	memoryFlavor := "memory-test"
	memoryPrefix := "test-memory-"
	memory, _ := resource.ParseQuantity("128M")

	cpuPrefix := "test-cpu"
	cpuFlavor := "cpu-test"
	cores := 7

	BeforeEach(func() {
		tests.BeforeTestCleanup()
		vmi = tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))
		vmi.Labels = map[string]string{flavorKey: memoryFlavor}

		selector := k8smetav1.LabelSelector{MatchLabels: map[string]string{flavorKey: memoryFlavor}}
		memoryPreset = &v1.VirtualMachineInstancePreset{
			ObjectMeta: k8smetav1.ObjectMeta{GenerateName: memoryPrefix},
			Spec: v1.VirtualMachineInstancePresetSpec{
				Selector: selector,
				Domain: &v1.DomainSpec{
					Resources: v1.ResourceRequirements{Requests: k8sv1.ResourceList{
						"memory": memory}},
				},
			},
		}

		selector = k8smetav1.LabelSelector{MatchLabels: map[string]string{flavorKey: cpuFlavor}}
		cpuPreset = &v1.VirtualMachineInstancePreset{
			ObjectMeta: k8smetav1.ObjectMeta{GenerateName: cpuPrefix},
			Spec: v1.VirtualMachineInstancePresetSpec{
				Selector: selector,
				Domain: &v1.DomainSpec{
					CPU: &v1.CPU{Cores: uint32(cores)},
				},
			},
		}
	})

	Context("CRD Validation", func() {

		It("[test_id:1595]Should reject POST if schema is invalid", func() {
			// Preset with missing selector should fail CRD validation
			jsonString := "{\"kind\":\"VirtualMachineInstancePreset\",\"apiVersion\":\"kubevirt.io/v1alpha3\",\"metadata\":{\"generateName\":\"test-memory-\",\"creationTimestamp\":null},\"spec\":{}}"

			result := virtClient.RestClient().Post().Resource("virtualmachineinstancepresets").Namespace(tests.NamespaceTestDefault).Body([]byte(jsonString)).SetHeader("Content-Type", "application/json").Do()

			// Verify validation failed.
			statusCode := 0
			result.StatusCode(&statusCode)
			Expect(statusCode).To(Equal(http.StatusUnprocessableEntity))
		})
		It("[test_id:1596]should reject POST if validation webhoook deems the spec is invalid", func() {
			preset := &v1.VirtualMachineInstancePreset{
				ObjectMeta: k8smetav1.ObjectMeta{GenerateName: "fake"},
				Spec: v1.VirtualMachineInstancePresetSpec{
					Selector: k8smetav1.LabelSelector{MatchLabels: map[string]string{"fake": "fake"}},
					Domain:   &v1.DomainSpec{},
				},
			}
			// disk with two targets is invalid
			preset.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
				DiskDevice: v1.DiskDevice{
					Disk:  &v1.DiskTarget{},
					CDRom: &v1.CDRomTarget{},
				},
			})
			result := virtClient.RestClient().Post().Resource("virtualmachineinstancepresets").Namespace(tests.NamespaceTestDefault).Body(preset).Do()
			// Verify validation failed.
			statusCode := 0
			result.StatusCode(&statusCode)
			Expect(statusCode).To(Equal(http.StatusUnprocessableEntity))

			reviewResponse := &k8smetav1.Status{}
			body, _ := result.Raw()
			err = json.Unmarshal(body, reviewResponse)
			Expect(err).To(BeNil())

			Expect(len(reviewResponse.Details.Causes)).To(Equal(1))
			Expect(reviewResponse.Details.Causes[0].Field).To(Equal("spec.domain.devices.disks[1]"))
		})
	})
	Context("Preset Matching", func() {

		It("[test_id:1597]Should be accepted on POST", func() {
			err := virtClient.RestClient().Post().Resource("virtualmachineinstancepresets").Namespace(tests.NamespaceTestDefault).Body(memoryPreset).Do().Error()
			Expect(err).To(BeNil())
		})

		It("[test_id:1598]Should reject a second submission of a VMIPreset", func() {
			// This test requires an explicit name or the resources won't conflict
			presetName := "test-preset"
			memoryPreset.Name = presetName
			err := virtClient.RestClient().Post().Resource("virtualmachineinstancepresets").Namespace(tests.NamespaceTestDefault).Body(memoryPreset).Do().Error()
			Expect(err).ToNot(HaveOccurred())

			waitForPreset(virtClient, presetName)

			b, err := virtClient.RestClient().Post().Resource("virtualmachineinstancepresets").Namespace(tests.NamespaceTestDefault).Body(memoryPreset).DoRaw()
			Expect(err).To(HaveOccurred())
			status := k8smetav1.Status{}
			err = json.Unmarshal(b, &status)
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:1599]Should return 404 if VMIPreset does not exist", func() {
			b, err := virtClient.RestClient().Get().Resource("virtualmachineinstancepresets").Namespace(tests.NamespaceTestDefault).Name("wrong").DoRaw()
			Expect(err).To(HaveOccurred())
			status := k8smetav1.Status{}
			err = json.Unmarshal(b, &status)
			Expect(err).ToNot(HaveOccurred())
			Expect(status.Code).To(Equal(int32(http.StatusNotFound)))
		})

		It("[test_id:1600]Should reject presets that conflict with VirtualMachineInstance settings", func() {
			err := virtClient.RestClient().Post().Resource("virtualmachineinstancepresets").Namespace(tests.NamespaceTestDefault).Body(memoryPreset).Do().Error()
			Expect(err).ToNot(HaveOccurred())

			newPreset := waitForPreset(virtClient, memoryPrefix)

			newVMI, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStart(vmi)

			Expect(newVMI.Labels[flavorKey]).To(Equal(memoryFlavor))
			Expect(newPreset.Spec.Selector.MatchLabels[flavorKey]).To(Equal(memoryFlavor))

			// check the annotations
			annotationKey := fmt.Sprintf("virtualmachinepreset.%s/%s", v1.GroupName, newPreset.Name)
			_, found := newVMI.Annotations[annotationKey]
			Expect(found).To(BeFalse())
		})

		It("[test_id:1601]Should accept presets that don't conflict with VirtualMachineInstance settings", func() {
			err := virtClient.RestClient().Post().Resource("virtualmachineinstancepresets").Namespace(tests.NamespaceTestDefault).Body(cpuPreset).Do().Error()
			Expect(err).ToNot(HaveOccurred())

			newPreset := waitForPreset(virtClient, cpuPrefix)

			vmi = tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))
			vmi.Labels = map[string]string{flavorKey: cpuFlavor}

			newVMI, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStart(vmi)

			Expect(newVMI.Labels[flavorKey]).To(Equal(cpuFlavor))
			Expect(newPreset.Spec.Selector.MatchLabels[flavorKey]).To(Equal(cpuFlavor))

			// check the annotations
			annotationKey := fmt.Sprintf("virtualmachinepreset.%s/%s", v1.GroupName, newPreset.Name)
			Expect(newVMI.Annotations[annotationKey]).To(Equal("kubevirt.io/v1alpha3"))

			// check a setting from the preset itself to show it was applied
			Expect(int(newVMI.Spec.Domain.CPU.Cores)).To(Equal(cores))
		})

		It("[test_id:1602]Should ignore VMIs that don't match", func() {
			err := virtClient.RestClient().Post().Resource("virtualmachineinstancepresets").Namespace(tests.NamespaceTestDefault).Body(memoryPreset).Do().Error()
			Expect(err).ToNot(HaveOccurred())

			newPreset := waitForPreset(virtClient, memoryPrefix)

			// reset the label so it will not match
			vmi = tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))
			newVMI, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())

			tests.WaitForSuccessfulVMIStart(vmi)

			// check the annotations
			annotationKey := fmt.Sprintf("virtualmachinepreset.%s/%s", v1.GroupName, newPreset.Name)
			_, found := newVMI.Annotations[annotationKey]
			Expect(found).To(BeFalse())

			Expect(newVMI.Status.Phase).ToNot(Equal(v1.Failed))
		})

		It("[test_id:1603]Should not be applied to existing VMIs", func() {
			// create the VirtualMachineInstance first
			newVMI, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStart(vmi)

			err = virtClient.RestClient().Post().Resource("virtualmachineinstancepresets").Namespace(tests.NamespaceTestDefault).Body(memoryPreset).Do().Error()
			Expect(err).ToNot(HaveOccurred())

			newPreset := waitForPreset(virtClient, memoryPrefix)

			// check the annotations
			annotationKey := fmt.Sprintf("virtualmachinepreset.%s/%s", v1.GroupName, newPreset.Name)
			_, found := newVMI.Annotations[annotationKey]
			Expect(found).To(BeFalse())
			Expect(newVMI.Status.Phase).ToNot(Equal(v1.Failed))
		})
	})

	Context("Exclusions", func() {
		It("[test_id:1604]Should not apply presets to VirtualMachineInstance's with the exclusion marking", func() {
			err := virtClient.RestClient().Post().Resource("virtualmachineinstancepresets").Namespace(tests.NamespaceTestDefault).Body(cpuPreset).Do().Error()
			Expect(err).ToNot(HaveOccurred())

			newPreset := waitForPreset(virtClient, cpuPrefix)

			vmi = tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))
			vmi.Labels = map[string]string{flavorKey: cpuFlavor}
			exclusionMarking := "virtualmachineinstancepresets.admission.kubevirt.io/exclude"
			vmi.Annotations = map[string]string{exclusionMarking: "true"}

			newVMI, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStart(vmi)

			// check the annotations
			annotationKey := fmt.Sprintf("virtualmachinepreset.%s/%s", v1.GroupName, newPreset.Name)
			_, ok := newVMI.Annotations[annotationKey]
			Expect(ok).To(BeFalse(), "Preset should not have been applied due to exclusion")

			// check a setting from the preset itself to show it was applied
			Expect(newVMI.Spec.Domain.CPU).To(BeNil(),
				"CPU should still have been the default value (not defined in spec)")
		})
	})

	Context("Conflict", func() {
		var conflictPreset *v1.VirtualMachineInstancePreset

		conflictKey := fmt.Sprintf("%s/conflict", v1.GroupName)
		conflictFlavor := "conflict-test"
		conflictMemory, _ := resource.ParseQuantity("256M")
		conflictPrefix := "test-conflict-"

		BeforeEach(func() {
			selector := k8smetav1.LabelSelector{MatchLabels: map[string]string{conflictKey: conflictFlavor}}
			conflictPreset = &v1.VirtualMachineInstancePreset{
				ObjectMeta: k8smetav1.ObjectMeta{GenerateName: conflictPrefix},
				Spec: v1.VirtualMachineInstancePresetSpec{
					Selector: selector,
					Domain: &v1.DomainSpec{
						Resources: v1.ResourceRequirements{Requests: k8sv1.ResourceList{
							"memory": conflictMemory}},
					},
				},
			}
		})

		It("[test_id:1605]should denied to start the VMI", func() {
			err := virtClient.RestClient().Post().Resource("virtualmachineinstancepresets").Namespace(tests.NamespaceTestDefault).Body(conflictPreset).Do().Error()
			Expect(err).ToNot(HaveOccurred())
			waitForPreset(virtClient, conflictPrefix)

			err = virtClient.RestClient().Post().Resource("virtualmachineinstancepresets").Namespace(tests.NamespaceTestDefault).Body(memoryPreset).Do().Error()
			Expect(err).ToNot(HaveOccurred())
			waitForPreset(virtClient, memoryPrefix)

			vmi.Labels = map[string]string{flavorKey: memoryFlavor, conflictKey: conflictFlavor}
			By("creating the VirtualMachineInstance")
			_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).To(HaveOccurred())
		})
	})
})

func waitForPreset(virtClient kubecli.KubevirtClient, prefix string) v1.VirtualMachineInstancePreset {
	preset := v1.VirtualMachineInstancePreset{}
	Eventually(func() bool {
		presetList := v1.VirtualMachineInstancePresetList{}
		err := virtClient.RestClient().Get().Resource("virtualmachineinstancepresets").Namespace(tests.NamespaceTestDefault).Do().Into(&presetList)
		Expect(err).ToNot(HaveOccurred())
		for _, thisPreset := range presetList.Items {
			if strings.HasPrefix(thisPreset.Name, prefix) {
				preset = thisPreset
				return true
			}
		}
		return false
	}, time.Duration(60)*time.Second).Should(Equal(true), "Timed out waiting for preset to appear")
	return preset
}
