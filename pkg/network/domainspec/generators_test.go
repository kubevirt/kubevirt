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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package domainspec

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"

	"kubevirt.io/kubevirt/pkg/network/istio"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/api/core/v1"
	api2 "kubevirt.io/client-go/api"

	dutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("Pod Network", func() {
	var mockNetwork *netdriver.MockNetworkHandler
	var ctrl *gomock.Controller
	var fakeMac net.HardwareAddr
	var tmpDir string
	var mtu int

	BeforeEach(func() {
		dutils.MockDefaultOwnershipManager()
		var err error
		tmpDir, err = os.MkdirTemp("/tmp", "interface-cache")
		Expect(err).ToNot(HaveOccurred())

		ctrl = gomock.NewController(GinkgoT())
		mockNetwork = netdriver.NewMockNetworkHandler(ctrl)
		mtu = 1410
		fakeMac, _ = net.ParseMAC("12:34:56:78:9A:BC")
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	Context("on successful setup", func() {
		Context("Slirp Plug", func() {
			var (
				domain *api.Domain
				vmi    *v1.VirtualMachineInstance
			)

			BeforeEach(func() {
				domain = NewDomainWithSlirpInterface()
				api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
				vmi = newVMISlirpInterface("testnamespace", "testVmName")
			})

			It("Should create an interface in the qemu command line and remove it from the interfaces", func() {
				specGenerator := NewSlirpLibvirtSpecGenerator(&vmi.Spec.Domain.Devices.Interfaces[0], domain)
				Expect(specGenerator.Generate()).To(Succeed())

				Expect(domain.Spec.Devices.Interfaces).To(BeEmpty())
				Expect(domain.Spec.QEMUCmd.QEMUArg).To(HaveLen(2))
				Expect(domain.Spec.QEMUCmd.QEMUArg[0]).To(Equal(api.Arg{Value: "-device"}))
				Expect(domain.Spec.QEMUCmd.QEMUArg[1]).To(Equal(api.Arg{Value: `{"driver":"e1000","netdev":"default","id":"default"}`}))
			})

			It("Should append MAC address to qemu arguments if set", func() {
				mac := "de-ad-00-00-be-af"
				device := fmt.Sprintf(`{"driver":"e1000","netdev":"default","id":"default","mac":%q}`, mac)

				vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = mac
				specGenerator := NewSlirpLibvirtSpecGenerator(&vmi.Spec.Domain.Devices.Interfaces[0], domain)
				Expect(specGenerator.Generate()).To(Succeed())

				Expect(domain.Spec.Devices.Interfaces).To(BeEmpty())
				Expect(domain.Spec.QEMUCmd.QEMUArg).To(HaveLen(2))
				Expect(domain.Spec.QEMUCmd.QEMUArg[0]).To(Equal(api.Arg{Value: "-device"}))
				Expect(domain.Spec.QEMUCmd.QEMUArg[1]).To(Equal(api.Arg{Value: device}))
			})
			It("Should create an interface in the qemu command line, remove it from the interfaces and leave the other interfaces inplace", func() {
				domain.Spec.Devices.Interfaces = append(domain.Spec.Devices.Interfaces, api.Interface{
					Model: &api.Model{
						Type: v1.VirtIO,
					},
					Type: "bridge",
					Source: api.InterfaceSource{
						Bridge: api.DefaultBridgeName,
					},
					Alias: api.NewUserDefinedAlias("default"),
				})
				specGenerator := NewSlirpLibvirtSpecGenerator(&vmi.Spec.Domain.Devices.Interfaces[0], domain)
				Expect(specGenerator.Generate()).To(Succeed())

				Expect(domain.Spec.Devices.Interfaces).To(HaveLen(1))
				Expect(domain.Spec.QEMUCmd.QEMUArg).To(HaveLen(2))
				Expect(domain.Spec.QEMUCmd.QEMUArg[0]).To(Equal(api.Arg{Value: "-device"}))
				Expect(domain.Spec.QEMUCmd.QEMUArg[1]).To(Equal(api.Arg{Value: `{"driver":"e1000","netdev":"default","id":"default"}`}))
			})
		})
		Context("Macvtap plug", func() {
			const primaryPodIfaceName = "eth0"

			var (
				domain        *api.Domain
				specGenerator *MacvtapLibvirtSpecGenerator
			)

			BeforeEach(func() {
				domain = NewDomainWithMacvtapInterface("default")
				api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
				vmi := newVMIMacvtapInterface("testnamespace", "testVmName", "default")
				macvtapInterface := &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Name: primaryPodIfaceName, MTU: mtu, HardwareAddr: fakeMac}}
				mockNetwork.EXPECT().LinkByName(primaryPodIfaceName).Return(macvtapInterface, nil)
				specGenerator = NewMacvtapLibvirtSpecGenerator(
					&vmi.Spec.Domain.Devices.Interfaces[0], domain, primaryPodIfaceName, mockNetwork)
			})

			It("Should pass a non-privileged macvtap interface to qemu", func() {
				Expect(specGenerator.Generate()).To(Succeed())

				Expect(domain.Spec.Devices.Interfaces).To(HaveLen(1), "should have a single interface")
				Expect(domain.Spec.Devices.Interfaces[0].Target).To(
					Equal(
						&api.InterfaceTarget{
							Device:  primaryPodIfaceName,
							Managed: "no",
						}), "should have an unmanaged interface")
				Expect(domain.Spec.Devices.Interfaces[0].MAC).To(Equal(&api.MAC{MAC: fakeMac.String()}), "should have the expected MAC address")
				Expect(domain.Spec.Devices.Interfaces[0].MTU).To(Equal(&api.MTU{Size: "1410"}), "should have the expected MTU")
			})
		})
		Context("Passt plug", func() {
			var specGenerator *PasstLibvirtSpecGenerator

			getPorts := func(specGenerator *PasstLibvirtSpecGenerator) string {
				return strings.Join(specGenerator.generatePorts(), " ")
			}

			createPasstInterface := func() *v1.Interface {
				return &v1.Interface{
					Name: "passt_test",
					InterfaceBindingMethod: v1.InterfaceBindingMethod{
						Passt: &v1.InterfacePasst{},
					},
				}
			}

			It("Should forward all ports if ports are not specified in spec.interfaces", func() {
				specGenerator = NewPasstLibvirtSpecGenerator(
					createPasstInterface(), nil, api2.NewMinimalVMI("passtVmi"))
				Expect(getPorts(specGenerator)).To(Equal("-t all -u all"))
			})

			It("Should forward the specified tcp and udp ports", func() {
				passtIface := createPasstInterface()
				passtIface.Ports = []v1.Port{{Port: 1}, {Protocol: "UdP", Port: 2}, {Protocol: "UDP", Port: 3}, {Protocol: "tcp", Port: 4}}
				specGenerator = NewPasstLibvirtSpecGenerator(
					passtIface, nil, api2.NewMinimalVMI("passtVmi"))
				Expect(getPorts(specGenerator)).To(Equal("-t 1,4 -u 2,3"))
			})

			It("Should forward the specified tcp ports", func() {
				passtIface := createPasstInterface()
				passtIface.Ports = []v1.Port{{Protocol: "TCP", Port: 1}, {Protocol: "TCP", Port: 4}}
				specGenerator = NewPasstLibvirtSpecGenerator(
					passtIface, nil, api2.NewMinimalVMI("passtVmi"))
				Expect(getPorts(specGenerator)).To(Equal("-t 1,4"))
			})

			It("Should forward the specified udp ports", func() {
				passtIface := createPasstInterface()
				passtIface.Ports = []v1.Port{{Protocol: "UDP", Port: 2}, {Protocol: "UDP", Port: 3}}
				specGenerator = NewPasstLibvirtSpecGenerator(
					passtIface, nil, api2.NewMinimalVMI("passtVmi"))
				Expect(getPorts(specGenerator)).To(Equal("-u 2,3"))
			})

			It("Should exclude istio ports", func() {
				passtIface := createPasstInterface()
				istioVmi := api2.NewMinimalVMI("passtVmi")
				istioVmi.Annotations = map[string]string{
					istio.ISTIO_INJECT_ANNOTATION: "true",
				}
				specGenerator = NewPasstLibvirtSpecGenerator(
					passtIface, nil, istioVmi)
				Expect(getPorts(specGenerator)).To(Equal("-t ~15000,~15001,~15004,~15006,~15008,~15009,~15020,~15021,~15053,~15090 -u all"))
			})
		})
	})
})
