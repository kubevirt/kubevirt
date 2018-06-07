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
	"regexp"
	"strconv"
	"time"

	"github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
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

	Describe("VirtualMachineInstance definition", func() {
		Context("with 3 CPU cores", func() {
			var vmi *v1.VirtualMachineInstance

			BeforeEach(func() {
				vmi = tests.NewRandomVMIWithEphemeralDisk(tests.RegistryDiskFor(tests.RegistryDiskAlpine))
			})
			It("should report 3 cpu cores under guest OS", func() {
				vmi.Spec.Domain.CPU = &v1.CPU{
					Cores: 3,
				}
				vmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: kubev1.ResourceList{
						kubev1.ResourceMemory: resource.MustParse("64M"),
					},
				}

				By("Starting a VirtualMachineInstance")
				vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStart(vmi)

				By("Expecting the VirtualMachineInstance console")
				expecter, _, err := tests.NewConsoleExpecter(virtClient, vmi, 10*time.Second)
				Expect(err).ToNot(HaveOccurred())
				defer expecter.Close()

				By("Checking the number of CPU cores under guest OS")
				_, err = expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: "Welcome to Alpine"},
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: "login"},
					&expect.BSnd{S: "root\n"},
					&expect.BExp{R: "#"},
					&expect.BSnd{S: "grep -c ^processor /proc/cpuinfo\n"},
					&expect.BExp{R: "3"},
				}, 250*time.Second)

				By("Checking the requested amount of memory allocated for a guest")
				Expect(vmi.Spec.Domain.Resources.Requests.Memory().String()).To(Equal("64M"))

				readyPod := tests.GetRunningPodByLabel(vmi.Name, v1.DomainLabel, tests.NamespaceTestDefault)
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

		Context("with hugepages", func() {
			var hugepagesVm *v1.VirtualMachine

			verifyHugepagesConsumption := func() {
				// TODO: we need to check hugepages state via node allocated resources, but currently it has the issue
				// https://github.com/kubernetes/kubernetes/issues/64691
				pods, err := virtClient.Core().Pods(tests.NamespaceTestDefault).List(tests.UnfinishedVMPodSelector(hugepagesVm))
				Expect(err).ToNot(HaveOccurred())
				Expect(len(pods.Items)).To(Equal(1))

				// Get a hugepages statistics from virt-launcher pod
				output, err := tests.ExecuteCommandOnPod(
					virtClient,
					&pods.Items[0],
					pods.Items[0].Spec.Containers[0].Name,
					[]string{"cat", "/proc/meminfo"},
				)
				Expect(err).ToNot(HaveOccurred())

				hostHugepages := regexp.MustCompile(`HugePages_Total:\s*([0-9]+)\nHugePages_Free:\s*([0-9]+)`)

				totalHugepages, err := strconv.Atoi(hostHugepages.FindStringSubmatch(output)[1])
				Expect(err).ToNot(HaveOccurred())

				freeHugepages, err := strconv.Atoi(hostHugepages.FindStringSubmatch(output)[2])
				Expect(err).ToNot(HaveOccurred())

				// Verify that the VM memory equals to a number of consumed hugepages
				hugepagesSize := resource.MustParse(hugepagesVm.Spec.Domain.Hugepages.Size)
				vmHugepagesConsumption := int64(totalHugepages-freeHugepages) * hugepagesSize.Value()
				vmMemory := hugepagesVm.Spec.Domain.Resources.Requests[kubev1.ResourceMemory]

				Expect(vmHugepagesConsumption).To(Equal(vmMemory.Value()))
			}

			BeforeEach(func() {
				hugepagesVm = tests.NewRandomVMWithEphemeralDiskAndUserdata(tests.RegistryDiskFor(tests.RegistryDiskCirros), "#!/bin/bash\necho 'hello'\n")
				hugepagesVm.Spec.Domain.Resources.Requests[kubev1.ResourceMemory] = resource.MustParse("64Mi")
			})

			table.DescribeTable("should consume hugepages ", func(resourceName kubev1.ResourceName, value string) {
				nodeWithHugepages := tests.GetNodeWithHugepages(virtClient, resourceName)
				if nodeWithHugepages == nil {
					Skip(fmt.Sprintf("No node with hugepages %s capacity", resourceName))
				}
				// initialHugepages := nodeWithHugepages.Status.Capacity[resourceName]
				hugepagesVm.Spec.Affinity = &v1.Affinity{
					NodeAffinity: &kubev1.NodeAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: &kubev1.NodeSelector{
							NodeSelectorTerms: []kubev1.NodeSelectorTerm{
								{
									MatchExpressions: []kubev1.NodeSelectorRequirement{
										{Key: "kubernetes.io/hostname", Operator: kubev1.NodeSelectorOpIn, Values: []string{nodeWithHugepages.Name}},
									},
								},
							},
						},
					},
				}
				hugepagesVm.Spec.Domain.Hugepages = &v1.Hugepages{}
				hugepagesVm.Spec.Domain.Hugepages.Size = value

				By("Starting a VM")
				_, err = virtClient.VM(tests.NamespaceTestDefault).Create(hugepagesVm)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMStart(hugepagesVm)

				By("Checking that the VM memory equals to a number of consumed hugepages")
				// TODO: we need to check hugepages state via node allocated resources, but currently it has the issue
				// https://github.com/kubernetes/kubernetes/issues/64691
				verifyHugepagesConsumption()
			},
				table.Entry("hugepages-2Mi", v1.Hugepage2MiResource, "2Mi"),
				table.Entry("hugepages-1Gi", v1.Hugepage1GiResource, "1Gi"),
			)
		})
	})

	Context("New VirtualMachineInstance with all supported drives", func() {

		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			// ordering:
			// use a small disk for the other ones
			containerImage := tests.RegistryDiskFor(tests.RegistryDiskCirros)
			// virtio - added by NewRandomVMIWithEphemeralDisk
			vmi = tests.NewRandomVMIWithEphemeralDiskAndUserdata(containerImage, "echo hi!\n")
			// sata
			tests.AddEphemeralDisk(vmi, "disk2", "sata", containerImage)
			// ide
			tests.AddEphemeralDisk(vmi, "disk3", "ide", containerImage)
			// floppy
			tests.AddEphemeralFloppy(vmi, "disk4", containerImage)
			// NOTE: we have one disk per bus, so we expect vda, sda, hda, fda

			// We need ide support for the test, q35 does not support ide
			vmi.Spec.Domain.Machine.Type = "pc"
		})

		// FIXME ide and floppy is not recognized by the used image right now
		It("should have all the device nodes", func() {
			vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStart(vmi)

			expecter, err := tests.LoggedInCirrosExpecter(vmi)
			Expect(err).ToNot(HaveOccurred())
			defer expecter.Close()
			res, err := expecter.ExpectBatch([]expect.Batcher{
				// keep the ordering!
				&expect.BSnd{S: "ls /dev/sda  /dev/vda  /dev/vdb\n"},
				&expect.BExp{R: "/dev/sda  /dev/vda  /dev/vdb"},
			}, 10*time.Second)
			log.DefaultLogger().Object(vmi).Infof("%v", res)

			Expect(err).ToNot(HaveOccurred())
		})
	})

})
