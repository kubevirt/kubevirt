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

package network_test

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"

	"github.com/vishvananda/netlink"

	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	network "kubevirt.io/kubevirt/pkg/network/setup"
)

var _ = Describe("Unpluggedpodnic", func() {
	var (
		mockHandler     *netdriver.MockNetworkHandler
		ctrl            *gomock.Controller
		unpluggedpodnic network.Unpluggedpodnic
		tapLink         *netlink.GenericLink
		dummyIfaceLink  *netlink.GenericLink
		bridgeLink      *netlink.GenericLink
	)
	const (
		tapName        = "tap16477688c0e"
		dummyIfaceName = "16477688c0e-nic"
		bridgeName     = "k6t-16477688c0e"
		networkName    = "blue"
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockHandler = netdriver.NewMockNetworkHandler(ctrl)

		unpluggedpodnic = network.NewUnpluggedpodnic(v1.Network{Name: networkName, NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{}}}, mockHandler)

		tapLink = &netlink.GenericLink{}
		dummyIfaceLink = &netlink.GenericLink{}
		bridgeLink = &netlink.GenericLink{}
	})

	Context("UnplugPhase1", func() {
		It("successfully when bridge, tap and dummy nic exist", func() {
			mockHandler.EXPECT().LinkByName(tapName).Return(tapLink, nil)
			mockHandler.EXPECT().LinkByName(dummyIfaceName).Return(dummyIfaceLink, nil)
			mockHandler.EXPECT().LinkByName(bridgeName).Return(bridgeLink, nil)
			mockHandler.EXPECT().LinkDel(tapLink).Return(nil)
			mockHandler.EXPECT().LinkDel(dummyIfaceLink).Return(nil)
			mockHandler.EXPECT().LinkDel(bridgeLink).Return(nil)

			Expect(unpluggedpodnic.UnplugPhase1()).To(Succeed())
		})
		It("successfully when dummy nic doesn't exist", func() {
			mockHandler.EXPECT().LinkByName(tapName).Return(tapLink, nil)
			mockHandler.EXPECT().LinkByName(dummyIfaceName).Return(nil, netlink.LinkNotFoundError{})
			mockHandler.EXPECT().LinkByName(bridgeName).Return(bridgeLink, nil)
			mockHandler.EXPECT().LinkDel(tapLink).Return(nil)
			mockHandler.EXPECT().LinkDel(bridgeLink).Return(nil)

			Expect(unpluggedpodnic.UnplugPhase1()).To(Succeed())
		})
		It("partially fails when some of the devices fail to be removed", func() {
			const (
				linkByNameErr1 = "link by name error1"
				linkByNameErr2 = "link by name error2"
			)

			mockHandler.EXPECT().LinkByName(tapName).Return(tapLink, nil)
			mockHandler.EXPECT().LinkByName(dummyIfaceName).Return(nil, fmt.Errorf(linkByNameErr1))
			mockHandler.EXPECT().LinkByName(bridgeName).Return(bridgeLink, fmt.Errorf(linkByNameErr2))
			mockHandler.EXPECT().LinkDel(tapLink).Return(nil)

			err := unpluggedpodnic.UnplugPhase1()
			Expect(err.Error()).To(ContainSubstring(linkByNameErr1))
			Expect(err.Error()).To(ContainSubstring(linkByNameErr1))
		})
	})
})
