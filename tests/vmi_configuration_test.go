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
	"bufio"
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode"

	"kubevirt.io/kubevirt/tests/decorators"

	"k8s.io/apimachinery/pkg/util/rand"

	"kubevirt.io/kubevirt/pkg/virt-controller/services"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/pborman/uuid"
	k8sv1 "k8s.io/api/core/v1"
	kubev1 "k8s.io/api/core/v1"
	nodev1 "k8s.io/api/node/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"

	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	. "kubevirt.io/kubevirt/tests/framework/matcher"

	kvutil "kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/tests/util"

	v1 "kubevirt.io/api/core/v1"
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	kubevirt_hooks_v1alpha2 "kubevirt.io/kubevirt/pkg/hooks/v1alpha2"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/util/cluster"
	hw_utils "kubevirt.io/kubevirt/pkg/util/hardware"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/libdv"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
	"kubevirt.io/kubevirt/tests/watcher"
)

var _ = Describe("[sig-compute]Configurations", decorators.SigCompute, func() {
	const enoughMemForSafeBiosEmulation = "32Mi"
	var virtClient kubecli.KubevirtClient

	const (
		cgroupV1MemoryUsagePath = "/sys/fs/cgroup/memory/memory.usage_in_bytes"
		cgroupV2MemoryUsagePath = "/sys/fs/cgroup/memory.current"
	)

	getPodMemoryUsage := func(pod *kubev1.Pod) (output string, err error) {
		output, err = exec.ExecuteCommandOnPod(
			virtClient,
			pod,
			"compute",
			[]string{"cat", cgroupV2MemoryUsagePath},
		)

		if err == nil {
			return
		}

		output, err = exec.ExecuteCommandOnPod(
			virtClient,
			pod,
			"compute",
			[]string{"cat", cgroupV1MemoryUsagePath},
		)

		return
	}

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("with all devices on the root PCI bus", func() {
		It("[test_id:4623]should start run the guest as usual", func() {
			vmi := libvmi.NewCirros(
				libvmi.WithAnnotation(v1.PlacePCIDevicesOnRootComplex, "true"),
				libvmi.WithRng(),
				libvmi.WithWatchdog(v1.WatchdogActionPoweroff),
			)
			vmi.Spec.Domain.Devices.Inputs = []v1.Input{{Name: "tablet", Bus: v1.VirtIO, Type: "tablet"}, {Name: "tablet1", Bus: "usb", Type: "tablet"}}
			vmi = tests.RunVMIAndExpectLaunch(vmi, 60)
			Expect(console.LoginToCirros(vmi)).To(Succeed())
			domSpec, err := tests.GetRunningVMIDomainSpec(vmi)
			Expect(err).ToNot(HaveOccurred())
			rootPortController := []api.Controller{}
			for _, c := range domSpec.Devices.Controllers {
				if c.Model == "pcie-root-port" {
					rootPortController = append(rootPortController, c)
				}
			}
			Expect(rootPortController).To(BeEmpty(), "libvirt should not add additional buses to the root one")
		})
	})

	Context("when requesting virtio-transitional models", func() {
		It("[test_id:6957]should start and run the guest", func() {
			vmi := libvmi.NewCirros(
				libvmi.WithRng(),
				libvmi.WithWatchdog(v1.WatchdogActionPoweroff),
			)
			vmi.Spec.Domain.Devices.Inputs = []v1.Input{{Name: "tablet", Bus: v1.VirtIO, Type: "tablet"}, {Name: "tablet1", Bus: "usb", Type: "tablet"}}
			vmi.Spec.Domain.Devices.UseVirtioTransitional = pointer.BoolPtr(true)
			vmi = tests.RunVMIAndExpectLaunch(vmi, 60)
			Expect(console.LoginToCirros(vmi)).To(Succeed())
			domSpec, err := tests.GetRunningVMIDomainSpec(vmi)
			Expect(err).ToNot(HaveOccurred())
			testutils.ExpectVirtioTransitionalOnly(domSpec)
		})
	})

	Context("[rfe_id:897][crit:medium][vendor:cnv-qe@redhat.com][level:component]for CPU and memory limits should", func() {

		It("[test_id:3110]lead to get the burstable QOS class assigned when limit and requests differ", func() {
			vmi := libvmi.NewAlpine()
			vmi = tests.RunVMIAndExpectScheduling(vmi, 60)

			Eventually(func() kubev1.PodQOSClass {
				vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.IsFinal()).To(BeFalse())
				if vmi.Status.QOSClass == nil {
					return ""
				}
				return *vmi.Status.QOSClass
			}, 10*time.Second, 1*time.Second).Should(Equal(kubev1.PodQOSBurstable))
		})

		It("[test_id:3111]lead to get the guaranteed QOS class assigned when limit and requests are identical", func() {
			vmi := libvmi.NewAlpine()
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
				vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.IsFinal()).To(BeFalse())
				if vmi.Status.QOSClass == nil {
					return ""
				}
				return *vmi.Status.QOSClass
			}, 10*time.Second, 1*time.Second).Should(Equal(kubev1.PodQOSGuaranteed))
		})

		It("[test_id:3112]lead to get the guaranteed QOS class assigned when only limits are set", func() {
			vmi := libvmi.NewAlpine()
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
				vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.IsFinal()).To(BeFalse())
				if vmi.Status.QOSClass == nil {
					return ""
				}
				return *vmi.Status.QOSClass
			}, 10*time.Second, 1*time.Second).Should(Equal(kubev1.PodQOSGuaranteed))

			vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Spec.Domain.Resources.Requests.Cpu().Cmp(*vmi.Spec.Domain.Resources.Limits.Cpu())).To(BeZero())
			Expect(vmi.Spec.Domain.Resources.Requests.Memory().Cmp(*vmi.Spec.Domain.Resources.Limits.Memory())).To(BeZero())
		})

	})

	Describe("VirtualMachineInstance definition", func() {
		fedoraWithUefiSecuredBoot := libvmi.NewFedora(
			libvmi.WithResourceMemory("1Gi"),
			libvmi.WithUefi(true),
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)
		alpineWithUefiWithoutSecureBoot := libvmi.NewAlpine(
			libvmi.WithResourceMemory("1Gi"),
			libvmi.WithUefi(false),
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)

		DescribeTable("with memory configuration", func(vmiOptions []libvmi.Option, expectedGuestMemory int) {
			vmi := libvmi.New(vmiOptions...)

			By("Starting a VirtualMachineInstance")
			vmi = tests.RunVMIAndExpectScheduling(vmi, 60)
			libwait.WaitForSuccessfulVMIStart(vmi)

			expectedMemoryInKiB := expectedGuestMemory * 1024
			expectedMemoryXMLStr := fmt.Sprintf("unit='KiB'>%d", expectedMemoryInKiB)

			domXml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(domXml).To(ContainSubstring(expectedMemoryXMLStr))

		},
			Entry("provided by domain spec directly",
				[]libvmi.Option{
					libvmi.WithGuestMemory("512Mi"),
				},
				512,
			),
			Entry("provided by resources limits",
				[]libvmi.Option{
					libvmi.WithLimitMemory("256Mi"),
					libvmi.WithLimitCPU("1"),
				},
				256,
			),
			Entry("provided by resources requests and limits",
				[]libvmi.Option{
					libvmi.WithResourceCPU("1"),
					libvmi.WithLimitCPU("1"),
					libvmi.WithResourceMemory("64Mi"),
					libvmi.WithLimitMemory("256Mi"),
				},
				64,
			),
		)

		Context("[Serial][rfe_id:2065][crit:medium][vendor:cnv-qe@redhat.com][level:component]with 3 CPU cores", Serial, func() {
			var availableNumberOfCPUs int
			var vmi *v1.VirtualMachineInstance

			BeforeEach(func() {
				availableNumberOfCPUs = libnode.GetHighestCPUNumberAmongNodes(virtClient)

				requiredNumberOfCpus := 3
				Expect(availableNumberOfCPUs).ToNot(BeNumerically("<", requiredNumberOfCpus),
					fmt.Sprintf("Test requires %d cpus, but only %d available!", requiredNumberOfCpus, availableNumberOfCPUs))
				vmi = libvmi.NewAlpine()
			})

			It("[test_id:1659]should report 3 cpu cores under guest OS", func() {
				vmi.Spec.Domain.CPU = &v1.CPU{
					Cores: 3,
				}
				vmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: kubev1.ResourceList{
						kubev1.ResourceMemory: resource.MustParse("100M"),
					},
				}

				By("Starting a VirtualMachineInstance")
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred(), "should start vmi")
				libwait.WaitForSuccessfulVMIStart(vmi)

				By("Expecting the VirtualMachineInstance console")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				By("Checking the number of CPU cores under guest OS")
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "grep -c ^processor /proc/cpuinfo\n"},
					&expect.BExp{R: console.RetValue("3")},
				}, 15)).To(Succeed(), "should report number of cores")

				By("Checking the requested amount of memory allocated for a guest")
				Expect(vmi.Spec.Domain.Resources.Requests.Memory().String()).To(Equal("100M"))

				readyPod := tests.GetRunningPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
				var computeContainer *kubev1.Container
				for _, container := range readyPod.Spec.Containers {
					if container.Name == "compute" {
						computeContainer = &container
						break
					}
				}
				if computeContainer == nil {
					util.PanicOnError(fmt.Errorf("could not find the compute container"))
				}
				Expect(computeContainer.Resources.Requests.Memory().ToDec().ScaledValue(resource.Mega)).To(Equal(int64(371)))

				Expect(err).ToNot(HaveOccurred())
			})
			It("[test_id:4624]should set a correct memory units", func() {
				vmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: kubev1.ResourceList{
						kubev1.ResourceMemory: resource.MustParse("64Mi"),
					},
				}
				expectedMemoryInKiB := 64 * 1024
				expectedMemoryXMLStr := fmt.Sprintf("unit='KiB'>%d", expectedMemoryInKiB)

				By("Starting a VirtualMachineInstance")
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(vmi)

				domXml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(domXml).To(ContainSubstring(expectedMemoryXMLStr))
			})

			It("[test_id:1660]should report 3 sockets under guest OS", func() {
				vmi.Spec.Domain.CPU = &v1.CPU{
					Sockets: 3,
					Cores:   2,
				}
				vmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: kubev1.ResourceList{
						kubev1.ResourceMemory: resource.MustParse("120M"),
					},
				}

				By("Starting a VirtualMachineInstance")
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred(), "should start vmi")
				libwait.WaitForSuccessfulVMIStart(vmi)

				By("Expecting the VirtualMachineInstance console")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				By("Checking the number of sockets under guest OS")
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "grep '^physical id' /proc/cpuinfo | uniq | wc -l\n"},
					&expect.BExp{R: console.RetValue("3")},
				}, 60)).To(Succeed(), "should report number of sockets")
			})

			It("[test_id:1661]should report 2 sockets from spec.domain.resources.requests under guest OS ", func() {
				vmi.Spec.Domain.CPU = nil
				vmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: kubev1.ResourceList{
						kubev1.ResourceCPU:    resource.MustParse("1200m"),
						kubev1.ResourceMemory: resource.MustParse("100M"),
					},
				}

				By("Starting a VirtualMachineInstance")
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred(), "should start vmi")
				libwait.WaitForSuccessfulVMIStart(vmi)

				By("Expecting the VirtualMachineInstance console")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				By("Checking the number of sockets under guest OS")
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "grep '^physical id' /proc/cpuinfo | uniq | wc -l\n"},
					&expect.BExp{R: console.RetValue("2")},
				}, 60)).To(Succeed(), "should report number of sockets")
			})

			It("[test_id:1662]should report 2 sockets from spec.domain.resources.limits under guest OS ", func() {
				vmi.Spec.Domain.CPU = nil
				vmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: kubev1.ResourceList{
						kubev1.ResourceMemory: resource.MustParse("100M"),
					},
					Limits: kubev1.ResourceList{
						kubev1.ResourceCPU: resource.MustParse("1200m"),
					},
				}

				By("Starting a VirtualMachineInstance")
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred(), "should start vmi")
				libwait.WaitForSuccessfulVMIStart(vmi)

				By("Expecting the VirtualMachineInstance console")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				By("Checking the number of sockets under guest OS")
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "grep '^physical id' /proc/cpuinfo | uniq | wc -l\n"},
					&expect.BExp{R: console.RetValue("2")},
				}, 60)).To(Succeed(), "should report number of sockets")
			})

			It("[test_id:1663]should report 4 vCPUs under guest OS", func() {
				vmi.Spec.Domain.CPU = &v1.CPU{
					Threads: 2,
					Sockets: 2,
					Cores:   1,
				}
				vmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: kubev1.ResourceList{
						kubev1.ResourceMemory: resource.MustParse("128M"),
					},
				}

				By("Starting a VirtualMachineInstance")
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred(), "should start vmi")
				libwait.WaitForSuccessfulVMIStart(vmi)

				By("Expecting the VirtualMachineInstance console")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				By("Checking the number of vCPUs under guest OS")
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "grep -c ^processor /proc/cpuinfo\n"},
					&expect.BExp{R: console.RetValue("4")},
				}, 60)).To(Succeed(), "should report number of threads")
			})

			It("[Serial][test_id:1664]should map cores to virtio block queues", Serial, func() {
				_true := true
				vmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: kubev1.ResourceList{
						kubev1.ResourceMemory: resource.MustParse("64M"),
						kubev1.ResourceCPU:    resource.MustParse("3"),
					},
				}
				vmi.Spec.Domain.Devices.BlockMultiQueue = &_true

				By("Starting a VirtualMachineInstance")
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(vmi)

				domXml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(domXml).To(ContainSubstring("queues='3'"))
			})

			It("[test_id:1665]should map cores to virtio net queues", func() {
				if testsuite.ShouldAllowEmulation(virtClient) {
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
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(vmi)

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
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(vmi)

				domXml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(domXml).ToNot(ContainSubstring("queues='"))
			})
		})

		Context("[rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]with no memory requested", func() {
			It("[test_id:3113]should failed to the VMI creation", func() {
				vmi := libvmi.New()
				By("Starting a VirtualMachineInstance")
				_, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("[Serial][rfe_id:609][crit:medium][vendor:cnv-qe@redhat.com][level:component]with cluster memory overcommit being applied", Serial, func() {
			BeforeEach(func() {
				kv := util.GetCurrentKv(virtClient)

				config := kv.Spec.Configuration
				config.DeveloperConfiguration.MemoryOvercommit = 200
				tests.UpdateKubeVirtConfigValueAndWait(config)
			})

			It("[test_id:3114]should set requested amount of memory according to the specified virtual memory", func() {
				vmi := libvmi.New()
				guestMemory := resource.MustParse("4096M")
				vmi.Spec.Domain.Memory = &v1.Memory{Guest: &guestMemory}
				vmi.Spec.Domain.Resources = v1.ResourceRequirements{}
				runningVMI := tests.RunVMI(vmi, 30)
				Expect(runningVMI.Spec.Domain.Resources.Requests.Memory().String()).To(Equal("2048M"))
			})
		})

		Context("with BIOS bootloader method and no disk", func() {
			It("[test_id:5265]should find no bootable device by default", func() {
				By("Creating a VMI with no disk and an explicit network interface")
				vmi := libvmi.New(
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				)
				vmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: kubev1.ResourceList{
						kubev1.ResourceMemory: resource.MustParse("32M"),
					},
				}

				By("Enabling BIOS serial output")
				vmi.Spec.Domain.Firmware = &v1.Firmware{
					Bootloader: &v1.Bootloader{
						BIOS: &v1.BIOS{
							UseSerial: tests.NewBool(true),
						},
					},
				}

				By("Ensuring network boot is disabled on the network interface")
				Expect(vmi.Spec.Domain.Devices.Interfaces[0].BootOrder).To(BeNil())

				By("Starting a VirtualMachineInstance")
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred())

				By("Expecting no bootable NIC")
				Expect(console.NetBootExpecter(vmi)).NotTo(Succeed())
				// The expecter *should* have error-ed since the network interface is not marked bootable
			})

			It("[test_id:5266]should boot to NIC rom if a boot order was set on a network interface", func() {
				By("Enabling network boot")
				var bootOrder uint = 1
				interfaceDeviceWithMasqueradeBinding := libvmi.InterfaceDeviceWithMasqueradeBinding()
				interfaceDeviceWithMasqueradeBinding.BootOrder = &bootOrder

				By("Creating a VMI with no disk and an explicit network interface")
				vmi := libvmi.New(
					libvmi.WithResourceMemory(enoughMemForSafeBiosEmulation),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithInterface(interfaceDeviceWithMasqueradeBinding),
					withSerialBIOS(),
				)

				By("Starting a VirtualMachineInstance")
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred())

				By("Expecting a bootable NIC")
				Expect(console.NetBootExpecter(vmi)).To(Succeed())
			})
		})

		DescribeTable("[rfe_id:2262][crit:medium][vendor:cnv-qe@redhat.com][level:component]with EFI bootloader method", func(vmi *v1.VirtualMachineInstance, loginTo console.LoginToFunction, msg string, fileName string) {
			By("Starting a VirtualMachineInstance")
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred())

			wp := watcher.WarningsPolicy{FailOnWarnings: false}
			libwait.WaitForVMIPhase(vmi,
				[]v1.VirtualMachineInstancePhase{v1.Running, v1.Failed},
				libwait.WithWarningsPolicy(&wp),
				libwait.WithTimeout(180),
				libwait.WithWaitForFail(true),
			)
			vmiMeta, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
			ExpectWithOffset(1, err).ToNot(HaveOccurred())

			switch vmiMeta.Status.Phase {
			case v1.Failed:
				// This Error is expected to be handled
				By("Getting virt-launcher logs")
				logs := func() string { return getVirtLauncherLogs(virtClient, vmi) }
				Eventually(logs,
					30*time.Second,
					500*time.Millisecond).
					Should(ContainSubstring("EFI OVMF rom missing"))
			default:
				libwait.WaitUntilVMIReady(vmi, loginTo)
				By(msg)
				domXml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(domXml).To(MatchRegexp(fileName))
			}
		},
			Entry("[Serial][test_id:1668]should use EFI without secure boot", Serial, alpineWithUefiWithoutSecureBoot, console.LoginToAlpine, "Checking if UEFI is enabled", `OVMF_CODE(\.secboot)?\.fd`),
			Entry("[Serial][test_id:4437]should enable EFI secure boot", Serial, fedoraWithUefiSecuredBoot, console.SecureBootExpecter, "Checking if SecureBoot is enabled in the libvirt XML", `OVMF_CODE\.secboot\.fd`),
		)

		Context("[rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]with diverging guest memory from requested memory", func() {
			It("[test_id:1669]should show the requested guest memory inside the VMI", func() {
				vmi := libvmi.NewCirros()
				guestMemory := resource.MustParse("256Mi")
				vmi.Spec.Domain.Memory = &v1.Memory{
					Guest: &guestMemory,
				}

				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(vmi)

				Expect(console.LoginToCirros(vmi)).To(Succeed())

				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "free -m | grep Mem: | tr -s ' ' | cut -d' ' -f2\n"},
					&expect.BExp{R: console.RetValue("225")},
				}, 10)).To(Succeed())

			})
		})

		Context("[rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]with diverging memory limit from memory request and no guest memory", func() {
			It("[test_id:3115]should show the memory request inside the VMI", func() {
				vmi := libvmi.NewCirros(
					libvmi.WithResourceMemory("256Mi"),
					libvmi.WithLimitMemory("512Mi"),
				)
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(vmi)

				Expect(console.LoginToCirros(vmi)).To(Succeed())

				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "free -m | grep Mem: | tr -s ' ' | cut -d' ' -f2\n"},
					&expect.BExp{R: console.RetValue("225")},
				}, 10)).To(Succeed())

			})
		})

		Context("[rfe_id:989]test cpu_allocation_ratio", func() {
			It("virt-launchers pod cpu requests should be proportional to the number of vCPUs", func() {
				vmi := libvmi.NewCirros()
				guestMemory := resource.MustParse("256Mi")
				vmi.Spec.Domain.Memory = &v1.Memory{
					Guest: &guestMemory,
				}
				vmi.Spec.Domain.CPU = &v1.CPU{
					Threads: 1,
					Sockets: 1,
					Cores:   6,
				}

				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(vmi)

				readyPod := tests.GetRunningPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
				var computeContainer *kubev1.Container
				for _, container := range readyPod.Spec.Containers {
					if container.Name == "compute" {
						computeContainer = &container
						break
					}
				}
				if computeContainer == nil {
					util.PanicOnError(fmt.Errorf("could not find the compute container"))
				}
				Expect(computeContainer.Resources.Requests.Cpu().String()).To(Equal("600m"))
			})

		})

		Context("[rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]with support memory over commitment", func() {
			It("[test_id:755]should show the requested memory different than guest memory", func() {
				vmi := libvmi.NewCirros(overcommitGuestOverhead())
				guestMemory := resource.MustParse("256Mi")
				vmi.Spec.Domain.Memory = &v1.Memory{
					Guest: &guestMemory,
				}

				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(vmi)

				Expect(console.LoginToCirros(vmi)).To(Succeed())

				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "[ $(free -m | grep Mem: | tr -s ' ' | cut -d' ' -f2) -gt 200 ] && echo 'pass'\n"},
					&expect.BExp{R: console.RetValue("pass")},
					&expect.BSnd{S: "swapoff -a && dd if=/dev/zero of=/dev/shm/test bs=1k count=100k\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: console.RetValue("0")},
				}, 15)).To(Succeed())

				pod := tests.GetRunningPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
				podMemoryUsage, err := getPodMemoryUsage(pod)
				Expect(err).ToNot(HaveOccurred())
				By("Converting pod memory usage")
				m, err := strconv.Atoi(strings.Trim(podMemoryUsage, "\n"))
				Expect(err).ToNot(HaveOccurred())
				By("Checking if pod memory usage is > 64Mi")
				Expect(m).To(BeNumerically(">", 67108864), "67108864 B = 64 Mi")
			})

		})

		Context("[rfe_id:609][crit:medium][vendor:cnv-qe@redhat.com][level:component]Support memory over commitment test", func() {
			var vmi *v1.VirtualMachineInstance

			BeforeEach(func() {
				var err error
				vmi = libvmi.NewCirros(overcommitGuestOverhead())
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(vmi)
			})

			It("[test_id:730]Check OverCommit VM Created and Started", func() {
				overcommitVmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(overcommitVmi)
			})
			It("[test_id:731]Check OverCommit status on VMI", func() {
				overcommitVmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(overcommitVmi.Spec.Domain.Resources.OvercommitGuestOverhead).To(BeTrue())
			})
			It("[test_id:732]Check Free memory on the VMI", func() {
				By("Expecting console")
				Expect(console.LoginToCirros(vmi)).To(Succeed())

				// Check on the VM, if the Free memory is roughly what we expected
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "[ $(free -m | grep Mem: | tr -s ' ' | cut -d' ' -f2) -gt 90 ] && echo 'pass'\n"},
					&expect.BExp{R: console.RetValue("pass")},
				}, 15)).To(Succeed())
			})
		})

		Context("[rfe_id:3078][crit:medium][vendor:cnv-qe@redhat.com][level:component]with usb controller", func() {
			It("[test_id:3117]should start the VMI with usb controller when usb device is present", func() {
				vmi := libvmi.NewAlpine()
				vmi.Spec.Domain.Devices.Inputs = []v1.Input{
					{
						Name: "tablet0",
						Type: "tablet",
						Bus:  "usb",
					},
				}
				By("Starting a VirtualMachineInstance")
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred(), "should start vmi")
				libwait.WaitForSuccessfulVMIStart(vmi)

				By("Expecting the VirtualMachineInstance console")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				By("Checking the number of usb under guest OS")
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "ls -l /sys/bus/usb/devices/usb* | wc -l\n"},
					&expect.BExp{R: console.RetValue("2")},
				}, 60)).To(Succeed(), "should report number of usb")
			})

			It("[test_id:3117]should start the VMI with usb controller when input device doesn't have bus", func() {
				vmi := libvmi.NewAlpine()
				vmi.Spec.Domain.Devices.Inputs = []v1.Input{
					{
						Name: "tablet0",
						Type: "tablet",
					},
				}
				By("Starting a VirtualMachineInstance")
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred(), "should start vmi")
				libwait.WaitForSuccessfulVMIStart(vmi)

				By("Expecting the VirtualMachineInstance console")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				By("Checking the number of usb under guest OS")
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "ls -l /sys/bus/usb/devices/usb* | wc -l\n"},
					&expect.BExp{R: console.RetValue("2")},
				}, 60)).To(Succeed(), "should report number of usb")
			})

			It("[test_id:3118]should start the VMI without usb controller", func() {
				vmi := libvmi.NewAlpine()
				By("Starting a VirtualMachineInstance")
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred(), "should start vmi")

				libwait.WaitForSuccessfulVMIStart(vmi)

				By("Expecting the VirtualMachineInstance console")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				By("Checking the number of usb under guest OS")
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "ls -l /sys/bus/usb/devices/usb* 2>/dev/null | wc -l\n"},
					&expect.BExp{R: console.RetValue("0")},
				}, 60)).To(Succeed(), "should report number of usb")
			})
		})

		Context("[rfe_id:3077][crit:medium][vendor:cnv-qe@redhat.com][level:component]with input devices", func() {
			It("[test_id:2642]should failed to start the VMI with wrong type of input device", func() {
				vmi := libvmi.NewCirros()
				vmi.Spec.Domain.Devices.Inputs = []v1.Input{
					{
						Name: "tablet0",
						Type: "keyboard",
						Bus:  v1.VirtIO,
					},
				}
				By("Starting a VirtualMachineInstance")
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).To(HaveOccurred(), "should not start vmi")
			})

			It("[test_id:3074]should failed to start the VMI with wrong bus of input device", func() {
				vmi := libvmi.NewCirros()
				vmi.Spec.Domain.Devices.Inputs = []v1.Input{
					{
						Name: "tablet0",
						Type: "tablet",
						Bus:  "ps2",
					},
				}
				By("Starting a VirtualMachineInstance")
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).To(HaveOccurred(), "should not start vmi")
			})

			It("[test_id:3072]should start the VMI with tablet input device with virtio bus", func() {
				vmi := libvmi.NewAlpine()
				vmi.Spec.Domain.Devices.Inputs = []v1.Input{
					{
						Name: "tablet0",
						Type: "tablet",
						Bus:  v1.VirtIO,
					},
				}
				By("Starting a VirtualMachineInstance")
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred(), "should start vmi")
				libwait.WaitForSuccessfulVMIStart(vmi)

				By("Expecting the VirtualMachineInstance console")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				By("Checking the tablet input under guest OS")
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "grep -rs '^QEMU Virtio Tablet' /sys/devices | wc -l\n"},
					&expect.BExp{R: console.RetValue("1")},
				}, 60)).To(Succeed(), "should report input device")
			})

			It("[test_id:3073]should start the VMI with tablet input device with usb bus", func() {
				vmi := libvmi.NewAlpine()
				vmi.Spec.Domain.Devices.Inputs = []v1.Input{
					{
						Name: "tablet0",
						Type: "tablet",
						Bus:  "usb",
					},
				}
				By("Starting a VirtualMachineInstance")
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred(), "should start vmi")
				libwait.WaitForSuccessfulVMIStart(vmi)

				By("Expecting the VirtualMachineInstance console")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				By("Checking the tablet input under guest OS")
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "grep -rs '^QEMU USB Tablet' /sys/devices | wc -l\n"},
					&expect.BExp{R: console.RetValue("1")},
				}, 60)).To(Succeed(), "should report input device")
			})
		})

		Context("with namespace different from provided", func() {
			It("should fail admission", func() {
				// create a namespace default limit
				limitRangeObj := kubev1.LimitRange{

					ObjectMeta: metav1.ObjectMeta{Name: "abc1", Namespace: testsuite.GetTestNamespace(nil)},
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
				_, err := virtClient.CoreV1().LimitRanges(testsuite.GetTestNamespace(nil)).Create(context.Background(), &limitRangeObj, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vmi := libvmi.NewAlpine()
				vmi.Namespace = testsuite.NamespaceTestAlternative
				vmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: kubev1.ResourceList{
						kubev1.ResourceMemory: resource.MustParse("64M"),
					},
					Limits: kubev1.ResourceList{
						kubev1.ResourceCPU: resource.MustParse("1000m"),
					},
				}

				By("Creating a VMI")
				Consistently(func() error {
					_, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi)
					return err
				}, 30*time.Second, time.Second).Should(And(HaveOccurred(), MatchError("the namespace of the provided object does not match the namespace sent on the request")))
			})
		})

		Context("[rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]with hugepages", func() {
			var hugepagesVmi *v1.VirtualMachineInstance

			verifyHugepagesConsumption := func() bool {
				// TODO: we need to check hugepages state via node allocated resources, but currently it has the issue
				// https://github.com/kubernetes/kubernetes/issues/64691
				pods, err := virtClient.CoreV1().Pods(hugepagesVmi.Namespace).List(context.Background(), tests.UnfinishedVMIPodSelector(hugepagesVmi))
				Expect(err).ToNot(HaveOccurred())
				Expect(pods.Items).To(HaveLen(1))

				hugepagesSize := resource.MustParse(hugepagesVmi.Spec.Domain.Memory.Hugepages.PageSize)
				hugepagesDir := fmt.Sprintf("/sys/kernel/mm/hugepages/hugepages-%dkB", hugepagesSize.Value()/int64(1024))

				// Get a hugepages statistics from virt-launcher pod
				output, err := exec.ExecuteCommandOnPod(
					virtClient,
					&pods.Items[0],
					pods.Items[0].Spec.Containers[0].Name,
					[]string{"cat", fmt.Sprintf("%s/nr_hugepages", hugepagesDir)},
				)
				Expect(err).ToNot(HaveOccurred())

				totalHugepages, err := strconv.Atoi(strings.Trim(output, "\n"))
				Expect(err).ToNot(HaveOccurred())

				output, err = exec.ExecuteCommandOnPod(
					virtClient,
					&pods.Items[0],
					pods.Items[0].Spec.Containers[0].Name,
					[]string{"cat", fmt.Sprintf("%s/free_hugepages", hugepagesDir)},
				)
				Expect(err).ToNot(HaveOccurred())

				freeHugepages, err := strconv.Atoi(strings.Trim(output, "\n"))
				Expect(err).ToNot(HaveOccurred())

				output, err = exec.ExecuteCommandOnPod(
					virtClient,
					&pods.Items[0],
					pods.Items[0].Spec.Containers[0].Name,
					[]string{"cat", fmt.Sprintf("%s/resv_hugepages", hugepagesDir)},
				)
				Expect(err).ToNot(HaveOccurred())

				resvHugepages, err := strconv.Atoi(strings.Trim(output, "\n"))
				Expect(err).ToNot(HaveOccurred())

				// Verify that the VM memory equals to a number of consumed hugepages
				vmHugepagesConsumption := int64(totalHugepages-freeHugepages+resvHugepages) * hugepagesSize.Value()
				vmMemory := hugepagesVmi.Spec.Domain.Resources.Requests[kubev1.ResourceMemory]
				if hugepagesVmi.Spec.Domain.Memory != nil && hugepagesVmi.Spec.Domain.Memory.Guest != nil {
					vmMemory = *hugepagesVmi.Spec.Domain.Memory.Guest
				}

				if vmHugepagesConsumption == vmMemory.Value() {
					return true
				}
				return false
			}
			BeforeEach(func() {
				hugepagesVmi = libvmi.NewCirros()
			})

			DescribeTable("should consume hugepages ", func(hugepageSize string, memory string, guestMemory string, option1, option2 libvmi.Option) {
				if option1 != nil && option2 != nil {
					hugepagesVmi = libvmi.NewCirros(option1, option2)
				}
				hugepageType := kubev1.ResourceName(kubev1.ResourceHugePagesPrefix + hugepageSize)
				v, err := cluster.GetKubernetesVersion()
				Expect(err).ShouldNot(HaveOccurred())
				if strings.Contains(v, "1.16") {
					hugepagesVmi.Annotations = map[string]string{
						v1.MemfdMemoryBackend: "false",
					}
					log.DefaultLogger().Object(hugepagesVmi).Infof("Fall back to use hugepages source file. Libvirt in the 1.16 provider version doesn't support memfd as memory backend")
				}

				nodeWithHugepages := libnode.GetNodeWithHugepages(virtClient, hugepageType)
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

				namespace := testsuite.GetTestNamespace(nil)
				if kvutil.IsPasstVMI(hugepagesVmi) {
					namespace = testsuite.NamespacePrivileged
				}

				By("Starting a VM")
				hugepagesVmi, err = virtClient.VirtualMachineInstance(namespace).Create(context.Background(), hugepagesVmi)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(hugepagesVmi)

				By("Checking that the VM memory equals to a number of consumed hugepages")
				Eventually(func() bool { return verifyHugepagesConsumption() }, 30*time.Second, 5*time.Second).Should(BeTrue())
			},
				Entry("[Serial][test_id:1671]hugepages-2Mi", Serial, "2Mi", "64Mi", "None", nil, nil),
				Entry("[Serial][test_id:1672]hugepages-1Gi", Serial, "1Gi", "1Gi", "None", nil, nil),
				Entry("[Serial][test_id:1672]hugepages-2Mi with guest memory set explicitly", Serial, "2Mi", "70Mi", "64Mi", nil, nil),
				Entry("[Serial]hugepages-2Mi with passt enabled", decorators.PasstGate, Serial, "2Mi", "64Mi", "None",
					libvmi.WithPasstInterfaceWithPort(), libvmi.WithNetwork(v1.DefaultPodNetwork())),
			)

			Context("with unsupported page size", func() {
				It("[test_id:1673]should failed to schedule the pod", func() {
					nodes, err := virtClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
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
					hugepagesVmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(hugepagesVmi)).Create(context.Background(), hugepagesVmi)
					Expect(err).ToNot(HaveOccurred())

					var vmiCondition v1.VirtualMachineInstanceCondition
					Eventually(func() bool {
						vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(hugepagesVmi)).Get(context.Background(), hugepagesVmi.Name, &metav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred())

						for _, cond := range vmi.Status.Conditions {
							if cond.Type == v1.VirtualMachineInstanceConditionType(kubev1.PodScheduled) && cond.Status == kubev1.ConditionFalse {
								vmiCondition = cond
								return true
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
				rngVmi = libvmi.NewAlpine(withNoRng())
			})

			It("[test_id:1674]should have the virtio rng device present when present", func() {
				rngVmi.Spec.Domain.Devices.Rng = &v1.Rng{}

				By("Starting a VirtualMachineInstance")
				rngVmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(rngVmi)).Create(context.Background(), rngVmi)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(rngVmi)

				By("Expecting the VirtualMachineInstance console")
				Expect(console.LoginToAlpine(rngVmi)).To(Succeed())

				By("Checking the virtio rng presence")
				Expect(console.SafeExpectBatch(rngVmi, []expect.Batcher{
					&expect.BSnd{S: "grep -c ^virtio /sys/devices/virtual/misc/hw_random/rng_available\n"},
					&expect.BExp{R: console.RetValue("1")},
				}, 400)).To(Succeed())
			})

			It("[test_id:1675]should not have the virtio rng device when not present", func() {
				By("Starting a VirtualMachineInstance")
				rngVmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(rngVmi)).Create(context.Background(), rngVmi)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(rngVmi)

				By("Expecting the VirtualMachineInstance console")
				Expect(console.LoginToAlpine(rngVmi)).To(Succeed())

				By("Checking the virtio rng presence")
				Expect(console.SafeExpectBatch(rngVmi, []expect.Batcher{
					&expect.BSnd{S: "[[ ! -e /sys/devices/virtual/misc/hw_random/rng_available ]] && echo non\n"},
					&expect.BExp{R: console.RetValue("non")},
				}, 400)).To(Succeed())
			})
		})

		Context("[rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]with guestAgent", func() {
			var agentVMI *v1.VirtualMachineInstance

			prepareAgentVM := func() *v1.VirtualMachineInstance {
				// TODO: actually review this once the VM image is present
				agentVMI := tests.NewRandomFedoraVMI()

				By("Starting a VirtualMachineInstance")
				agentVMI, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(agentVMI)).Create(context.Background(), agentVMI)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI successfully")
				libwait.WaitForSuccessfulVMIStart(agentVMI)

				getOptions := metav1.GetOptions{}
				var freshVMI *v1.VirtualMachineInstance

				By("VMI has the guest agent connected condition")
				Eventually(func() []v1.VirtualMachineInstanceCondition {
					freshVMI, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(agentVMI)).Get(context.Background(), agentVMI.Name, &getOptions)
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
				agentVMI = libvmi.NewAlpine()
				By("Starting a VirtualMachineInstance")
				agentVMI, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(agentVMI)).Create(context.Background(), agentVMI)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI successfully")
				libwait.WaitForSuccessfulVMIStart(agentVMI)

				getOptions := metav1.GetOptions{}
				var freshVMI *v1.VirtualMachineInstance

				freshVMI, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(agentVMI)).Get(context.Background(), agentVMI.Name, &getOptions)
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

				freshVMI, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(agentVMI)).Get(context.Background(), agentVMI.Name, &getOptions)
				Expect(err).ToNot(HaveOccurred(), "Should get VMI ")
				Expect(freshVMI.Status.Conditions).To(
					ContainElement(
						MatchFields(
							IgnoreExtras,
							Fields{"Type": Equal(v1.VirtualMachineInstanceAgentConnected)})),
					"agent should already be connected")

			})

			It("[test_id:4625]should remove condition when agent is off", func() {
				agentVMI := prepareAgentVM()
				By("Expecting the VirtualMachineInstance console")
				Expect(console.LoginToFedora(agentVMI)).To(Succeed())

				By("Terminating guest agent and waiting for it to disappear.")
				Expect(console.SafeExpectBatch(agentVMI, []expect.Batcher{
					&expect.BSnd{S: "systemctl stop qemu-guest-agent\n"},
					&expect.BExp{R: console.PromptExpression},
				}, 400)).To(Succeed())

				By("VMI has the guest agent connected condition")
				Eventually(matcher.ThisVMI(agentVMI), 240*time.Second, 2).Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstanceAgentConnected))
			})

			Context("[Serial]with cluster config changes", Serial, func() {
				BeforeEach(func() {
					kv := util.GetCurrentKv(virtClient)

					config := kv.Spec.Configuration
					config.SupportedGuestAgentVersions = []string{"X.*"}
					tests.UpdateKubeVirtConfigValueAndWait(config)
				})

				It("[test_id:5267]VMI condition should signal unsupported agent presence", func() {
					agentVMI := tests.NewRandomFedoraVMIWithBlacklistGuestAgent("guest-shutdown")
					By("Starting a VirtualMachineInstance")
					agentVMI, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(agentVMI)).Create(context.Background(), agentVMI)
					Expect(err).ToNot(HaveOccurred(), "Should create VMI successfully")
					libwait.WaitForSuccessfulVMIStart(agentVMI)

					Eventually(matcher.ThisVMI(agentVMI), 240*time.Second, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceUnsupportedAgent))
				})

				It("[test_id:6958]VMI condition should not signal unsupported agent presence for optional commands", func() {
					agentVMI := tests.NewRandomFedoraVMIWithBlacklistGuestAgent("guest-exec,guest-set-password")
					By("Starting a VirtualMachineInstance")
					agentVMI, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(agentVMI)).Create(context.Background(), agentVMI)
					Expect(err).ToNot(HaveOccurred(), "Should create VMI successfully")
					libwait.WaitForSuccessfulVMIStart(agentVMI)

					By("VMI has the guest agent connected condition")
					Eventually(matcher.ThisVMI(agentVMI), 240*time.Second, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

					By("fetching the VMI after agent has connected")
					Expect(matcher.ThisVMI(agentVMI)()).To(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstanceUnsupportedAgent))
				})
			})

			It("[test_id:4626]should have guestosinfo in status when agent is present", func() {
				agentVMI := prepareAgentVM()
				getOptions := metav1.GetOptions{}
				var updatedVmi *v1.VirtualMachineInstance
				var err error

				By("Expecting the Guest VM information")
				Eventually(func() bool {
					updatedVmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(agentVMI)).Get(context.Background(), agentVMI.Name, &getOptions)
					if err != nil {
						return false
					}
					return updatedVmi.Status.GuestOSInfo.Name != ""
				}, 240*time.Second, 2).Should(BeTrue(), "Should have guest OS Info in vmi status")

				Expect(err).ToNot(HaveOccurred())
				Expect(updatedVmi.Status.GuestOSInfo.Name).To(ContainSubstring("Fedora"))
			})

			It("[test_id:4627]should return the whole data when agent is present", func() {
				agentVMI := prepareAgentVM()

				By("Expecting the Guest VM information")
				Eventually(func() bool {
					guestInfo, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(agentVMI)).GuestOsInfo(context.Background(), agentVMI.Name)
					if err != nil {
						// invalid request, retry
						return false
					}

					return guestInfo.Hostname != "" &&
						guestInfo.Timezone != "" &&
						guestInfo.GAVersion != "" &&
						guestInfo.OS.Name != "" &&
						len(guestInfo.FSInfo.Filesystems) > 0

				}, 240*time.Second, 2).Should(BeTrue(), "Should have guest OS Info in subresource")
			})

			It("[test_id:4628]should not return the whole data when agent is not present", func() {
				agentVMI := prepareAgentVM()

				By("Expecting the VirtualMachineInstance console")
				Expect(console.LoginToFedora(agentVMI)).To(Succeed())

				By("Terminating guest agent and waiting for it to disappear.")
				Expect(console.SafeExpectBatch(agentVMI, []expect.Batcher{
					&expect.BSnd{S: "systemctl stop qemu-guest-agent\n"},
					&expect.BExp{R: console.PromptExpression},
				}, 400)).To(Succeed())

				By("Expecting the Guest VM information")
				Eventually(func() string {
					_, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(agentVMI)).GuestOsInfo(context.Background(), agentVMI.Name)
					if err != nil {
						return err.Error()
					}
					return ""
				}, 240*time.Second, 2).Should(ContainSubstring("VMI does not have guest agent connected"), "Should have not have guest info in subresource")
			})

			It("[test_id:4629]should return user list", func() {
				agentVMI := prepareAgentVM()

				Expect(console.LoginToFedora(agentVMI)).To(Succeed())

				By("Expecting the Guest VM information")
				Eventually(func() bool {
					userList, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(agentVMI)).UserList(context.Background(), agentVMI.Name)
					if err != nil {
						// invalid request, retry
						return false
					}

					return len(userList.Items) > 0 && userList.Items[0].UserName == "fedora"

				}, 240*time.Second, 2).Should(BeTrue(), "Should have fedora users")
			})

			It("[test_id:4630]should return filesystem list", func() {
				agentVMI := prepareAgentVM()

				By("Expecting the Guest VM information")
				Eventually(func() bool {
					fsList, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(agentVMI)).FilesystemList(context.Background(), agentVMI.Name)
					if err != nil {
						// invalid request, retry
						return false
					}

					return len(fsList.Items) > 0 && fsList.Items[0].DiskName != "" && fsList.Items[0].MountPoint != ""

				}, 240*time.Second, 2).Should(BeTrue(), "Should have some filesystem")
			})

		})

		Context("[rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]with serial-number", func() {
			var snVmi *v1.VirtualMachineInstance

			BeforeEach(func() {
				snVmi = libvmi.NewAlpine()
			})

			It("[test_id:3121]should have serial-number set when present", func() {
				snVmi.Spec.Domain.Firmware = &v1.Firmware{Serial: "4b2f5496-f3a3-460b-a375-168223f68845"}

				By("Starting a VirtualMachineInstance")
				snVmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(snVmi)).Create(context.Background(), snVmi)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(snVmi)

				getOptions := metav1.GetOptions{}
				var freshVMI *v1.VirtualMachineInstance

				freshVMI, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(snVmi)).Get(context.Background(), snVmi.Name, &getOptions)
				Expect(err).ToNot(HaveOccurred(), "Should get VMI ")

				domXML, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, freshVMI)
				Expect(err).ToNot(HaveOccurred(), "Should return XML from VMI")

				Expect(domXML).To(ContainSubstring("<entry name='serial'>4b2f5496-f3a3-460b-a375-168223f68845</entry>"), "Should have serial-number present")
			})

			It("[test_id:3122]should not have serial-number set when not present", func() {
				By("Starting a VirtualMachineInstance")
				snVmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(snVmi)).Create(context.Background(), snVmi)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(snVmi)

				getOptions := metav1.GetOptions{}
				var freshVMI *v1.VirtualMachineInstance

				freshVMI, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(snVmi)).Get(context.Background(), snVmi.Name, &getOptions)
				Expect(err).ToNot(HaveOccurred(), "Should get VMI ")

				domXML, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, freshVMI)
				Expect(err).ToNot(HaveOccurred(), "Should return XML from VMI")

				Expect(domXML).ToNot(ContainSubstring("<entry name='serial'>"), "Should have serial-number present")
			})
		})

		Context("with TSC timer", func() {
			featureSupportedInAtLeastOneNode := func(nodes *k8sv1.NodeList, feature string) bool {
				for _, node := range nodes.Items {
					for label := range node.Labels {
						if strings.Contains(label, services.NFD_CPU_FEATURE_PREFIX) && strings.Contains(label, feature) {
							return true
						}
					}
				}
				return false
			}
			It("[test_id:6843]should set a TSC fequency and have the CPU flag avaliable in the guest", decorators.Invtsc, decorators.TscFrequencies, func() {
				nodes := libnode.GetAllSchedulableNodes(virtClient)
				Expect(featureSupportedInAtLeastOneNode(nodes, "invtsc")).To(BeTrue(), "To run this test at least one node should support invtsc feature")
				vmi := libvmi.NewCirros()
				vmi.Spec.Domain.CPU = &v1.CPU{
					Features: []v1.CPUFeature{
						{
							Name:   "invtsc",
							Policy: "require",
						},
					},
				}
				By("Expecting the VirtualMachineInstance start")
				vmi = tests.RunVMIAndExpectLaunch(vmi, 180)

				By("Checking the TSC frequency on the VMI")
				vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Status.TopologyHints).ToNot(BeNil())
				Expect(vmi.Status.TopologyHints.TSCFrequency).ToNot(BeNil())

				By("Checking the TSC frequency on the Domain XML")
				domainSpec, err := tests.GetRunningVMIDomainSpec(vmi)
				Expect(err).ToNot(HaveOccurred())
				timerFrequency := ""
				for _, timer := range domainSpec.Clock.Timer {
					if timer.Name == "tsc" {
						timerFrequency = timer.Frequency
					}
				}
				Expect(timerFrequency).ToNot(BeEmpty())

				By("Expecting the VirtualMachineInstance console")
				Expect(console.LoginToCirros(vmi)).To(Succeed())

				By("Checking the CPU model under the guest OS")
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: fmt.Sprintf("grep '%s' /proc/cpuinfo > /dev/null\n", "nonstop_tsc")},
					&expect.BExp{R: fmt.Sprintf(console.PromptExpression)},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: console.RetValue("0")},
				}, 10)).To(Succeed())
			})
		})

		Context("with Clock and timezone", func() {

			It("[sig-compute][test_id:5268]guest should see timezone", func() {
				vmi := libvmi.NewCirros()
				timezone := "America/New_York"
				tz := v1.ClockOffsetTimezone(timezone)
				vmi.Spec.Domain.Clock = &v1.Clock{
					ClockOffset: v1.ClockOffset{
						Timezone: &tz,
					},
					Timer: &v1.Timer{},
				}

				By("Creating a VMI with timezone set")
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for successful start of VMI")
				libwait.WaitForSuccessfulVMIStart(vmi)

				By("Logging to VMI")
				Expect(console.LoginToCirros(vmi)).To(Succeed())

				loc, err := time.LoadLocation(timezone)
				Expect(err).ToNot(HaveOccurred())
				now := time.Now().In(loc)
				nowplus := now.Add(20 * time.Second)
				nowminus := now.Add(-20 * time.Second)
				By("Checking hardware clock time")
				expected := fmt.Sprintf("(%02d:%02d:|%02d:%02d:|%02d:%02d:)", nowminus.Hour(), nowminus.Minute(), now.Hour(), now.Minute(), nowplus.Hour(), nowplus.Minute())
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "sudo hwclock --localtime \n"},
					&expect.BExp{R: expected},
				}, 20)).To(Succeed(), "Expected the VM time to be within 20 seconds of "+now.String())

			})
		})

		Context("with volumes, disks and filesystem defined", func() {

			var vmi *v1.VirtualMachineInstance

			BeforeEach(func() {
				vmi = tests.NewRandomVMI()
			})

			It("[test_id:6960]should reject disk with missing volume", func() {
				const diskName = "testdisk"
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name: diskName,
				})
				_, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).To(HaveOccurred())
				const expectedErrMessage = "denied the request: spec.domain.devices.disks[0].Name '" + diskName + "' not found."
				Expect(err.Error()).To(ContainSubstring(expectedErrMessage))
			})
		})

		Context("[Serial]using defaultRuntimeClass configuration", Serial, func() {
			var runtimeClassName string

			BeforeEach(func() {
				// use random runtime class to avoid collisions with cleanup where a
				// runtime class is still in the process of being deleted because pod
				// cleanup is still in progress
				runtimeClassName = "fake-runtime-class" + "-" + rand.String(5)
				By("Creating a runtime class")
				Expect(createRuntimeClass(runtimeClassName, "fake-handler")).To(Succeed())
			})

			AfterEach(func() {
				By("Cleaning up runtime class")
				Expect(deleteRuntimeClass(runtimeClassName)).To(Succeed())
			})

			It("should apply runtimeClassName to pod when set", func() {
				By("Configuring a default runtime class")
				config := util.GetCurrentKv(virtClient).Spec.Configuration.DeepCopy()
				config.DefaultRuntimeClass = runtimeClassName
				tests.UpdateKubeVirtConfigValueAndWait(*config)

				By("Creating a new VMI")
				vmi := tests.NewRandomVMI()
				// Runtime class related warnings are expected since we created a fake runtime class that isn't supported
				wp := watcher.WarningsPolicy{FailOnWarnings: true, WarningsIgnoreList: []string{"RuntimeClass"}}
				vmi = tests.RunVMIAndExpectSchedulingWithWarningPolicy(vmi, 30, wp)

				By("Checking for presence of runtimeClassName")
				pod := tests.GetPodByVirtualMachineInstance(vmi)
				Expect(pod.Spec.RuntimeClassName).ToNot(BeNil())
				Expect(*pod.Spec.RuntimeClassName).To(BeEquivalentTo(runtimeClassName))
			})
		})
		It("should not apply runtimeClassName to pod when not set", func() {
			By("verifying no default runtime class name is set")
			config := util.GetCurrentKv(virtClient).Spec.Configuration
			Expect(config.DefaultRuntimeClass).To(BeEmpty())
			By("Creating a VMI")
			vmi := tests.RunVMIAndExpectLaunch(tests.NewRandomVMI(), 60)

			By("Checking for absence of runtimeClassName")
			pod := tests.GetRunningPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
			Expect(pod.Spec.RuntimeClassName).To(BeNil())
		})

		Context("[Serial]with geust-to-request memory ", Serial, func() {
			setHeadroom := func(ratioStr string) {
				kv := util.GetCurrentKv(virtClient)

				config := kv.Spec.Configuration
				config.AdditionalGuestMemoryOverheadRatio = &ratioStr
				tests.UpdateKubeVirtConfigValueAndWait(config)
			}

			getComputeMemoryRequest := func(vmi *virtv1.VirtualMachineInstance) resource.Quantity {
				launcherPod := tests.GetPodByVirtualMachineInstance(vmi)
				computeContainer := tests.GetComputeContainerOfPod(launcherPod)
				return computeContainer.Resources.Requests[kubev1.ResourceMemory]
			}

			It("should add guest-to-memory headroom", func() {
				const guestMemoryStr = "1024M"
				origVmiWithoutHeadroom := libvmi.New(libvmi.WithResourceMemory(guestMemoryStr))
				origVmiWithHeadroom := libvmi.New(libvmi.WithResourceMemory(guestMemoryStr))

				By("Running a vmi without additional headroom")
				vmiWithoutHeadroom := tests.RunVMIAndExpectScheduling(origVmiWithoutHeadroom, 60)

				By("Setting a headroom ratio in Kubevirt CR")
				const ratio = "1.567"
				setHeadroom(ratio)

				By("Running a vmi with additional headroom")
				vmiWithHeadroom := tests.RunVMIAndExpectScheduling(origVmiWithHeadroom, 60)

				requestWithoutHeadroom := getComputeMemoryRequest(vmiWithoutHeadroom)
				requestWithHeadroom := getComputeMemoryRequest(vmiWithHeadroom)

				overheadWithoutHeadroom := services.GetMemoryOverhead(vmiWithoutHeadroom, runtime.GOARCH, nil)
				overheadWithHeadroom := services.GetMemoryOverhead(vmiWithoutHeadroom, runtime.GOARCH, pointer.String(ratio))

				expectedDiffBetweenRequests := overheadWithHeadroom.DeepCopy()
				expectedDiffBetweenRequests.Sub(overheadWithoutHeadroom)

				actualDiffBetweenRequests := requestWithHeadroom.DeepCopy()
				actualDiffBetweenRequests.Sub(requestWithoutHeadroom)

				By("Ensuring memory request is as expected")
				const errFmt = "ratio: %s, request without headroom: %s, request with headroom: %s, overhead without headroom: %s, overhead with headroom: %s, expected diff between requests: %s, actual diff between requests: %s"
				Expect(actualDiffBetweenRequests.Cmp(expectedDiffBetweenRequests)).To(Equal(0),
					fmt.Sprintf(errFmt, ratio, requestWithoutHeadroom.String(), requestWithHeadroom.String(), overheadWithoutHeadroom.String(), overheadWithHeadroom.String(), expectedDiffBetweenRequests.String(), actualDiffBetweenRequests.String()))

				By("Ensure no memory specifications had been changed on VMIs")
				Expect(origVmiWithHeadroom.Spec.Domain.Resources).To(Equal(vmiWithHeadroom.Spec.Domain.Resources), "vmi resources are not expected to change")
				Expect(origVmiWithHeadroom.Spec.Domain.Memory).To(Equal(vmiWithHeadroom.Spec.Domain.Memory), "vmi guest memory is not expected to change")
				Expect(origVmiWithoutHeadroom.Spec.Domain.Resources).To(Equal(vmiWithoutHeadroom.Spec.Domain.Resources), "vmi resources are not expected to change")
				Expect(origVmiWithoutHeadroom.Spec.Domain.Memory).To(Equal(vmiWithoutHeadroom.Spec.Domain.Memory), "vmi guest memory is not expected to change")
			})
		})
	})

	Context("[rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]with CPU spec", func() {
		var cpuVmi *v1.VirtualMachineInstance
		var nodes *k8sv1.NodeList

		parseCPUNiceName := func(name string) string {
			updatedCPUName := strings.Replace(name, "\n", "", -1)
			if strings.Contains(updatedCPUName, ":") {
				updatedCPUName = strings.Split(name, ":")[1]

			}
			updatedCPUName = strings.Replace(updatedCPUName, " ", "", 1)
			updatedCPUName = strings.Replace(updatedCPUName, "(", "", -1)
			updatedCPUName = strings.Replace(updatedCPUName, ")", "", -1)

			updatedCPUName = strings.Split(updatedCPUName, "-")[0]
			updatedCPUName = strings.Split(updatedCPUName, "_")[0]

			for i, char := range updatedCPUName {
				if unicode.IsUpper(char) && i != 0 {
					updatedCPUName = strings.Split(updatedCPUName, string(char))[0]
				}
			}
			return updatedCPUName
		}

		BeforeEach(func() {
			nodes = libnode.GetAllSchedulableNodes(virtClient)
			Expect(nodes.Items).ToNot(BeEmpty(), "There should be some compute node")

			cpuVmi = libvmi.NewCirros()
		})

		Context("[rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]when CPU model defined", func() {
			It("[test_id:1678]should report defined CPU model", func() {
				supportedCPUs := tests.GetSupportedCPUModels(*nodes)
				Expect(supportedCPUs).ToNot(BeEmpty())
				cpuVmi.Spec.Domain.CPU = &v1.CPU{
					Model: supportedCPUs[0],
				}
				niceName := parseCPUNiceName(supportedCPUs[0])

				By("Starting a VirtualMachineInstance")
				cpuVmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(cpuVmi)).Create(context.Background(), cpuVmi)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(cpuVmi)

				By("Expecting the VirtualMachineInstance console")
				Expect(console.LoginToCirros(cpuVmi)).To(Succeed())

				By("Checking the CPU model under the guest OS")
				Expect(console.SafeExpectBatch(cpuVmi, []expect.Batcher{
					&expect.BSnd{S: fmt.Sprintf("grep %s /proc/cpuinfo\n", niceName)},
					&expect.BExp{R: fmt.Sprintf(".*model name.*%s.*", niceName)},
				}, 10)).To(Succeed())
			})
		})

		Context("[rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]when CPU model equals to passthrough", func() {
			It("[test_id:1679]should report exactly the same model as node CPU", func() {
				cpuVmi.Spec.Domain.CPU = &v1.CPU{
					Model: "host-passthrough",
				}

				By("Starting a VirtualMachineInstance")
				cpuVmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(cpuVmi)).Create(context.Background(), cpuVmi)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(cpuVmi)

				By("Checking the CPU model under the guest OS")
				output := tests.RunCommandOnVmiPod(cpuVmi, []string{"grep", "-m1", "model name", "/proc/cpuinfo"})

				niceName := parseCPUNiceName(output)

				By("Expecting the VirtualMachineInstance console")
				Expect(console.LoginToCirros(cpuVmi)).To(Succeed())

				By("Checking the CPU model under the guest OS")
				Expect(console.SafeExpectBatch(cpuVmi, []expect.Batcher{
					&expect.BSnd{S: fmt.Sprintf("grep '%s' /proc/cpuinfo\n", niceName)},
					&expect.BExp{R: fmt.Sprintf(".*model name.*%s.*", niceName)},
				}, 10)).To(Succeed())
			})
		})

		Context("[rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]when CPU model not defined", func() {
			It("[test_id:1680]should report CPU model from libvirt capabilities", func() {
				By("Starting a VirtualMachineInstance")
				cpuVmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(cpuVmi)).Create(context.Background(), cpuVmi)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(cpuVmi)

				output := tests.RunCommandOnVmiPod(cpuVmi, []string{"grep", "-m1", "model name", "/proc/cpuinfo"})

				niceName := parseCPUNiceName(output)

				By("Expecting the VirtualMachineInstance console")
				Expect(console.LoginToCirros(cpuVmi)).To(Succeed())

				By("Checking the CPU model under the guest OS")
				console.SafeExpectBatch(cpuVmi, []expect.Batcher{
					&expect.BSnd{S: fmt.Sprintf("grep '%s' /proc/cpuinfo\n", niceName)},
					&expect.BExp{R: fmt.Sprintf(".*model name.*%s.*", niceName)},
				}, 10)
			})
		})

		Context("when CPU features defined", func() {
			It("[test_id:3123]should start a Virtual Machine with matching features", func() {
				supportedCPUFeatures := tests.GetSupportedCPUFeatures(*nodes)
				Expect(supportedCPUFeatures).ToNot(BeEmpty())
				cpuVmi.Spec.Domain.CPU = &v1.CPU{
					Features: []v1.CPUFeature{
						{
							Name: supportedCPUFeatures[0],
						},
					},
				}

				By("Starting a VirtualMachineInstance")
				cpuVmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(cpuVmi)).Create(context.Background(), cpuVmi)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(cpuVmi)

				By("Expecting the VirtualMachineInstance console")
				Expect(console.LoginToCirros(cpuVmi)).To(Succeed())
			})
		})
	})

	Context("[Serial][rfe_id:2869][crit:medium][vendor:cnv-qe@redhat.com][level:component]with machine type settings", Serial, func() {
		testEmulatedMachines := []string{"q35*", "pc-q35*", "pc*"}

		BeforeEach(func() {
			kv := util.GetCurrentKv(virtClient)

			config := kv.Spec.Configuration
			config.MachineType = ""
			config.ArchitectureConfiguration = &v1.ArchConfiguration{Amd64: &v1.ArchSpecificConfiguration{}, Arm64: &v1.ArchSpecificConfiguration{}, Ppc64le: &v1.ArchSpecificConfiguration{}}
			config.ArchitectureConfiguration.Amd64.EmulatedMachines = testEmulatedMachines
			config.ArchitectureConfiguration.Arm64.EmulatedMachines = testEmulatedMachines
			config.ArchitectureConfiguration.Ppc64le.EmulatedMachines = testEmulatedMachines

			tests.UpdateKubeVirtConfigValueAndWait(config)
		})

		It("[test_id:3124]should set machine type from VMI spec", func() {
			vmi := libvmi.New(
				libvmi.WithResourceMemory(enoughMemForSafeBiosEmulation),
				withMachineType("pc"),
			)
			vmi = tests.RunVMIAndExpectLaunch(vmi, 30)
			runningVMISpec, err := tests.GetRunningVMIDomainSpec(vmi)

			Expect(err).ToNot(HaveOccurred())
			Expect(runningVMISpec.OS.Type.Machine).To(ContainSubstring("pc-i440"))

			Expect(vmi.Status.Machine).ToNot(BeNil())
			Expect(vmi.Status.Machine.Type).To(Equal(runningVMISpec.OS.Type.Machine))
		})

		It("[test_id:3125]should allow creating VM without Machine defined", func() {
			vmi := tests.NewRandomVMI()
			vmi.Spec.Domain.Machine = nil
			tests.RunVMIAndExpectLaunch(vmi, 30)
			runningVMISpec, err := tests.GetRunningVMIDomainSpec(vmi)

			Expect(err).ToNot(HaveOccurred())
			Expect(runningVMISpec.OS.Type.Machine).To(ContainSubstring("q35"))
		})

		It("[test_id:6964]should allow creating VM defined with Machine with an empty Type", func() {
			// This is needed to provide backward compatibility since our example VMIs used to be defined in this way
			vmi := libvmi.New(
				libvmi.WithResourceMemory(enoughMemForSafeBiosEmulation),
				withMachineType(""),
			)

			vmi = tests.RunVMIAndExpectLaunch(vmi, 30)
			runningVMISpec, err := tests.GetRunningVMIDomainSpec(vmi)

			Expect(err).ToNot(HaveOccurred())
			Expect(runningVMISpec.OS.Type.Machine).To(ContainSubstring("q35"))
		})

		It("[Serial][test_id:3126]should set machine type from kubevirt-config", Serial, func() {
			kv := util.GetCurrentKv(virtClient)
			testEmulatedMachines := []string{"pc"}

			config := kv.Spec.Configuration

			config.ArchitectureConfiguration = &v1.ArchConfiguration{Amd64: &v1.ArchSpecificConfiguration{}, Arm64: &v1.ArchSpecificConfiguration{}, Ppc64le: &v1.ArchSpecificConfiguration{}}
			config.ArchitectureConfiguration.Amd64.MachineType = "pc"
			config.ArchitectureConfiguration.Arm64.MachineType = "pc"
			config.ArchitectureConfiguration.Ppc64le.MachineType = "pc"
			config.ArchitectureConfiguration.Amd64.EmulatedMachines = testEmulatedMachines
			config.ArchitectureConfiguration.Arm64.EmulatedMachines = testEmulatedMachines
			config.ArchitectureConfiguration.Ppc64le.EmulatedMachines = testEmulatedMachines
			tests.UpdateKubeVirtConfigValueAndWait(config)

			vmi := tests.NewRandomVMI()
			vmi.Spec.Domain.Machine = nil
			tests.RunVMIAndExpectLaunch(vmi, 30)
			runningVMISpec, err := tests.GetRunningVMIDomainSpec(vmi)

			Expect(err).ToNot(HaveOccurred())
			Expect(runningVMISpec.OS.Type.Machine).To(ContainSubstring("pc-i440"))
		})
	})

	Context("with a custom scheduler", func() {
		It("[test_id:4631]should set the custom scheduler on the pod", func() {
			vmi := libvmi.New(
				libvmi.WithResourceMemory(enoughMemForSafeBiosEmulation),
				WithSchedulerName("my-custom-scheduler"),
			)
			runningVMI := tests.RunVMIAndExpectScheduling(vmi, 30)
			launcherPod, err := libvmi.GetPodByVirtualMachineInstance(runningVMI, testsuite.GetTestNamespace(vmi))
			Expect(err).ToNot(HaveOccurred())
			Expect(launcherPod.Spec.SchedulerName).To(Equal("my-custom-scheduler"))
		})
	})

	Context("[rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]with CPU request settings", func() {

		It("[test_id:3127]should set CPU request from VMI spec", func() {
			vmi := libvmi.New(
				libvmi.WithResourceMemory(enoughMemForSafeBiosEmulation),
				libvmi.WithResourceCPU("500m"),
			)
			runningVMI := tests.RunVMIAndExpectScheduling(vmi, 30)

			readyPod, err := libvmi.GetPodByVirtualMachineInstance(runningVMI, testsuite.GetTestNamespace(vmi))
			Expect(err).ToNot(HaveOccurred())
			computeContainer := tests.GetComputeContainerOfPod(readyPod)
			cpuRequest := computeContainer.Resources.Requests[kubev1.ResourceCPU]
			Expect(cpuRequest.String()).To(Equal("500m"))
		})

		It("[test_id:3128]should set CPU request when it is not provided", func() {
			vmi := tests.NewRandomVMI()
			runningVMI := tests.RunVMIAndExpectScheduling(vmi, 30)

			readyPod, err := libvmi.GetPodByVirtualMachineInstance(runningVMI, testsuite.GetTestNamespace(vmi))
			Expect(err).ToNot(HaveOccurred())
			computeContainer := tests.GetComputeContainerOfPod(readyPod)
			cpuRequest := computeContainer.Resources.Requests[kubev1.ResourceCPU]
			Expect(cpuRequest.String()).To(Equal("100m"))
		})

		It("[Serial][test_id:3129]should set CPU request from kubevirt-config", Serial, func() {
			kv := util.GetCurrentKv(virtClient)

			config := kv.Spec.Configuration
			configureCPURequest := resource.MustParse("800m")
			config.CPURequest = &configureCPURequest
			tests.UpdateKubeVirtConfigValueAndWait(config)

			vmi := tests.NewRandomVMI()
			runningVMI := tests.RunVMIAndExpectScheduling(vmi, 30)

			readyPod, err := libvmi.GetPodByVirtualMachineInstance(runningVMI, testsuite.GetTestNamespace(vmi))
			Expect(err).ToNot(HaveOccurred())
			computeContainer := tests.GetComputeContainerOfPod(readyPod)
			cpuRequest := computeContainer.Resources.Requests[kubev1.ResourceCPU]
			Expect(cpuRequest.String()).To(Equal("800m"))
		})
	})

	Context("[Serial]with automatic CPU limit configured in the CR", Serial, func() {
		BeforeEach(func() {
			By("Adding a label selector to the CR for auto CPU limit")
			kv := util.GetCurrentKv(virtClient)
			config := kv.Spec.Configuration
			config.AutoCPULimitNamespaceLabelSelector = &metav1.LabelSelector{
				MatchLabels: map[string]string{"autocpulimit": "true"},
			}
			tests.UpdateKubeVirtConfigValueAndWait(config)
		})
		It("should not set a CPU limit if the namespace doesn't match the selector", func() {
			By("Creating a running VMI")
			vmi := tests.NewRandomVMI()
			runningVMI := tests.RunVMIAndExpectScheduling(vmi, 30)

			By("Ensuring no CPU limit is set")
			readyPod, err := libvmi.GetPodByVirtualMachineInstance(runningVMI, testsuite.GetTestNamespace(vmi))
			Expect(err).ToNot(HaveOccurred())
			computeContainer := tests.GetComputeContainerOfPod(readyPod)
			_, exists := computeContainer.Resources.Limits[kubev1.ResourceCPU]
			Expect(exists).To(BeFalse(), "CPU limit set on the compute container when none was expected")
		})
		It("should set a CPU limit if the namespace matches the selector", func() {
			By("Creating a VMI object")
			vmi := tests.NewRandomVMI()

			By("Adding the right label to VMI namespace")
			namespace, err := virtClient.CoreV1().Namespaces().Get(context.Background(), vmi.Namespace, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			namespace.Labels["autocpulimit"] = "true"
			namespace, err = virtClient.CoreV1().Namespaces().Update(context.Background(), namespace, metav1.UpdateOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("Starting the VMI")
			runningVMI := tests.RunVMIAndExpectScheduling(vmi, 30)

			By("Ensuring the CPU limit is set to the correct value")
			readyPod, err := libvmi.GetPodByVirtualMachineInstance(runningVMI, testsuite.GetTestNamespace(vmi))
			Expect(err).ToNot(HaveOccurred())
			computeContainer := tests.GetComputeContainerOfPod(readyPod)
			limits, exists := computeContainer.Resources.Limits[kubev1.ResourceCPU]
			Expect(exists).To(BeTrue(), "expected CPU limit not set on the compute container")
			Expect(limits.String()).To(Equal("1"))
		})
	})

	Context("with automatic resource limits FG enabled", decorators.AutoResourceLimitsGate, func() {

		When("there is no ResourceQuota with memory and cpu limits associated with the creation namespace", func() {
			It("should not automatically set memory limits in the virt-launcher pod", func() {
				vmi := libvmi.NewCirros()
				By("Creating a running VMI")
				runningVMI := tests.RunVMIAndExpectScheduling(vmi, 30)

				By("Ensuring no memory and cpu limits are set")
				readyPod, err := libvmi.GetPodByVirtualMachineInstance(runningVMI, testsuite.GetTestNamespace(vmi))
				Expect(err).ToNot(HaveOccurred())
				computeContainer := tests.GetComputeContainerOfPod(readyPod)
				_, exists := computeContainer.Resources.Limits[kubev1.ResourceMemory]
				Expect(exists).To(BeFalse(), "Memory limits set on the compute container when none was expected")
				_, exists = computeContainer.Resources.Limits[kubev1.ResourceCPU]
				Expect(exists).To(BeFalse(), "CPU limits set on the compute container when none was expected")
			})
		})

		When("a ResourceQuota with memory and cpu limits is associated to the creation namespace", func() {
			var (
				vmi                       *virtv1.VirtualMachineInstance
				expectedLauncherMemLimits *resource.Quantity
				expectedLauncherCPULimits resource.Quantity
				vmiRequest                resource.Quantity
			)

			BeforeEach(func() {
				vmiRequest = resource.MustParse("256Mi")
				delta := resource.MustParse("100Mi")
				vmi = libvmi.NewCirros(
					libvmi.WithResourceMemory(vmiRequest.String()),
					libvmi.WithCPUCount(1, 1, 1),
				)
				vmiPodRequest := services.GetMemoryOverhead(vmi, runtime.GOARCH, nil)
				vmiPodRequest.Add(vmiRequest)
				value := int64(float64(vmiPodRequest.Value()) * services.DefaultMemoryLimitOverheadRatio)

				expectedLauncherMemLimits = resource.NewQuantity(value, vmiPodRequest.Format)
				expectedLauncherCPULimits = resource.MustParse("1")

				// Add a delta to not saturate the rq
				rqLimit := expectedLauncherMemLimits.DeepCopy()
				rqLimit.Add(delta)
				By("Creating a Resource Quota with memory limits")
				rq := &kubev1.ResourceQuota{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:    testsuite.GetTestNamespace(nil),
						GenerateName: "test-quota",
					},
					Spec: k8sv1.ResourceQuotaSpec{
						Hard: kubev1.ResourceList{
							k8sv1.ResourceLimitsMemory: resource.MustParse(rqLimit.String()),
							k8sv1.ResourceLimitsCPU:    resource.MustParse("1500m"),
						},
					},
				}
				_, err := virtClient.CoreV1().ResourceQuotas(testsuite.GetTestNamespace(nil)).Create(context.Background(), rq, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			It("should set a memory limit in the virt-launcher pod", func() {
				By("Starting the VMI")
				runningVMI := tests.RunVMIAndExpectScheduling(vmi, 30)

				By("Ensuring the memory and cpu limits are set to the correct values")
				readyPod, err := libvmi.GetPodByVirtualMachineInstance(runningVMI, testsuite.GetTestNamespace(vmi))
				Expect(err).ToNot(HaveOccurred())
				computeContainer := tests.GetComputeContainerOfPod(readyPod)
				memLimits, exists := computeContainer.Resources.Limits[kubev1.ResourceMemory]
				Expect(exists).To(BeTrue(), "expected memory limits set on the compute container")
				Expect(memLimits.Value()).To(BeEquivalentTo(expectedLauncherMemLimits.Value()))
				cpuLimits, exists := computeContainer.Resources.Limits[kubev1.ResourceCPU]
				Expect(exists).To(BeTrue(), "expected cpu limits set on the compute container")
				Expect(cpuLimits.Value()).To(BeEquivalentTo(expectedLauncherCPULimits.Value()))
			})
		})
	})

	Context("[Serial][rfe_id:904][crit:medium][vendor:cnv-qe@redhat.com][level:component][storage-req]with driver cache and io settings and PVC", Serial, decorators.StorageReq, func() {
		var dataVolume *cdiv1.DataVolume

		BeforeEach(func() {
			var err error
			if !checks.HasFeature(virtconfig.HostDiskGate) {
				Skip("Cluster has the HostDisk featuregate disabled, skipping  the tests")
			}

			dataVolume, err = createBlockDataVolume(virtClient)
			Expect(err).ToNot(HaveOccurred())
			if dataVolume == nil {
				Skip("Skip test when Block storage is not present")
			}

			libstorage.EventuallyDV(dataVolume, 240, Or(HaveSucceeded(), BeInPhase(cdiv1.WaitForFirstConsumer)))
		})

		AfterEach(func() {
			libstorage.DeleteDataVolume(&dataVolume)
		})

		It("[test_id:1681]should set appropriate cache modes", func() {
			vmi := tests.NewRandomVMI()

			By("adding disks to a VMI")
			tests.AddEphemeralDisk(vmi, "ephemeral-disk1", v1.DiskBusVirtio, cd.ContainerDiskFor(cd.ContainerDiskCirros))
			vmi.Spec.Domain.Devices.Disks[0].Cache = v1.CacheNone

			tests.AddEphemeralDisk(vmi, "ephemeral-disk2", v1.DiskBusVirtio, cd.ContainerDiskFor(cd.ContainerDiskCirros))
			vmi.Spec.Domain.Devices.Disks[1].Cache = v1.CacheWriteThrough

			tests.AddEphemeralDisk(vmi, "ephemeral-disk5", v1.DiskBusVirtio, cd.ContainerDiskFor(cd.ContainerDiskCirros))
			vmi.Spec.Domain.Devices.Disks[2].Cache = v1.CacheWriteBack

			tests.AddEphemeralDisk(vmi, "ephemeral-disk3", v1.DiskBusVirtio, cd.ContainerDiskFor(cd.ContainerDiskCirros))
			tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\necho 'hello'\n")
			tmpHostDiskDir := tests.RandTmpDir()
			tests.AddHostDisk(vmi, filepath.Join(tmpHostDiskDir, "test-disk.img"), v1.HostDiskExistsOrCreate, "hostdisk")
			tests.RunVMIAndExpectLaunch(vmi, 60)
			runningVMISpec, err := tests.GetRunningVMIDomainSpec(vmi)
			Expect(err).ToNot(HaveOccurred())
			vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
			defer tests.RemoveHostDiskImage(tmpHostDiskDir, vmiPod.Spec.NodeName)

			disks := runningVMISpec.Devices.Disks
			By("checking if number of attached disks is equal to real disks number")
			Expect(vmi.Spec.Domain.Devices.Disks).To(HaveLen(len(disks)))

			cacheNone := string(v1.CacheNone)
			cacheWritethrough := string(v1.CacheWriteThrough)
			cacheWriteback := string(v1.CacheWriteBack)

			By("checking if requested cache 'none' has been set")
			Expect(disks[0].Alias.GetName()).To(Equal("ephemeral-disk1"))
			Expect(disks[0].Driver.Cache).To(Equal(cacheNone))

			By("checking if requested cache 'writethrough' has been set")
			Expect(disks[1].Alias.GetName()).To(Equal("ephemeral-disk2"))
			Expect(disks[1].Driver.Cache).To(Equal(cacheWritethrough))

			By("checking if requested cache 'writeback' has been set")
			Expect(disks[2].Alias.GetName()).To(Equal("ephemeral-disk5"))
			Expect(disks[2].Driver.Cache).To(Equal(cacheWriteback))

			By("checking if default cache 'none' has been set to ephemeral disk")
			Expect(disks[3].Alias.GetName()).To(Equal("ephemeral-disk3"))
			Expect(disks[3].Driver.Cache).To(Equal(cacheNone))

			By("checking if default cache 'none' has been set to cloud-init disk")
			Expect(disks[4].Alias.GetName()).To(Equal("cloud-init"))
			Expect(disks[4].Driver.Cache).To(Equal(cacheNone))

			By("checking if default cache 'writethrough' has been set to fs which does not support direct I/O")
			Expect(disks[5].Alias.GetName()).To(Equal("hostdisk"))
			Expect(disks[5].Driver.Cache).To(Equal(cacheWritethrough))

		})

		It("[test_id:5360]should set appropriate IO modes", func() {
			vmi := tests.NewRandomVMI()

			By("adding disks to a VMI")
			// disk[0]:  File, sparsed, no user-input, cache=none
			tests.AddEphemeralDisk(vmi, "ephemeral-disk1", v1.DiskBusVirtio, cd.ContainerDiskFor(cd.ContainerDiskCirros))
			vmi.Spec.Domain.Devices.Disks[0].Cache = v1.CacheNone

			// disk[1]:  Block, no user-input, cache=none
			tests.AddPVCDisk(vmi, "block-pvc", v1.DiskBusVirtio, dataVolume.Name)

			// disk[2]: File, not-sparsed, no user-input, cache=none
			tests.AddPVCDisk(vmi, "hostpath-pvc", v1.DiskBusVirtio, tests.DiskAlpineHostPath)

			// disk[3]:  File, sparsed, user-input=threads, cache=none
			tests.AddEphemeralDisk(vmi, "ephemeral-disk2", v1.DiskBusVirtio, cd.ContainerDiskFor(cd.ContainerDiskCirros))
			vmi.Spec.Domain.Devices.Disks[3].Cache = v1.CacheNone
			vmi.Spec.Domain.Devices.Disks[3].IO = v1.IOThreads

			tests.RunVMIAndExpectLaunch(vmi, 60)
			runningVMISpec, err := tests.GetRunningVMIDomainSpec(vmi)
			Expect(err).ToNot(HaveOccurred())

			disks := runningVMISpec.Devices.Disks
			By("checking if number of attached disks is equal to real disks number")
			Expect(vmi.Spec.Domain.Devices.Disks).To(HaveLen(len(disks)))

			ioNative := v1.IONative
			ioThreads := v1.IOThreads
			ioNone := ""

			By("checking if default io has not been set for sparsed file")
			Expect(disks[0].Alias.GetName()).To(Equal("ephemeral-disk1"))
			Expect(string(disks[0].Driver.IO)).To(Equal(ioNone))

			By("checking if default io mode has been set to 'native' for block device")
			Expect(disks[1].Alias.GetName()).To(Equal("block-pvc"))
			Expect(disks[1].Driver.IO).To(Equal(ioNative))

			By("checking if default cache 'none' has been set to pvc disk")
			Expect(disks[2].Alias.GetName()).To(Equal("hostpath-pvc"))
			// PVC is mounted as tmpfs on kind, which does not support direct I/O.
			// As such, it behaves as plugging in a hostDisk - check disks[6].
			if checks.IsRunningOnKindInfra() {
				// The cache mode is set to cacheWritethrough
				Expect(string(disks[2].Driver.IO)).To(Equal(ioNone))
			} else {
				// The cache mode is set to cacheNone
				Expect(disks[2].Driver.IO).To(Equal(ioNative))
			}

			By("checking if requested io mode 'threads' has been set")
			Expect(disks[3].Alias.GetName()).To(Equal("ephemeral-disk2"))
			Expect(disks[3].Driver.IO).To(Equal(ioThreads))

		})
	})

	Context("Block size configuration set", func() {

		It("[test_id:6965][storage-req]Should set BlockIO when using custom block sizes", decorators.StorageReq, func() {
			By("creating a block volume")
			dataVolume, err := createBlockDataVolume(virtClient)
			Expect(err).ToNot(HaveOccurred())
			if dataVolume == nil {
				Skip("Skip test when Block storage is not present")
			}

			libstorage.EventuallyDV(dataVolume, 240, Or(HaveSucceeded(), BeInPhase(cdiv1.WaitForFirstConsumer)))

			vmi := tests.NewRandomVMIWithPVC(dataVolume.Name)

			By("setting the disk to use custom block sizes")
			logicalSize := uint(16384)
			physicalSize := uint(16384)
			vmi.Spec.Domain.Devices.Disks[0].BlockSize = &v1.BlockSize{
				Custom: &v1.CustomBlockSize{
					Logical:  logicalSize,
					Physical: physicalSize,
				},
			}

			By("initializing the VM")
			tests.RunVMIAndExpectLaunch(vmi, 60)
			runningVMISpec, err := tests.GetRunningVMIDomainSpec(vmi)
			Expect(err).ToNot(HaveOccurred())

			By("checking if number of attached disks is equal to real disks number")
			disks := runningVMISpec.Devices.Disks
			Expect(vmi.Spec.Domain.Devices.Disks).To(HaveLen(len(disks)))

			By("checking if BlockIO is set to the custom block size")
			Expect(disks[0].Alias.GetName()).To(Equal("disk0"))
			Expect(disks[0].BlockIO).ToNot(BeNil())
			Expect(disks[0].BlockIO.LogicalBlockSize).To(Equal(logicalSize))
			Expect(disks[0].BlockIO.PhysicalBlockSize).To(Equal(physicalSize))
		})

		It("[test_id:6966][storage-req]Should set BlockIO when set to match volume block sizes on block devices", decorators.StorageReq, func() {
			By("creating a block volume")
			dataVolume, err := createBlockDataVolume(virtClient)
			Expect(err).ToNot(HaveOccurred())
			if dataVolume == nil {
				Skip("Skip test when Block storage is not present")
			}

			libstorage.EventuallyDV(dataVolume, 240, Or(HaveSucceeded(), BeInPhase(cdiv1.WaitForFirstConsumer)))

			vmi := tests.NewRandomVMIWithPVC(dataVolume.Name)

			By("setting the disk to match the volume block sizes")
			vmi.Spec.Domain.Devices.Disks[0].BlockSize = &v1.BlockSize{
				MatchVolume: &v1.FeatureState{},
			}

			By("initializing the VM")
			tests.RunVMIAndExpectLaunch(vmi, 60)
			runningVMISpec, err := tests.GetRunningVMIDomainSpec(vmi)
			Expect(err).ToNot(HaveOccurred())

			By("checking if number of attached disks is equal to real disks number")
			disks := runningVMISpec.Devices.Disks
			Expect(vmi.Spec.Domain.Devices.Disks).To(HaveLen(len(disks)))

			By("checking if BlockIO is set for the disk")
			Expect(disks[0].Alias.GetName()).To(Equal("disk0"))
			Expect(disks[0].BlockIO).ToNot(BeNil())
			// Block devices should be one of 512n, 512e or 4096n so accept 512 and 4096 values.
			expectedDiskSizes := SatisfyAny(Equal(uint(512)), Equal(uint(4096)))
			Expect(disks[0].BlockIO.LogicalBlockSize).To(expectedDiskSizes)
			Expect(disks[0].BlockIO.PhysicalBlockSize).To(expectedDiskSizes)
		})

		It("[test_id:6967]Should set BlockIO when set to match volume block sizes on files", func() {
			if !checks.HasFeature(virtconfig.HostDiskGate) {
				Skip("Cluster has the HostDisk featuregate disabled, skipping  the tests")
			}

			By("creating a disk image")
			var nodeName string
			tmpHostDiskDir := tests.RandTmpDir()
			tmpHostDiskPath := filepath.Join(tmpHostDiskDir, fmt.Sprintf("disk-%s.img", uuid.NewRandom().String()))

			job := tests.CreateHostDiskImage(tmpHostDiskPath)
			job, err := virtClient.CoreV1().Pods(testsuite.NamespacePrivileged).Create(context.Background(), job, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(ThisPod(job), 30*time.Second, 1*time.Second).Should(BeInPhase(k8sv1.PodSucceeded))
			pod, err := ThisPod(job)()
			Expect(err).NotTo(HaveOccurred())
			nodeName = pod.Spec.NodeName
			defer tests.RemoveHostDiskImage(tmpHostDiskDir, nodeName)

			vmi := tests.NewRandomVMIWithHostDisk(tmpHostDiskPath, v1.HostDiskExistsOrCreate, nodeName)

			By("setting the disk to match the volume block sizes")
			vmi.Spec.Domain.Devices.Disks[0].BlockSize = &v1.BlockSize{
				MatchVolume: &v1.FeatureState{},
			}

			By("initializing the VM")
			tests.RunVMIAndExpectLaunch(vmi, 60)
			runningVMISpec, err := tests.GetRunningVMIDomainSpec(vmi)
			Expect(err).ToNot(HaveOccurred())

			By("checking if number of attached disks is equal to real disks number")
			disks := runningVMISpec.Devices.Disks
			Expect(vmi.Spec.Domain.Devices.Disks).To(HaveLen(len(disks)))

			By("checking if BlockIO is set for the disk")
			Expect(disks[0].Alias.GetName()).To(Equal("host-disk"))
			Expect(disks[0].BlockIO).ToNot(BeNil())
			// The default for most filesystems nowadays is 4096 but it can be changed.
			// As such, relying on a specific value is flakey.
			// As long as we have a value, the exact value doesn't matter.
			Expect(disks[0].BlockIO.LogicalBlockSize).ToNot(BeZero())
			// A filesystem only has a single size so logical == physical
			Expect(disks[0].BlockIO.LogicalBlockSize).To(Equal(disks[0].BlockIO.PhysicalBlockSize))
		})
	})

	Context("[rfe_id:898][crit:medium][vendor:cnv-qe@redhat.com][level:component]New VirtualMachineInstance with all supported drives", func() {

		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			// ordering:
			// use a small disk for the other ones
			containerImage := cd.ContainerDiskFor(cd.ContainerDiskCirros)
			// virtio - added by NewRandomVMIWithEphemeralDisk
			vmi = tests.NewRandomVMIWithEphemeralDiskAndUserdata(containerImage, "echo hi!\n")
			// sata
			tests.AddEphemeralDisk(vmi, "disk2", v1.DiskBusSATA, containerImage)
			// NOTE: we have one disk per bus, so we expect vda, sda
		})
		checkPciAddress := func(vmi *v1.VirtualMachineInstance, expectedPciAddress string) {
			err := console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "grep DEVNAME /sys/bus/pci/devices/" + expectedPciAddress + "/*/block/vda/uevent|awk -F= '{ print $2 }'\n"},
				&expect.BExp{R: "vda"},
			}, 15)
			Expect(err).ToNot(HaveOccurred())
		}

		It("[test_id:1682]should have all the device nodes", func() {
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi)

			Expect(console.LoginToCirros(vmi)).To(Succeed())

			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				// keep the ordering!
				&expect.BSnd{S: "ls /dev/vda  /dev/vdb\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "echo $?\n"},
				&expect.BExp{R: console.RetValue("0")},
			}, 10)).To(Succeed())
		})

		It("[test_id:3906]should configure custom Pci address", func() {
			By("checking disk1 Pci address")
			vmi.Spec.Domain.Devices.Disks[0].Disk.PciAddress = "0000:00:10.0"
			vmi.Spec.Domain.Devices.Disks[0].Disk.Bus = v1.DiskBusVirtio
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitUntilVMIReady(vmi, console.LoginToCirros)

			checkPciAddress(vmi, vmi.Spec.Domain.Devices.Disks[0].Disk.PciAddress)
		})

		It("[test_id:1020]should not create the VM with wrong PCI address", func() {
			By("setting disk1 Pci address")

			wrongPciAddress := "0000:04:10.0"

			vmi.Spec.Domain.Devices.Disks[0].Disk.PciAddress = wrongPciAddress
			vmi.Spec.Domain.Devices.Disks[0].Disk.Bus = v1.DiskBusVirtio
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred())

			var vmiCondition v1.VirtualMachineInstanceCondition
			// TODO
			Eventually(func() bool {
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				for _, cond := range vmi.Status.Conditions {
					if cond.Type == v1.VirtualMachineInstanceConditionType(v1.VirtualMachineInstanceSynchronized) && cond.Status == kubev1.ConditionFalse {
						vmiCondition = cond
						return true
					}
				}
				return false
			}, 120*time.Second, time.Second).Should(BeTrue())

			Expect(vmiCondition.Message).To(ContainSubstring("Invalid PCI address " + wrongPciAddress))
			Expect(vmiCondition.Reason).To(Equal("Synchronizing with the Domain failed."))
		})
	})
	Describe("[rfe_id:897][crit:medium][arm64][vendor:cnv-qe@redhat.com][level:component]VirtualMachineInstance with CPU pinning", func() {
		var nodes *kubev1.NodeList

		isNodeHasCPUManagerLabel := func(nodeName string) bool {
			Expect(nodeName).ToNot(BeEmpty())

			nodeObject, err := virtClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
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
			var err error
			checks.SkipTestIfNoCPUManager()
			nodes, err = virtClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			if len(nodes.Items) == 1 {
				Skip("Skip cpu pinning test that requires multiple nodes when only one node is present.")
			}
		})

		Context("[Serial]with cpu pinning enabled", Serial, func() {

			It("[test_id:1684]should set the cpumanager label to false when it's not running", func() {

				By("adding a cpumanger=true label to a node")
				nodes, err := virtClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{LabelSelector: v1.CPUManager + "=" + "false"})
				Expect(err).ToNot(HaveOccurred())
				if len(nodes.Items) == 0 {
					Skip("Skip CPU manager test on clusters where CPU manager is running on all worker/compute nodes")
				}

				node := &nodes.Items[0]
				node, err = virtClient.CoreV1().Nodes().Patch(context.Background(), node.Name, types.StrategicMergePatchType, []byte(fmt.Sprintf(`{"metadata": { "labels": {"%s": "true"}}}`, v1.CPUManager)), metav1.PatchOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("setting the cpumanager label back to false")
				Eventually(func() string {
					n, err := virtClient.CoreV1().Nodes().Get(context.Background(), node.Name, metav1.GetOptions{})
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
				cpuVmi := libvmi.NewCirros()
				cpuVmi.Spec.Domain.CPU = &v1.CPU{
					Cores:                 2,
					DedicatedCPUPlacement: true,
				}

				By("Starting a VirtualMachineInstance")
				cpuVmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(cpuVmi)).Create(context.Background(), cpuVmi)
				Expect(err).ToNot(HaveOccurred())
				node := libwait.WaitForSuccessfulVMIStart(cpuVmi).Status.NodeName

				By("Checking that the VMI QOS is guaranteed")
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(cpuVmi)).Get(context.Background(), cpuVmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Status.QOSClass).ToNot(BeNil())
				Expect(*vmi.Status.QOSClass).To(Equal(kubev1.PodQOSGuaranteed))

				Expect(isNodeHasCPUManagerLabel(node)).To(BeTrue())

				By("Checking that the pod QOS is guaranteed")
				readyPod := tests.GetRunningPodByVirtualMachineInstance(cpuVmi, testsuite.GetTestNamespace(cpuVmi))
				podQos := readyPod.Status.QOSClass
				Expect(podQos).To(Equal(kubev1.PodQOSGuaranteed))

				var computeContainer *kubev1.Container
				for _, container := range readyPod.Spec.Containers {
					if container.Name == "compute" {
						computeContainer = &container
					}
				}
				if computeContainer == nil {
					util.PanicOnError(fmt.Errorf("could not find the compute container"))
				}

				output, err := tests.GetPodCPUSet(readyPod)
				log.Log.Infof("%v", output)
				Expect(err).ToNot(HaveOccurred())
				output = strings.TrimSuffix(output, "\n")
				pinnedCPUsList, err := hw_utils.ParseCPUSetLine(output, 100)
				Expect(err).ToNot(HaveOccurred())

				Expect(pinnedCPUsList).To(HaveLen(int(cpuVmi.Spec.Domain.CPU.Cores)))

				By("Expecting the VirtualMachineInstance console")
				Expect(console.LoginToCirros(cpuVmi)).To(Succeed())

				By("Checking the number of CPU cores under guest OS")
				Expect(console.SafeExpectBatch(cpuVmi, []expect.Batcher{
					&expect.BSnd{S: "grep -c ^processor /proc/cpuinfo\n"},
					&expect.BExp{R: "2"},
				}, 15)).To(Succeed())

				By("Check values in domain XML")
				domXML, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, cpuVmi)
				Expect(err).ToNot(HaveOccurred(), "Should return XML from VMI")
				Expect(domXML).To(ContainSubstring("<hint-dedicated state='on'/>"), "should container the hint-dedicated feature")
			})
			It("[test_id:4632]should be able to start a vm with guest memory different from requested and keep guaranteed qos", func() {
				Skip("Skip test till issue https://github.com/kubevirt/kubevirt/issues/3910 is fixed")
				cpuVmi := libvmi.NewCirros()
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
				cpuVmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(cpuVmi)).Create(context.Background(), cpuVmi)
				Expect(err).ToNot(HaveOccurred())
				node := libwait.WaitForSuccessfulVMIStart(cpuVmi).Status.NodeName

				By("Checking that the VMI QOS is guaranteed")
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(cpuVmi)).Get(context.Background(), cpuVmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Status.QOSClass).ToNot(BeNil())
				Expect(*vmi.Status.QOSClass).To(Equal(kubev1.PodQOSGuaranteed))

				Expect(isNodeHasCPUManagerLabel(node)).To(BeTrue())

				By("Checking that the pod QOS is guaranteed")
				readyPod := tests.GetRunningPodByVirtualMachineInstance(cpuVmi, testsuite.GetTestNamespace(vmi))
				podQos := readyPod.Status.QOSClass
				Expect(podQos).To(Equal(kubev1.PodQOSGuaranteed))

				//-------------------------------------------------------------------
				Expect(console.LoginToCirros(vmi)).To(Succeed())

				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "[ $(free -m | grep Mem: | tr -s ' ' | cut -d' ' -f2) -lt 80 ] && echo 'pass'\n"},
					&expect.BExp{R: console.RetValue("pass")},
					&expect.BSnd{S: "swapoff -a && dd if=/dev/zero of=/dev/shm/test bs=1k count=118k\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: console.RetValue("0")},
				}, 15)).To(Succeed())

				pod := tests.GetRunningPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
				podMemoryUsage, err := getPodMemoryUsage(pod)
				Expect(err).ToNot(HaveOccurred())
				By("Converting pod memory usage")
				m, err := strconv.Atoi(strings.Trim(podMemoryUsage, "\n"))
				Expect(err).ToNot(HaveOccurred())
				By("Checking if pod memory usage is > 80Mi")
				Expect(m).To(BeNumerically(">", 83886080), "83886080 B = 80 Mi")
			})
			DescribeTable("[test_id:4023]should start a vmi with dedicated cpus and isolated emulator thread", func(resources *v1.ResourceRequirements) {
				cpuVmi := libvmi.NewCirros()
				cpuVmi.Spec.Domain.CPU = &v1.CPU{
					Cores:                 2,
					DedicatedCPUPlacement: true,
					IsolateEmulatorThread: true,
				}
				if resources != nil {
					cpuVmi.Spec.Domain.Resources = *resources
				}

				By("Starting a VirtualMachineInstance")
				cpuVmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(cpuVmi)).Create(context.Background(), cpuVmi)
				Expect(err).ToNot(HaveOccurred())
				node := libwait.WaitForSuccessfulVMIStart(cpuVmi).Status.NodeName

				By("Checking that the VMI QOS is guaranteed")
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(cpuVmi)).Get(context.Background(), cpuVmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Status.QOSClass).ToNot(BeNil())
				Expect(*vmi.Status.QOSClass).To(Equal(kubev1.PodQOSGuaranteed))

				Expect(isNodeHasCPUManagerLabel(node)).To(BeTrue())

				By("Checking that the pod QOS is guaranteed")
				readyPod := tests.GetRunningPodByVirtualMachineInstance(cpuVmi, testsuite.GetTestNamespace(vmi))
				podQos := readyPod.Status.QOSClass
				Expect(podQos).To(Equal(kubev1.PodQOSGuaranteed))

				var computeContainer *kubev1.Container
				for _, container := range readyPod.Spec.Containers {
					if container.Name == "compute" {
						computeContainer = &container
					}
				}
				if computeContainer == nil {
					util.PanicOnError(fmt.Errorf("could not find the compute container"))
				}

				output, err := tests.GetPodCPUSet(readyPod)
				log.Log.Infof("%v", output)
				Expect(err).ToNot(HaveOccurred())
				output = strings.TrimSuffix(output, "\n")
				pinnedCPUsList, err := hw_utils.ParseCPUSetLine(output, 100)
				Expect(err).ToNot(HaveOccurred())

				output, err = tests.ListCgroupThreads(readyPod)
				Expect(err).ToNot(HaveOccurred())
				pids := strings.Split(output, "\n")

				getProcessNameErrors := 0
				By("Expecting only vcpu threads on root of pod cgroup")
				for _, pid := range pids {
					if len(pid) == 0 {
						continue
					}
					output, err = tests.GetProcessName(readyPod, pid)
					if err != nil {
						getProcessNameErrors++
						continue
					}
					Expect(output).To(ContainSubstring("CPU "))
					Expect(output).To(ContainSubstring("KVM"))
				}
				Expect(getProcessNameErrors).Should(BeNumerically("<=", 1))

				// 1 additioan pcpus should be allocated on the pod for the emulation threads
				Expect(pinnedCPUsList).To(HaveLen(int(cpuVmi.Spec.Domain.CPU.Cores) + 1))

				By("Expecting the VirtualMachineInstance console")
				Expect(console.LoginToCirros(cpuVmi)).To(Succeed())

				By("Checking the number of CPU cores under guest OS")
				Expect(console.SafeExpectBatch(cpuVmi, []expect.Batcher{
					&expect.BSnd{S: "grep -c ^processor /proc/cpuinfo\n"},
					&expect.BExp{R: "2"},
				}, 15)).To(Succeed())

				emulator, err := tests.GetRunningVMIEmulator(vmi)
				Expect(err).ToNot(HaveOccurred())
				emulator = filepath.Base(emulator)

				virtClient := kubevirt.Client()
				pidCmd := []string{"pidof", emulator}
				qemuPid, err := exec.ExecuteCommandOnPod(virtClient, readyPod, "compute", pidCmd)
				// do not check for kvm-pit thread if qemu is not in use
				if err != nil {
					return
				}
				kvmpitmask, err := tests.GetKvmPitMask(strings.TrimSpace(qemuPid), node)
				Expect(err).ToNot(HaveOccurred())

				vcpuzeromask, err := tests.GetVcpuMask(readyPod, emulator, "0")
				Expect(err).ToNot(HaveOccurred())

				Expect(kvmpitmask).To(Equal(vcpuzeromask))
			},
				Entry(" with explicit resources set", &virtv1.ResourceRequirements{
					Requests: kubev1.ResourceList{
						kubev1.ResourceCPU:    resource.MustParse("2"),
						kubev1.ResourceMemory: resource.MustParse("256Mi"),
					},
					Limits: kubev1.ResourceList{
						kubev1.ResourceCPU:    resource.MustParse("2"),
						kubev1.ResourceMemory: resource.MustParse("256Mi"),
					},
				}),
				Entry("without resource requirements set", nil),
			)

			It("[test_id:4024]should fail the vmi creation if IsolateEmulatorThread requested without dedicated cpus", func() {
				cpuVmi := libvmi.NewCirros()
				cpuVmi.Spec.Domain.CPU = &v1.CPU{
					Cores:                 2,
					IsolateEmulatorThread: true,
				}

				By("Starting a VirtualMachineInstance")
				cpuVmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(cpuVmi)).Create(context.Background(), cpuVmi)
				Expect(err).To(HaveOccurred())
			})

			It("[test_id:802]should configure correct number of vcpus with requests.cpus", func() {
				cpuVmi := libvmi.NewCirros()
				cpuVmi.Spec.Domain.CPU = &v1.CPU{
					DedicatedCPUPlacement: true,
				}
				cpuVmi.Spec.Domain.Resources.Requests[k8sv1.ResourceCPU] = resource.MustParse("2")

				By("Starting a VirtualMachineInstance")
				cpuVmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(cpuVmi)).Create(context.Background(), cpuVmi)
				Expect(err).ToNot(HaveOccurred())
				node := libwait.WaitForSuccessfulVMIStart(cpuVmi).Status.NodeName
				Expect(isNodeHasCPUManagerLabel(node)).To(BeTrue())

				By("Expecting the VirtualMachineInstance console")
				Expect(console.LoginToCirros(cpuVmi)).To(Succeed())

				By("Checking the number of CPU cores under guest OS")
				Expect(console.SafeExpectBatch(cpuVmi, []expect.Batcher{
					&expect.BSnd{S: "grep -c ^processor /proc/cpuinfo\n"},
					&expect.BExp{R: "2"},
				}, 15)).To(Succeed())
			})

			It("[test_id:1688]should fail the vmi creation if the requested resources are inconsistent", func() {
				cpuVmi := libvmi.NewCirros()
				cpuVmi.Spec.Domain.CPU = &v1.CPU{
					Cores:                 2,
					DedicatedCPUPlacement: true,
				}

				cpuVmi.Spec.Domain.Resources.Requests[k8sv1.ResourceCPU] = resource.MustParse("3")

				By("Starting a VirtualMachineInstance")
				cpuVmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(cpuVmi)).Create(context.Background(), cpuVmi)
				Expect(err).To(HaveOccurred())
			})
			It("[test_id:1689]should fail the vmi creation if cpu is not an integer", func() {
				cpuVmi := libvmi.NewCirros()
				cpuVmi.Spec.Domain.CPU = &v1.CPU{
					DedicatedCPUPlacement: true,
				}

				cpuVmi.Spec.Domain.Resources.Requests[k8sv1.ResourceCPU] = resource.MustParse("300m")

				By("Starting a VirtualMachineInstance")
				cpuVmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(cpuVmi)).Create(context.Background(), cpuVmi)
				Expect(err).To(HaveOccurred())
			})
			It("[test_id:1690]should fail the vmi creation if Guaranteed QOS cannot be set", func() {
				cpuVmi := libvmi.NewCirros()
				cpuVmi.Spec.Domain.CPU = &v1.CPU{
					DedicatedCPUPlacement: true,
				}
				cpuVmi.Spec.Domain.Resources.Requests[k8sv1.ResourceCPU] = resource.MustParse("2")
				cpuVmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Limits: kubev1.ResourceList{
						kubev1.ResourceCPU: resource.MustParse("4"),
					},
				}
				By("Starting a VirtualMachineInstance")
				cpuVmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(cpuVmi)).Create(context.Background(), cpuVmi)
				Expect(err).To(HaveOccurred())
			})
			It("[test_id:830]should start a vm with no cpu pinning after a vm with cpu pinning on same node", func() {
				Vmi := libvmi.NewCirros()
				cpuVmi := libvmi.NewCirros()
				cpuVmi.Spec.Domain.CPU = &v1.CPU{
					DedicatedCPUPlacement: true,
				}

				cpuVmi.Spec.Domain.Resources.Requests[k8sv1.ResourceCPU] = resource.MustParse("2")
				Vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceCPU] = resource.MustParse("1")
				Vmi.Spec.NodeSelector = map[string]string{v1.CPUManager: "true"}

				By("Starting a VirtualMachineInstance with dedicated cpus")
				cpuVmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(cpuVmi)).Create(context.Background(), cpuVmi)
				Expect(err).ToNot(HaveOccurred())
				node := libwait.WaitForSuccessfulVMIStart(cpuVmi).Status.NodeName
				Expect(isNodeHasCPUManagerLabel(node)).To(BeTrue())

				By("Starting a VirtualMachineInstance without dedicated cpus")
				Vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(cpuVmi)).Create(context.Background(), Vmi)
				Expect(err).ToNot(HaveOccurred())
				node = libwait.WaitForSuccessfulVMIStart(Vmi).Status.NodeName
				Expect(isNodeHasCPUManagerLabel(node)).To(BeTrue())
			})
		})

		Context("[Serial]cpu pinning with fedora images, dedicated and non dedicated cpu should be possible on same node via spec.domain.cpu.cores", Serial, func() {

			var cpuvmi, vmi *v1.VirtualMachineInstance
			var node string

			BeforeEach(func() {

				nodes := libnode.GetAllSchedulableNodes(virtClient)
				Expect(nodes.Items).ToNot(BeEmpty(), "There should be some nodes")
				node = nodes.Items[1].Name

				vmi = libvmi.NewFedora()

				cpuvmi = libvmi.NewFedora()
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
				cpuvmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(cpuvmi)).Create(context.Background(), cpuvmi)
				Expect(err).ToNot(HaveOccurred())
				node1 := libwait.WaitForSuccessfulVMIStart(cpuvmi).Status.NodeName
				Expect(isNodeHasCPUManagerLabel(node1)).To(BeTrue())
				Expect(node1).To(Equal(node))

				By("Expecting the VirtualMachineInstance console")
				Expect(console.LoginToFedora(cpuvmi)).To(Succeed())

				By("Starting a VirtualMachineInstance without dedicated cpus")
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred())
				node2 := libwait.WaitForSuccessfulVMIStart(vmi).Status.NodeName
				Expect(isNodeHasCPUManagerLabel(node2)).To(BeTrue())
				Expect(node2).To(Equal(node))

				By("Expecting the VirtualMachineInstance console")
				Expect(console.LoginToFedora(vmi)).To(Succeed())
			})

			It("[test_id:832]should start a vm with cpu pinning after a vm with no cpu pinning on same node", func() {

				By("Starting a VirtualMachineInstance without dedicated cpus")
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred())
				node2 := libwait.WaitForSuccessfulVMIStart(vmi).Status.NodeName
				Expect(isNodeHasCPUManagerLabel(node2)).To(BeTrue())
				Expect(node2).To(Equal(node))

				By("Expecting the VirtualMachineInstance console")
				Expect(console.LoginToFedora(vmi)).To(Succeed())

				By("Starting a VirtualMachineInstance with dedicated cpus")
				cpuvmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(cpuvmi)).Create(context.Background(), cpuvmi)
				Expect(err).ToNot(HaveOccurred())
				node1 := libwait.WaitForSuccessfulVMIStart(cpuvmi).Status.NodeName
				Expect(isNodeHasCPUManagerLabel(node1)).To(BeTrue())
				Expect(node1).To(Equal(node))

				By("Expecting the VirtualMachineInstance console")
				Expect(console.LoginToFedora(cpuvmi)).To(Succeed())
			})
		})
	})

	Context("[rfe_id:2926][crit:medium][vendor:cnv-qe@redhat.com][level:component]Check Chassis value", func() {

		It("[Serial][test_id:2927]Test Chassis value in a newly created VM", Serial, func() {
			vmi := tests.NewRandomFedoraVMIWithEphemeralDiskHighMemory()
			vmi.Spec.Domain.Chassis = &v1.Chassis{
				Asset: "Test-123",
			}

			By("Starting a VirtualMachineInstance")
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi)

			By("Check values on domain XML")
			domXml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(domXml).To(ContainSubstring("<entry name='asset'>Test-123</entry>"))

			By("Expecting console")
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			By("Check value in VM with dmidecode")
			// Check on the VM, if expected values are there with dmidecode
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "[ $(sudo dmidecode -s chassis-asset-tag | tr -s ' ') = Test-123 ] && echo 'pass'\n"},
				&expect.BExp{R: console.RetValue("pass")},
			}, 10)).To(Succeed())
		})
	})

	Context("[Serial][rfe_id:2926][crit:medium][vendor:cnv-qe@redhat.com][level:component]Check SMBios with default and custom values", Serial, func() {

		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = tests.NewRandomFedoraVMIWithEphemeralDiskHighMemory()
		})

		It("[test_id:2751]test default SMBios", func() {
			kv := util.GetCurrentKv(virtClient)

			config := kv.Spec.Configuration
			// Clear SMBios values if already set in kubevirt-config, for testing default values.
			test_smbios := &v1.SMBiosConfiguration{Family: "", Product: "", Manufacturer: ""}
			config.SMBIOSConfig = test_smbios
			tests.UpdateKubeVirtConfigValueAndWait(config)

			By("Starting a VirtualMachineInstance")
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi)

			By("Check values in domain XML")
			domXml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(domXml).To(ContainSubstring("<entry name='family'>KubeVirt</entry>"))
			Expect(domXml).To(ContainSubstring("<entry name='product'>None</entry>"))
			Expect(domXml).To(ContainSubstring("<entry name='manufacturer'>KubeVirt</entry>"))

			By("Expecting console")
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			By("Check values in dmidecode")
			// Check on the VM, if expected values are there with dmidecode
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "[ $(sudo dmidecode -s system-family | tr -s ' ') = KubeVirt ] && echo 'pass'\n"},
				&expect.BExp{R: console.RetValue("pass")},
				&expect.BSnd{S: "[ $(sudo dmidecode -s system-product-name | tr -s ' ') = None ] && echo 'pass'\n"},
				&expect.BExp{R: console.RetValue("pass")},
				&expect.BSnd{S: "[ $(sudo dmidecode -s system-manufacturer | tr -s ' ') = KubeVirt ] && echo 'pass'\n"},
				&expect.BExp{R: console.RetValue("pass")},
			}, 1)).To(Succeed())
		})

		It("[test_id:2752]test custom SMBios values", func() {
			kv := util.GetCurrentKv(virtClient)
			config := kv.Spec.Configuration
			// Set a custom test SMBios
			test_smbios := &v1.SMBiosConfiguration{Family: "test", Product: "test", Manufacturer: "None", Sku: "1.0", Version: "1.0"}
			config.SMBIOSConfig = test_smbios
			tests.UpdateKubeVirtConfigValueAndWait(config)

			By("Starting a VirtualMachineInstance")
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi)

			domXml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(domXml).To(ContainSubstring("<entry name='family'>test</entry>"))
			Expect(domXml).To(ContainSubstring("<entry name='product'>test</entry>"))
			Expect(domXml).To(ContainSubstring("<entry name='manufacturer'>None</entry>"))
			Expect(domXml).To(ContainSubstring("<entry name='sku'>1.0</entry>"))
			Expect(domXml).To(ContainSubstring("<entry name='version'>1.0</entry>"))

			By("Expecting console")
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			By("Check values in dmidecode")

			// Check on the VM, if expected values are there with dmidecode
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "[ $(sudo dmidecode -s system-family | tr -s ' ') = test ] && echo 'pass'\n"},
				&expect.BExp{R: console.RetValue("pass")},
				&expect.BSnd{S: "[ $(sudo dmidecode -s system-product-name | tr -s ' ') = test ] && echo 'pass'\n"},
				&expect.BExp{R: console.RetValue("pass")},
				&expect.BSnd{S: "[ $(sudo dmidecode -s system-manufacturer | tr -s ' ') = None ] && echo 'pass'\n"},
				&expect.BExp{R: console.RetValue("pass")},
			}, 1)).To(Succeed())
		})
	})

	Context("With ephemeral CD-ROM", func() {
		var vmi *v1.VirtualMachineInstance
		var DiskBusIDE v1.DiskBus = "ide"

		BeforeEach(func() {
			vmi = tests.NewRandomFedoraVMIWithEphemeralDiskHighMemory()
		})

		DescribeTable("For various bus types", func(bus v1.DiskBus, errMsg string) {
			tests.AddEphemeralCdrom(vmi, "cdrom-0", bus, cd.ContainerDiskFor(cd.ContainerDiskCirros))

			By(fmt.Sprintf("Starting a VMI with a %s CD-ROM", bus))
			_, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			if errMsg == "" {
				Expect(err).ToNot(HaveOccurred())
			} else {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(errMsg))
			}
		},
			Entry("[test_id:3777] Should be accepted when using sata", v1.DiskBusSATA, ""),
			Entry("[test_id:3809] Should be accepted when using scsi", v1.DiskBusSCSI, ""),
			Entry("[test_id:3776] Should be rejected when using virtio", v1.DiskBusVirtio, "Bus type virtio is invalid"),
			Entry("[test_id:3808] Should be rejected when using ide", DiskBusIDE, "IDE bus is not supported"),
		)
	})

	Context("Custom PCI Addresses configuration", func() {
		// The aim of the test is to validate the configurability of a range of PCI slots
		// on the root PCI bus 0. We would like to test slots 2..1a (slots 0,1 and beyond 1a are reserved).
		// In addition , we test usage of PCI functions on a single slot
		// by occupying all the functions 1..7 on random port 2.

		addrPrefix := "0000:00" // PCI bus 0
		numOfSlotsToTest := 24  // slots 2..1a
		numOfFuncsToTest := 8
		var vmi *v1.VirtualMachineInstance

		createDisks := func(numOfDisks int, vmi *v1.VirtualMachineInstance) {
			for i := 0; i < numOfDisks; i++ {
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks,
					v1.Disk{
						Name: fmt.Sprintf("test%v", i),
						DiskDevice: v1.DiskDevice{
							Disk: &v1.DiskTarget{
								Bus: v1.DiskBusVirtio,
							},
						},
					})
				vmi.Spec.Volumes = append(vmi.Spec.Volumes,
					v1.Volume{
						Name: fmt.Sprintf("test%v", i),
						VolumeSource: v1.VolumeSource{
							EmptyDisk: &v1.EmptyDiskSource{
								Capacity: resource.MustParse("1Mi"),
							},
						},
					})
			}
		}
		assignDisksToSlots := func(startIndex int, vmi *v1.VirtualMachineInstance) {
			var addr string

			for i, disk := range vmi.Spec.Domain.Devices.Disks {
				addr = fmt.Sprintf("%x", i+startIndex)
				if len(addr) == 1 {
					disk.DiskDevice.Disk.PciAddress = fmt.Sprintf("%s:0%v.0", addrPrefix, addr)
				} else {
					disk.DiskDevice.Disk.PciAddress = fmt.Sprintf("%s:%v.0", addrPrefix, addr)
				}
			}
		}

		assignDisksToFunctions := func(startIndex int, vmi *v1.VirtualMachineInstance) {
			for i, disk := range vmi.Spec.Domain.Devices.Disks {
				disk.DiskDevice.Disk.PciAddress = fmt.Sprintf("%s:02.%v", addrPrefix, fmt.Sprintf("%x", i+startIndex))
			}
		}

		BeforeEach(func() {
			var bootOrder uint = 1
			vmi = tests.NewRandomFedoraVMI()
			vmi.Spec.Domain.Resources.Requests[kubev1.ResourceMemory] = resource.MustParse("1024M")
			vmi.Spec.Domain.Devices.Disks[0].BootOrder = &bootOrder
		})

		DescribeTable("should configure custom pci address", func(startIndex, numOfDevices int, testingPciFunctions bool) {
			currentDisks := len(vmi.Spec.Domain.Devices.Disks)
			numOfDisksToAdd := numOfDevices - currentDisks

			createDisks(numOfDisksToAdd, vmi)
			if testingPciFunctions {
				assignDisksToFunctions(startIndex, vmi)
			} else {
				tests.DisableFeatureGate(virtconfig.ExpandDisksGate)
				assignDisksToSlots(startIndex, vmi)
			}
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitUntilVMIReady(vmi, console.LoginToFedora)
			Expect(vmi.Spec.Domain.Devices.Disks).Should(HaveLen(numOfDevices))

			err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Delete(context.Background(), vmi.Name, &metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())
		},
			Entry("[Serial][test_id:5269]across all available PCI root bus slots", Serial, 2, numOfSlotsToTest, false),
			Entry("[test_id:5270]across all available PCI functions of a single slot", 0, numOfFuncsToTest, true),
		)
	})

	Context("Check KVM CPUID advertisement", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			checks.SkipIfRunningOnKindInfra("Skip KVM MSR prescence test on kind")

			vmi = tests.NewRandomFedoraVMIWithEphemeralDiskHighMemory()
		})

		It("[test_id:5271]test cpuid hidden", func() {
			vmi.Spec.Domain.Features = &v1.Features{
				KVM: &v1.FeatureKVM{Hidden: true},
			}

			By("Starting a VirtualMachineInstance")
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi)

			By("Check values in domain XML")
			domXml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(domXml).To(ContainSubstring("<hidden state='on'/>"))

			By("Expecting console")
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			By("Check virt-what-cpuid-helper does not match KVM")
			Expect(console.ExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "/usr/libexec/virt-what-cpuid-helper > /dev/null 2>&1 && echo 'pass'\n"},
				&expect.BExp{R: console.RetValue("pass")},
				&expect.BSnd{S: "$(sudo /usr/libexec/virt-what-cpuid-helper | grep -q KVMKVMKVM) || echo 'pass'\n"},
				&expect.BExp{R: console.RetValue("pass")},
			}, 2*time.Second)).To(Succeed())
		})

		It("[test_id:5272]test cpuid default", func() {
			By("Starting a VirtualMachineInstance")
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi)

			By("Expecting console")
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			By("Check virt-what-cpuid-helper matches KVM")
			Expect(console.ExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "/usr/libexec/virt-what-cpuid-helper > /dev/null 2>&1 && echo 'pass'\n"},
				&expect.BExp{R: console.RetValue("pass")},
				&expect.BSnd{S: "$(sudo /usr/libexec/virt-what-cpuid-helper | grep -q KVMKVMKVM) && echo 'pass'\n"},
				&expect.BExp{R: console.RetValue("pass")},
			}, 1*time.Second)).To(Succeed())
		})
	})
	Context("virt-launcher processes memory usage", func() {
		doesntExceedMemoryUsage := func(processRss *map[string]resource.Quantity, process string, memoryLimit resource.Quantity) {
			actual := (*processRss)[process]
			ExpectWithOffset(1, (&actual).Cmp(memoryLimit)).To(Equal(-1),
				"the %s process is taking too much RAM! (%s > %s). All processes: %v",
				process, actual.String(), memoryLimit.String(), processRss)
		}
		It("should be lower than allocated size", func() {
			By("Starting a VirtualMachineInstance")
			vmi := tests.NewRandomFedoraVMI()
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi)

			By("Expecting console")
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			By("Running ps in virt-launcher")
			pods, err := virtClient.CoreV1().Pods(vmi.Namespace).List(context.Background(), metav1.ListOptions{
				LabelSelector: v1.CreatedByLabel + "=" + string(vmi.GetUID()),
			})
			Expect(err).ToNot(HaveOccurred(), "Should list pods successfully")
			var stdout, stderr string
			errorMassageFormat := "failed after running the `ps` command with stdout:\n %v \n stderr:\n %v \n err: \n %v \n"
			Eventually(func() error {
				stdout, stderr, err = exec.ExecuteCommandOnPodWithResults(virtClient, &pods.Items[0], "compute",
					[]string{
						"ps",
						"--no-header",
						"axo",
						"rss,command",
					})
				return err
			}, time.Second, 50*time.Millisecond).Should(BeNil(), fmt.Sprintf(errorMassageFormat, stdout, stderr, err))

			By("Parsing the output of ps")
			processRss := make(map[string]resource.Quantity)
			scanner := bufio.NewScanner(strings.NewReader(stdout))
			for scanner.Scan() {
				fields := strings.Fields(scanner.Text())
				Expect(len(fields)).To(BeNumerically(">=", 2))
				rss := fields[0]
				command := filepath.Base(fields[1])
				// Handle the qemu binary: e.g. qemu-kvm or qemu-system-x86_64
				if command == "qemu-kvm" || strings.HasPrefix(command, "qemu-system-") {
					command = "qemu"
				}
				switch command {
				case "virt-launcher-monitor", "virt-launcher", "virtlogd", "virtqemud", "qemu":
					Expect(processRss).ToNot(HaveKey(command), "multiple %s processes found", command)
					value := resource.MustParse(rss + "Ki")
					processRss[command] = value
				}
			}
			for _, process := range []string{"virt-launcher-monitor", "virt-launcher", "virtlogd", "virtqemud", "qemu"} {
				Expect(processRss).To(HaveKey(process), "no %s process found", process)
			}

			By("Ensuring no process is using too much ram")
			doesntExceedMemoryUsage(&processRss, "virt-launcher-monitor", resource.MustParse(services.VirtLauncherMonitorOverhead))
			doesntExceedMemoryUsage(&processRss, "virt-launcher", resource.MustParse(services.VirtLauncherOverhead))
			doesntExceedMemoryUsage(&processRss, "virtlogd", resource.MustParse(services.VirtlogdOverhead))
			doesntExceedMemoryUsage(&processRss, "virtqemud", resource.MustParse(services.VirtqemudOverhead))
			qemuExpected := resource.MustParse(services.QemuOverhead)
			qemuExpected.Add(vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory])
			doesntExceedMemoryUsage(&processRss, "qemu", qemuExpected)
		})
	})

	Context("When topology spread constraints are defined for the VMI", func() {
		It("they should be applied to the launcher pod", func() {
			vmi := libvmi.NewCirros()
			tsc := []k8sv1.TopologySpreadConstraint{
				{
					MaxSkew:           1,
					TopologyKey:       "zone",
					WhenUnsatisfiable: "DoNotSchedule",
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"foo": "bar",
						},
					},
				},
			}
			vmi.Spec.TopologySpreadConstraints = tsc

			By("Starting a VirtualMachineInstance")
			vmi = tests.RunVMIAndExpectScheduling(vmi, 30)

			By("Ensuring that pod has expected topologySpreadConstraints")
			pod := tests.GetPodByVirtualMachineInstance(vmi)
			Expect(pod.Spec.TopologySpreadConstraints).To(Equal(tsc))
		})
	})
})

