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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package network

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	nadv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kvirtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/client-go/kubecli"
	netvmispec "kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/util"
)

const dummyInterfaceName = "dummy0"

var _ = SIGDescribe("Infosource", func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		var err error
		virtClient, err = kubecli.GetKubevirtClient()
		Expect(err).NotTo(HaveOccurred(), "Should successfully initialize an API client")

		tests.BeforeTestCleanup()
	})

	Context("VMI with 3 interfaces", func() {
		var vmi *kvirtv1.VirtualMachineInstance

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
			Expect(createNAD(virtClient, util.NamespaceTestDefault, nadName)).To(Succeed())

			defaultBridgeInterface := libvmi.InterfaceDeviceWithBridgeBinding(primaryNetwork)
			secondaryLinuxBridgeInterface1 := libvmi.InterfaceDeviceWithBridgeBinding(secondaryNetwork1.Name)
			secondaryLinuxBridgeInterface2 := libvmi.InterfaceDeviceWithBridgeBinding(secondaryNetwork2.Name)
			vmiSpec := libvmi.NewFedora(
				libvmi.WithInterface(*libvmi.InterfaceWithMac(&defaultBridgeInterface, primaryInterfaceMac)),
				libvmi.WithNetwork(kvirtv1.DefaultPodNetwork()),
				libvmi.WithInterface(*libvmi.InterfaceWithMac(&secondaryLinuxBridgeInterface1, secondaryInterface1Mac)),
				libvmi.WithInterface(*libvmi.InterfaceWithMac(&secondaryLinuxBridgeInterface2, secondaryInterface2Mac)),
				libvmi.WithNetwork(secondaryNetwork1),
				libvmi.WithNetwork(secondaryNetwork2),
				libvmi.WithCloudInitNoCloudUserData(manipulateGuestLinksScript(primaryInterfaceNewMac, dummyInterfaceMac), false))

			var err error
			vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmiSpec)
			Expect(err).NotTo(HaveOccurred())
			tests.WaitForSuccessfulVMIStart(vmi)
			tests.WaitAgentConnected(virtClient, vmi)
		})

		It("should have the expected entries in vmi status", func() {
			expectedInterfaces := []kvirtv1.VirtualMachineInstanceNetworkInterface{
				{
					InfoSource: netvmispec.InfoSourceDomain,
					MAC:        primaryInterfaceMac,
					Name:       primaryNetwork,
				},
				{
					InfoSource:    netvmispec.InfoSourceDomainAndGA,
					InterfaceName: "eth1",
					MAC:           secondaryInterface1Mac,
					Name:          secondaryInterface1Name,
				},
				{
					InfoSource: netvmispec.InfoSourceDomain,
					MAC:        secondaryInterface2Mac,
					Name:       secondaryInterface2Name,
				},
				{
					InfoSource:    netvmispec.InfoSourceGuestAgent,
					InterfaceName: "eth0",
					MAC:           primaryInterfaceNewMac,
				},
				{
					InfoSource:    netvmispec.InfoSourceGuestAgent,
					InterfaceName: dummyInterfaceName,
					MAC:           dummyInterfaceMac,
				},
			}

			// once the dummy interface appears in the status, it means there was a guest-agent report
			// and then we can compare the rest of the expected info.
			Eventually(func() bool {
				var err error
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				return dummyInterfaceExists(vmi)
			}, 120*time.Second, 2*time.Second).Should(Equal(true))

			networkInterface := netvmispec.LookupInterfaceStatusByMac(vmi.Status.Interfaces, primaryInterfaceMac)
			Expect(networkInterface).NotTo(BeNil(), "interface not found")
			Expect(networkInterface.IP).To(BeEmpty())

			guestInterface := netvmispec.LookupInterfaceStatusByMac(vmi.Status.Interfaces, primaryInterfaceNewMac)
			Expect(guestInterface).NotTo(BeNil(), "interface not found")
			Expect(guestInterface.IP).NotTo(BeEmpty())

			for i := range vmi.Status.Interfaces {
				vmi.Status.Interfaces[i].IP = ""
				vmi.Status.Interfaces[i].IPs = nil
			}

			Expect(expectedInterfaces).To(ConsistOf(vmi.Status.Interfaces))
		})
	})
})

func newNetworkAttachmentDefinition(networkName string) *nadv1.NetworkAttachmentDefinition {
	config := fmt.Sprintf(`{"cniVersion": "0.3.1", "name": "%s", "type": "cnv-bridge", "bridge": "%s"}`, networkName, networkName)
	return &nadv1.NetworkAttachmentDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: networkName,
		},
		Spec: nadv1.NetworkAttachmentDefinitionSpec{Config: config},
	}
}

func dummyInterfaceExists(vmi *kvirtv1.VirtualMachineInstance) bool {
	for i := range vmi.Status.Interfaces {
		if vmi.Status.Interfaces[i].InterfaceName == dummyInterfaceName {
			return true
		}
	}
	return false
}

func createNAD(virtClient kubecli.KubevirtClient, namespace, nadName string) error {
	nadSpec := newNetworkAttachmentDefinition(nadName)
	_, err := virtClient.NetworkClient().K8sCniCncfIoV1().NetworkAttachmentDefinitions(namespace).Create(context.TODO(), nadSpec, metav1.CreateOptions{})
	return err
}

func manipulateGuestLinksScript(eth0NewMac, dummyInterfaceMac string) string {
	changeEth0Mac := "ip link set dev eth0 address " + eth0NewMac + "\n"
	createDummyInterface := "ip link add " + dummyInterfaceName + " type dummy\n" +
		"ip link set dev " + dummyInterfaceName + " address " + dummyInterfaceMac + "\n"
	moveEth2ToOtherNS := "ip netns add testns\n" +
		"ip link set eth2 netns testns\n"

	return "#!/bin/bash\n" + changeEth0Mac + createDummyInterface + moveEth2ToOtherNS
}
