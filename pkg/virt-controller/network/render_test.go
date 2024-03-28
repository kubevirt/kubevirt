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
 * Copyright 2024 Red Hat, Inc.
 *
 */

package network_test

import (
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-controller/network"

	fakenetworkclient "kubevirt.io/client-go/generated/network-attachment-definition-client/clientset/versioned/fake"
	"kubevirt.io/client-go/kubecli"
)

var _ = Describe("ExtractNetworkToResourceMap", func() {
	var (
		virtClient     *kubecli.MockKubevirtClient
		networkClient  *fakenetworkclient.Clientset
		multusNetworks []v1.Network
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)

		networkClient = fakenetworkclient.NewSimpleClientset()
		virtClient.EXPECT().NetworkClient().Return(networkClient).AnyTimes()

		multusNetworks = []v1.Network{
			logicalSecondaryNetwork(redNetworkLogicalName, redNetworkNadName),
			logicalSecondaryNetwork(blueNetworkLogicalName, blueNetworkNadName),
		}

		Expect(createNADs(networkClient, namespace, multusNetworks, map[string]struct{}{})).To(Succeed())
	})

	It("should return map the expected networkToResourceMap", func() {
		nads, err := network.GetNetworkAttachmentDefinitions(virtClient, namespace, multusNetworks)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(nads)).To(Equal(2))
		for networkName, nad := range nads {
			Expect(nad.Name).To(Equal(networkName + nadSuffix))
			Expect(nad.Namespace).To(Equal(namespace))
		}

		networkToResourceMap := network.ExtractNetworkToResourceMap(nads)
		Expect(networkToResourceMap).To(Equal(map[string]string{"red": resourceName, "blue": resourceName}))
	})
})
