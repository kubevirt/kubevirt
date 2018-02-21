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
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("VMPreset", func() {

	dockerTag := os.Getenv("docker_tag")
	if dockerTag == "" {
		dockerTag = "latest"
	}

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	var vm *v1.VirtualMachine
	var memoryPreset *v1.VirtualMachinePreset
	var cpuPreset *v1.VirtualMachinePreset

	flavorKey := fmt.Sprintf("%s/flavor", v1.GroupName)
	memoryFlavor := "memory-test"
	memory, _ := resource.ParseQuantity("128M")

	cpuFlavor := "cpu-test"
	cores := 7

	BeforeEach(func() {
		tests.BeforeTestCleanup()
		vm = tests.NewRandomVMWithEphemeralDisk(tests.RegistryDiskFor(tests.RegistryDiskAlpine))
		vm.Labels = map[string]string{flavorKey: memoryFlavor}

		selector := k8smetav1.LabelSelector{MatchLabels: map[string]string{flavorKey: memoryFlavor}}
		memoryPreset = &v1.VirtualMachinePreset{
			ObjectMeta: k8smetav1.ObjectMeta{GenerateName: "test-memory-"},
			Spec: v1.VirtualMachinePresetSpec{
				Selector: selector,
				Domain: &v1.DomainSpec{
					Resources: v1.ResourceRequirements{Requests: k8sv1.ResourceList{
						"memory": memory}},
				},
			},
		}

		selector = k8smetav1.LabelSelector{MatchLabels: map[string]string{flavorKey: cpuFlavor}}
		cpuPreset = &v1.VirtualMachinePreset{
			ObjectMeta: k8smetav1.ObjectMeta{GenerateName: "test-cpu-"},
			Spec: v1.VirtualMachinePresetSpec{
				Selector: selector,
				Domain: &v1.DomainSpec{
					CPU: &v1.CPU{Cores: uint32(cores)},
				},
			},
		}
	})

	Context("Preset Matching", func() {

		It("Should be accepted on POST", func() {
			err := virtClient.RestClient().Post().Resource("virtualmachinepresets").Namespace(tests.NamespaceTestDefault).Body(memoryPreset).Do().Error()
			Expect(err).To(BeNil())
		})

		It("Should reject a second submission of a VMPreset", func() {
			// This test requires an explicit name or the resources won't conflict
			memoryPreset.Name = "test-preset"
			err := virtClient.RestClient().Post().Resource("virtualmachinepresets").Namespace(tests.NamespaceTestDefault).Body(memoryPreset).Do().Error()
			Expect(err).ToNot(HaveOccurred())

			waitForPreset(virtClient)

			b, err := virtClient.RestClient().Post().Resource("virtualmachinepresets").Namespace(tests.NamespaceTestDefault).Body(memoryPreset).DoRaw()
			Expect(err).To(HaveOccurred())
			status := k8smetav1.Status{}
			err = json.Unmarshal(b, &status)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should return 404 if VMPreset does not exist", func() {
			b, err := virtClient.RestClient().Get().Resource("virtualmachinepresets").Namespace(tests.NamespaceTestDefault).Name("wrong").DoRaw()
			Expect(err).To(HaveOccurred())
			status := k8smetav1.Status{}
			err = json.Unmarshal(b, &status)
			Expect(err).ToNot(HaveOccurred())
			Expect(status.Code).To(Equal(int32(http.StatusNotFound)))
		})

		It("Should reject presets that conflict with VM settings", func() {
			err := virtClient.RestClient().Post().Resource("virtualmachinepresets").Namespace(tests.NamespaceTestDefault).Body(memoryPreset).Do().Error()
			Expect(err).ToNot(HaveOccurred())

			newPreset := waitForPreset(virtClient)

			err = virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).Do().Error()
			Expect(err).ToNot(HaveOccurred())

			newVm := v1.VirtualMachine{}
			err = virtClient.RestClient().Get().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Name(vm.Name).Do().Into(&newVm)
			Expect(err).ToNot(HaveOccurred())

			Expect(newVm.Labels[flavorKey]).To(Equal(memoryFlavor))
			Expect(newPreset.Spec.Selector.MatchLabels[flavorKey]).To(Equal(memoryFlavor))

			// check the annotations
			Expect(len(newVm.Annotations)).To(Equal(0))
		})

		It("Should accept presets that don't conflict with VM settings", func() {
			err := virtClient.RestClient().Post().Resource("virtualmachinepresets").Namespace(tests.NamespaceTestDefault).Body(cpuPreset).Do().Error()
			Expect(err).ToNot(HaveOccurred())

			newPreset := waitForPreset(virtClient)

			vm = tests.NewRandomVMWithEphemeralDisk("kubevirt/alpine-registry-disk-demo:devel")
			vm.Labels = map[string]string{flavorKey: cpuFlavor}

			err = virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).Do().Error()
			Expect(err).ToNot(HaveOccurred())

			newVm := v1.VirtualMachine{}
			err = virtClient.RestClient().Get().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Name(vm.Name).Do().Into(&newVm)
			Expect(err).ToNot(HaveOccurred())

			Expect(newVm.Labels[flavorKey]).To(Equal(cpuFlavor))
			Expect(newPreset.Spec.Selector.MatchLabels[flavorKey]).To(Equal(cpuFlavor))

			// check the annotations
			Expect(len(newVm.Annotations)).To(Equal(1))
			annotationKey := fmt.Sprintf("virtualmachinepreset.%s/%s", v1.GroupName, newPreset.Name)
			Expect(newVm.Annotations[annotationKey]).To(Equal("kubevirt.io/v1alpha1"))

			// check a setting from the preset itself to show it was applied
			Expect(int(newVm.Spec.Domain.CPU.Cores)).To(Equal(cores))
		})

		It("Should ignore VMs that don't match", func() {
			err := virtClient.RestClient().Post().Resource("virtualmachinepresets").Namespace(tests.NamespaceTestDefault).Body(memoryPreset).Do().Error()
			Expect(err).ToNot(HaveOccurred())

			// reset the label so it will not match
			vm = tests.NewRandomVMWithEphemeralDisk("kubevirt/alpine-registry-disk-demo:devel")
			err = virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).Do().Error()
			Expect(err).ToNot(HaveOccurred())

			newVm := v1.VirtualMachine{}
			err = virtClient.RestClient().Get().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Name(vm.Name).Do().Into(&newVm)
			Expect(err).ToNot(HaveOccurred())

			Expect(len(newVm.Annotations)).To(Equal(0))
		})

		It("Should not be applied to existing VMs", func() {
			// create the VM first
			err = virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).Do().Error()
			Expect(err).ToNot(HaveOccurred())

			err := virtClient.RestClient().Post().Resource("virtualmachinepresets").Namespace(tests.NamespaceTestDefault).Body(memoryPreset).Do().Error()
			Expect(err).ToNot(HaveOccurred())

			newVm := v1.VirtualMachine{}
			err = virtClient.RestClient().Get().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Name(vm.Name).Do().Into(&newVm)
			Expect(err).ToNot(HaveOccurred())

			Expect(len(newVm.Annotations)).To(Equal(0))
		})
	})
})

func waitForPreset(virtClient kubecli.KubevirtClient) v1.VirtualMachinePreset {
	preset := v1.VirtualMachinePreset{}
	Eventually(func() bool {
		presetList := v1.VirtualMachinePresetList{}
		err := virtClient.RestClient().Get().Resource("virtualmachinepresets").Namespace(tests.NamespaceTestDefault).Do().Into(&presetList)
		Expect(err).ToNot(HaveOccurred())
		if len(presetList.Items) == 1 {
			preset = presetList.Items[0]
		}
		return true
	}, time.Duration(60)*time.Second).Should(Equal(true), "Timed out waiting for preset to appear")
	return preset
}
