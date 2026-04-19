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
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmici "kubevirt.io/kubevirt/pkg/libvmi/cloudinit"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnet/cloudinit"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("Secondary network IPv6", decorators.Multus, func() {
	const (
		nadName    = "ipv6-secondary"
		bridgeName = "br10"
		podIPv6    = "fd10:0:2::"
		guestIPv6  = "2001:db8:1::1/64"
	)

	BeforeEach(func() {
		config := libnet.NewNetConfig(nadName,
			libnet.NewNetPluginConfig("bridge", map[string]interface{}{
				"bridge": bridgeName,
				"ipam": map[string]interface{}{
					"type": "host-local",
					"ranges": [][]map[string]interface{}{
						{{"subnet": "10.10.10.0/24"}},
						{{"subnet": "fd10:0:2::0/120"}},
					},
				},
			}),
		)
		nad := libnet.NewNetAttachDef(nadName, config)
		_, err := libnet.CreateNetAttachDef(context.Background(), testsuite.GetTestNamespace(nil), nad)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should show guest IPv6 in status, not pod IPAM IPv6", func() {
		networkData, err := cloudinit.NewNetworkData(
			cloudinit.WithEthernet("eth1", cloudinit.WithAddresses(guestIPv6)),
		)
		Expect(err).NotTo(HaveOccurred())

		vmi := libvmifact.NewFedora(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding("secondary")),
			libvmi.WithNetwork(libvmi.MultusNetwork("secondary", nadName)),
			libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudNetworkData(networkData)),
		)

		virtClient := kubevirt.Client()
		vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).
			Create(context.Background(), vmi, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToFedora)
		Eventually(matcher.ThisVMI(vmi), 2*time.Minute, 2*time.Second).
			Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

		Eventually(func() error {
			return verifySecondaryIPv6Status(virtClient, vmi.Name, testsuite.GetTestNamespace(nil), "secondary", "2001:db8:1:", podIPv6)
		}, 2*time.Minute, 2*time.Second).Should(Succeed())

		Consistently(func() error {
			return verifySecondaryIPv6Status(virtClient, vmi.Name, testsuite.GetTestNamespace(nil), "secondary", "2001:db8:1:", podIPv6)
		}, 30*time.Second, 2*time.Second).Should(Succeed())
	})
})

func verifySecondaryIPv6Status(virtClient kubecli.KubevirtClient, vmiName, namespace, ifaceName, expectedIPv6Prefix, forbiddenIPv6Prefix string) error {
	vmi, err := virtClient.VirtualMachineInstance(namespace).Get(context.Background(), vmiName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	ifaceByName := indexInterfaceStatusByName(vmi)
	iface, exists := ifaceByName[ifaceName]
	if !exists {
		return fmt.Errorf("interface %s not found in status", ifaceName)
	}

	hasExpectedIPv6 := false
	for _, ip := range iface.IPs {
		if strings.HasPrefix(ip, expectedIPv6Prefix) {
			hasExpectedIPv6 = true
		}
		if strings.HasPrefix(ip, forbiddenIPv6Prefix) {
			return fmt.Errorf("status contains forbidden IPv6 prefix %s: %s", forbiddenIPv6Prefix, ip)
		}
	}
	if !hasExpectedIPv6 {
		return fmt.Errorf("status missing expected IPv6 prefix %s", expectedIPv6Prefix)
	}
	return nil
}
