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
 * Copyright The KubeVirt Authors.
 *
 */

package network

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmici "kubevirt.io/kubevirt/pkg/libvmi/cloudinit"
	"kubevirt.io/kubevirt/pkg/network/namescheme"
	network "kubevirt.io/kubevirt/pkg/network/setup"
	netvmispec "kubevirt.io/kubevirt/pkg/network/vmispec"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const dummyInterfaceName = "dummy0"

var _ = Describe(SIG("Infosource", func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("VMI with 3 interfaces", func() {
		var vmi *v1.VirtualMachineInstance

		const (
			nadName                 = "infosrc"
			primaryNetwork          = "default"
			primaryInterfaceMac     = "02:00:05:05:05:05"
			primaryInterfaceNewMac  = "02:00:b5:b5:b5:b5"
			secondaryInterface1Name = "bridge1-unchanged"
			secondaryInterface1Mac  = "02:00:a0:a0:a0:a0"
			secondaryInterface2Name = "bridge2-setns"
			secondaryInterface2Mac  = "02:00:a1:a1:a1:a1"
			dummyInterfaceMac       = "02:00:b0:b0:b0:b0"
		)

		secondaryNetwork1 := libvmi.MultusNetwork(secondaryInterface1Name, nadName)
		secondaryNetwork2 := libvmi.MultusNetwork(secondaryInterface2Name, nadName)

		BeforeEach(func() {
			By("Create NetworkAttachmentDefinition")
			netAttachDef := libnet.NewBridgeNetAttachDef(nadName, nadName)
			_, err := libnet.CreateNetAttachDef(context.Background(), testsuite.NamespaceTestDefault, netAttachDef)
			Expect(err).NotTo(HaveOccurred())

			defaultBridgeInterface := libvmi.InterfaceDeviceWithBridgeBinding(primaryNetwork)
			secondaryLinuxBridgeInterface1 := libvmi.InterfaceDeviceWithBridgeBinding(secondaryNetwork1.Name)
			secondaryLinuxBridgeInterface2 := libvmi.InterfaceDeviceWithBridgeBinding(secondaryNetwork2.Name)
			vmiSpec := libvmifact.NewFedora(
				libvmi.WithInterface(*libvmi.InterfaceWithMac(&defaultBridgeInterface, primaryInterfaceMac)),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithInterface(*libvmi.InterfaceWithMac(&secondaryLinuxBridgeInterface1, secondaryInterface1Mac)),
				libvmi.WithInterface(*libvmi.InterfaceWithMac(&secondaryLinuxBridgeInterface2, secondaryInterface2Mac)),
				libvmi.WithNetwork(secondaryNetwork1),
				libvmi.WithNetwork(secondaryNetwork2),
				libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudUserData(manipulateGuestLinksScript(primaryInterfaceNewMac, dummyInterfaceMac))))

			vmi, err = virtClient.VirtualMachineInstance(testsuite.NamespaceTestDefault).Create(context.Background(), vmiSpec, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi)
			Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
		})

		It("should have the expected entries in vmi status", func() {
			infoSourceDomainAndMultusStatus := netvmispec.NewInfoSource(
				netvmispec.InfoSourceDomain, netvmispec.InfoSourceMultusStatus)
			infoSourceDomainAndGAAndMultusStatus := netvmispec.NewInfoSource(
				netvmispec.InfoSourceDomain, netvmispec.InfoSourceGuestAgent, netvmispec.InfoSourceMultusStatus)

			const linkStateUp = "up"

			expectedInterfaces := []v1.VirtualMachineInstanceNetworkInterface{
				{
					InfoSource:       netvmispec.InfoSourceDomain,
					MAC:              primaryInterfaceMac,
					Name:             primaryNetwork,
					PodInterfaceName: namescheme.PrimaryPodInterfaceName,
					QueueCount:       network.DefaultInterfaceQueueCount,
					LinkState:        linkStateUp,
				},
				{
					InfoSource:       infoSourceDomainAndGAAndMultusStatus,
					InterfaceName:    "eth1",
					MAC:              secondaryInterface1Mac,
					Name:             secondaryInterface1Name,
					PodInterfaceName: namescheme.GenerateHashedInterfaceName(secondaryInterface1Name),
					QueueCount:       network.DefaultInterfaceQueueCount,
					LinkState:        linkStateUp,
				},
				{
					InfoSource:       infoSourceDomainAndMultusStatus,
					MAC:              secondaryInterface2Mac,
					Name:             secondaryInterface2Name,
					PodInterfaceName: namescheme.GenerateHashedInterfaceName(secondaryInterface2Name),
					QueueCount:       network.DefaultInterfaceQueueCount,
					LinkState:        linkStateUp,
				},
				{
					InfoSource:    netvmispec.InfoSourceGuestAgent,
					InterfaceName: "eth0",
					MAC:           primaryInterfaceNewMac,
					QueueCount:    network.UnknownInterfaceQueueCount,
				},
				{
					InfoSource:    netvmispec.InfoSourceGuestAgent,
					InterfaceName: dummyInterfaceName,
					MAC:           dummyInterfaceMac,
					QueueCount:    network.UnknownInterfaceQueueCount,
				},
			}

			// once the dummy interface appears in the status, it means there was a guest-agent report
			// and then we can compare the rest of the expected info.
			Eventually(func() bool {
				var err error
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				return dummyInterfaceExists(vmi)
			}, 120*time.Second, 2*time.Second).Should(BeTrue())

			networkInterface := netvmispec.LookupInterfaceStatusByMac(vmi.Status.Interfaces, primaryInterfaceMac)
			Expect(networkInterface).NotTo(BeNil(), "interface not found")
			Expect(networkInterface.IP).NotTo(BeEmpty())

			guestInterface := netvmispec.LookupInterfaceStatusByMac(vmi.Status.Interfaces, primaryInterfaceNewMac)
			Expect(guestInterface).NotTo(BeNil(), "interface not found")
			Expect(guestInterface.IP).NotTo(BeEmpty())

			for i := range vmi.Status.Interfaces {
				vmi.Status.Interfaces[i].IP = ""
				vmi.Status.Interfaces[i].IPs = nil
			}

			Expect(vmi.Status.Interfaces).To(ConsistOf(expectedInterfaces))
		})
	})
}))

func dummyInterfaceExists(vmi *v1.VirtualMachineInstance) bool {
	for i := range vmi.Status.Interfaces {
		if vmi.Status.Interfaces[i].InterfaceName == dummyInterfaceName {
			return true
		}
	}
	return false
}

func manipulateGuestLinksScript(eth0NewMac, dummyInterfaceMac string) string {
	changeEth0Mac := "ip link set dev eth0 address " + eth0NewMac + "\n"
	createDummyInterface := "ip link add " + dummyInterfaceName + " type dummy\n" +
		"ip link set dev " + dummyInterfaceName + " address " + dummyInterfaceMac + "\n"
	moveEth2ToOtherNS := "ip netns add testns\n" +
		"ip link set eth2 netns testns\n"

	return "#!/bin/bash\n" + changeEth0Mac + createDummyInterface + moveEth2ToOtherNS
}
