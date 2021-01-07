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

package network

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("Common Methods", func() {
	pid := "self"
	Context("Functions Read and Write from cache", func() {
		It("should persist interface payload", func() {
			tmpDir, err := ioutil.TempDir("", "commontest")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(tmpDir)
			setInterfaceCacheFile(tmpDir + "/cache-%s.json")

			ifaceName := "iface_name"
			iface := api.Interface{Type: "fake_type", Source: api.InterfaceSource{Bridge: "fake_br"}}
			err = writeToCachedFile(&iface, interfaceCacheFile, pid, ifaceName)
			Expect(err).ToNot(HaveOccurred())

			var cached_iface api.Interface
			isExist, err := readFromCachedFile(pid, ifaceName, interfaceCacheFile, &cached_iface)
			Expect(err).ToNot(HaveOccurred())
			Expect(isExist).To(BeTrue())

			Expect(iface).To(Equal(cached_iface))
		})
		It("should persist qemu arg payload", func() {
			tmpDir, err := ioutil.TempDir("", "commontest")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(tmpDir)
			setInterfaceCacheFile(tmpDir + "/cache-%s.json")

			qemuArgName := "iface_name"
			qemuArg := api.Arg{Value: "test_value"}
			err = writeToCachedFile(&qemuArg, interfaceCacheFile, pid, qemuArgName)
			Expect(err).ToNot(HaveOccurred())

			var cached_qemuArg api.Arg
			isExist, err := readFromCachedFile(pid, qemuArgName, interfaceCacheFile, &cached_qemuArg)
			Expect(err).ToNot(HaveOccurred())
			Expect(isExist).To(BeTrue())

			Expect(qemuArg).To(Equal(cached_qemuArg))
		})
	})
	Context("GetAvailableAddrsFromCIDR function", func() {
		It("Should return 2 addresses", func() {
			networkHandler := NetworkUtilsHandler{}
			gw, vm, err := networkHandler.GetHostAndGwAddressesFromCIDR("10.0.0.0/30")
			Expect(err).ToNot(HaveOccurred())
			Expect(gw).To(Equal("10.0.0.1/30"))
			Expect(vm).To(Equal("10.0.0.2/30"))
		})
		It("Should return 2 IPV6 addresses", func() {
			networkHandler := NetworkUtilsHandler{}
			gw, vm, err := networkHandler.GetHostAndGwAddressesFromCIDR("fd10:0:2::/120")
			Expect(err).ToNot(HaveOccurred())
			Expect(gw).To(Equal("fd10:0:2::1/120"))
			Expect(vm).To(Equal("fd10:0:2::2/120"))
		})
		It("Should fail when the subnet is too small", func() {
			networkHandler := NetworkUtilsHandler{}
			_, _, err := networkHandler.GetHostAndGwAddressesFromCIDR("10.0.0.0/31")
			Expect(err).To(HaveOccurred())
		})
		It("Should fail when the IPV6 subnet is too small", func() {
			networkHandler := NetworkUtilsHandler{}
			_, _, err := networkHandler.GetHostAndGwAddressesFromCIDR("fd10:0:2::/127")
			Expect(err).To(HaveOccurred())
		})
	})
	Context("GenerateRandomMac function", func() {
		It("should return a valid mac address", func() {
			networkHandler := NetworkUtilsHandler{}
			mac, err := networkHandler.GenerateRandomMac()
			Expect(err).ToNot(HaveOccurred())
			Expect(strings.HasPrefix(mac.String(), "02:00:00")).To(BeTrue())
		})
	})
})

var _ = Describe("VIF", func() {
	const ipv4Cidr = "10.0.0.200/24"
	const ipv4Address = "10.0.0.200"
	const ipv4Mask = "ffffff00"
	const ipv6Cidr = "fd10:0:2::2/120"
	const mac = "de:ad:00:00:be:ef"
	const ipv4Gateway = "10.0.0.1"
	const vifName = "test-vif"

	Context("String", func() {
		It("returns correct string representation", func() {
			vif := createDummyVIF(vifName, ipv4Cidr, ipv4Gateway, "", mac)
			Expect(vif.String()).To(Equal(fmt.Sprintf("VIF: { Name: %s, IP: %s, Mask: %s, IPv6: <nil>, MAC: %s, Gateway: %s, IPAMDisabled: false}", vifName, ipv4Address, ipv4Mask, mac, ipv4Gateway)))
		})
		It("returns correct string representation with ipv6", func() {
			vif := createDummyVIF(vifName, ipv4Cidr, ipv4Gateway, ipv6Cidr, mac)
			Expect(vif.String()).To(Equal(fmt.Sprintf("VIF: { Name: %s, IP: %s, Mask: %s, IPv6: %s, MAC: %s, Gateway: %s, IPAMDisabled: false}", vifName, ipv4Address, ipv4Mask, ipv6Cidr, mac, ipv4Gateway)))
		})
	})
})

func createDummyVIF(vifName, ipv4cidr, ipv4gateway, ipv6cidr, macStr string) *VIF {
	addr, _ := netlink.ParseAddr(ipv4cidr)
	mac, _ := net.ParseMAC(macStr)
	gw := net.ParseIP(ipv4gateway)
	vif := &VIF{
		Name:    vifName,
		IP:      *addr,
		MAC:     mac,
		Gateway: gw,
	}
	if ipv6cidr != "" {
		ipv6Addr, _ := netlink.ParseAddr(ipv6cidr)
		vif.IPv6 = *ipv6Addr
	}

	return vif
}
