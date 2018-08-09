/*
 * This file is part of the kubevirt project
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
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"

	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Bridge", func() {

	bridgeIP := map[string]string{"red": "172.16.98.1", "blue": "172.16.99.1"}

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)
	tests.BeforeAll(func() {
		waitForPodToFinish := func(pod *k8sv1.Pod) k8sv1.PodPhase {
			Eventually(func() k8sv1.PodPhase {
				j, err := virtClient.Core().Pods(tests.NamespaceTestDefault).Get(pod.ObjectMeta.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return j.Status.Phase
			}, 60*time.Second, 1*time.Second).Should(Or(Equal(k8sv1.PodSucceeded), Equal(k8sv1.PodFailed)))
			j, err := virtClient.Core().Pods(tests.NamespaceTestDefault).Get(pod.ObjectMeta.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return j.Status.Phase
		}

		addBridgeToHost := func(name string) {
			// create bridge on the node
			parameters := []string{"link", "add", name, "type", "bridge"}
			job := tests.RenderIPRouteJob(fmt.Sprintf("ip-add-%s", name), parameters)

			// make sure that both jobs are happening on the same node
			listOptions := metav1.ListOptions{}
			nodeList, err := virtClient.CoreV1().Nodes().List(listOptions)
			Expect(err).ToNot(HaveOccurred())
			Expect(nodeList.Items).NotTo(HaveLen(0))
			node := nodeList.Items[0]
			nodeSelector := map[string]string{"kubernetes.io/hostname": node.Name}

			job.Spec.NodeSelector = nodeSelector
			job, err = virtClient.CoreV1().Pods(tests.NamespaceTestDefault).Create(job)
			Expect(err).ToNot(HaveOccurred())
			waitForPodToFinish(job)
			// dont check results, as this may fail because bridge is already there
			// if there was any issue with creating the bridges the following "set" command would indicate the failure

			// set the bridge to "up" mode
			parameters = []string{"link", "set", "dev", name, "up"}
			job = tests.RenderIPRouteJob(fmt.Sprintf("ip-set-%s", name), parameters)
			job.Spec.NodeSelector = nodeSelector
			job, err = virtClient.CoreV1().Pods(tests.NamespaceTestDefault).Create(job)
			Expect(err).ToNot(HaveOccurred())
			phase := waitForPodToFinish(job)
			Expect(phase).To(Equal(k8sv1.PodSucceeded))

			// set IP on bridge
			parameters = []string{"addr", "add", bridgeIP[name] + "/24", "dev", name}
			job = tests.RenderIPRouteJob(fmt.Sprintf("ip-addr-%s", name), parameters)
			job.Spec.NodeSelector = nodeSelector
			job, err = virtClient.CoreV1().Pods(tests.NamespaceTestDefault).Create(job)
			Expect(err).ToNot(HaveOccurred())
			phase = waitForPodToFinish(job)
			// don't check address setting result
		}

		// add red and blue bridges to host
		addBridgeToHost("red")
		addBridgeToHost("blue")
	})

	Context("Exposing a network to the VM via bridge device plugin", func() {
		var vmi *v1.VirtualMachineInstance
		const networkName = "red"
		const macAddress = "de:ad:00:00:be:af"
		tests.BeforeAll(func() {
			vmi = tests.NewRandomVMIWithBridgeNetworkEphemeralDiskAndUserdata(tests.RegistryDiskFor(tests.RegistryDiskCirros),
				"#!/bin/bash\necho 'hello'\n",
				networkName,
				networkName)

			// set the MAC address on the L2 interface
			vmi.Spec.Domain.Devices.Interfaces[1].MacAddress = macAddress

			vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStartWithTimeout(vmi, 120)
		})

		It("Should create 2 interfaces on the VM", func() {
			const ifaceName = "eth1"
			expecter, err := tests.LoggedInCirrosExpecter(vmi)
			Expect(err).ToNot(HaveOccurred())
			defer expecter.Close()

			out, err := expecter.ExpectBatch([]expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: fmt.Sprintf("ip link show %s &> /dev/null; echo $?\n", ifaceName)},
				&expect.BExp{R: "0"},
			}, 180*time.Second)
			log.Log.Infof("%v", out)
			Expect(err).ToNot(HaveOccurred())
		})

		It("VM should be able to connect to the outside world over the default interface", func() {
			expecter, err := tests.LoggedInCirrosExpecter(vmi)
			defer expecter.Close()
			Expect(err).ToNot(HaveOccurred())

			out, err := expecter.ExpectBatch([]expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: "curl -o /dev/null -s -w \"%{http_code}\\n\" -k https://google.com\n"},
				&expect.BExp{R: "301"},
			}, 180*time.Second)
			log.Log.Infof("%v", out)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should have MAC address set correctly", func() {
			const ifaceName = "eth1"
			expecter, err := tests.LoggedInCirrosExpecter(vmi)
			Expect(err).ToNot(HaveOccurred())
			defer expecter.Close()

			out, err := expecter.ExpectBatch([]expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: fmt.Sprintf("ip link show %s | tail -1 | awk '{print $2}'\n", ifaceName)},
				&expect.BExp{R: macAddress},
			}, 180*time.Second)
			log.Log.Infof("%v", out)
			Expect(err).ToNot(HaveOccurred())
		})

		It("VM should be able to ping its bridge", func() {
			const IP = "172.16.98.100"
			const ifaceName = "eth1"
			addIPToVMI(IP+"/24", ifaceName, vmi)

			pingExpectOK(bridgeIP[networkName], ifaceName, vmi)
		})
	})

	Context("Exposing multiple networks to the VM via host bridge", func() {
		var vmi *v1.VirtualMachineInstance
		const networkName1 = "red"
		const networkName2 = "blue"
		tests.BeforeAll(func() {
			vmi = tests.NewRandomVMIWithBridgeNetworkEphemeralDiskAndUserdata(tests.RegistryDiskFor(tests.RegistryDiskCirros),
				"#!/bin/bash\necho 'hello'\n",
				networkName1,
				networkName1)

			// add the "blue" interface and network
			vmi.Spec.Domain.Devices.Interfaces = append(vmi.Spec.Domain.Devices.Interfaces,
				v1.Interface{Name: networkName2,
					InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}})
			vmi.Spec.Networks = append(vmi.Spec.Networks,
				v1.Network{Name: networkName2,
					NetworkSource: v1.NetworkSource{
						HostBridge: &v1.HostBridge{BridgeName: networkName2}}})

			vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStartWithTimeout(vmi, 120)
		})

		It("Should create 3 interfaces on the VM", func() {
			const ifaceName = "eth2"
			expecter, err := tests.LoggedInCirrosExpecter(vmi)
			Expect(err).ToNot(HaveOccurred())
			defer expecter.Close()

			out, err := expecter.ExpectBatch([]expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: fmt.Sprintf("ip link show %s &> /dev/null; echo $?\n", ifaceName)},
				&expect.BExp{R: "0"},
			}, 180*time.Second)
			log.Log.Infof("%v", out)
			Expect(err).ToNot(HaveOccurred())
		})

		It("VM should be able to ping its bridges on different networks", func() {
			const IP1 = "172.16.98.10"
			const IP2 = "172.16.99.10"
			const ifaceName1 = "eth1"
			const ifaceName2 = "eth2"
			addIPToVMI(IP1+"/24", ifaceName1, vmi)
			addIPToVMI(IP2+"/24", ifaceName2, vmi)

			pingExpectOK(bridgeIP[networkName1], ifaceName1, vmi)
			pingExpectOK(bridgeIP[networkName2], ifaceName2, vmi)
		})
	})

	Context("Exposing multiple interfaces for the same network to the VM via host bridge", func() {
		var vmi *v1.VirtualMachineInstance
		const networkName = "blue"
		const IP1 = "172.16.99.30"
		const IP2 = "172.16.99.40"
		const ifaceName1 = "eth1"
		const ifaceName2 = "eth2"
		tests.BeforeAll(func() {
			vmi = tests.NewRandomVMIWithBridgeNetworkEphemeralDiskAndUserdata(tests.RegistryDiskFor(tests.RegistryDiskCirros),
				"#!/bin/bash\necho 'hello'\n",
				networkName+"1",
				networkName)

			// add another interface to the same network
			vmi.Spec.Domain.Devices.Interfaces = append(vmi.Spec.Domain.Devices.Interfaces,
				v1.Interface{Name: networkName + "2",
					InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}})
			vmi.Spec.Networks = append(vmi.Spec.Networks,
				v1.Network{Name: networkName + "2",
					NetworkSource: v1.NetworkSource{
						HostBridge: &v1.HostBridge{BridgeName: networkName}}})

			vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStartWithTimeout(vmi, 120)

			// add IPs to the interfaces on the VM
			addIPToVMI(IP1+"/24", ifaceName1, vmi)
			addIPToVMI(IP2+"/24", ifaceName2, vmi)
		})

		It("Should create 3 interfaces on the VM", func() {
			const ifaceName = "eth2"
			expecter, err := tests.LoggedInCirrosExpecter(vmi)
			Expect(err).ToNot(HaveOccurred())
			defer expecter.Close()

			out, err := expecter.ExpectBatch([]expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: fmt.Sprintf("ip link show %s &> /dev/null; echo $?\n", ifaceName)},
				&expect.BExp{R: "0"},
			}, 180*time.Second)
			log.Log.Infof("%v", out)
			Expect(err).ToNot(HaveOccurred())
		})

		It("VM should be able to connect to the outside world over the default interface", func() {
			expecter, err := tests.LoggedInCirrosExpecter(vmi)
			defer expecter.Close()
			Expect(err).ToNot(HaveOccurred())

			out, err := expecter.ExpectBatch([]expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: "curl -o /dev/null -s -w \"%{http_code}\\n\" -k https://google.com\n"},
				&expect.BExp{R: "301"},
			}, 180*time.Second)
			log.Log.Infof("%v", out)
			Expect(err).ToNot(HaveOccurred())
		})

		It("VMs should be able to ping the bridge on via different interfaces", func() {
			pingExpectOK(bridgeIP[networkName], ifaceName1, vmi)
			pingExpectOK(bridgeIP[networkName], ifaceName2, vmi)
		})

		It("Ping should fail over the default interface", func() {
			pingExpectFail(bridgeIP[networkName], "eth0", vmi)
		})
	})

	Context("Define 2 VMs over the same network", func() {
		var vmi1 *v1.VirtualMachineInstance
		var vmi2 *v1.VirtualMachineInstance
		const networkName = "red"
		const IP1 = "172.16.98.50"
		const IP2 = "172.16.98.60"
		const ifaceName = "eth1"

		tests.BeforeAll(func() {
			createVMWithNetworkandIP := func(networkName string, cidr string) (vmi *v1.VirtualMachineInstance) {
				vmi = tests.NewRandomVMIWithBridgeNetworkEphemeralDiskAndUserdata(tests.RegistryDiskFor(tests.RegistryDiskCirros),
					"#!/bin/bash\necho 'hello'\n",
					networkName,
					networkName)

				vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStartWithTimeout(vmi, 120)

				addIPToVMI(cidr, ifaceName, vmi)
				return
			}

			vmi1 = createVMWithNetworkandIP(networkName, IP1+"/24")
			vmi2 = createVMWithNetworkandIP(networkName, IP2+"/24")
		})

		It("VMs should be able to ping the bridge", func() {
			pingExpectOK(bridgeIP[networkName], ifaceName, vmi1)
			pingExpectOK(bridgeIP[networkName], ifaceName, vmi2)
		})

		It("Ping should fail over the default interface", func() {
			pingExpectFail(bridgeIP[networkName], "eth0", vmi1)
			pingExpectFail(bridgeIP[networkName], "eth0", vmi2)
		})

		It("VMs should be able to ping one another", func() {
			Skip("ping between 2 VMs don't work")
			pingExpectOK(IP2, ifaceName, vmi1)
			pingExpectOK(IP1, ifaceName, vmi2)
		})
	})
})

func addIPToVMI(cidr string, ifaceName string, vmi *v1.VirtualMachineInstance) {
	// add IP addresses on the interfaces
	expecter, err := tests.LoggedInCirrosExpecter(vmi)
	defer expecter.Close()
	Expect(err).ToNot(HaveOccurred())

	out, err := expecter.ExpectBatch([]expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: "\\$ "},
		&expect.BSnd{S: fmt.Sprintf("sudo ip addr add %s dev %s && echo ok\n", cidr, ifaceName)},
		&expect.BExp{R: "ok"},
	}, 180*time.Second)
	log.Log.Infof("%v", out)
	Expect(err).ToNot(HaveOccurred())
}

func pingExpectOK(ip string, ifaceName string, vmi *v1.VirtualMachineInstance) {
	expecter, err := tests.LoggedInCirrosExpecter(vmi)
	defer expecter.Close()
	Expect(err).ToNot(HaveOccurred())

	out, err := expecter.ExpectBatch([]expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: "\\$ "},
		&expect.BSnd{S: fmt.Sprintf("ping %s -I %s -q -c 2 -w 10 && echo ok\n", ip, ifaceName)},
		&expect.BExp{R: "ok"},
	}, 30*time.Second)
	log.Log.Infof("%v", out)
	Expect(err).ToNot(HaveOccurred())
}

func pingExpectFail(ip string, ifaceName string, vmi *v1.VirtualMachineInstance) {
	expecter, err := tests.LoggedInCirrosExpecter(vmi)
	Expect(err).ToNot(HaveOccurred())
	defer expecter.Close()

	out, err := expecter.ExpectBatch([]expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: "\\$ "},
		&expect.BSnd{S: fmt.Sprintf("ping %s -I %s -q -c 2 -w 10 || echo fail\n", ip, ifaceName)},
		&expect.BExp{R: "fail"},
	}, 30*time.Second)
	log.Log.Infof("%v", out)
	Expect(err).ToNot(HaveOccurred())
}
