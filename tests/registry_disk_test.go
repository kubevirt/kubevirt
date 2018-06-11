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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package tests_test

import (
	"flag"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/runtime"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("RegistryDisk", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	LaunchVMI := func(vm *v1.VirtualMachineInstance) runtime.Object {
		By("Starting a VirtualMachineInstance")
		obj, err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vm).Do().Get()
		Expect(err).To(BeNil())
		return obj
	}

	VerifyRegistryDiskVMI := func(vm *v1.VirtualMachineInstance, obj runtime.Object, ignoreWarnings bool) {
		_, ok := obj.(*v1.VirtualMachineInstance)
		Expect(ok).To(BeTrue(), "Object is not of type *v1.VirtualMachineInstance")
		if ignoreWarnings == true {
			tests.WaitForSuccessfulVMIStartIgnoreWarnings(obj)
		} else {
			tests.WaitForSuccessfulVMIStart(obj)
		}

		// Verify Registry Disks are Online
		pods, err := virtClient.CoreV1().Pods(tests.NamespaceTestDefault).List(tests.UnfinishedVMIPodSelector(vm))
		Expect(err).To(BeNil())

		By("Checking the number of VirtualMachineInstance disks")
		disksFound := 0
		for _, pod := range pods.Items {
			if pod.ObjectMeta.DeletionTimestamp != nil {
				continue
			}
			for _, containerStatus := range pod.Status.ContainerStatuses {
				if strings.HasPrefix(containerStatus.Name, "volume") == false {
					// only check readiness of disk containers
					continue
				}
				disksFound++
			}
			break
		}
		Expect(disksFound).To(Equal(1))
	}

	Describe("Starting and stopping the same VirtualMachineInstance", func() {
		Context("with ephemeral registry disk", func() {
			It("should success multiple times", func() {
				vm := tests.NewRandomVMIWithEphemeralDisk(tests.RegistryDiskFor(tests.RegistryDiskCirros))
				num := 2
				for i := 0; i < num; i++ {
					By("Starting the VirtualMachineInstance")
					obj, err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vm).Do().Get()
					Expect(err).To(BeNil())
					tests.WaitForSuccessfulVMIStartWithTimeout(obj, 180)

					By("Stopping the VirtualMachineInstance")
					_, err = virtClient.RestClient().Delete().Resource("virtualmachineinstances").Namespace(vm.GetObjectMeta().GetNamespace()).Name(vm.GetObjectMeta().GetName()).Do().Get()
					Expect(err).To(BeNil())
					By("Waiting until the VirtualMachineInstance is gone")
					tests.WaitForVirtualMachineToDisappearWithTimeout(vm, 120)
				}
			})
		})
	})

	Describe("Starting a VirtualMachineInstance", func() {
		Context("with ephemeral registry disk", func() {
			It("should not modify the spec on status update", func() {
				vm := tests.NewRandomVMIWithEphemeralDisk(tests.RegistryDiskFor(tests.RegistryDiskCirros))
				v1.SetObjectDefaults_VirtualMachineInstance(vm)

				By("Starting the VirtualMachineInstance")
				vm, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vm)
				Expect(err).To(BeNil())
				tests.WaitForSuccessfulVMIStartWithTimeout(vm, 60)
				startedVMI, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vm.ObjectMeta.Name, metav1.GetOptions{})
				Expect(err).To(BeNil())
				By("Checking that the VirtualMachineInstance spec did not change")
				Expect(startedVMI.Spec).To(Equal(vm.Spec))
			})
		})
	})

	Describe("Starting multiple VMIs", func() {
		Context("with ephemeral registry disk", func() {
			It("should success", func() {
				num := 5
				vms := make([]*v1.VirtualMachineInstance, 0, num)
				objs := make([]runtime.Object, 0, num)
				for i := 0; i < num; i++ {
					vm := tests.NewRandomVMIWithEphemeralDisk(tests.RegistryDiskFor(tests.RegistryDiskCirros))
					// FIXME if we give too much ram, the vms really boot and eat all our memory (cache?)
					vm.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1M")
					obj := LaunchVMI(vm)
					vms = append(vms, vm)
					objs = append(objs, obj)
				}

				for idx, vm := range vms {
					// TODO once networking is implemented properly set ignoreWarnings == false here.
					// We have to ignore warnings because VMIs started in parallel
					// may cause libvirt to fail to create the macvtap device in
					// the host network.
					// The new network implementation we're working on should resolve this.
					// NOTE the VirtualMachineInstance still starts successfully regardless of this warning.
					// It just requires virt-handler to retry the Start command at the moment.
					VerifyRegistryDiskVMI(vm, objs[idx], true)
				}
			}) // Timeout is long because this test involves multiple parallel VirtualMachineInstance launches.
		})
	})
})
