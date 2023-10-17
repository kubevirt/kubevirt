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
 * Copyright 2017-2023 Red Hat, Inc.
 *
 */

package vmi_configuration

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	kubev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/util"

	hw_utils "kubevirt.io/kubevirt/pkg/util/hardware"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = ConfigDescribe("[rfe_id:897][crit:medium][arm64][vendor:cnv-qe@redhat.com][level:component]VirtualMachineInstance with CPU pinning", func() {
	const (
		cgroupV1MemoryUsagePath = "/sys/fs/cgroup/memory/memory.usage_in_bytes"
		cgroupV2MemoryUsagePath = "/sys/fs/cgroup/memory.current"
	)

	var virtClient kubecli.KubevirtClient
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
