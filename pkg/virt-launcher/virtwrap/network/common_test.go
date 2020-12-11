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
	Context("String", func() {
		It("returns correct string representation", func() {
			addr, _ := netlink.ParseAddr("10.0.0.200/24")
			mac, _ := net.ParseMAC("de:ad:00:00:be:ef")
			gw := net.ParseIP("10.0.0.1")
			vif := &VIF{
				Name:      "test-vif",
				IP:        *addr,
				MAC:       mac,
				Gateway:   gw,
				Mtu:       1450,
				TapDevice: "myTap0",
			}
			Expect(vif.String()).To(Equal("VIF: { Name: test-vif, IP: 10.0.0.200, Mask: ffffff00, MAC: de:ad:00:00:be:ef, Gateway: 10.0.0.1, MTU: 1450, IPAMDisabled: false, TapDevice: myTap0}"))
		})
	})
})
