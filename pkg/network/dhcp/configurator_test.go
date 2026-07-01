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
 */

package dhcp

import (
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/cache"
)

var _ = Describe("DHCP configurator", func() {

	const (
		bridgeName = "br0"
		ifaceName  = "eth0"
	)

	var (
		dhcpStartedDir string
		cfg            *configurator
		dhcpConfig     cache.DHCPConfig
		dhcpOptions    *v1.DHCPOptions
	)

	BeforeEach(func() {
		var err error
		dhcpStartedDir, err = os.MkdirTemp("", "dhcp-started-")
		Expect(err).NotTo(HaveOccurred())

		dhcpConfig = cache.DHCPConfig{
			Name: ifaceName,
			Mtu:  1400,
		}
	})

	AfterEach(func() {
		Expect(os.RemoveAll(dhcpStartedDir)).To(Succeed())
	})

	newConfigurator := func(startFunc dhcpStartFunc) *configurator {
		return &configurator{
			advertisingIfaceName: bridgeName,
			dhcpStartedDirectory: dhcpStartedDir,
			startDHCPFunc:        startFunc,
		}
	}

	Context("EnsureDHCPServerStarted", func() {
		It("should succeed when DHCP server starts", func() {
			cfg = newConfigurator(func(_ *cache.DHCPConfig, _ string, _ *v1.DHCPOptions) error {
				return nil
			})

			Expect(cfg.EnsureDHCPServerStarted(ifaceName, dhcpConfig, dhcpOptions)).To(Succeed())
		})

		It("should only start the DHCP server once for the same interface", func() {
			callCount := 0
			cfg = newConfigurator(func(_ *cache.DHCPConfig, _ string, _ *v1.DHCPOptions) error {
				callCount++
				return nil
			})

			Expect(cfg.EnsureDHCPServerStarted(ifaceName, dhcpConfig, dhcpOptions)).To(Succeed())
			Expect(cfg.EnsureDHCPServerStarted(ifaceName, dhcpConfig, dhcpOptions)).To(Succeed())
			Expect(callCount).To(Equal(1))
		})

		It("should fail when DHCP server fails to start", func() {
			cfg = newConfigurator(func(_ *cache.DHCPConfig, _ string, _ *v1.DHCPOptions) error {
				return fmt.Errorf("failed to start DHCP server")
			})

			Expect(cfg.EnsureDHCPServerStarted(ifaceName, dhcpConfig, dhcpOptions)).To(HaveOccurred())
		})

		When("IPAM is disabled", func() {
			BeforeEach(func() {
				dhcpConfig.IPAMDisabled = true
			})

			It("should skip starting the DHCP server", func() {
				cfg = newConfigurator(func(_ *cache.DHCPConfig, _ string, _ *v1.DHCPOptions) error {
					Fail("startDHCP should not be called when IPAM is disabled")
					return nil
				})

				Expect(cfg.EnsureDHCPServerStarted(ifaceName, dhcpConfig, dhcpOptions)).To(Succeed())
			})
		})
	})
})
