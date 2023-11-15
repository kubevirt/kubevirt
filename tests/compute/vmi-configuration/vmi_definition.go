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
 * Copyright the KubeVirt Authors.
 *
 */

package vmi_configuration

import (
	"context"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests/libpod"

	"k8s.io/apimachinery/pkg/util/rand"

	"kubevirt.io/kubevirt/pkg/virt-controller/services"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	k8sv1 "k8s.io/api/core/v1"
	kubev1 "k8s.io/api/core/v1"
	nodev1 "k8s.io/api/node/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	v1 "kubevirt.io/api/core/v1"
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	kvutil "kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/util"

	"kubevirt.io/kubevirt/pkg/util/cluster"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
	"kubevirt.io/kubevirt/tests/watcher"
)

var _ = ConfigDescribe("VirtualMachineInstance definition", func() {

	const enoughMemForSafeBiosEmulation = "32Mi"
	const (
		cgroupV1MemoryUsagePath = "/sys/fs/cgroup/memory/memory.usage_in_bytes"
		cgroupV2MemoryUsagePath = "/sys/fs/cgroup/memory.current"
	)

	var virtClient kubecli.KubevirtClient

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
				libvmi.WithSerialBIOS(),
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
			logs := func() string { return libpod.GetVirtLauncherLogs(virtClient, vmi) }
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
			vmi := libvmi.NewCirros(libvmi.WithOvercommitGuestOverhead())
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
			vmi = libvmi.NewCirros(libvmi.WithOvercommitGuestOverhead())
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
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = libvmi.NewAlpine(libvmi.WithoutRNG())
		})

		It("[test_id:1674]should have the virtio rng device present when present", func() {
			vmi.Spec.Domain.Devices.Rng = &v1.Rng{}

			By("Starting a VirtualMachineInstance")
			rngVmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
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
			rngVmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
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

		createRuntimeClass := func(name, handler string) error {
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

		deleteRuntimeClass := func(name string) error {
			virtCli := kubevirt.Client()

			return virtCli.NodeV1().RuntimeClasses().Delete(context.Background(), name, metav1.DeleteOptions{})
		}

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
