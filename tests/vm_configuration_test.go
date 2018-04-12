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
	"fmt"
	"time"

	"github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	kubev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Configurations", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	Describe("VM definition", func() {
		Context("with 3 CPU cores", func() {
			var vm *v1.VirtualMachine

			BeforeEach(func() {
				vm = tests.NewRandomVMWithEphemeralDisk(tests.RegistryDiskFor(tests.RegistryDiskAlpine))
			})
			It("should report 3 cpu cores under guest OS", func() {
				vm.Spec.Domain.CPU = &v1.CPU{
					Cores: 3,
				}
				vm.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: kubev1.ResourceList{
						kubev1.ResourceMemory: resource.MustParse("64M"),
					},
				}

				By("Starting a VM")
				vm, err = virtClient.VM(tests.NamespaceTestDefault).Create(vm)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMStart(vm)

				By("Expecting the VM console")
				expecter, _, err := tests.NewConsoleExpecter(virtClient, vm, 10*time.Second)
				Expect(err).ToNot(HaveOccurred())
				defer expecter.Close()

				By("Checking the number of CPU cores under guest OS")
				_, err = expecter.ExpectBatch([]expect.Batcher{
					&expect.BExp{R: "Welcome to Alpine"},
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: "login"},
					&expect.BSnd{S: "root\n"},
					&expect.BExp{R: "#"},
					&expect.BSnd{S: "grep -c ^processor /proc/cpuinfo\n"},
					&expect.BExp{R: "3"},
				}, 250*time.Second)

				By("Checking the requested amount of memory allocated for a guest")
				Expect(vm.Spec.Domain.Resources.Requests.Memory().String()).To(Equal("64M"))

				readyPod := tests.GetRunningPodByLabel(vm.Name, v1.DomainLabel, tests.NamespaceTestDefault)
				var computeContainer *kubev1.Container
				for _, container := range readyPod.Spec.Containers {
					println(container.Name)
					if container.Name == "compute" {
						computeContainer = &container
					}
				}
				if computeContainer == nil {
					tests.PanicOnError(fmt.Errorf("could not find the compute container"))
				}
				Expect(computeContainer.Resources.Requests.Memory().ToDec().ScaledValue(resource.Mega)).To(Equal(int64(179)))

				Expect(err).ToNot(HaveOccurred())
			}, 300)
		})
	})

	Context("New VM with all supported drives", func() {

		var vm *v1.VirtualMachine

		BeforeEach(func() {
			// ordering:
			// use a small disk for the other ones
			containerImage := tests.RegistryDiskFor(tests.RegistryDiskCirros)
			// virtio - added by NewRandomVMWithEphemeralDisk
			vm = tests.NewRandomVMWithEphemeralDiskAndUserdata(containerImage, "echo hi!\n")
			// sata
			tests.AddEphemeralDisk(vm, "disk2", "sata", containerImage)
			// ide
			tests.AddEphemeralDisk(vm, "disk3", "ide", containerImage)
			// floppy
			tests.AddEphemeralFloppy(vm, "disk4", containerImage)
			// NOTE: we have one disk per bus, so we expect vda, sda, hda, fda

			// We need ide support for the test, q35 does not support ide
			vm.Spec.Domain.Machine.Type = "pc"
		})

		// FIXME ide and floppy is not recognized by the used image right now
		It("should have all the device nodes", func() {
			vm, err = virtClient.VM(tests.NamespaceTestDefault).Create(vm)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMStart(vm)

			expecter, err := tests.LoggedInCirrosExpecter(vm)
			Expect(err).ToNot(HaveOccurred())
			defer expecter.Close()
			res, err := expecter.ExpectBatch([]expect.Batcher{
				// keep the ordering!
				&expect.BSnd{S: "ls /dev/sda  /dev/vda  /dev/vdb\n"},
				&expect.BExp{R: "/dev/sda  /dev/vda  /dev/vdb"},
			}, 10*time.Second)
			log.DefaultLogger().Object(vm).Infof("%v", res)

			Expect(err).ToNot(HaveOccurred())
		})
	})

})
