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
	"net"
	"os"
	"runtime"
	"strconv"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/api/core/v1"
	api2 "kubevirt.io/client-go/api"

	dutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	"kubevirt.io/kubevirt/pkg/network/istio"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var istioPortForwardRange = []api.InterfacePortForwardRange{
	{Start: 15000, Exclude: "yes"}, {Start: 15001, Exclude: "yes"},
	{Start: 15004, Exclude: "yes"}, {Start: 15006, Exclude: "yes"},
	{Start: 15008, Exclude: "yes"}, {Start: 15009, Exclude: "yes"},
	{Start: 15020, Exclude: "yes"}, {Start: 15021, Exclude: "yes"},
	{Start: 15053, Exclude: "yes"}, {Start: 15090, Exclude: "yes"},
}

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
					&vmi.Spec.Domain.Devices.Interfaces[0], domain, primaryPodIfaceName, mockNetwork)
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

		Context("Passt plug", func() {
			const podIfaceName = "eth0"
			var specGenerator *PasstLibvirtSpecGenerator

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
					createPasstInterface(), nil, podIfaceName, api2.NewMinimalVMI("passtVmi"))
				expectedPortFwd := []api.InterfacePortForward{
					{Proto: "tcp"}, {Proto: "udp"},
				}
				Expect(specGenerator.generatePortForward()).To(Equal(expectedPortFwd))
			})

			It("Should forward the specified tcp and udp ports", func() {
				passtIface := createPasstInterface()
				passtIface.Ports = []v1.Port{{Port: 1}, {Protocol: "UdP", Port: 2}, {Protocol: "UDP", Port: 3}, {Protocol: "tcp", Port: 4}}
				specGenerator = NewPasstLibvirtSpecGenerator(
					passtIface, nil, podIfaceName, api2.NewMinimalVMI("passtVmi"))

				expectedPortFwd := []api.InterfacePortForward{
					{
						Proto: "tcp",
						Ranges: []api.InterfacePortForwardRange{
							{Start: 1}, {Start: 4},
						},
					},
					{
						Proto: "udp",
						Ranges: []api.InterfacePortForwardRange{
							{Start: 2}, {Start: 3},
						},
					},
				}
				Expect(specGenerator.generatePortForward()).To(Equal(expectedPortFwd))
			})

			It("Should forward the specified tcp ports", func() {
				passtIface := createPasstInterface()
				passtIface.Ports = []v1.Port{{Protocol: "TCP", Port: 1}, {Protocol: "TCP", Port: 4}}
				specGenerator = NewPasstLibvirtSpecGenerator(
					passtIface, nil, podIfaceName, api2.NewMinimalVMI("passtVmi"))

				expectedPortFwd := []api.InterfacePortForward{
					{
						Proto: "tcp",
						Ranges: []api.InterfacePortForwardRange{
							{Start: 1}, {Start: 4},
						},
					},
				}

				Expect(specGenerator.generatePortForward()).To(Equal(expectedPortFwd))
			})

			It("Should forward the specified udp ports", func() {
				passtIface := createPasstInterface()
				passtIface.Ports = []v1.Port{{Protocol: "UDP", Port: 2}, {Protocol: "UDP", Port: 3}}
				specGenerator = NewPasstLibvirtSpecGenerator(
					passtIface, nil, podIfaceName, api2.NewMinimalVMI("passtVmi"))

				expectedPortFwd := []api.InterfacePortForward{
					{
						Proto: "udp",
						Ranges: []api.InterfacePortForwardRange{
							{Start: 2}, {Start: 3},
						},
					},
				}

				Expect(specGenerator.generatePortForward()).To(Equal(expectedPortFwd))
			})

			It("Should exclude istio ports", func() {
				passtIface := createPasstInterface()
				istioVmi := api2.NewMinimalVMI("passtVmi")
				istioVmi.Annotations = map[string]string{
					istio.IstioInjectAnnotation: "true",
				}
				specGenerator = NewPasstLibvirtSpecGenerator(
					passtIface, nil, podIfaceName, istioVmi)

				expectedPortFwd := []api.InterfacePortForward{
					{
						Proto:  "tcp",
						Ranges: istioPortForwardRange,
					},
				}

				Expect(specGenerator.generatePortForward()).To(Equal(expectedPortFwd))
			})

			It("should set passt domain interface", func() {
				istioVmi := api2.NewMinimalVMI("test")
				istioVmi.Annotations = map[string]string{istio.IstioInjectAnnotation: "true"}

				testDom := api.NewMinimalDomain("test")
				testAlias := api.NewUserDefinedAlias("default")
				testModel := &api.Model{Type: "virtio"}
				testDomIface := api.Interface{Alias: testAlias, Model: testModel}
				testDom.Spec.Devices.Interfaces = append(testDom.Spec.Devices.Interfaces, testDomIface)

				vmiSpecIface := &v1.Interface{
					Name:                   "default",
					MacAddress:             "02:02:02:02:02:02",
					InterfaceBindingMethod: v1.InterfaceBindingMethod{Passt: &v1.InterfacePasst{}},
					Ports: []v1.Port{
						{Protocol: "udp", Port: 100}, {Protocol: "udp", Port: 200},
						{Protocol: "tcp", Port: 8080},
						{Port: 80},
					},
				}

				specGenerator = NewPasstLibvirtSpecGenerator(vmiSpecIface, testDom, podIfaceName, istioVmi)

				expectedIface := &api.Interface{
					Type:    "user",
					Backend: &api.InterfaceBackend{Type: "passt", LogFile: PasstLogFile},
					Source:  api.InterfaceSource{Device: podIfaceName},
					Alias:   testAlias,
					Model:   testModel,
					MAC:     &api.MAC{MAC: "02:02:02:02:02:02"},
					PortForward: []api.InterfacePortForward{
						{
							Proto: "tcp",
							Ranges: append(
								istioPortForwardRange,
								api.InterfacePortForwardRange{Start: 8080},
								api.InterfacePortForwardRange{Start: 80},
							),
						},
						{
							Proto: "udp",
							Ranges: []api.InterfacePortForwardRange{
								{Start: 100}, {Start: 200},
							},
						},
					},
				}
				copy := testDomIface.DeepCopy()
				Expect(specGenerator.generateInterface(copy)).To(Equal(expectedIface))
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
