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
	"strings"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	kubev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	hw_utils "kubevirt.io/kubevirt/pkg/util/hardware"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Configurations", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	getComputeContainerOfPod := func(pod *kubev1.Pod) *kubev1.Container {
		var computeContainer *kubev1.Container
		for _, container := range pod.Spec.Containers {
			if container.Name == "compute" {
				computeContainer = &container
				break
			}
		}
		if computeContainer == nil {
			tests.PanicOnError(fmt.Errorf("could not find the compute container"))
		}
		return computeContainer
	}

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	Describe("[rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]VirtualMachineInstance definition", func() {
		Context("with 3 CPU cores", func() {
			var vmi *v1.VirtualMachineInstance

			BeforeEach(func() {
				vmi = tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))
			})
			It("[test_id:1659]should report 3 cpu cores under guest OS", func() {
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
				Expect(err).ToNot(HaveOccurred(), "should start vmi")
				tests.WaitForSuccessfulVMIStart(vmi)

				By("Expecting the VirtualMachineInstance console")
				expecter, err := tests.LoggedInAlpineExpecter(vmi)
				Expect(err).ToNot(HaveOccurred(), "should start console")
				defer expecter.Close()

				By("Checking the number of CPU cores under guest OS")
				_, err = expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: "grep -c ^processor /proc/cpuinfo\n"},
					&expect.BExp{R: "3"},
				}, 15*time.Second)
				Expect(err).ToNot(HaveOccurred(), "should report number of cores")

				By("Checking the requested amount of memory allocated for a guest")
				Expect(vmi.Spec.Domain.Resources.Requests.Memory().String()).To(Equal("64M"))

				readyPod := tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)
				var computeContainer *kubev1.Container
				for _, container := range readyPod.Spec.Containers {
					if container.Name == "compute" {
						computeContainer = &container
						break
					}
				}
				if computeContainer == nil {
					tests.PanicOnError(fmt.Errorf("could not find the compute container"))
				}
				Expect(computeContainer.Resources.Requests.Memory().ToDec().ScaledValue(resource.Mega)).To(Equal(int64(243)))

				Expect(err).ToNot(HaveOccurred())
			})

			It("[test_id:1660]should report 3 sockets under guest OS", func() {
				vmi.Spec.Domain.CPU = &v1.CPU{
					Sockets: 3,
					Cores:   2,
				}
				vmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: kubev1.ResourceList{
						kubev1.ResourceMemory: resource.MustParse("64M"),
					},
				}

				By("Starting a VirtualMachineInstance")
				vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred(), "should start vmi")
				tests.WaitForSuccessfulVMIStart(vmi)

				By("Expecting the VirtualMachineInstance console")
				expecter, err := tests.LoggedInAlpineExpecter(vmi)
				Expect(err).ToNot(HaveOccurred(), "should start console")
				defer expecter.Close()

				By("Checking the number of sockets under guest OS")
				_, err = expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: "grep '^physical id' /proc/cpuinfo | uniq | wc -l\n"},
					&expect.BExp{R: "3"},
				}, 60*time.Second)
				Expect(err).ToNot(HaveOccurred(), "should report number of sockets")
			})

			It("[test_id:1661]should report 2 sockets from spec.domain.resources.requests under guest OS ", func() {
				vmi.Spec.Domain.CPU = nil
				vmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: kubev1.ResourceList{
						kubev1.ResourceCPU:    resource.MustParse("1200m"),
						kubev1.ResourceMemory: resource.MustParse("64M"),
					},
				}

				By("Starting a VirtualMachineInstance")
				vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred(), "should start vmi")
				tests.WaitForSuccessfulVMIStart(vmi)

				By("Expecting the VirtualMachineInstance console")
				expecter, err := tests.LoggedInAlpineExpecter(vmi)
				Expect(err).ToNot(HaveOccurred(), "should start console")
				defer expecter.Close()

				By("Checking the number of sockets under guest OS")
				_, err = expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: "grep '^physical id' /proc/cpuinfo | uniq | wc -l\n"},
					&expect.BExp{R: "2"},
				}, 60*time.Second)
				Expect(err).ToNot(HaveOccurred(), "should report number of sockets")
			})

			It("[test_id:1662]should report 2 sockets from spec.domain.resources.limits under guest OS ", func() {
				vmi.Spec.Domain.CPU = nil
				vmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: kubev1.ResourceList{
						kubev1.ResourceMemory: resource.MustParse("64M"),
					},
					Limits: kubev1.ResourceList{
						kubev1.ResourceCPU: resource.MustParse("1200m"),
					},
				}

				By("Starting a VirtualMachineInstance")
				vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred(), "should start vmi")
				tests.WaitForSuccessfulVMIStart(vmi)

				By("Expecting the VirtualMachineInstance console")
				expecter, err := tests.LoggedInAlpineExpecter(vmi)
				Expect(err).ToNot(HaveOccurred(), "should start console")
				defer expecter.Close()

				By("Checking the number of sockets under guest OS")
				_, err = expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: "grep '^physical id' /proc/cpuinfo | uniq | wc -l\n"},
					&expect.BExp{R: "2"},
				}, 60*time.Second)
				Expect(err).ToNot(HaveOccurred(), "should report number of sockets")
			})

			It("[test_id:1663]should report 4 vCPUs under guest OS", func() {
				vmi.Spec.Domain.CPU = &v1.CPU{
					Threads: 2,
					Sockets: 2,
					Cores:   1,
				}
				vmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: kubev1.ResourceList{
						kubev1.ResourceMemory: resource.MustParse("64M"),
					},
				}

				By("Starting a VirtualMachineInstance")
				vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred(), "should start vmi")
				tests.WaitForSuccessfulVMIStart(vmi)

				By("Expecting the VirtualMachineInstance console")
				expecter, err := tests.LoggedInAlpineExpecter(vmi)
				Expect(err).ToNot(HaveOccurred(), "should start console")
				defer expecter.Close()

				By("Checking the number of vCPUs under guest OS")
				_, err = expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: "grep -c ^processor /proc/cpuinfo\n"},
					&expect.BExp{R: "4"},
				}, 60*time.Second)
				Expect(err).ToNot(HaveOccurred(), "should report number of threads")
			})

			It("[test_id:1664]should map cores to virtio block queues", func() {
				_true := true
				vmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: kubev1.ResourceList{
						kubev1.ResourceMemory: resource.MustParse("64M"),
						kubev1.ResourceCPU:    resource.MustParse("3"),
					},
				}
				vmi.Spec.Domain.Devices.BlockMultiQueue = &_true

				By("Starting a VirtualMachineInstance")
				vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStart(vmi)

				domXml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(domXml).To(ContainSubstring("queues='3'"))
			})

			It("[test_id:1665]should map cores to virtio net queues", func() {
				if shouldUseEmulation(virtClient) {
					Skip("Software emulation should not be enabled for this test to run")
				}

				_true := true
				vmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: kubev1.ResourceList{
						kubev1.ResourceMemory: resource.MustParse("64M"),
						kubev1.ResourceCPU:    resource.MustParse("3"),
					},
				}

				vmi.Spec.Domain.Devices.NetworkInterfaceMultiQueue = &_true

				By("Starting a VirtualMachineInstance")
				vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStart(vmi)

				domXml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(domXml).To(ContainSubstring("driver name='vhost' queues='3'"))
			})

			It("[test_id:1667]should not enforce explicitly rejected virtio block queues without cores", func() {
				_false := false
				vmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: kubev1.ResourceList{
						kubev1.ResourceMemory: resource.MustParse("64M"),
					},
				}
				vmi.Spec.Domain.Devices.BlockMultiQueue = &_false

				By("Starting a VirtualMachineInstance")
				vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStart(vmi)

				domXml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(domXml).ToNot(ContainSubstring("queues='"))
			})
		})

		Context("with EFI bootloader method", func() {

			It("[test_id:1668]should use EFI", func() {
				vmi := tests.NewRandomVMIWithEFIBootloader()

				By("Starting a VirtualMachineInstance")
				vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitUntilVMIReady(vmi, tests.LoggedInAlpineExpecter)

				By("Checking if UEFI is enabled")
				domXml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(domXml).To(ContainSubstring("OVMF_CODE"))
			})
		})

		Context("[rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]with diverging guest memory from requested memory", func() {
			It("[test_id:1669]should show the requested guest memory inside the VMI", func() {
				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				guestMemory := resource.MustParse("64M")
				vmi.Spec.Domain.Resources.Requests[kubev1.ResourceMemory] = resource.MustParse("64M")
				guestMemory.Add(*vmi.Spec.Domain.Resources.Requests.Memory())
				vmi.Spec.Domain.Memory = &v1.Memory{
					Guest: &guestMemory,
				}

				vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStart(vmi)

				expecter, err := tests.LoggedInCirrosExpecter(vmi)
				Expect(err).ToNot(HaveOccurred())
				defer expecter.Close()

				res, err := expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: "free -m | grep Mem: | tr -s ' ' | cut -d' ' -f2\n"},
					&expect.BExp{R: "105"},
				}, 10*time.Second)
				log.DefaultLogger().Object(vmi).Infof("%v", res)
				Expect(err).ToNot(HaveOccurred())

			})

		})

		Context("with usb controller", func() {
			It("should start the VMI with usb controller when usb device is present", func() {
				vmi := tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))
				vmi.Spec.Domain.Devices.Inputs = []v1.Input{
					{
						Name: "tablet0",
						Type: "tablet",
						Bus:  "usb",
					},
				}
				By("Starting a VirtualMachineInstance")
				vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred(), "should start vmi")
				tests.WaitForSuccessfulVMIStart(vmi)

				By("Expecting the VirtualMachineInstance console")
				expecter, err := tests.LoggedInAlpineExpecter(vmi)
				Expect(err).ToNot(HaveOccurred(), "should get console")
				defer expecter.Close()

				By("Checking the number of usb under guest OS")
				_, err = expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: "ls -l /sys/bus/usb/devices/usb* | wc -l\n"},
					&expect.BExp{R: "2"},
				}, 60*time.Second)
				Expect(err).ToNot(HaveOccurred(), "should report number of usb")
			})

			It("should start the VMI without usb controller", func() {
				vmi := tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))
				By("Starting a VirtualMachineInstance")
				vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred(), "should start vmi")

				tests.WaitForSuccessfulVMIStart(vmi)

				By("Expecting the VirtualMachineInstance console")
				expecter, err := tests.LoggedInAlpineExpecter(vmi)
				Expect(err).ToNot(HaveOccurred(), "should start console")
				defer expecter.Close()
				By("Checking the number of usb under guest OS")
				_, err = expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: "ls -l /sys/bus/usb/devices/usb* 2>/dev/null | wc -l\n"},
					&expect.BExp{R: "0"},
				}, 60*time.Second)
				Expect(err).ToNot(HaveOccurred(), "should report number of usb")
			})
		})

		Context("with input devices", func() {
			It("should failed to start the VMI with wrong type of input device", func() {
				vmi := tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskFedora))
				vmi.Spec.Domain.Devices.Inputs = []v1.Input{
					{
						Name: "tablet0",
						Type: "keyboard",
						Bus:  "virtio",
					},
				}
				By("Starting a VirtualMachineInstance")
				vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).To(HaveOccurred(), "should not start vmi")
			})

			It("should failed to start the VMI with wrong bus of input device", func() {
				vmi := tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskFedora))
				vmi.Spec.Domain.Devices.Inputs = []v1.Input{
					{
						Name: "tablet0",
						Type: "tablet",
						Bus:  "ps2",
					},
				}
				By("Starting a VirtualMachineInstance")
				vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).To(HaveOccurred(), "should not start vmi")
			})

			It("should start the VMI with tablet input device with virtio bus", func() {
				vmi := tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))
				vmi.Spec.Domain.Devices.Inputs = []v1.Input{
					{
						Name: "tablet0",
						Type: "tablet",
						Bus:  "virtio",
					},
				}
				By("Starting a VirtualMachineInstance")
				vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred(), "should start vmi")
				tests.WaitForSuccessfulVMIStart(vmi)

				By("Expecting the VirtualMachineInstance console")
				expecter, err := tests.LoggedInAlpineExpecter(vmi)
				Expect(err).ToNot(HaveOccurred(), "should start console")
				defer expecter.Close()

				By("Checking the tablet input under guest OS")
				_, err = expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: "grep -rs '^QEMU Virtio Tablet' /sys/devices | wc -l\n"},
					&expect.BExp{R: "1"},
				}, 60*time.Second)
				Expect(err).ToNot(HaveOccurred(), "should report input device")
			})

			It("should start the VMI with tablet input device with usb bus", func() {
				vmi := tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))
				vmi.Spec.Domain.Devices.Inputs = []v1.Input{
					{
						Name: "tablet0",
						Type: "tablet",
						Bus:  "usb",
					},
				}
				By("Starting a VirtualMachineInstance")
				vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred(), "should start vmi")
				tests.WaitForSuccessfulVMIStart(vmi)

				By("Expecting the VirtualMachineInstance console")
				expecter, err := tests.LoggedInAlpineExpecter(vmi)
				Expect(err).ToNot(HaveOccurred(), "should start console")
				defer expecter.Close()

				By("Checking the tablet input under guest OS")
				_, err = expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: "grep -rs '^QEMU USB Tablet' /sys/devices | wc -l\n"},
					&expect.BExp{R: "1"},
				}, 60*time.Second)
				Expect(err).ToNot(HaveOccurred(), "should report input device")
			})
		})

		Context("[rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]with namespace memory limits lower than VMI required memory", func() {
			var vmi *v1.VirtualMachineInstance
			It("[test_id:1670]should failed to start the VMI", func() {
				// create a namespace default limit
				limitRangeObj := kubev1.LimitRange{

					ObjectMeta: metav1.ObjectMeta{Name: "abc1", Namespace: tests.NamespaceTestDefault},
					Spec: kubev1.LimitRangeSpec{
						Limits: []kubev1.LimitRangeItem{
							{
								Type: kubev1.LimitTypeContainer,
								Default: kubev1.ResourceList{
									kubev1.ResourceMemory: resource.MustParse("32Mi"),
								},
							},
						},
					},
				}
				_, err := virtClient.Core().LimitRanges(tests.NamespaceTestDefault).Create(&limitRangeObj)
				Expect(err).ToNot(HaveOccurred())

				By("Starting a VirtualMachineInstance")
				// Retrying up to 5 sec, then if you still succeeds in VMI creation, things must be going wrong.
				Eventually(func() error {
					vmi = tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))
					vmi.Spec.Domain.Resources = v1.ResourceRequirements{
						Requests: kubev1.ResourceList{
							kubev1.ResourceMemory: resource.MustParse("64M"),
						},
					}
					vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
					virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Delete(vmi.Name, &metav1.DeleteOptions{})
					return err
				}, 5*time.Second, 1*time.Second).Should(MatchError("admission webhook \"virtualmachineinstances-create-validator.kubevirt.io\" denied the request: spec.domain.resources.requests.memory '64M' is greater than spec.domain.resources.limits.memory '32Mi'"))
			})
		})

		Context("with namespace cpu limits lower than VMI required cpu", func() {
			var vmi *v1.VirtualMachineInstance
			It("should fail to start the VMI", func() {
				// create a namespace default limit
				limitRangeObj := kubev1.LimitRange{

					ObjectMeta: metav1.ObjectMeta{Name: "abc1", Namespace: tests.NamespaceTestDefault},
					Spec: kubev1.LimitRangeSpec{
						Limits: []kubev1.LimitRangeItem{
							{
								Type: kubev1.LimitTypeContainer,
								Default: kubev1.ResourceList{
									kubev1.ResourceCPU: resource.MustParse("500m"),
								},
							},
						},
					},
				}
				_, err := virtClient.Core().LimitRanges(tests.NamespaceTestDefault).Create(&limitRangeObj)
				Expect(err).ToNot(HaveOccurred())

				By("Starting a VirtualMachineInstance")
				// Retrying up to 5 sec, then if you still succeeds in VMI creation, things must be going wrong.
				Eventually(func() error {
					vmi = tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))
					vmi.Spec.Domain.Resources = v1.ResourceRequirements{
						Requests: kubev1.ResourceList{
							kubev1.ResourceCPU: resource.MustParse("800m"),
						},
					}
					vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
					virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Delete(vmi.Name, &metav1.DeleteOptions{})
					return err
				}, 5*time.Second, 1*time.Second).Should(MatchError("admission webhook \"virtualmachineinstances-create-validator.kubevirt.io\" denied the request: spec.domain.resources.requests.cpu '800m' is greater than spec.domain.resources.limits.cpu '500m'"))
			})
		})

		Context("with namespace limits higher than VMI requests", func() {
			var vmi *v1.VirtualMachineInstance
			It("should start the VMI with the right default settings from namespace limits", func() {
				// create a namespace default limit
				limitRangeObj := kubev1.LimitRange{

					ObjectMeta: metav1.ObjectMeta{Name: "abc1", Namespace: tests.NamespaceTestDefault},
					Spec: kubev1.LimitRangeSpec{
						Limits: []kubev1.LimitRangeItem{
							{
								Type: kubev1.LimitTypeContainer,
								Default: kubev1.ResourceList{
									kubev1.ResourceCPU:    resource.MustParse("2000m"),
									kubev1.ResourceMemory: resource.MustParse("512M"),
								},
								DefaultRequest: kubev1.ResourceList{
									kubev1.ResourceCPU: resource.MustParse("500m"),
								},
							},
						},
					},
				}
				_, err := virtClient.Core().LimitRanges(tests.NamespaceTestDefault).Create(&limitRangeObj)
				Expect(err).ToNot(HaveOccurred())

				vmi = tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))
				vmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: kubev1.ResourceList{
						kubev1.ResourceMemory: resource.MustParse("64M"),
					},
					Limits: kubev1.ResourceList{
						kubev1.ResourceCPU: resource.MustParse("1000m"),
					},
				}

				By("Starting a VirtualMachineInstance")
				vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred(), "should start vmi")
				tests.WaitForSuccessfulVMIStart(vmi)

				Expect(vmi.Spec.Domain.Resources.Requests.Memory().ToDec().ScaledValue(resource.Mega)).To(Equal(int64(64)))
				Expect(vmi.Spec.Domain.Resources.Limits.Memory().ToDec().ScaledValue(resource.Mega)).To(Equal(int64(512)))
				Expect(vmi.Spec.Domain.Resources.Requests.Cpu().MilliValue()).To(Equal(int64(500)))
				Expect(vmi.Spec.Domain.Resources.Limits.Cpu().MilliValue()).To(Equal(int64(1000)))

				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("with hugepages", func() {
			var hugepagesVmi *v1.VirtualMachineInstance

			verifyHugepagesConsumption := func() {
				// TODO: we need to check hugepages state via node allocated resources, but currently it has the issue
				// https://github.com/kubernetes/kubernetes/issues/64691
				pods, err := virtClient.Core().Pods(tests.NamespaceTestDefault).List(tests.UnfinishedVMIPodSelector(hugepagesVmi))
				Expect(err).ToNot(HaveOccurred())
				Expect(len(pods.Items)).To(Equal(1))

				hugepagesSize := resource.MustParse(hugepagesVmi.Spec.Domain.Memory.Hugepages.PageSize)
				hugepagesDir := fmt.Sprintf("/sys/kernel/mm/hugepages/hugepages-%dkB", hugepagesSize.Value()/int64(1024))

				// Get a hugepages statistics from virt-launcher pod
				output, err := tests.ExecuteCommandOnPod(
					virtClient,
					&pods.Items[0],
					pods.Items[0].Spec.Containers[0].Name,
					[]string{"cat", fmt.Sprintf("%s/nr_hugepages", hugepagesDir)},
				)
				Expect(err).ToNot(HaveOccurred())

				totalHugepages, err := strconv.Atoi(strings.Trim(output, "\n"))
				Expect(err).ToNot(HaveOccurred())

				output, err = tests.ExecuteCommandOnPod(
					virtClient,
					&pods.Items[0],
					pods.Items[0].Spec.Containers[0].Name,
					[]string{"cat", fmt.Sprintf("%s/free_hugepages", hugepagesDir)},
				)
				Expect(err).ToNot(HaveOccurred())

				freeHugepages, err := strconv.Atoi(strings.Trim(output, "\n"))
				Expect(err).ToNot(HaveOccurred())

				// Verify that the VM memory equals to a number of consumed hugepages
				vmHugepagesConsumption := int64(totalHugepages-freeHugepages) * hugepagesSize.Value()
				vmMemory := hugepagesVmi.Spec.Domain.Resources.Requests[kubev1.ResourceMemory]

				Expect(vmHugepagesConsumption).To(Equal(vmMemory.Value()))
			}

			BeforeEach(func() {
				hugepagesVmi = tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
			})

			table.DescribeTable("should consume hugepages ", func(hugepageSize string, memory string) {
				hugepageType := kubev1.ResourceName(kubev1.ResourceHugePagesPrefix + hugepageSize)

				nodeWithHugepages := tests.GetNodeWithHugepages(virtClient, hugepageType)
				if nodeWithHugepages == nil {
					Skip(fmt.Sprintf("No node with hugepages %s capacity", hugepageType))
				}
				// initialHugepages := nodeWithHugepages.Status.Capacity[resourceName]
				hugepagesVmi.Spec.Affinity = &kubev1.Affinity{
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
				hugepagesVmi.Spec.Domain.Resources.Requests[kubev1.ResourceMemory] = resource.MustParse(memory)

				hugepagesVmi.Spec.Domain.Memory = &v1.Memory{
					Hugepages: &v1.Hugepages{PageSize: hugepageSize},
				}

				By("Starting a VM")
				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(hugepagesVmi)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStart(hugepagesVmi)

				By("Checking that the VM memory equals to a number of consumed hugepages")
				verifyHugepagesConsumption()
			},
				table.Entry("[test_id:1671]hugepages-2Mi", "2Mi", "64Mi"),
				table.Entry("[test_id:1672]hugepages-1Gi", "1Gi", "1Gi"),
			)

			Context("with usupported page size", func() {
				It("[test_id:1673]should failed to schedule the pod", func() {
					nodes, err := virtClient.Core().Nodes().List(metav1.ListOptions{})
					Expect(err).ToNot(HaveOccurred())

					hugepageType2Mi := kubev1.ResourceName(kubev1.ResourceHugePagesPrefix + "2Mi")
					for _, node := range nodes.Items {
						if _, ok := node.Status.Capacity[hugepageType2Mi]; !ok {
							Skip("No nodes with hugepages support")
						}
					}

					hugepagesVmi.Spec.Domain.Resources.Requests[kubev1.ResourceMemory] = resource.MustParse("66Mi")

					hugepagesVmi.Spec.Domain.Memory = &v1.Memory{
						Hugepages: &v1.Hugepages{PageSize: "3Mi"},
					}

					By("Starting a VM")
					_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(hugepagesVmi)
					Expect(err).ToNot(HaveOccurred())

					var vmiCondition v1.VirtualMachineInstanceCondition
					Eventually(func() bool {
						vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(hugepagesVmi.Name, &metav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred())

						if len(vmi.Status.Conditions) > 0 {
							for _, cond := range vmi.Status.Conditions {
								if cond.Type == v1.VirtualMachineInstanceConditionType(kubev1.PodScheduled) && cond.Status == kubev1.ConditionFalse {
									vmiCondition = vmi.Status.Conditions[0]
									return true
								}
							}
						}
						return false
					}, 30*time.Second, time.Second).Should(BeTrue())
					Expect(vmiCondition.Message).To(ContainSubstring("Insufficient hugepages-3Mi"))
					Expect(vmiCondition.Reason).To(Equal("Unschedulable"))
				})
			})
		})

		Context("[rfe_id:893][crit:medium][vendor:cnv-qe@redhat.com][level:component]with rng", func() {
			var rngVmi *v1.VirtualMachineInstance

			BeforeEach(func() {
				rngVmi = tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))
			})

			It("[test_id:1674]should have the virtio rng device present when present", func() {
				rngVmi.Spec.Domain.Devices.Rng = &v1.Rng{}

				By("Starting a VirtualMachineInstance")
				rngVmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(rngVmi)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStart(rngVmi)

				By("Expecting the VirtualMachineInstance console")
				expecter, err := tests.LoggedInAlpineExpecter(rngVmi)
				Expect(err).ToNot(HaveOccurred())
				defer expecter.Close()

				By("Checking the virtio rng presence")
				_, err = expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: "grep -c ^virtio /sys/devices/virtual/misc/hw_random/rng_available\n"},
					&expect.BExp{R: "1"},
				}, 400*time.Second)
				Expect(err).ToNot(HaveOccurred())
			})

			It("[test_id:1675]should not have the virtio rng device when not present", func() {
				By("Starting a VirtualMachineInstance")
				rngVmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(rngVmi)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStart(rngVmi)

				By("Expecting the VirtualMachineInstance console")
				expecter, err := tests.LoggedInAlpineExpecter(rngVmi)
				Expect(err).ToNot(HaveOccurred())
				defer expecter.Close()

				By("Checking the virtio rng presence")
				_, err = expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: "[[ ! -e /sys/devices/virtual/misc/hw_random/rng_available ]] && echo non\n"},
					&expect.BExp{R: "non"},
				}, 400*time.Second)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("with guestAgent", func() {
			var agentVMI *v1.VirtualMachineInstance

			It("[test_id:1676]should have attached a guest agent channel by default", func() {

				agentVMI = tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))
				By("Starting a VirtualMachineInstance")
				agentVMI, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(agentVMI)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI successfully")
				tests.WaitForSuccessfulVMIStart(agentVMI)

				getOptions := metav1.GetOptions{}
				var freshVMI *v1.VirtualMachineInstance

				freshVMI, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(agentVMI.Name, &getOptions)
				Expect(err).ToNot(HaveOccurred(), "Should get VMI ")

				domXML, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, freshVMI)
				Expect(err).ToNot(HaveOccurred(), "Should return XML from VMI")

				Expect(domXML).To(ContainSubstring("<channel type='unix'>"), "Should contain at least one channel")
				Expect(domXML).To(ContainSubstring("<target type='virtio' name='org.qemu.guest_agent.0' state='disconnected'/>"), "Should have guest agent channel present")
				Expect(domXML).To(ContainSubstring("<alias name='channel0'/>"), "Should have guest channel present")
			})

			It("[test_id:1677]VMI condition should signal agent presence", func() {

				// TODO: actually review this once the VM image is present
				agentVMI := tests.NewRandomFedoraVMIWitGuestAgent()

				By("Starting a VirtualMachineInstance")
				agentVMI, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(agentVMI)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI successfully")
				tests.WaitForSuccessfulVMIStart(agentVMI)

				getOptions := metav1.GetOptions{}
				var freshVMI *v1.VirtualMachineInstance

				By("VMI has the guest agent connected condition")
				Eventually(func() []v1.VirtualMachineInstanceCondition {
					freshVMI, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(agentVMI.Name, &getOptions)
					Expect(err).ToNot(HaveOccurred(), "Should get VMI ")
					return freshVMI.Status.Conditions
				}, 240*time.Second, 2).Should(
					ContainElement(
						MatchFields(
							IgnoreExtras,
							Fields{"Type": Equal(v1.VirtualMachineInstanceAgentConnected)})),
					"Should have agent connected condition")

				By("Expecting the VirtualMachineInstance console")
				expecter, err := tests.LoggedInFedoraExpecter(agentVMI)
				Expect(err).ToNot(HaveOccurred())
				defer expecter.Close()

				By("Terminating guest agent and waiting for it to dissappear.")
				res, err := expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: "systemctl stop guestagent\n"},
				}, 400*time.Second)
				log.DefaultLogger().Object(agentVMI).Infof("Login: %v", res)
				Expect(err).ToNot(HaveOccurred())

				By("VMI has the guest agent connected condition")
				Eventually(func() []v1.VirtualMachineInstanceCondition {
					freshVMI, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(agentVMI.Name, &getOptions)
					Expect(err).ToNot(HaveOccurred(), "Should get VMI ")
					return freshVMI.Status.Conditions
				}, 240*time.Second, 2).ShouldNot(
					ContainElement(
						MatchFields(
							IgnoreExtras,
							Fields{"Type": Equal(v1.VirtualMachineInstanceAgentConnected)})),
					"Agent condition should be gone")
			})
		})

		Context("with serial-number", func() {
			var snVmi *v1.VirtualMachineInstance

			BeforeEach(func() {
				snVmi = tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))
			})

			It("should have serial-number set when present", func() {
				snVmi.Spec.Domain.Firmware = &v1.Firmware{Serial: "4b2f5496-f3a3-460b-a375-168223f68845"}

				By("Starting a VirtualMachineInstance")
				snVmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(snVmi)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStart(snVmi)

				getOptions := metav1.GetOptions{}
				var freshVMI *v1.VirtualMachineInstance

				freshVMI, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(snVmi.Name, &getOptions)
				Expect(err).ToNot(HaveOccurred(), "Should get VMI ")

				domXML, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, freshVMI)
				Expect(err).ToNot(HaveOccurred(), "Should return XML from VMI")

				Expect(domXML).To(ContainSubstring("<entry name='serial'>4b2f5496-f3a3-460b-a375-168223f68845</entry>"), "Should have serial-number present")
			})

			It("should not have serial-number set when not present", func() {
				By("Starting a VirtualMachineInstance")
				snVmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(snVmi)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStart(snVmi)

				getOptions := metav1.GetOptions{}
				var freshVMI *v1.VirtualMachineInstance

				freshVMI, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(snVmi.Name, &getOptions)
				Expect(err).ToNot(HaveOccurred(), "Should get VMI ")

				domXML, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, freshVMI)
				Expect(err).ToNot(HaveOccurred(), "Should return XML from VMI")

				Expect(domXML).ToNot(ContainSubstring("<entry name='serial'>"), "Should have serial-number present")
			})
		})

	})

	Context("[rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]with CPU spec", func() {
		libvirtCPUModelRegexp := regexp.MustCompile(`<model>(\w+)\-*\w*</model>`)
		libvirtCPUVendorRegexp := regexp.MustCompile(`<vendor>(\w+)</vendor>`)
		libvirtCPUFeatureRegexp := regexp.MustCompile(`<feature name='(\w+)'/>`)
		cpuModelNameRegexp := regexp.MustCompile(`Model name:\s*([\s\w\-@\.\(\)]+)`)

		var libvirtCpuModel string
		var libvirtCpuVendor string
		var cpuModelName string
		var cpuFeatures []string
		var cpuVmi *v1.VirtualMachineInstance

		// Collect capabilities once for all tests
		tests.BeforeAll(func() {
			vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
			_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			node := tests.WaitForSuccessfulVMIStart(vmi)

			virshCaps := tests.GetNodeLibvirtCapabilities(vmi)

			model := libvirtCPUModelRegexp.FindStringSubmatch(virshCaps)
			Expect(len(model)).To(Equal(2))
			libvirtCpuModel = model[1]

			vendor := libvirtCPUVendorRegexp.FindStringSubmatch(virshCaps)
			Expect(len(vendor)).To(Equal(2))
			libvirtCpuVendor = vendor[1]

			cpuFeaturesList := libvirtCPUFeatureRegexp.FindAllStringSubmatch(virshCaps, -1)

			for _, cpuFeature := range cpuFeaturesList {
				cpuFeatures = append(cpuFeatures, cpuFeature[1])
			}

			cpuInfo := tests.GetNodeCPUInfo(vmi)
			modelName := cpuModelNameRegexp.FindStringSubmatch(cpuInfo)
			Expect(len(modelName)).To(Equal(2))
			cpuModelName = modelName[1]

			cpuVmi = tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
			cpuVmi.Spec.Affinity = &kubev1.Affinity{
				NodeAffinity: &kubev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &kubev1.NodeSelector{
						NodeSelectorTerms: []kubev1.NodeSelectorTerm{
							{
								MatchExpressions: []kubev1.NodeSelectorRequirement{
									{Key: "kubernetes.io/hostname", Operator: kubev1.NodeSelectorOpIn, Values: []string{node}},
								},
							},
						},
					},
				},
			}

			// Best to also delete the VMI, in the case that there is only one spot free for scheduling
			err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Delete(vmi.Name, &metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() bool {
				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
				if errors.IsNotFound(err) {
					return true
				}
				return false
			}, 30*time.Second, 1*time.Second).Should(BeTrue())
		})

		Context("[rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]when CPU model defined", func() {
			It("[test_id:1678]should report defined CPU model", func() {
				vmiModel := "Conroe"
				if libvirtCpuVendor == "AMD" {
					vmiModel = "Opteron_G1"
				}
				cpuVmi.Spec.Domain.CPU = &v1.CPU{
					Model: vmiModel,
				}

				By("Starting a VirtualMachineInstance")
				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(cpuVmi)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStart(cpuVmi)

				By("Expecting the VirtualMachineInstance console")
				expecter, err := tests.LoggedInCirrosExpecter(cpuVmi)
				Expect(err).ToNot(HaveOccurred())
				defer expecter.Close()

				By("Checking the CPU model under the guest OS")
				_, err = expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: fmt.Sprintf("grep %s /proc/cpuinfo\n", vmiModel)},
					&expect.BExp{R: "model name"},
				}, 10*time.Second)
			})
		})

		Context("[rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]when CPU model equals to passthrough", func() {
			It("[test_id:1679]should report exactly the same model as node CPU", func() {
				cpuVmi.Spec.Domain.CPU = &v1.CPU{
					Model: "host-passthrough",
				}

				By("Starting a VirtualMachineInstance")
				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(cpuVmi)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStart(cpuVmi)

				By("Expecting the VirtualMachineInstance console")
				expecter, err := tests.LoggedInCirrosExpecter(cpuVmi)
				Expect(err).ToNot(HaveOccurred())
				defer expecter.Close()

				By("Checking the CPU model under the guest OS")
				_, err = expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: fmt.Sprintf("grep %s /proc/cpuinfo\n", cpuModelName)},
					&expect.BExp{R: "model name"},
				}, 10*time.Second)
			})
		})

		Context("[rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]when CPU model not defined", func() {
			It("[test_id:1680]should report CPU model from libvirt capabilities", func() {
				By("Starting a VirtualMachineInstance")
				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(cpuVmi)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStart(cpuVmi)

				By("Expecting the VirtualMachineInstance console")
				expecter, err := tests.LoggedInCirrosExpecter(cpuVmi)
				Expect(err).ToNot(HaveOccurred())
				defer expecter.Close()

				By("Checking the CPU model under the guest OS")
				_, err = expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: fmt.Sprintf("grep %s /proc/cpuinfo\n", libvirtCpuModel)},
					&expect.BExp{R: "model name"},
				}, 10*time.Second)
			})
		})

		Context("when CPU features defined", func() {
			It("should start a Virtaul Machine with matching features", func() {
				cpuVmi.Spec.Domain.CPU = &v1.CPU{
					Features: []v1.CPUFeature{
						{
							Name: cpuFeatures[0],
						},
					},
				}

				By("Starting a VirtualMachineInstance")
				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(cpuVmi)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStart(cpuVmi)

				By("Expecting the VirtualMachineInstance console")
				expecter, err := tests.LoggedInCirrosExpecter(cpuVmi)
				Expect(err).ToNot(HaveOccurred())
				defer expecter.Close()

				By("Checking the CPU features under the guest OS")
				_, err = expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: fmt.Sprintf("grep %s /proc/cpuinfo\n", cpuFeatures[0])},
					&expect.BExp{R: "flags"},
				}, 10*time.Second)
				Expect(err).ToNot(HaveOccurred())

			})
		})
	})

	Context("with machine type settings", func() {
		defaultMachineTypeKey := "machine-type"

		AfterEach(func() {
			cfgMap, err := virtClient.CoreV1().ConfigMaps(tests.KubeVirtInstallNamespace).Get(kubevirtConfig, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			cfgMap.Data[defaultMachineTypeKey] = ""

			_, err = virtClient.CoreV1().ConfigMaps(tests.KubeVirtInstallNamespace).Update(cfgMap)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should set machine type from VMI spec", func() {
			vmi := tests.NewRandomVMI()
			vmi.Spec.Domain.Machine.Type = "pc-q35-3.0"
			tests.RunVMIAndExpectLaunch(vmi, 30)
			runningVMISpec, err := tests.GetRunningVMISpec(vmi)

			Expect(err).ToNot(HaveOccurred())
			Expect(runningVMISpec.OS.Type.Machine).To(Equal("pc-q35-3.0"))
		})

		It("should set default machine type when it is not provided", func() {
			vmi := tests.NewRandomVMI()
			vmi.Spec.Domain.Machine.Type = ""
			tests.RunVMIAndExpectLaunch(vmi, 30)
			runningVMISpec, err := tests.GetRunningVMISpec(vmi)

			Expect(err).ToNot(HaveOccurred())
			Expect(runningVMISpec.OS.Type.Machine).To(ContainSubstring("q35"))
		})

		It("should set machine type from kubevirt-config", func() {
			cfgMap, err := virtClient.CoreV1().ConfigMaps(tests.KubeVirtInstallNamespace).Get(kubevirtConfig, metav1.GetOptions{})
			Expect(err).To(BeNil())
			cfgMap.Data[defaultMachineTypeKey] = "pc-q35-3.0"
			_, err = virtClient.CoreV1().ConfigMaps(tests.KubeVirtInstallNamespace).Update(cfgMap)
			Expect(err).ToNot(HaveOccurred())

			vmi := tests.NewRandomVMI()
			vmi.Spec.Domain.Machine.Type = ""
			tests.RunVMIAndExpectLaunch(vmi, 30)
			runningVMISpec, err := tests.GetRunningVMISpec(vmi)

			Expect(err).ToNot(HaveOccurred())
			Expect(runningVMISpec.OS.Type.Machine).To(Equal("pc-q35-3.0"))
		})
	})

	Context("with CPU request settings", func() {
		defaultCPURequestKey := "cpu-request"

		AfterEach(func() {
			cfgMap, err := virtClient.CoreV1().ConfigMaps(tests.KubeVirtInstallNamespace).Get(kubevirtConfig, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			cfgMap.Data[defaultCPURequestKey] = ""

			_, err = virtClient.CoreV1().ConfigMaps(tests.KubeVirtInstallNamespace).Update(cfgMap)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should set CPU request from VMI spec", func() {
			vmi := tests.NewRandomVMI()
			vmi.Spec.Domain.Resources.Requests[kubev1.ResourceCPU] = resource.MustParse("500m")
			runningVMI := tests.RunVMIAndExpectScheduling(vmi, 30)

			readyPod := tests.GetPodByVirtualMachineInstance(runningVMI, tests.NamespaceTestDefault)
			computeContainer := getComputeContainerOfPod(readyPod)
			cpuRequest := computeContainer.Resources.Requests[kubev1.ResourceCPU]
			Expect(cpuRequest.String()).To(Equal("500m"))
		})

		It("should set CPU request when it is not provided", func() {
			vmi := tests.NewRandomVMI()
			vmi.Spec.Domain.Resources = v1.ResourceRequirements{
				Requests: kubev1.ResourceList{
					kubev1.ResourceMemory: resource.MustParse("64M"),
				},
			}
			runningVMI := tests.RunVMIAndExpectScheduling(vmi, 30)

			readyPod := tests.GetPodByVirtualMachineInstance(runningVMI, tests.NamespaceTestDefault)
			computeContainer := getComputeContainerOfPod(readyPod)
			cpuRequest := computeContainer.Resources.Requests[kubev1.ResourceCPU]
			Expect(cpuRequest.String()).To(Equal("100m"))
		})

		It("should set CPU request from kubevirt-config", func() {
			cfgMap, err := virtClient.CoreV1().ConfigMaps(tests.KubeVirtInstallNamespace).Get(kubevirtConfig, metav1.GetOptions{})
			Expect(err).To(BeNil())
			cfgMap.Data[defaultCPURequestKey] = "800m"
			_, err = virtClient.CoreV1().ConfigMaps(tests.KubeVirtInstallNamespace).Update(cfgMap)
			Expect(err).ToNot(HaveOccurred())

			vmi := tests.NewRandomVMI()
			vmi.Spec.Domain.Resources = v1.ResourceRequirements{
				Requests: kubev1.ResourceList{
					kubev1.ResourceMemory: resource.MustParse("64M"),
				},
			}
			runningVMI := tests.RunVMIAndExpectScheduling(vmi, 30)

			readyPod := tests.GetPodByVirtualMachineInstance(runningVMI, tests.NamespaceTestDefault)
			computeContainer := getComputeContainerOfPod(readyPod)
			cpuRequest := computeContainer.Resources.Requests[kubev1.ResourceCPU]
			Expect(cpuRequest.String()).To(Equal("800m"))
		})
	})

	Context("[rfe_id:904][crit:medium][vendor:cnv-qe@redhat.com][level:component]with driver cache settings", func() {
		blockPVName := "block-pv-" + rand.String(48)

		BeforeEach(func() {
			// create a new PV and PVC (PVs can't be reused)
			tests.CreateBlockVolumePvAndPvc(blockPVName, "1Gi")
		}, 60)

		AfterEach(func() {
			tests.DeletePvAndPvc(blockPVName)
		}, 60)

		It("[test_id:1681]should set appropriate cache modes", func() {
			vmi := tests.NewRandomVMI()
			vmi.Spec.Domain.Resources.Requests[kubev1.ResourceMemory] = resource.MustParse("64M")

			By("adding disks to a VMI")
			tests.AddEphemeralDisk(vmi, "ephemeral-disk1", "virtio", tests.ContainerDiskFor(tests.ContainerDiskCirros))
			vmi.Spec.Domain.Devices.Disks[0].Cache = v1.CacheNone

			tests.AddEphemeralDisk(vmi, "ephemeral-disk2", "virtio", tests.ContainerDiskFor(tests.ContainerDiskCirros))
			vmi.Spec.Domain.Devices.Disks[1].Cache = v1.CacheWriteThrough

			tests.AddEphemeralDisk(vmi, "ephemeral-disk3", "virtio", tests.ContainerDiskFor(tests.ContainerDiskCirros))
			tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\necho 'hello'\n")
			tests.AddPVCDisk(vmi, "hostpath-pvc", "virtio", tests.DiskAlpineHostPath)
			tests.AddPVCDisk(vmi, "block-pvc", "virtio", blockPVName)
			tests.AddHostDisk(vmi, "/run/kubevirt-private/vm-disks/test-disk.img", v1.HostDiskExistsOrCreate, "hostdisk")
			tests.RunVMIAndExpectLaunch(vmi, 60)

			runningVMISpec, err := tests.GetRunningVMISpec(vmi)
			Expect(err).ToNot(HaveOccurred())

			disks := runningVMISpec.Devices.Disks
			By("checking if number of attached disks is equal to real disks number")
			Expect(len(vmi.Spec.Domain.Devices.Disks)).To(Equal(len(disks)))

			cacheNone := string(v1.CacheNone)
			cacheWritethrough := string(v1.CacheWriteThrough)

			By("checking if requested cache 'none' has been set")
			Expect(disks[0].Alias.Name).To(Equal("ephemeral-disk1"))
			Expect(disks[0].Driver.Cache).To(Equal(cacheNone))

			By("checking if requested cache 'writethrough' has been set")
			Expect(disks[1].Alias.Name).To(Equal("ephemeral-disk2"))
			Expect(disks[1].Driver.Cache).To(Equal(cacheWritethrough))

			By("checking if default cache 'none' has been set to ephemeral disk")
			Expect(disks[2].Alias.Name).To(Equal("ephemeral-disk3"))
			Expect(disks[2].Driver.Cache).To(Equal(cacheNone))

			By("checking if default cache 'none' has been set to cloud-init disk")
			Expect(disks[3].Alias.Name).To(Equal("cloud-init"))
			Expect(disks[3].Driver.Cache).To(Equal(cacheNone))

			By("checking if default cache 'none' has been set to pvc disk")
			Expect(disks[4].Alias.Name).To(Equal("hostpath-pvc"))
			Expect(disks[4].Driver.Cache).To(Equal(cacheNone))

			By("checking if default cache 'none' has been set to block pvc")
			Expect(disks[5].Alias.Name).To(Equal("block-pvc"))
			Expect(disks[5].Driver.Cache).To(Equal(cacheNone))

			By("checking if default cache 'writethrough' has been set to fs which does not support direct I/O")
			Expect(disks[6].Alias.Name).To(Equal("hostdisk"))
			Expect(disks[6].Driver.Cache).To(Equal(cacheWritethrough))
		})
	})

	Context("[rfe_id:898][crit:medium][vendor:cnv-qe@redhat.com][level:component]New VirtualMachineInstance with all supported drives", func() {

		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			// ordering:
			// use a small disk for the other ones
			containerImage := tests.ContainerDiskFor(tests.ContainerDiskCirros)
			// virtio - added by NewRandomVMIWithEphemeralDisk
			vmi = tests.NewRandomVMIWithEphemeralDiskAndUserdata(containerImage, "echo hi!\n")
			// sata
			tests.AddEphemeralDisk(vmi, "disk2", "sata", containerImage)
			// NOTE: we have one disk per bus, so we expect vda, sda
		})
		checkPciAddress := func(vmi *v1.VirtualMachineInstance, expectedPciAddress string, prompt string) {
			err := tests.CheckForTextExpecter(vmi, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: prompt},
				&expect.BSnd{S: "grep DEVNAME /sys/bus/pci/devices/" + expectedPciAddress + "/*/block/vda/uevent|awk -F= '{ print $2 }'\n"},
				&expect.BExp{R: "vda"},
			}, 15)
			Expect(err).ToNot(HaveOccurred())
		}

		It("[test_id:1682]should have all the device nodes", func() {
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

		It("[test_id:1683]should configure custom Pci address", func() {
			By("checking disk1 Pci address")
			vmi.Spec.Domain.Devices.Disks[0].Disk.PciAddress = "0000:00:10.0"
			vmi.Spec.Domain.Devices.Disks[0].Disk.Bus = "virtio"
			vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitUntilVMIReady(vmi, tests.LoggedInCirrosExpecter)

			checkPciAddress(vmi, vmi.Spec.Domain.Devices.Disks[0].Disk.PciAddress, "\\$")
		})
	})
	Describe("[rfe_id:897][crit:medium][vendor:cnv-qe@redhat.com][level:component]VirtualMachineInstance with CPU pinning", func() {
		var nodes *kubev1.NodeList

		isNodeHasCPUManagerLabel := func(nodeName string) bool {
			Expect(nodeName).ToNot(BeEmpty())

			nodeObject, err := virtClient.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			nodeHaveCpuManagerLabel := false
			nodeLabels := nodeObject.GetLabels()

			for label, val := range nodeLabels {
				if label == v1.CPUManager && val == "true" {
					nodeHaveCpuManagerLabel = true
					break
				}
			}
			return nodeHaveCpuManagerLabel
		}

		BeforeEach(func() {
			nodes, err = virtClient.CoreV1().Nodes().List(metav1.ListOptions{})
			tests.PanicOnError(err)
			if len(nodes.Items) == 1 {
				Skip("Skip cpu pinning test that requires multiple nodes when only one node is present.")
			}
		})

		Context("with cpu pinning enabled", func() {
			It("[test_id:1684]should set the cpumanager label to false when it's not running", func() {

				By("adding a cpumanger=true lable to a node")
				nodes, err := virtClient.CoreV1().Nodes().List(metav1.ListOptions{LabelSelector: v1.CPUManager + "=" + "false"})
				Expect(err).ToNot(HaveOccurred())
				Expect(nodes.Items).To(HaveLen(1))

				node := &nodes.Items[0]
				node, err = virtClient.CoreV1().Nodes().Patch(node.Name, types.StrategicMergePatchType, []byte(fmt.Sprintf(`{"metadata": { "labels": {"%s": "true"}}}`, v1.CPUManager)))
				Expect(err).ToNot(HaveOccurred())

				By("setting the cpumanager label back to false")
				Eventually(func() string {
					n, err := virtClient.CoreV1().Nodes().Get(node.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return n.Labels[v1.CPUManager]
				}, 2*time.Minute, 2*time.Second).Should(Equal("false"))
			})
			It("[test_id:1685]non master node should have a cpumanager label", func() {
				cpuManagerEnabled := false
				for idx := 1; idx < len(nodes.Items); idx++ {
					labels := nodes.Items[idx].GetLabels()
					for label, val := range labels {
						if label == "cpumanager" && val == "true" {
							cpuManagerEnabled = true
						}
					}
				}
				Expect(cpuManagerEnabled).To(BeTrue())
			})
			It("[test_id:1686]should be scheduled on a node with running cpu manager", func() {
				cpuVmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				cpuVmi.Spec.Domain.CPU = &v1.CPU{
					Cores:                 2,
					DedicatedCPUPlacement: true,
				}
				cpuVmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: kubev1.ResourceList{
						kubev1.ResourceMemory: resource.MustParse("64M"),
					},
				}

				By("Starting a VirtualMachineInstance")
				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(cpuVmi)
				Expect(err).ToNot(HaveOccurred())
				node := tests.WaitForSuccessfulVMIStart(cpuVmi)

				Expect(isNodeHasCPUManagerLabel(node)).To(BeTrue())

				By("Checking that the pod QOS is guaranteed")
				readyPod := tests.GetRunningPodByVirtualMachineInstance(cpuVmi, tests.NamespaceTestDefault)
				podQos := readyPod.Status.QOSClass
				Expect(podQos).To(Equal(kubev1.PodQOSGuaranteed))

				var computeContainer *kubev1.Container
				for _, container := range readyPod.Spec.Containers {
					if container.Name == "compute" {
						computeContainer = &container
					}
				}
				if computeContainer == nil {
					tests.PanicOnError(fmt.Errorf("could not find the compute container"))
				}

				output, err := tests.ExecuteCommandOnPod(
					virtClient,
					readyPod,
					"compute",
					[]string{"cat", hw_utils.CPUSET_PATH},
				)
				log.Log.Infof("%v", output)
				Expect(err).ToNot(HaveOccurred())
				output = strings.TrimSuffix(output, "\n")
				pinnedCPUsList, err := hw_utils.ParseCPUSetLine(output)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(pinnedCPUsList)).To(Equal(int(cpuVmi.Spec.Domain.CPU.Cores)))

				By("Expecting the VirtualMachineInstance console")
				expecter, err := tests.LoggedInCirrosExpecter(cpuVmi)
				Expect(err).ToNot(HaveOccurred())
				defer expecter.Close()

				By("Checking the number of CPU cores under guest OS")
				res, err := expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: "grep -c ^processor /proc/cpuinfo\n"},
					&expect.BExp{R: "2"},
				}, 15*time.Second)
				log.DefaultLogger().Object(cpuVmi).Infof("%v", res)
				Expect(err).ToNot(HaveOccurred())
			})

			It("[test_id:1687]should configure correct number of vcpus with requests.cpus", func() {
				cpuVmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				cpuVmi.Spec.Domain.CPU = &v1.CPU{
					DedicatedCPUPlacement: true,
				}
				cpuVmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: kubev1.ResourceList{
						kubev1.ResourceCPU:    resource.MustParse("2"),
						kubev1.ResourceMemory: resource.MustParse("64M"),
					},
				}
				By("Starting a VirtualMachineInstance")
				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(cpuVmi)
				Expect(err).ToNot(HaveOccurred())
				node := tests.WaitForSuccessfulVMIStart(cpuVmi)
				Expect(isNodeHasCPUManagerLabel(node)).To(BeTrue())

				By("Expecting the VirtualMachineInstance console")
				expecter, err := tests.LoggedInCirrosExpecter(cpuVmi)
				Expect(err).ToNot(HaveOccurred())
				defer expecter.Close()

				By("Checking the number of CPU cores under guest OS")
				res, err := expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: "grep -c ^processor /proc/cpuinfo\n"},
					&expect.BExp{R: "2"},
				}, 15*time.Second)
				log.DefaultLogger().Object(cpuVmi).Infof("%v", res)
				Expect(err).ToNot(HaveOccurred())
			})

			It("[test_id:1688]should fail the vmi creation if the requested resources are inconsistent", func() {
				cpuVmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				cpuVmi.Spec.Domain.CPU = &v1.CPU{
					Cores:                 2,
					DedicatedCPUPlacement: true,
				}
				cpuVmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: kubev1.ResourceList{
						kubev1.ResourceCPU:    resource.MustParse("3"),
						kubev1.ResourceMemory: resource.MustParse("64M"),
					},
				}
				By("Starting a VirtualMachineInstance")
				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(cpuVmi)
				Expect(err).To(HaveOccurred())
			})
			It("[test_id:1689]should fail the vmi creation if cpu is not an integer", func() {
				cpuVmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				cpuVmi.Spec.Domain.CPU = &v1.CPU{
					DedicatedCPUPlacement: true,
				}
				cpuVmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: kubev1.ResourceList{
						kubev1.ResourceCPU:    resource.MustParse("300m"),
						kubev1.ResourceMemory: resource.MustParse("64M"),
					},
				}
				By("Starting a VirtualMachineInstance")
				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(cpuVmi)
				Expect(err).To(HaveOccurred())
			})
			It("[test_id:1690]should fail the vmi creation if Guaranteed QOS cannot be set", func() {
				cpuVmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				cpuVmi.Spec.Domain.CPU = &v1.CPU{
					DedicatedCPUPlacement: true,
				}
				cpuVmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: kubev1.ResourceList{
						kubev1.ResourceCPU:    resource.MustParse("2"),
						kubev1.ResourceMemory: resource.MustParse("64M"),
					},
					Limits: kubev1.ResourceList{
						kubev1.ResourceCPU: resource.MustParse("4"),
					},
				}
				By("Starting a VirtualMachineInstance")
				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(cpuVmi)
				Expect(err).To(HaveOccurred())
			})
			It("[test_id:1691]should start a vm with no cpu pinning after a vm with cpu pinning on same node", func() {
				Vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				cpuVmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				cpuVmi.Spec.Domain.CPU = &v1.CPU{
					DedicatedCPUPlacement: true,
				}
				cpuVmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: kubev1.ResourceList{
						kubev1.ResourceCPU:    resource.MustParse("2"),
						kubev1.ResourceMemory: resource.MustParse("64M"),
					},
				}
				Vmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: kubev1.ResourceList{
						kubev1.ResourceCPU: resource.MustParse("1"),
					},
				}
				Vmi.Spec.NodeSelector = map[string]string{v1.CPUManager: "true"}

				By("Starting a VirtualMachineInstance with dedicated cpus")
				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(cpuVmi)
				Expect(err).ToNot(HaveOccurred())
				node := tests.WaitForSuccessfulVMIStart(cpuVmi)
				Expect(isNodeHasCPUManagerLabel(node)).To(BeTrue())

				By("Starting a VirtualMachineInstance without dedicated cpus")
				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(Vmi)
				Expect(err).ToNot(HaveOccurred())
				node = tests.WaitForSuccessfulVMIStart(cpuVmi)
				Expect(isNodeHasCPUManagerLabel(node)).To(BeTrue())
			})
		})
	})
})
