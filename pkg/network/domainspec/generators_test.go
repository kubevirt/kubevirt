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

package domainspec

import (
	"net"
	"os"
	"runtime"
	"strconv"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/api/core/v1"

	dutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("Pod Network", func() {
	var mockNetwork *netdriver.MockNetworkHandler
	var ctrl *gomock.Controller
	var fakeMac net.HardwareAddr
	var tmpDir string
	const mtu = "1410"

	BeforeEach(func() {
		dutils.MockDefaultOwnershipManager()
		var err error
		tmpDir, err = os.MkdirTemp("/tmp", "interface-cache")
		Expect(err).ToNot(HaveOccurred())

		ctrl = gomock.NewController(GinkgoT())
		mockNetwork = netdriver.NewMockNetworkHandler(ctrl)
		fakeMac, _ = net.ParseMAC("12:34:56:78:9A:BC")
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	Context("on successful setup", func() {
		Context("tap generator", func() {
			const primaryPodIfaceName = "eth0"
			const tapName = "tap0"
			const specMAC = "11:22:33:44:55:66"

			var (
				domain        *api.Domain
				specGenerator *TapLibvirtSpecGenerator
				tapInterface  netlink.Link
				vmi           *v1.VirtualMachineInstance
			)
			BeforeEach(func() {
				domain = NewDomainInterface("default")
				api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
				vmi = newVMIMasqueradeInterface("testnamespace", "testVmName")
				vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = specMAC
				mtuVal, _ := strconv.Atoi(mtu)
				iface := &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Name: primaryPodIfaceName, MTU: mtuVal, HardwareAddr: fakeMac}}
				tapInterface = &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Name: tapName}}
				mockNetwork.EXPECT().LinkByName(primaryPodIfaceName).Return(iface, nil)
				specGenerator = NewTapLibvirtSpecGenerator(
					&vmi.Spec.Domain.Devices.Interfaces[0],
					vmi.Spec.Networks[0],
					domain,
					primaryPodIfaceName,
					mockNetwork,
				)
			})

			It("Should use the tap device as the target", func() {
				mockNetwork.EXPECT().LinkByName(tapName).Return(tapInterface, nil)

				Expect(specGenerator.Generate()).To(Succeed())

				verifyTapDomain(domain.Spec.Devices.Interfaces, tapName, mtu, specMAC)
			})

			It("Should use the pod interface as the target", func() {
				mockNetwork.EXPECT().LinkByName(tapName).Return(nil, netlink.LinkNotFoundError{})

				Expect(specGenerator.Generate()).To(Succeed())

				verifyTapDomain(domain.Spec.Devices.Interfaces, primaryPodIfaceName, mtu, specMAC)
			})

			It("Should use the pod interface MAC address", func() {
				mockNetwork.EXPECT().LinkByName(tapName).Return(tapInterface, nil)
				vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = ""

				Expect(specGenerator.Generate()).To(Succeed())

				verifyTapDomain(domain.Spec.Devices.Interfaces, tapName, mtu, fakeMac.String())
			})
		})
	})
})

func verifyTapDomain(domainIfaces []api.Interface, expectedTargetName, expectedMTU, expectedMAC string) {
	ExpectWithOffset(1, domainIfaces).To(HaveLen(1), "should have a single interface")
	ExpectWithOffset(1, domainIfaces[0].Target).To(
		Equal(
			&api.InterfaceTarget{
				Device:  expectedTargetName,
				Managed: "no",
			}), "should have an unmanaged interface")
	ExpectWithOffset(1, domainIfaces[0].MAC).To(Equal(&api.MAC{MAC: expectedMAC}), "should have the expected MAC address")
	ExpectWithOffset(1, domainIfaces[0].MTU).To(Equal(&api.MTU{Size: expectedMTU}), "should have the expected MTU")
}
