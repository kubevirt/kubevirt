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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package link

import (
	"errors"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/api/core/v1"

	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
)

var _ = Describe("DiscoverByNetwork", func() {
	const (
		testNetworkName         = "blue"
		testNetIfaceName        = "pod16477688c0e"
		testNetOrdinalIfaceName = "net1"
	)

	var (
		ctrl               *gomock.Controller
		mockNetworkHandler *netdriver.MockNetworkHandler
	)

	testNetworks := func() []v1.Network {
		return []v1.Network{
			*v1.DefaultPodNetwork(),
			multusNetwork(testNetworkName),
			multusNetwork("A"),
		}
	}

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockNetworkHandler = netdriver.NewMockNetworkHandler(ctrl)
	})

	It("should fail when the given no networks", func() {
		_, err := DiscoverByNetwork(mockNetworkHandler, nil, v1.Network{Name: "ensp1f2"})

		Expect(err).To(HaveOccurred())
	})

	It("should fail given a network name that not exist in networks slice", func() {
		_, err := DiscoverByNetwork(mockNetworkHandler, testNetworks(), v1.Network{Name: "ensp1f2"})

		Expect(err).To(HaveOccurred())
	})

	It("should fail when link not found", func() {
		mockNetworkHandler.EXPECT().LinkByName(testNetIfaceName).Return(nil, errors.New("test fail"))
		mockNetworkHandler.EXPECT().LinkByName(testNetOrdinalIfaceName).Return(nil, errors.New("test fail"))

		_, err := DiscoverByNetwork(mockNetworkHandler, testNetworks(), multusNetwork(testNetworkName))

		Expect(err).To(HaveOccurred())
	})

	It("should get default network iface link", func() {
		mockNetworkHandler.EXPECT().LinkByName("eth0").Return(&netlink.Bridge{LinkAttrs: netlink.LinkAttrs{Name: "eth0"}}, nil)

		actualLink, err := DiscoverByNetwork(mockNetworkHandler, testNetworks(), *v1.DefaultPodNetwork())

		Expect(err).ToNot(HaveOccurred())
		Expect(actualLink.Attrs().Name).To(Equal("eth0"))
	})

	It("should get network iface link", func() {
		mockNetworkHandler.EXPECT().LinkByName(testNetIfaceName).Return(&netlink.Bridge{LinkAttrs: netlink.LinkAttrs{Name: testNetIfaceName}}, nil)

		actualLink, err := DiscoverByNetwork(mockNetworkHandler, testNetworks(), multusNetwork(testNetworkName))

		Expect(err).ToNot(HaveOccurred())
		Expect(actualLink.Attrs().Name).To(Equal(testNetIfaceName))
	})

	It("when network iface link not found, should get link using ordinal iface name", func() {
		mockNetworkHandler.EXPECT().LinkByName(testNetIfaceName).Return(nil, errors.New("test fail"))
		mockNetworkHandler.EXPECT().LinkByName(testNetOrdinalIfaceName).Return(&netlink.Bridge{LinkAttrs: netlink.LinkAttrs{Name: testNetOrdinalIfaceName}}, nil)

		actualLink, err := DiscoverByNetwork(mockNetworkHandler, testNetworks(), multusNetwork(testNetworkName))

		Expect(err).ToNot(HaveOccurred())
		Expect(actualLink.Attrs().Name).To(Equal(testNetOrdinalIfaceName))
	})
})

func multusNetwork(name string) v1.Network {
	return v1.Network{
		Name:          name,
		NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: name + "net"}},
	}
}