func createRuntimeClass(name, handler string) error {
	virtCli := kubevirt.Client()

	_, err := virtCli.NodeV1().RuntimeClasses().Create(
		context.Background(),
		&nodev1.RuntimeClass{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Handler:    handler,
		},
		metav1.CreateOptions{},
	)
	return err
}

func deleteRuntimeClass(name string) error {
	virtCli := kubevirt.Client()

	return virtCli.NodeV1().RuntimeClasses().Delete(context.Background(), name, metav1.DeleteOptions{})
}

func withNoRng() libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.Devices.Rng = nil
	}
}

func overcommitGuestOverhead() libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.Resources.OvercommitGuestOverhead = true
	}
}

func withMachineType(machineType string) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.Machine = &v1.Machine{Type: machineType}
	}
}

func WithSchedulerName(schedulerName string) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.SchedulerName = schedulerName
	}
}

func withSerialBIOS() libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Firmware == nil {
			vmi.Spec.Domain.Firmware = &v1.Firmware{}
		}
		if vmi.Spec.Domain.Firmware.Bootloader == nil {
			vmi.Spec.Domain.Firmware.Bootloader = &v1.Bootloader{}
		}
		if vmi.Spec.Domain.Firmware.Bootloader.BIOS == nil {
			vmi.Spec.Domain.Firmware.Bootloader.BIOS = &v1.BIOS{}
		}
		vmi.Spec.Domain.Firmware.Bootloader.BIOS.UseSerial = pointer.Bool(true)
	}
}

func createBlockDataVolume(virtClient kubecli.KubevirtClient) (*cdiv1.DataVolume, error) {
	sc, foundSC := libstorage.GetBlockStorageClass(k8sv1.ReadWriteOnce)
	if !foundSC {
		return nil, nil
	}

	dataVolume := libdv.NewDataVolume(
		libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros)),
		libdv.WithPVC(libdv.PVCWithStorageClass(sc), libdv.PVCWithBlockVolumeMode()),
	)

	return virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dataVolume, metav1.CreateOptions{})
}
