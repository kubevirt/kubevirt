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
 * Copyright The KubeVirt Authors.
 *
 */

package compute

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	kvconfig "kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("CPU", func() {
	const enoughMemForSafeBiosEmulation = "32Mi"
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("[rfe_id:2065][crit:medium][vendor:cnv-qe@redhat.com][level:component]with 3 CPU cores", Serial, func() {
		var availableNumberOfCPUs int

		BeforeEach(func() {
			availableNumberOfCPUs = libnode.GetHighestCPUNumberAmongNodes(virtClient)

			requiredNumberOfCpus := 3
			Expect(availableNumberOfCPUs).ToNot(BeNumerically("<", requiredNumberOfCpus),
				fmt.Sprintf("Test requires %d cpus, but only %d available!", requiredNumberOfCpus, availableNumberOfCPUs))
		})

		It("[test_id:1659]should report 3 cpu cores under guest OS", func() {
			vmi := libvmifact.NewAlpine(
				libvmi.WithCPUCount(3, 0, 0),
				libvmi.WithMemoryRequest("128Mi"),
			)

			By("Starting a VirtualMachineInstance")
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
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
			Expect(vmi.Spec.Domain.Resources.Requests.Memory().String()).To(Equal("128Mi"))

			readyPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(err).NotTo(HaveOccurred())

			computeContainer := libpod.LookupComputeContainer(readyPod)
			Expect(computeContainer.Resources.Requests.Memory().ToDec().ScaledValue(resource.Mega)).To(Equal(int64(399)))
		})

		It("[test_id:1660]should report 3 sockets under guest OS", func() {
			vmi := libvmifact.NewAlpine(
				libvmi.WithCPUCount(2, 0, 3),
				libvmi.WithMemoryRequest("128Mi"),
			)

			By("Starting a VirtualMachineInstance")
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
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
			vmi := libvmifact.NewAlpine(
				libvmi.WithCPURequest("1200m"),
				libvmi.WithMemoryRequest("128Mi"),
			)
			vmi.Spec.Domain.CPU = nil

			By("Starting a VirtualMachineInstance")
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
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
			vmi := libvmifact.NewAlpine(
				libvmi.WithCPULimit("1200m"),
				libvmi.WithMemoryRequest("128Mi"),
			)
			vmi.Spec.Domain.CPU = nil

			By("Starting a VirtualMachineInstance")
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
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

		It("[test_id:1663]should report 2 vCPUs under guest OS", decorators.WgS390x, func() {
			vmi := libvmifact.NewAlpine(
				libvmi.WithCPUCount(1, 1, 2),
				libvmi.WithMemoryRequest("128M"),
			)

			By("Starting a VirtualMachineInstance")
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred(), "should start vmi")
			libwait.WaitForSuccessfulVMIStart(vmi)

			By("Expecting the VirtualMachineInstance console")
			Expect(console.LoginToAlpine(vmi)).To(Succeed())

			By("Checking the number of vCPUs under guest OS")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "grep -c ^processor /proc/cpuinfo\n"},
				&expect.BExp{R: console.RetValue("2")},
			}, 60)).To(Succeed(), "should report number of threads")
		})

		It("[test_id:1665]should map cores to virtio net queues", func() {
			_false := false
			vmi := libvmifact.NewAlpine(
				libvmi.WithMemoryRequest("128Mi"),
				libvmi.WithCPURequest("3"),
				libvmi.WithNetworkInterfaceMultiQueue(true),
			)
			vmi.Spec.Domain.Devices.BlockMultiQueue = &_false

			By("Starting a VirtualMachineInstance")
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi)

			By("Expecting console")
			Expect(console.LoginToAlpine(vmi)).To(Succeed())

			By("Check network interface queues in guest")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "ls /sys/class/net/eth0/queues/ | grep rx | wc -l\n"},
				&expect.BExp{R: console.RetValue("3")},
			}, 15)).To(Succeed())

			By("Check block device does not have multiple queues")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "ls -1 /sys/block/vda/mq | wc -l\n"},
				&expect.BExp{R: console.RetValue("1")},
			}, 15)).To(Succeed())
		})
	})

	Context("[rfe_id:989]test cpu_allocation_ratio", func() {
		It("virt-launchers pod cpu requests should be proportional to the number of vCPUs", func() {
			vmi := libvmifact.NewAlpine(
				libvmi.WithGuestMemory("256Mi"),
				libvmi.WithCPUCount(6, 1, 1),
			)

			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi)

			readyPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(err).NotTo(HaveOccurred())

			computeContainer := libpod.LookupComputeContainer(readyPod)
			Expect(computeContainer.Resources.Requests.Cpu().String()).To(Equal("600m"))
		})
	})

	Context("[rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]with CPU spec", func() {
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
		})

		Context("[rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]when CPU model defined", func() {
			It("[test_id:1678]should report defined CPU model", func() {
				supportedCPUs := libnode.GetSupportedCPUModels(*nodes)
				Expect(supportedCPUs).ToNot(BeEmpty())
				cpuVmi := libvmifact.NewAlpine(libvmi.WithCPUModel(supportedCPUs[0]))

				niceName := parseCPUNiceName(supportedCPUs[0])

				By("Starting a VirtualMachineInstance")
				cpuVmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(cpuVmi)).Create(context.Background(), cpuVmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(cpuVmi)

				By("Expecting the VirtualMachineInstance console")
				Expect(console.LoginToAlpine(cpuVmi)).To(Succeed())

				By("Checking the CPU model under the guest OS")
				Expect(console.SafeExpectBatch(cpuVmi, []expect.Batcher{
					&expect.BSnd{S: fmt.Sprintf("grep %s /proc/cpuinfo\n", niceName)},
					&expect.BExp{R: fmt.Sprintf(".*model name.*%s.*", niceName)},
				}, 10)).To(Succeed())
			})
		})

		Context("[rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]when CPU model equals to passthrough", func() {
			It("[test_id:1679]should report exactly the same model as node CPU", func() {
				cpuVmi := libvmifact.NewAlpine(libvmi.WithCPUModel("host-passthrough"))

				By("Starting a VirtualMachineInstance")
				cpuVmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(cpuVmi)).Create(context.Background(), cpuVmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(cpuVmi)

				By("Checking the CPU model under the guest OS")
				output := libpod.RunCommandOnVmiPod(cpuVmi, []string{"grep", "-m1", "model name", "/proc/cpuinfo"})

				niceName := parseCPUNiceName(output)

				By("Expecting the VirtualMachineInstance console")
				Expect(console.LoginToAlpine(cpuVmi)).To(Succeed())

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
				cpuVmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), libvmifact.NewAlpine(), metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(cpuVmi)

				output := libpod.RunCommandOnVmiPod(cpuVmi, []string{"grep", "-m1", "model name", "/proc/cpuinfo"})

				niceName := parseCPUNiceName(output)

				By("Expecting the VirtualMachineInstance console")
				Expect(console.LoginToAlpine(cpuVmi)).To(Succeed())

				By("Checking the CPU model under the guest OS")
				Expect(console.SafeExpectBatch(cpuVmi, []expect.Batcher{
					&expect.BSnd{S: fmt.Sprintf("grep '%s' /proc/cpuinfo\n", niceName)},
					&expect.BExp{R: fmt.Sprintf(".*model name.*%s.*", niceName)},
				}, 10)).To(Succeed())
			})
		})

		Context("when CPU features defined", func() {
			It("[test_id:3123]should start a Virtual Machine with matching features", func() {
				supportedCPUFeatures := libnode.GetSupportedCPUFeatures(*nodes)
				Expect(supportedCPUFeatures).ToNot(BeEmpty())
				cpuVmi := libvmifact.NewAlpine(libvmi.WithCPUFeature(supportedCPUFeatures[0], ""))

				By("Starting a VirtualMachineInstance")
				cpuVmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(cpuVmi)).Create(context.Background(), cpuVmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(cpuVmi)

				By("Expecting the VirtualMachineInstance console")
				Expect(console.LoginToAlpine(cpuVmi)).To(Succeed())
			})
		})
	})

	Context("[rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]with CPU request settings", func() {

		It("[test_id:3127]should set CPU request from VMI spec", func() {
			vmi := libvmi.New(
				libvmi.WithMemoryRequest(enoughMemForSafeBiosEmulation),
				libvmi.WithCPURequest("500m"),
			)
			runningVMI := libvmops.RunVMIAndExpectScheduling(vmi, 30)

			readyPod, err := libpod.GetPodByVirtualMachineInstance(runningVMI, testsuite.GetTestNamespace(vmi))
			Expect(err).ToNot(HaveOccurred())
			computeContainer := libpod.LookupComputeContainer(readyPod)
			cpuRequest := computeContainer.Resources.Requests[k8sv1.ResourceCPU]
			Expect(cpuRequest.String()).To(Equal("500m"))
		})

		It("[test_id:3128]should set CPU request when it is not provided", func() {
			vmi := libvmifact.NewGuestless()
			runningVMI := libvmops.RunVMIAndExpectScheduling(vmi, 30)

			readyPod, err := libpod.GetPodByVirtualMachineInstance(runningVMI, testsuite.GetTestNamespace(vmi))
			Expect(err).ToNot(HaveOccurred())
			computeContainer := libpod.LookupComputeContainer(readyPod)
			cpuRequest := computeContainer.Resources.Requests[k8sv1.ResourceCPU]
			Expect(cpuRequest.String()).To(Equal("100m"))
		})

		It("[test_id:3129]should set CPU request from kubevirt-config", Serial, func() {
			kv := libkubevirt.GetCurrentKv(virtClient)

			config := kv.Spec.Configuration
			configureCPURequest := resource.MustParse("800m")
			config.CPURequest = &configureCPURequest
			kvconfig.UpdateKubeVirtConfigValueAndWait(config)

			vmi := libvmifact.NewGuestless()
			runningVMI := libvmops.RunVMIAndExpectScheduling(vmi, 30)

			readyPod, err := libpod.GetPodByVirtualMachineInstance(runningVMI, testsuite.GetTestNamespace(vmi))
			Expect(err).ToNot(HaveOccurred())
			computeContainer := libpod.LookupComputeContainer(readyPod)
			cpuRequest := computeContainer.Resources.Requests[k8sv1.ResourceCPU]
			Expect(cpuRequest.String()).To(Equal("800m"))
		})
	})

	Context("with automatic CPU limit configured in the CR", Serial, func() {
		const autoCPULimitLabel = "autocpulimit"
		BeforeEach(func() {
			By("Adding a label selector to the CR for auto CPU limit")
			kv := libkubevirt.GetCurrentKv(virtClient)
			config := kv.Spec.Configuration
			config.AutoCPULimitNamespaceLabelSelector = &metav1.LabelSelector{
				MatchLabels: map[string]string{autoCPULimitLabel: "true"},
			}
			kvconfig.UpdateKubeVirtConfigValueAndWait(config)
		})
		It("should not set a CPU limit if the namespace doesn't match the selector", func() {
			By("Creating a running VMI")
			vmi := libvmifact.NewGuestless()
			runningVMI := libvmops.RunVMIAndExpectScheduling(vmi, 30)

			By("Ensuring no CPU limit is set")
			readyPod, err := libpod.GetPodByVirtualMachineInstance(runningVMI, testsuite.GetTestNamespace(vmi))
			Expect(err).ToNot(HaveOccurred())
			computeContainer := libpod.LookupComputeContainer(readyPod)
			_, exists := computeContainer.Resources.Limits[k8sv1.ResourceCPU]
			Expect(exists).To(BeFalse(), "CPU limit set on the compute container when none was expected")
		})
		It("should set a CPU limit if the namespace matches the selector", func() {
			By("Creating a VMI object")
			vmi := libvmifact.NewGuestless()

			By("Adding the right label to VMI namespace")
			namespace, err := virtClient.CoreV1().Namespaces().Get(context.Background(), testsuite.GetTestNamespace(vmi), metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			patchData := []byte(fmt.Sprintf(`{"metadata": { "labels": {"%s": "true"}}}`, autoCPULimitLabel))
			_, err = virtClient.CoreV1().Namespaces().Patch(context.Background(), namespace.Name, types.StrategicMergePatchType, patchData, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Starting the VMI")
			runningVMI := libvmops.RunVMIAndExpectScheduling(vmi, 30)

			By("Ensuring the CPU limit is set to the correct value")
			readyPod, err := libpod.GetPodByVirtualMachineInstance(runningVMI, testsuite.GetTestNamespace(vmi))
			Expect(err).ToNot(HaveOccurred())
			computeContainer := libpod.LookupComputeContainer(readyPod)
			limits, exists := computeContainer.Resources.Limits[k8sv1.ResourceCPU]
			Expect(exists).To(BeTrue(), "expected CPU limit not set on the compute container")
			Expect(limits.String()).To(Equal("1"))
		})
	})
}))
