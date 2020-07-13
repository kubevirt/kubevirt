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
	"encoding/json"
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

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	kubevirt_hooks_v1alpha2 "kubevirt.io/kubevirt/pkg/hooks/v1alpha2"
	hw_utils "kubevirt.io/kubevirt/pkg/util/hardware"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Configurations", func() {

	tests.FlagParse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	Context("with all devices on the root PCI bus", func() {
		It("should start run the guest as usual", func() {
			vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
			vmi.Annotations = map[string]string{
				v1.PlacePCIDevicesOnRootComplex: "true",
			}
			vmi.Spec.Domain.Devices.Rng = &v1.Rng{}
			vmi.Spec.Domain.Devices.Inputs = []v1.Input{{Name: "tablet", Bus: "virtio", Type: "tablet"}, {Name: "tablet1", Bus: "usb", Type: "tablet"}}
			vmi.Spec.Domain.Devices.Watchdog = &v1.Watchdog{Name: "watchdog", WatchdogDevice: v1.WatchdogDevice{I6300ESB: &v1.I6300ESBWatchdog{Action: v1.WatchdogActionPoweroff}}}
			vmi = tests.RunVMIAndExpectLaunch(vmi, 60)
			expecter, err := tests.LoggedInCirrosExpecter(vmi)
			Expect(err).ToNot(HaveOccurred())
			defer expecter.Close()
			domSpec, err := tests.GetRunningVMIDomainSpec(vmi)
			rootPortController := []api.Controller{}
			for _, c := range domSpec.Devices.Controllers {
				if c.Model == "pcie-root-port" {
					rootPortController = append(rootPortController, c)
				}
			}
			Expect(rootPortController).To(HaveLen(0), "libvirt should not add additional buses to the root one")
		})

	})

	Context("[rfe_id:897][crit:medium][vendor:cnv-qe@redhat.com][level:component]for CPU and memory limits should", func() {

		It("[test_id:3110]lead to get the burstable QOS class assigned when limit and requests differ", func() {
			vmi := tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))
			vmi = tests.RunVMIAndExpectScheduling(vmi, 60)

			Eventually(func() kubev1.PodQOSClass {
				vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.IsFinal()).To(BeFalse())
				if vmi.Status.QOSClass == nil {
					return ""
				}
				return *vmi.Status.QOSClass
			}, 10*time.Second, 1*time.Second).Should(Equal(kubev1.PodQOSBurstable))
		})

		It("[test_id:3111]lead to get the guaranteed QOS class assigned when limit and requests are identical", func() {
			vmi := tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))
			By("specifying identical limits and requests")
			vmi.Spec.Domain.Resources = v1.ResourceRequirements{
				Requests: kubev1.ResourceList{
					kubev1.ResourceCPU:    resource.MustParse("1"),
					kubev1.ResourceMemory: resource.MustParse("64M"),
				},
				Limits: kubev1.ResourceList{
					kubev1.ResourceCPU:    resource.MustParse("1"),
					kubev1.ResourceMemory: resource.MustParse("64M"),
				},
			}

			By("adding a sidecar to ensure it gets limits assigned too")
			vmi.ObjectMeta.Annotations = RenderSidecar(kubevirt_hooks_v1alpha2.Version)
			vmi = tests.RunVMIAndExpectScheduling(vmi, 60)

			Eventually(func() kubev1.PodQOSClass {
				vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.IsFinal()).To(BeFalse())
				if vmi.Status.QOSClass == nil {
					return ""
				}
				return *vmi.Status.QOSClass
			}, 10*time.Second, 1*time.Second).Should(Equal(kubev1.PodQOSGuaranteed))
		})

		It("[test_id:3112]lead to get the guaranteed QOS class assigned when only limits are set", func() {
			vmi := tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))
			By("specifying identical limits and requests")
			vmi.Spec.Domain.Resources = v1.ResourceRequirements{
				Requests: kubev1.ResourceList{},
				Limits: kubev1.ResourceList{
					kubev1.ResourceCPU:    resource.MustParse("1"),
					kubev1.ResourceMemory: resource.MustParse("64M"),
				},
			}

			By("adding a sidecar to ensure it gets limits assigned too")
			vmi.ObjectMeta.Annotations = RenderSidecar(kubevirt_hooks_v1alpha2.Version)
			vmi = tests.RunVMIAndExpectScheduling(vmi, 60)

			Eventually(func() kubev1.PodQOSClass {
				vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.IsFinal()).To(BeFalse())
				if vmi.Status.QOSClass == nil {
					return ""
				}
				return *vmi.Status.QOSClass
			}, 10*time.Second, 1*time.Second).Should(Equal(kubev1.PodQOSGuaranteed))

			vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Spec.Domain.Resources.Requests.Cpu().Cmp(*vmi.Spec.Domain.Resources.Limits.Cpu())).To(BeZero())
			Expect(vmi.Spec.Domain.Resources.Requests.Memory().Cmp(*vmi.Spec.Domain.Resources.Limits.Memory())).To(BeZero())
		})

	})

	Describe("VirtualMachineInstance definition", func() {
		Context("[rfe_id:2065][crit:medium][vendor:cnv-qe@redhat.com][level:component]with 3 CPU cores", func() {
			availableNumberOfCPUs := tests.GetHighestCPUNumberAmongNodes(virtClient)

			var vmi *v1.VirtualMachineInstance

			BeforeEach(func() {
				requiredNumberOfCpus := 3
				Expect(availableNumberOfCPUs).ToNot(BeNumerically("<", requiredNumberOfCpus),
					fmt.Sprintf("Test requires %d cpus, but only %d available!", requiredNumberOfCpus, availableNumberOfCPUs))
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
				_false := false
				vmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: kubev1.ResourceList{
						kubev1.ResourceMemory: resource.MustParse("64M"),
						kubev1.ResourceCPU:    resource.MustParse("3"),
					},
				}

				vmi.Spec.Domain.Devices.NetworkInterfaceMultiQueue = &_true
				vmi.Spec.Domain.Devices.BlockMultiQueue = &_false

				By("Starting a VirtualMachineInstance")
				vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStart(vmi)

				domXml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(domXml).To(ContainSubstring("driver name='vhost' queues='3'"))
				// make sure that there are not block queues configured
				Expect(domXml).ToNot(ContainSubstring("cache='none' queues='3'"))
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

		Context("[rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]with no memory requested", func() {
			It("[test_id:3113]should failed to the VMI creation", func() {
				vmi := tests.NewRandomVMI()
				vmi.Spec.Domain.Resources = v1.ResourceRequirements{}
				By("Starting a VirtualMachineInstance")
				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("[rfe_id:609][crit:medium][vendor:cnv-qe@redhat.com][level:component]with cluster memory overcommit being applied", func() {
			BeforeEach(func() {
				tests.UpdateClusterConfigValueAndWait("memory-overcommit", "200")
			})

			It("[test_id:3114]should set requested amount of memory according to the specified virtual memory", func() {
				vmi := tests.NewRandomVMI()
				guestMemory := resource.MustParse("4096M")
				vmi.Spec.Domain.Memory = &v1.Memory{Guest: &guestMemory}
				vmi.Spec.Domain.Resources = v1.ResourceRequirements{}
				runningVMI := tests.RunVMI(vmi, 30)
				Expect(runningVMI.Spec.Domain.Resources.Requests.Memory().String()).To(Equal("2048M"))
			})
		})

		Context("[rfe_id:2262][crit:medium][vendor:cnv-qe@redhat.com][level:component]with EFI bootloader method", func() {

			It("[test_id:1668]should use EFI", func() {
				vmi := tests.NewRandomVMIWithEFIBootloader()

				By("Starting a VirtualMachineInstance")
				vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitUntilVMIReady(vmi, tests.LoggedInAlpineExpecter)

				By("Checking if UEFI is enabled")
				domXml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(domXml).To(ContainSubstring("OVMF_CODE.fd"))
			})

			It("[test_id:4437]should enable EFI secure boot", func() {
				vmi := tests.NewRandomVMIWithSecureBoot()

				By("Starting a VirtualMachineInstance")
				vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())

				By("Checking if SecureBoot is enabled in Linux")
				tests.WaitUntilVMIReady(vmi, tests.SecureBootExpecter)

				By("Checking if SecureBoot is enabled in the libvirt XML")
				domXml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(domXml).To(ContainSubstring("OVMF_CODE.secboot.fd"))
			})
		})

		Context("[rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]with diverging guest memory from requested memory", func() {
			It("[test_id:1669]should show the requested guest memory inside the VMI", func() {
				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				vmi.Spec.Domain.Resources.Requests[kubev1.ResourceMemory] = resource.MustParse("64M")
				guestMemory := resource.MustParse("128M")
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

		Context("[rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]with diverging memory limit from memory request and no guest memory", func() {
			It("[test_id:3115]should show the memory limit inside the VMI", func() {
				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				vmi.Spec.Domain.Resources.Requests[kubev1.ResourceMemory] = resource.MustParse("64M")
				vmi.Spec.Domain.Resources.Limits = kubev1.ResourceList{
					kubev1.ResourceMemory: resource.MustParse("128M"),
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

		Context("[rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]with support memory over commitment", func() {
			It("[test_id:755]should show the requested memory different than guest memory", func() {
				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				guestMemory := resource.MustParse("256Mi")
				vmi.Spec.Domain.Resources.Requests[kubev1.ResourceMemory] = resource.MustParse("64Mi")
				vmi.Spec.Domain.Resources.OvercommitGuestOverhead = true
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
					&expect.BSnd{S: "[ $(free -m | grep Mem: | tr -s ' ' | cut -d' ' -f2) -gt 200 ] && echo 'pass' || echo 'fail'\n"},
					&expect.BExp{R: "pass"},
					&expect.BSnd{S: "swapoff -a && dd if=/dev/zero of=/dev/shm/test bs=1k count=118k; echo $?\n"},
					&expect.BExp{R: "0"},
				}, 15*time.Second)
				log.DefaultLogger().Object(vmi).Infof("%v", res)
				Expect(err).ToNot(HaveOccurred())

				pod := tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)
				podMemoryUsage, err := tests.ExecuteCommandOnPod(
					virtClient,
					pod,
					"compute",
					[]string{"/usr/bin/bash", "-c", "cat /sys/fs/cgroup/memory/memory.usage_in_bytes"},
				)
				Expect(err).ToNot(HaveOccurred())
				By("Converting pod memory usage")
				m, err := strconv.Atoi(strings.Trim(podMemoryUsage, "\n"))
				Expect(err).ToNot(HaveOccurred())
				By("Checking if pod memory usage is > 64Mi")
				Expect(m > 67108864).To(BeTrue(), "67108864 B = 64 Mi")
			})

		})

		Context("[rfe_id:609][crit:medium][vendor:cnv-qe@redhat.com][level:component]Support memory over commitment test", func() {
			vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
			vmi.Spec.Domain.Resources.Requests[kubev1.ResourceMemory] = resource.MustParse("128M")
			vmi.Spec.Domain.Resources.OvercommitGuestOverhead = true

			// Create VMI inside BeforeEach because it gets cleanedup, see BeforeEach at the top
			BeforeEach(func() {
				vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStart(vmi)
			})

			It("[test_id:730]Check OverCommit VM Created and Started", func() {
				overcommitVmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStart(overcommitVmi)
			})
			It("[test_id:731]Check OverCommit status on VMI", func() {
				overcommitVmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(overcommitVmi.Spec.Domain.Resources.OvercommitGuestOverhead).To(BeTrue())
			})
			It("[test_id:732]Check Free memory on the VMI", func() {
				By("Expecting console")
				expecter, err := tests.LoggedInCirrosExpecter(vmi)
				Expect(err).ToNot(HaveOccurred())
				defer expecter.Close()

				// Check on the VM, if the Free memory is roughly what we expected
				res, err := expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: "[ $(free -m | grep Mem: | tr -s ' ' | cut -d' ' -f2) -gt 95 ] && echo 'pass' || echo 'fail'\n"},
					&expect.BExp{R: "pass"},
				}, 15*time.Second)
				log.DefaultLogger().Object(vmi).Infof("%v", res)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("[rfe_id:3078][crit:medium][vendor:cnv-qe@redhat.com][level:component]with usb controller", func() {
			It("[test_id:3117]should start the VMI with usb controller when usb device is present", func() {
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

			It("[test_id:3117]should start the VMI with usb controller when input device doesn't have bus", func() {
				vmi := tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))
				vmi.Spec.Domain.Devices.Inputs = []v1.Input{
					{
						Name: "tablet0",
						Type: "tablet",
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

			It("[test_id:3118]should start the VMI without usb controller", func() {
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

		Context("[rfe_id:3077][crit:medium][vendor:cnv-qe@redhat.com][level:component]with input devices", func() {
			It("[test_id:2642]should failed to start the VMI with wrong type of input device", func() {
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

			It("[test_id:3074]should failed to start the VMI with wrong bus of input device", func() {
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

			It("[test_id:3072]should start the VMI with tablet input device with virtio bus", func() {
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

			It("[test_id:3073]should start the VMI with tablet input device with usb bus", func() {
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
				_, err := virtClient.CoreV1().LimitRanges(tests.NamespaceTestDefault).Create(&limitRangeObj)
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

		Context("[rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]with namespace cpu limits lower than VMI required cpu", func() {
			var vmi *v1.VirtualMachineInstance
			It("[test_id:3119]should fail to start the VMI", func() {
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
				_, err := virtClient.CoreV1().LimitRanges(tests.NamespaceTestDefault).Create(&limitRangeObj)
				Expect(err).ToNot(HaveOccurred())

				By("Starting a VirtualMachineInstance")
				// Retrying up to 5 sec, then if you still succeeds in VMI creation, things must be going wrong.
				Eventually(func() error {
					vmi = tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))
					vmi.Spec.Domain.Resources = v1.ResourceRequirements{
						Requests: kubev1.ResourceList{
							kubev1.ResourceCPU:    resource.MustParse("800m"),
							kubev1.ResourceMemory: resource.MustParse("512M"),
						},
					}
					vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
					virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Delete(vmi.Name, &metav1.DeleteOptions{})
					return err
				}, 5*time.Second, 1*time.Second).Should(MatchError("admission webhook \"virtualmachineinstances-create-validator.kubevirt.io\" denied the request: spec.domain.resources.requests.cpu '800m' is greater than spec.domain.resources.limits.cpu '500m'"))
			})
		})

		Context("[rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]with namespace limits higher than VMI requests", func() {
			var vmi *v1.VirtualMachineInstance
			It("[test_id:3120]should start the VMI with the right default settings from namespace limits", func() {
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
				_, err := virtClient.CoreV1().LimitRanges(tests.NamespaceTestDefault).Create(&limitRangeObj)
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

		Context("[rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]with hugepages", func() {
			var hugepagesVmi *v1.VirtualMachineInstance

			verifyHugepagesConsumption := func() {
				// TODO: we need to check hugepages state via node allocated resources, but currently it has the issue
				// https://github.com/kubernetes/kubernetes/issues/64691
				pods, err := virtClient.CoreV1().Pods(tests.NamespaceTestDefault).List(tests.UnfinishedVMIPodSelector(hugepagesVmi))
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
				if hugepagesVmi.Spec.Domain.Memory != nil && hugepagesVmi.Spec.Domain.Memory.Guest != nil {
					vmMemory = *hugepagesVmi.Spec.Domain.Memory.Guest
				}
				Expect(vmHugepagesConsumption).To(Equal(vmMemory.Value()))
			}

			BeforeEach(func() {
				hugepagesVmi = tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
			})

			table.DescribeTable("should consume hugepages ", func(hugepageSize string, memory string, guestMemory string) {
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
				if guestMemory != "None" {
					guestMemReq := resource.MustParse(guestMemory)
					hugepagesVmi.Spec.Domain.Memory.Guest = &guestMemReq
				}

				By("Starting a VM")
				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(hugepagesVmi)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStart(hugepagesVmi)

				By("Checking that the VM memory equals to a number of consumed hugepages")
				verifyHugepagesConsumption()
			},
				table.Entry("[test_id:1671]hugepages-2Mi", "2Mi", "64Mi", "None"),
				table.Entry("[test_id:1672]hugepages-1Gi", "1Gi", "1Gi", "None"),
				table.Entry("[test_id:1672]hugepages-1Gi", "2Mi", "70Mi", "64Mi"),
			)

			Context("with unsupported page size", func() {
				It("[test_id:1673]should failed to schedule the pod", func() {
					nodes, err := virtClient.CoreV1().Nodes().List(metav1.ListOptions{})
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

		Context("[rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]with guestAgent", func() {
			var agentVMI *v1.VirtualMachineInstance

			prepareAgentVM := func() *v1.VirtualMachineInstance {
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

				return agentVMI
			}

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
				agentVMI := prepareAgentVM()
				getOptions := metav1.GetOptions{}

				freshVMI, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(agentVMI.Name, &getOptions)
				Expect(err).ToNot(HaveOccurred(), "Should get VMI ")
				Expect(freshVMI.Status.Conditions).To(
					ContainElement(
						MatchFields(
							IgnoreExtras,
							Fields{"Type": Equal(v1.VirtualMachineInstanceAgentConnected)})),
					"agent should already be connected")

			})

			It("should remove condition when agent is off", func() {
				agentVMI := prepareAgentVM()
				getOptions := metav1.GetOptions{}

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
					freshVMI, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(agentVMI.Name, &getOptions)
					Expect(err).ToNot(HaveOccurred(), "Should get VMI ")
					return freshVMI.Status.Conditions
				}, 240*time.Second, 2).ShouldNot(
					ContainElement(
						MatchFields(
							IgnoreExtras,
							Fields{"Type": Equal(v1.VirtualMachineInstanceAgentConnected)})),
					"Agent condition should be gone")
			})

			Context("with cluster config changes", func() {
				supportedGuestAgentKey := "supported-guest-agent"

				BeforeEach(func() {
					tests.UpdateClusterConfigValueAndWait(supportedGuestAgentKey, "X.*")
				})

				It("[test_id:]VMI condition should signal unsupported agent presence", func() {
					agentVMI := prepareAgentVM()
					getOptions := metav1.GetOptions{}

					freshVMI, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(agentVMI.Name, &getOptions)
					Expect(err).ToNot(HaveOccurred(), "Should get VMI ")

					Eventually(func() []v1.VirtualMachineInstanceCondition {
						freshVMI, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(agentVMI.Name, &getOptions)
						Expect(err).ToNot(HaveOccurred(), "Should get VMI ")
						return freshVMI.Status.Conditions
					}, 240*time.Second, 2).Should(
						ContainElement(
							MatchFields(
								IgnoreExtras,
								Fields{"Type": Equal(v1.VirtualMachineInstanceUnsupportedAgent)})),
						"Should have unsupported agent connected condition")

				})
			})

			It("should have guestosinfo in status when agent is present", func() {
				agentVMI := prepareAgentVM()
				getOptions := metav1.GetOptions{}
				var updatedVmi *v1.VirtualMachineInstance
				var err error

				By("Expecting the Guest VM information")
				Eventually(func() bool {
					updatedVmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(agentVMI.Name, &getOptions)
					if err != nil {
						return false
					}
					return updatedVmi.Status.GuestOSInfo.Name != ""
				}, 240*time.Second, 2).Should(BeTrue(), "Should have guest OS Info in vmi status")

				Expect(err).ToNot(HaveOccurred())
				Expect(updatedVmi.Status.GuestOSInfo.Name).To(Equal("Fedora"))
			})

			It("should return the whole data when agent is present", func() {
				agentVMI := prepareAgentVM()

				By("Expecting the Guest VM information")
				Eventually(func() bool {
					guestInfo, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).GuestOsInfo(agentVMI.Name)
					if err != nil {
						// invalid request, retry
						return false
					}
					if guestInfo.Hostname != "" &&
						guestInfo.Timezone != "" &&
						guestInfo.GAVersion != "" &&
						guestInfo.OS.Name != "" &&
						len(guestInfo.FSInfo.Filesystems) > 0 {
						return true
					}
					return false
				}, 240*time.Second, 2).Should(BeTrue(), "Should have guest OS Info in subresource")
			})

			It("should not return the whole data when agent is not present", func() {
				agentVMI := prepareAgentVM()

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

				By("Expecting the Guest VM information")
				Eventually(func() string {
					_, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).GuestOsInfo(agentVMI.Name)
					if err != nil {
						return err.Error()
					}
					return ""
				}, 240*time.Second, 2).Should(ContainSubstring("VMI does not have guest agent connected"), "Should have not have guest info in subresource")
			})

			It("should return user list", func() {
				agentVMI := prepareAgentVM()

				expecter, err := tests.LoggedInFedoraExpecter(agentVMI)
				Expect(err).ToNot(HaveOccurred())
				defer expecter.Close()

				// do some action on VM just to make sure user is active
				expecter.Send("ls")

				By("Expecting the Guest VM information")
				Eventually(func() bool {
					userList, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).UserList(agentVMI.Name)
					if err != nil {
						// invalid request, retry
						return false
					}
					if len(userList.Items) > 0 && userList.Items[0].UserName == "fedora" {
						return true
					}
					return false
				}, 240*time.Second, 2).Should(BeTrue(), "Should have fedora users")
			})

			It("should return filesystem list", func() {
				agentVMI := prepareAgentVM()

				By("Expecting the Guest VM information")
				Eventually(func() bool {
					fsList, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).FilesystemList(agentVMI.Name)
					if err != nil {
						// invalid request, retry
						return false
					}
					if len(fsList.Items) > 0 && fsList.Items[0].DiskName != "" && fsList.Items[0].MountPoint != "" {
						return true
					}
					return false
				}, 240*time.Second, 2).Should(BeTrue(), "Should have some filesystem")
			})

		})

		Context("[rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]with serial-number", func() {
			var snVmi *v1.VirtualMachineInstance

			BeforeEach(func() {
				snVmi = tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))
			})

			It("[test_id:3121]should have serial-number set when present", func() {
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

			It("[test_id:3122]should not have serial-number set when not present", func() {
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
			It("[test_id:3123]should start a Virtaul Machine with matching features", func() {
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

	Context("[rfe_id:2869][crit:medium][vendor:cnv-qe@redhat.com][level:component]with machine type settings", func() {
		defaultMachineTypeKey := "machine-type"
		defaultEmulatedMachineType := "emulated-machines"

		BeforeEach(func() {
			tests.UpdateClusterConfigValueAndWait(defaultEmulatedMachineType, "q35*,pc-q35*,pc*")
		})

		It("[test_id:3124]should set machine type from VMI spec", func() {
			vmi := tests.NewRandomVMI()
			vmi.Spec.Domain.Machine.Type = "pc"
			tests.RunVMIAndExpectLaunch(vmi, 30)
			runningVMISpec, err := tests.GetRunningVMIDomainSpec(vmi)

			Expect(err).ToNot(HaveOccurred())
			Expect(runningVMISpec.OS.Type.Machine).To(ContainSubstring("pc-i440"))
		})

		It("[test_id:3125]should set default machine type when it is not provided", func() {
			vmi := tests.NewRandomVMI()
			vmi.Spec.Domain.Machine.Type = ""
			tests.RunVMIAndExpectLaunch(vmi, 30)
			runningVMISpec, err := tests.GetRunningVMIDomainSpec(vmi)

			Expect(err).ToNot(HaveOccurred())
			Expect(runningVMISpec.OS.Type.Machine).To(ContainSubstring("q35"))
		})

		It("[test_id:3126]should set machine type from kubevirt-config", func() {
			tests.UpdateClusterConfigValueAndWait(defaultMachineTypeKey, "pc")

			vmi := tests.NewRandomVMI()
			vmi.Spec.Domain.Machine.Type = ""
			tests.RunVMIAndExpectLaunch(vmi, 30)
			runningVMISpec, err := tests.GetRunningVMIDomainSpec(vmi)

			Expect(err).ToNot(HaveOccurred())
			Expect(runningVMISpec.OS.Type.Machine).To(ContainSubstring("pc-i440"))
		})
	})

	Context("with a custom scheduler", func() {
		It("schould set the custom scheduler on the pod", func() {
			vmi := tests.NewRandomVMI()
			vmi.Spec.SchedulerName = "my-custom-scheduler"
			runningVMI := tests.RunVMIAndExpectScheduling(vmi, 30)
			launcherPod := tests.GetPodByVirtualMachineInstance(runningVMI, tests.NamespaceTestDefault)
			Expect(launcherPod.Spec.SchedulerName).To(Equal("my-custom-scheduler"))
		})
	})

	Context("[rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]with CPU request settings", func() {
		defaultCPURequestKey := "cpu-request"

		It("[test_id:3127]should set CPU request from VMI spec", func() {
			vmi := tests.NewRandomVMI()
			vmi.Spec.Domain.Resources.Requests[kubev1.ResourceCPU] = resource.MustParse("500m")
			runningVMI := tests.RunVMIAndExpectScheduling(vmi, 30)

			readyPod := tests.GetPodByVirtualMachineInstance(runningVMI, tests.NamespaceTestDefault)
			computeContainer := tests.GetComputeContainerOfPod(readyPod)
			cpuRequest := computeContainer.Resources.Requests[kubev1.ResourceCPU]
			Expect(cpuRequest.String()).To(Equal("500m"))
		})

		It("[test_id:3128]should set CPU request when it is not provided", func() {
			vmi := tests.NewRandomVMI()
			vmi.Spec.Domain.Resources = v1.ResourceRequirements{
				Requests: kubev1.ResourceList{
					kubev1.ResourceMemory: resource.MustParse("64M"),
				},
			}
			runningVMI := tests.RunVMIAndExpectScheduling(vmi, 30)

			readyPod := tests.GetPodByVirtualMachineInstance(runningVMI, tests.NamespaceTestDefault)
			computeContainer := tests.GetComputeContainerOfPod(readyPod)
			cpuRequest := computeContainer.Resources.Requests[kubev1.ResourceCPU]
			Expect(cpuRequest.String()).To(Equal("100m"))
		})

		It("[test_id:3129]should set CPU request from kubevirt-config", func() {
			tests.UpdateClusterConfigValueAndWait(defaultCPURequestKey, "800m")

			vmi := tests.NewRandomVMI()
			vmi.Spec.Domain.Resources = v1.ResourceRequirements{
				Requests: kubev1.ResourceList{
					kubev1.ResourceMemory: resource.MustParse("64M"),
				},
			}
			runningVMI := tests.RunVMIAndExpectScheduling(vmi, 30)

			readyPod := tests.GetPodByVirtualMachineInstance(runningVMI, tests.NamespaceTestDefault)
			computeContainer := tests.GetComputeContainerOfPod(readyPod)
			cpuRequest := computeContainer.Resources.Requests[kubev1.ResourceCPU]
			Expect(cpuRequest.String()).To(Equal("800m"))
		})
	})

	Context("[rfe_id:904][crit:medium][vendor:cnv-qe@redhat.com][level:component]with driver cache settings and PVC", func() {
		var cfgMap *kubev1.ConfigMap
		var originalFeatureGates string

		BeforeEach(func() {
			cfgMap, err = virtClient.CoreV1().ConfigMaps(tests.KubeVirtInstallNamespace).Get(kubevirtConfig, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			originalFeatureGates = cfgMap.Data[virtconfig.FeatureGatesKey]
			tests.EnableFeatureGate(virtconfig.HostDiskGate)
			// create a new PV and PVC (PVs can't be reused)
			tests.CreateBlockVolumePvAndPvc("1Gi")
		}, 60)

		AfterEach(func() {
			tests.UpdateClusterConfigValueAndWait(virtconfig.FeatureGatesKey, originalFeatureGates)
		})

		It("[test_id:1681]should set appropriate cache modes", func() {
			tests.SkipPVCTestIfRunnigOnKindInfra()

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
			tests.AddPVCDisk(vmi, "block-pvc", "virtio", tests.BlockDiskForTest)
			tests.AddHostDisk(vmi, tests.RandTmpDir()+"/test-disk.img", v1.HostDiskExistsOrCreate, "hostdisk")
			tests.RunVMIAndExpectLaunch(vmi, 60)

			runningVMISpec, err := tests.GetRunningVMIDomainSpec(vmi)
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
			// PVC is mounted as tmpfs on kind, which does not support direct I/O.
			// As such, it behaves as plugging in a hostDisk - check disks[6].
			if tests.IsRunningOnKindInfra() {
				Expect(disks[4].Driver.Cache).To(Equal(cacheWritethrough))
			} else {
				Expect(disks[4].Driver.Cache).To(Equal(cacheNone))
			}

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

		It("[test_id:3906]should configure custom Pci address", func() {
			By("checking disk1 Pci address")
			vmi.Spec.Domain.Devices.Disks[0].Disk.PciAddress = "0000:00:10.0"
			vmi.Spec.Domain.Devices.Disks[0].Disk.Bus = "virtio"
			vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitUntilVMIReady(vmi, tests.LoggedInCirrosExpecter)

			checkPciAddress(vmi, vmi.Spec.Domain.Devices.Disks[0].Disk.PciAddress, "\\$")
		})

		It("[test_id:1020]should not create the VM with wrong PCI adress", func() {
			By("setting disk1 Pci address")

			wrongPciAddress := "0000:04:10.0"

			vmi.Spec.Domain.Devices.Disks[0].Disk.PciAddress = wrongPciAddress
			vmi.Spec.Domain.Devices.Disks[0].Disk.Bus = "virtio"
			vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())

			var vmiCondition v1.VirtualMachineInstanceCondition
			Eventually(func() bool {
				vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				if len(vmi.Status.Conditions) > 0 {
					for _, cond := range vmi.Status.Conditions {
						if cond.Type == v1.VirtualMachineInstanceConditionType(v1.VirtualMachineInstanceSynchronized) && cond.Status == kubev1.ConditionFalse {
							vmiCondition = cond
							return true
						}
					}
				}
				return false
			}, 120*time.Second, time.Second).Should(BeTrue())
			Expect(vmiCondition.Message).To(ContainSubstring("Invalid PCI address " + wrongPciAddress))
			Expect(vmiCondition.Reason).To(Equal("Synchronizing with the Domain failed."))
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
			if !tests.HasFeature(virtconfig.CPUManager) {
				Skip("Skip tests requiring CPUManager if feature gate is not enabled.")
			}
		})

		Context("with cpu pinning enabled", func() {
			It("[test_id:1684]should set the cpumanager label to false when it's not running", func() {

				By("adding a cpumanger=true label to a node")
				nodes, err := virtClient.CoreV1().Nodes().List(metav1.ListOptions{LabelSelector: v1.CPUManager + "=" + "false"})
				Expect(err).ToNot(HaveOccurred())
				if len(nodes.Items) == 0 {
					Skip("Skip CPU manager test on clusters where CPU manager is running on all worker/compute nodes")
				}

				node := &nodes.Items[0]
				node, err = virtClient.CoreV1().Nodes().Patch(node.Name, types.StrategicMergePatchType, []byte(fmt.Sprintf(`{"metadata": { "labels": {"%s": "true"}}}`, v1.CPUManager)))
				Expect(err).ToNot(HaveOccurred())

				By("setting the cpumanager label back to false")
				Eventually(func() string {
					n, err := virtClient.CoreV1().Nodes().Get(node.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return n.Labels[v1.CPUManager]
				}, 3*time.Minute, 2*time.Second).Should(Equal("false"))
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
			It("[test_id:991]should be scheduled on a node with running cpu manager", func() {
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

				By("Checking that the VMI QOS is guaranteed")
				vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(cpuVmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Status.QOSClass).ToNot(BeNil())
				Expect(*vmi.Status.QOSClass).To(Equal(kubev1.PodQOSGuaranteed))

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
			It("should be able to start a vm with guest memory different from requested and keed guaranteed qos", func() {
				cpuVmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				cpuVmi.Spec.Domain.CPU = &v1.CPU{
					Sockets:               2,
					Cores:                 1,
					DedicatedCPUPlacement: true,
				}
				guestMemory := resource.MustParse("64M")
				cpuVmi.Spec.Domain.Memory = &v1.Memory{Guest: &guestMemory}
				cpuVmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: kubev1.ResourceList{
						kubev1.ResourceMemory: resource.MustParse("80M"),
					},
				}

				By("Starting a VirtualMachineInstance")
				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(cpuVmi)
				Expect(err).ToNot(HaveOccurred())
				node := tests.WaitForSuccessfulVMIStart(cpuVmi)

				By("Checking that the VMI QOS is guaranteed")
				vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(cpuVmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Status.QOSClass).ToNot(BeNil())
				Expect(*vmi.Status.QOSClass).To(Equal(kubev1.PodQOSGuaranteed))

				Expect(isNodeHasCPUManagerLabel(node)).To(BeTrue())

				By("Checking that the pod QOS is guaranteed")
				readyPod := tests.GetRunningPodByVirtualMachineInstance(cpuVmi, tests.NamespaceTestDefault)
				podQos := readyPod.Status.QOSClass
				Expect(podQos).To(Equal(kubev1.PodQOSGuaranteed))

				//-------------------------------------------------------------------
				expecter, err := tests.LoggedInCirrosExpecter(vmi)
				Expect(err).ToNot(HaveOccurred())
				defer expecter.Close()

				res, err := expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: "[ $(free -m | grep Mem: | tr -s ' ' | cut -d' ' -f2) -lt 80 ] && echo 'pass' || echo 'fail'\n"},
					&expect.BExp{R: "pass"},
					&expect.BSnd{S: "swapoff -a && dd if=/dev/zero of=/dev/shm/test bs=1k count=118k; echo $?\n"},
					&expect.BExp{R: "0"},
				}, 15*time.Second)
				log.DefaultLogger().Object(vmi).Infof("%v", res)
				Expect(err).ToNot(HaveOccurred())

				pod := tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)
				podMemoryUsage, err := tests.ExecuteCommandOnPod(
					virtClient,
					pod,
					"compute",
					[]string{"/usr/bin/bash", "-c", "cat /sys/fs/cgroup/memory/memory.usage_in_bytes"},
				)
				Expect(err).ToNot(HaveOccurred())
				By("Converting pod memory usage")
				m, err := strconv.Atoi(strings.Trim(podMemoryUsage, "\n"))
				Expect(err).ToNot(HaveOccurred())
				By("Checking if pod memory usage is > 80Mi")
				Expect(m > 83886080).To(BeTrue(), "83886080 B = 80 Mi")
			})
			It("[test_id:4023]should start a vmi with dedicated cpus and isolated emulator thread", func() {

				cpuVmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				cpuVmi.Spec.Domain.CPU = &v1.CPU{
					Cores:                 2,
					DedicatedCPUPlacement: true,
					IsolateEmulatorThread: true,
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

				By("Checking that the VMI QOS is guaranteed")
				vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(cpuVmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Status.QOSClass).ToNot(BeNil())
				Expect(*vmi.Status.QOSClass).To(Equal(kubev1.PodQOSGuaranteed))

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

				// 1 additioan pcpus should be allocated on the pod for the emulation threads
				Expect(len(pinnedCPUsList)).To(Equal(int(cpuVmi.Spec.Domain.CPU.Cores) + 1))

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

			It("[test_id:4024]should fail the vmi creation if IsolateEmulatorThread requested without dedicated cpus", func() {
				cpuVmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				cpuVmi.Spec.Domain.CPU = &v1.CPU{
					Cores:                 2,
					IsolateEmulatorThread: true,
				}

				cpuVmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: kubev1.ResourceList{
						kubev1.ResourceMemory: resource.MustParse("64M"),
					},
				}
				By("Starting a VirtualMachineInstance")
				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(cpuVmi)
				Expect(err).To(HaveOccurred())
			})

			It("[test_id:802]should configure correct number of vcpus with requests.cpus", func() {
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
			It("[test_id:830]should start a vm with no cpu pinning after a vm with cpu pinning on same node", func() {
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
						kubev1.ResourceCPU:    resource.MustParse("1"),
						kubev1.ResourceMemory: resource.MustParse("64M"),
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

		Context("cpu pinning with fedora images, dedicated and non dedicated cpu should be possible on same node via spec.domain.cpu.cores", func() {

			var cpuvmi, vmi *v1.VirtualMachineInstance
			var node string

			BeforeEach(func() {

				nodes := tests.GetAllSchedulableNodes(virtClient)
				Expect(nodes.Items).ToNot(BeEmpty(), "There should be some nodes")
				node = nodes.Items[1].Name

				vmi = tests.NewRandomVMIWithEphemeralDiskAndUserdata(
					tests.ContainerDiskFor(
						tests.ContainerDiskFedora), "#!/bin/bash\necho \"fedora\" | passwd fedora --stdin\n")

				cpuvmi = tests.NewRandomVMIWithEphemeralDiskAndUserdata(
					tests.ContainerDiskFor(
						tests.ContainerDiskFedora), "#!/bin/bash\necho \"fedora\" | passwd fedora --stdin\n")
				cpuvmi.Spec.Domain.CPU = &v1.CPU{
					Cores:                 2,
					DedicatedCPUPlacement: true,
				}
				cpuvmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: kubev1.ResourceList{
						kubev1.ResourceMemory: resource.MustParse("512M"),
					},
				}
				cpuvmi.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": node}

				vmi.Spec.Domain.CPU = &v1.CPU{
					Cores: 2,
				}
				vmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: kubev1.ResourceList{
						kubev1.ResourceMemory: resource.MustParse("512M"),
					},
				}
				vmi.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": node}
			})

			It("[test_id:829]should start a vm with no cpu pinning after a vm with cpu pinning on same node", func() {

				By("Starting a VirtualMachineInstance with dedicated cpus")
				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(cpuvmi)
				Expect(err).ToNot(HaveOccurred())
				node1 := tests.WaitForSuccessfulVMIStart(cpuvmi)
				Expect(isNodeHasCPUManagerLabel(node1)).To(BeTrue())
				Expect(node1).To(Equal(node))

				By("Expecting the VirtualMachineInstance console")
				expecter1, err := tests.LoggedInFedoraExpecter(cpuvmi)
				Expect(err).ToNot(HaveOccurred())
				defer expecter1.Close()

				By("Starting a VirtualMachineInstance without dedicated cpus")
				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())
				node2 := tests.WaitForSuccessfulVMIStart(vmi)
				Expect(isNodeHasCPUManagerLabel(node2)).To(BeTrue())
				Expect(node2).To(Equal(node))

				By("Expecting the VirtualMachineInstance console")
				expecter2, err := tests.LoggedInFedoraExpecter(vmi)
				Expect(err).ToNot(HaveOccurred())
				defer expecter2.Close()
			})

			It("[test_id:832]should start a vm with cpu pinning after a vm with no cpu pinning on same node", func() {

				By("Starting a VirtualMachineInstance without dedicated cpus")
				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())
				node2 := tests.WaitForSuccessfulVMIStart(vmi)
				Expect(isNodeHasCPUManagerLabel(node2)).To(BeTrue())
				Expect(node2).To(Equal(node))

				By("Expecting the VirtualMachineInstance console")
				expecter1, err := tests.LoggedInFedoraExpecter(vmi)
				Expect(err).ToNot(HaveOccurred())
				defer expecter1.Close()

				By("Starting a VirtualMachineInstance with dedicated cpus")
				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(cpuvmi)
				Expect(err).ToNot(HaveOccurred())
				node1 := tests.WaitForSuccessfulVMIStart(cpuvmi)
				Expect(isNodeHasCPUManagerLabel(node1)).To(BeTrue())
				Expect(node1).To(Equal(node))

				By("Expecting the VirtualMachineInstance console")
				expecter2, err := tests.LoggedInFedoraExpecter(cpuvmi)
				Expect(err).ToNot(HaveOccurred())
				defer expecter2.Close()
			})
		})
	})

	Context("[rfe_id:2926][crit:medium][vendor:cnv-qe@redhat.com][level:component]Check Chassis value", func() {

		It("[test_id:2927]Test Chassis value in a newly created VM", func() {
			vmi := tests.NewRandomFedoraVMIWithDmidecode()
			vmi.Spec.Domain.Chassis = &v1.Chassis{
				Asset: "Test-123",
			}

			By("Starting a VirtualMachineInstance")
			vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStart(vmi)

			By("Check values on domain XML")
			domXml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(domXml).To(ContainSubstring("<entry name='asset'>Test-123</entry>"))

			By("Expecting console")
			expecter, err := tests.LoggedInFedoraExpecter(vmi)
			Expect(err).ToNot(HaveOccurred())
			defer expecter.Close()

			By("Check value in VM with dmidecode")
			// Check on the VM, if expected values are there with dmidecode
			res, err := expecter.ExpectBatch([]expect.Batcher{
				&expect.BSnd{S: "[ $(sudo dmidecode -s chassis-asset-tag | tr -s ' ') -eq Test-123 ] && echo 'pass' || echo 'fail'\n"},
				&expect.BExp{R: "pass"},
			}, 1*time.Second)
			log.DefaultLogger().Object(vmi).Infof("%v", res)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("[rfe_id:2926][crit:medium][vendor:cnv-qe@redhat.com][level:component]Check SMBios with default and custom values", func() {

		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = tests.NewRandomFedoraVMIWithDmidecode()
		})

		It("[test_id:2751]test default SMBios", func() {

			By("Starting a VirtualMachineInstance")
			vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStart(vmi)

			test_smbios := &cmdv1.SMBios{Family: "", Product: "", Manufacturer: ""}
			smbiosJson, err := json.Marshal(test_smbios)
			Expect(err).ToNot(HaveOccurred())
			tests.UpdateClusterConfigValueAndWait(virtconfig.SmbiosConfigKey, string(smbiosJson))

			By("Check values in domain XML")
			domXml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(domXml).To(ContainSubstring("<entry name='family'>KubeVirt</entry>"))
			Expect(domXml).To(ContainSubstring("<entry name='product'>None</entry>"))
			Expect(domXml).To(ContainSubstring("<entry name='manufacturer'>KubeVirt</entry>"))

			By("Expecting console")
			expecter, err := tests.LoggedInFedoraExpecter(vmi)
			Expect(err).ToNot(HaveOccurred())
			defer expecter.Close()

			By("Check values in dmidecode")
			// Check on the VM, if expected values are there with dmidecode
			res, err := expecter.ExpectBatch([]expect.Batcher{
				&expect.BSnd{S: "[ $(sudo dmidecode -s system-family | tr -s ' ') -eq KubeVirt ] && echo 'pass' || echo 'fail'\n"},
				&expect.BExp{R: "pass"},
				&expect.BSnd{S: "[ $(sudo dmidecode -s system-product-name | tr -s ' ') -eq None ] && echo 'pass' || echo 'fail'\n"},
				&expect.BExp{R: "pass"},
				&expect.BSnd{S: "[ $(sudo dmidecode -s system-manufacturer | tr -s ' ') -eq KubeVirt ] && echo 'pass' || echo 'fail'\n"},
				&expect.BExp{R: "pass"},
			}, 1*time.Second)
			log.DefaultLogger().Object(vmi).Infof("%v", res)
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:2752]test custom SMBios values", func() {
			// Set a custom test SMBios
			test_smbios := &cmdv1.SMBios{Family: "test", Product: "test", Manufacturer: "None", Sku: "1.0", Version: "1.0"}
			smbiosJson, err := json.Marshal(test_smbios)
			Expect(err).ToNot(HaveOccurred())
			tests.UpdateClusterConfigValueAndWait(virtconfig.SmbiosConfigKey, string(smbiosJson))

			By("Starting a VirtualMachineInstance")
			vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStart(vmi)

			domXml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(domXml).To(ContainSubstring("<entry name='family'>test</entry>"))
			Expect(domXml).To(ContainSubstring("<entry name='product'>test</entry>"))
			Expect(domXml).To(ContainSubstring("<entry name='manufacturer'>None</entry>"))
			Expect(domXml).To(ContainSubstring("<entry name='sku'>1.0</entry>"))
			Expect(domXml).To(ContainSubstring("<entry name='version'>1.0</entry>"))

			By("Expecting console")
			expecter, err := tests.LoggedInFedoraExpecter(vmi)
			Expect(err).ToNot(HaveOccurred())
			defer expecter.Close()

			By("Check values in dmidecode")

			// Check on the VM, if expected values are there with dmidecode
			res, err := expecter.ExpectBatch([]expect.Batcher{
				&expect.BSnd{S: "[ $(sudo dmidecode -s system-family | tr -s ' ') -eq test ] && echo 'pass' || echo 'fail'\n"},
				&expect.BExp{R: "pass"},
				&expect.BSnd{S: "[ $(sudo dmidecode -s system-product-name | tr -s ' ') -eq test ] && echo 'pass' || echo 'fail'\n"},
				&expect.BExp{R: "pass"},
				&expect.BSnd{S: "[ $(sudo dmidecode -s system-manufacturer | tr -s ' ') -eq None ] && echo 'pass' || echo 'fail'\n"},
				&expect.BExp{R: "pass"},
			}, 1*time.Second)
			log.DefaultLogger().Object(vmi).Infof("%v", res)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("With ephemeral CD-ROM", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = tests.NewRandomFedoraVMIWithDmidecode()
		})

		table.DescribeTable("For various bus types", func(bus string, errMsg string) {
			tests.AddEphemeralCdrom(vmi, "cdrom-0", bus, tests.ContainerDiskFor(tests.ContainerDiskCirros))

			By(fmt.Sprintf("Starting a VMI with a %s CD-ROM", bus))
			_, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			if errMsg == "" {
				Expect(err).ToNot(HaveOccurred())
			} else {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(errMsg))
			}
		},
			table.Entry("[test_id:3777] Should be accepted when using sata", "sata", ""),
			table.Entry("[test_id:3809] Should be accepted when using scsi", "scsi", ""),
			table.Entry("[test_id:3776] Should be rejected when using virtio", "virtio", "Bus type virtio is invalid"),
			table.Entry("[test_id:3808] Should be rejected when using ide", "ide", "IDE bus is not supported"),
		)
	})

	It("[test_id:4153]VMI with masquerade binding and guest agent should expose Pod IP as its public address", func() {
		vmi := tests.NewRandomFedoraVMIWitGuestAgent()

		By("Starting a VirtualMachineInstance")
		vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
		Expect(err).ToNot(HaveOccurred(), "Should successfully create VMI")
		tests.WaitForSuccessfulVMIStart(vmi)

		vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred(), "Should successfully get VMI")

		vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)

		tests.WaitAgentConnected(virtClient, vmi)

		// Ensure that VMI IP Stays equal to PodIP even after Guest-Agent kicks in
		Consistently(func() bool {
			vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred(), "Should successfully get VMI")
			return vmi.Status.Interfaces[0].IP == vmiPod.Status.PodIP
		}, 30*time.Second, 1*time.Second).Should(BeTrue(), "VMI status IP should match VMI Pod IP")
	})
})
