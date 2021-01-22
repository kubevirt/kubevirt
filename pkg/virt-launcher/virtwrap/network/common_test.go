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
	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

func mockVirtLauncherCachedPattern(path string) {
	virtLauncherCachedPattern = path
}

var _ = Describe("Common Methods", func() {
	pid := "self"
	Context("Functions Read and Write from cache", func() {
		It("should persist interface payload", func() {
			tmpDir, err := ioutil.TempDir("", "commontest")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(tmpDir)
			mockVirtLauncherCachedPattern(tmpDir + "/cache-%s.json")

			ifaceName := "iface_name"
			iface := api.Interface{Type: "fake_type", Source: api.InterfaceSource{Bridge: "fake_br"}}
			err = writeToVirtLauncherCachedFile(&iface, pid, ifaceName)
			Expect(err).ToNot(HaveOccurred())

			var cached_iface api.Interface
			err = readFromVirtLauncherCachedFile(&cached_iface, pid, ifaceName)
			Expect(err).ToNot(HaveOccurred())

			Expect(iface).To(Equal(cached_iface))
		})
		It("should persist qemu arg payload", func() {
			tmpDir, err := ioutil.TempDir("", "commontest")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(tmpDir)
			mockVirtLauncherCachedPattern(tmpDir + "/cache-%s.json")

			qemuArgName := "iface_name"
			qemuArg := api.Arg{Value: "test_value"}
			err = writeToVirtLauncherCachedFile(&qemuArg, pid, qemuArgName)
			Expect(err).ToNot(HaveOccurred())

			var cached_qemuArg api.Arg
			err = readFromVirtLauncherCachedFile(&cached_qemuArg, pid, qemuArgName)
			Expect(err).ToNot(HaveOccurred())

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
	const mtu = 1450
	const vifName = "test-vif"

	Context("String", func() {
		It("returns correct string representation", func() {
			vif := createDummyVIF(vifName, ipv4Cidr, ipv4Gateway, "", mac, mtu)
			Expect(vif.String()).To(Equal(fmt.Sprintf("VIF: { Name: %s, IP: %s, Mask: %s, IPv6: <nil>, MAC: %s, Gateway: %s, MTU: %d, IPAMDisabled: false}", vifName, ipv4Address, ipv4Mask, mac, ipv4Gateway, mtu)))
		})
		It("returns correct string representation with ipv6", func() {
			vif := createDummyVIF(vifName, ipv4Cidr, ipv4Gateway, ipv6Cidr, mac, mtu)
			Expect(vif.String()).To(Equal(fmt.Sprintf("VIF: { Name: %s, IP: %s, Mask: %s, IPv6: %s, MAC: %s, Gateway: %s, MTU: %d, IPAMDisabled: false}", vifName, ipv4Address, ipv4Mask, ipv6Cidr, mac, ipv4Gateway, mtu)))
		})
	})
})

func createDummyVIF(vifName, ipv4cidr, ipv4gateway, ipv6cidr, macStr string, mtu uint16) *VIF {
	addr, _ := netlink.ParseAddr(ipv4cidr)
	mac, _ := net.ParseMAC(macStr)
	gw := net.ParseIP(ipv4gateway)
	vif := &VIF{
		Name:    vifName,
		IP:      *addr,
		MAC:     mac,
		Gateway: gw,
		Mtu:     mtu,
	}
	if ipv6cidr != "" {
		ipv6Addr, _ := netlink.ParseAddr(ipv6cidr)
		vif.IPv6 = *ipv6Addr
	}

	return vif
}

var _ = Describe("infocache", func() {
	type SillyType struct{ A int }
	It("readFromCachedFile reads what writeToCachedFile had written", func() {
		written := SillyType{7}
		read := SillyType{0}
		tmpfile, err := ioutil.TempFile("", "silly")
		Expect(err).ToNot(HaveOccurred())
		defer os.Remove(tmpfile.Name())

		writeToCachedFile(written, tmpfile.Name())
		readFromCachedFile(&read, tmpfile.Name())
		Expect(written).To(Equal(read))
	})
	It("ReadFromVirtHandlerCachedFile reads what WriteToVirtHandlerCachedFile had written", func() {
		vmiuid := types.UID("123")
		written := SillyType{7}
		read := SillyType{0}

		err := CreateVirtHandlerCacheDir(vmiuid)
		Expect(err).ToNot(HaveOccurred())
		defer RemoveVirtHandlerCacheDir(vmiuid)

		err = WriteToVirtHandlerCachedFile(written, vmiuid, "eth0")
		Expect(err).ToNot(HaveOccurred())
		err = ReadFromVirtHandlerCachedFile(&read, vmiuid, "eth0")
		Expect(err).ToNot(HaveOccurred())
		Expect(written).To(Equal(read))
	})
	It("readFromVirtLauncherCachedFile reads what writeToVirtLauncherCachedFile had written", func() {
		tmpDir, err := ioutil.TempDir("", "commontest")
		Expect(err).ToNot(HaveOccurred())
		defer os.RemoveAll(tmpDir)
		mockVirtLauncherCachedPattern(tmpDir + "/cache-%s.json")

		pid := "123"
		written := SillyType{7}
		read := SillyType{0}

		err = writeToVirtLauncherCachedFile(written, pid, "eth0")
		Expect(err).ToNot(HaveOccurred())
		err = readFromVirtLauncherCachedFile(&read, pid, "eth0")
		Expect(err).ToNot(HaveOccurred())
		Expect(written).To(Equal(read))
	})
})
