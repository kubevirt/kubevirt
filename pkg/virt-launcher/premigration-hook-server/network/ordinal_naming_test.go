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

package network_test

import (
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"libvirt.org/go/libvirtxml"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-launcher/premigration-hook-server/network"
)

var _ = Describe("UpgradeOrdinalNamingScheme", func() {
	const (
		primaryNetworkName = "default"
		primaryTapName     = "tap0"

		secondaryNetwork1Name    = "secondary1"
		secondaryNetwork1NADName = "nad1"
		secondaryOrdinal1TapName = "tap1"
		secondaryHashed1TapName  = "tapbf1967901de"

		secondaryNetwork2Name    = "secondary2"
		secondaryNetwork2NADName = "nad2"
		secondaryOrdinal2TapName = "tap2"
		secondaryHashed2TapName  = "tap00a86454af9"
	)

	It("should do nothing if there aren't any interfaces", func() {
		domain := newDomainWithInterfaces(nil)
		Expect(network.UpgradeOrdinalNamingScheme(libvmi.New(), &domain)).To(Succeed())
		Expect(domain).To(Equal(newDomainWithInterfaces(nil)))
	})

	It("should do nothing if there is only a primary interface with a tap based binding", func() {
		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)

		existingIfaces := []libvirtxml.DomainInterface{
			newIface(primaryNetworkName, primaryTapName),
		}

		expectedIfaces := slices.Clone(existingIfaces)

		domain := newDomainWithInterfaces(existingIfaces)
		Expect(network.UpgradeOrdinalNamingScheme(vmi, &domain)).To(Succeed())

		expectedDomain := newDomainWithInterfaces(expectedIfaces)
		Expect(domain).To(Equal(expectedDomain))
	})

	It("should do nothing if there are secondary networks with hashed naming scheme", func() {
		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(secondaryNetwork1Name)),
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(secondaryNetwork2Name)),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetwork1Name, secondaryNetwork1NADName)),
			libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetwork2Name, secondaryNetwork2NADName)),
		)

		existingIfaces := []libvirtxml.DomainInterface{
			newIface(primaryNetworkName, primaryTapName),
			newIface(secondaryNetwork1Name, secondaryHashed1TapName),
			newIface(secondaryNetwork2Name, secondaryHashed2TapName),
		}

		expectedIfaces := slices.Clone(existingIfaces)

		domain := newDomainWithInterfaces(existingIfaces)
		Expect(network.UpgradeOrdinalNamingScheme(vmi, &domain)).To(Succeed())

		expectedDomain := newDomainWithInterfaces(expectedIfaces)
		Expect(domain).To(Equal(expectedDomain))
	})

	It("should convert the ordinal to hashed naming scheme", func() {
		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(secondaryNetwork1Name)),
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(secondaryNetwork2Name)),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetwork1Name, secondaryNetwork1NADName)),
			libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetwork2Name, secondaryNetwork2NADName)),
		)

		existingIfaces := []libvirtxml.DomainInterface{
			newIface(primaryNetworkName, primaryTapName),
			newIface(secondaryNetwork1Name, secondaryOrdinal1TapName),
			newIface(secondaryNetwork2Name, secondaryOrdinal2TapName),
		}

		domain := newDomainWithInterfaces(existingIfaces)
		Expect(network.UpgradeOrdinalNamingScheme(vmi, &domain)).To(Succeed())

		expectedIfaces := []libvirtxml.DomainInterface{
			newIface(primaryNetworkName, primaryTapName),
			newIface(secondaryNetwork1Name, secondaryHashed1TapName),
			newIface(secondaryNetwork2Name, secondaryHashed2TapName),
		}
		expectedDomain := newDomainWithInterfaces(expectedIfaces)
		Expect(domain).To(Equal(expectedDomain))
	})
})

func newDomainWithInterfaces(ifaces []libvirtxml.DomainInterface) libvirtxml.Domain {
	return libvirtxml.Domain{
		Devices: &libvirtxml.DomainDeviceList{Interfaces: ifaces},
	}
}

func newIface(netName, devName string) libvirtxml.DomainInterface {
	return libvirtxml.DomainInterface{
		Alias: newAlias(netName),
		Target: &libvirtxml.DomainInterfaceTarget{
			Dev: devName,
		},
	}
}

func newAlias(netName string) *libvirtxml.DomainAlias {
	return &libvirtxml.DomainAlias{
		Name: "ua-" + netName,
	}
}
