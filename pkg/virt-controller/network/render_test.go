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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/virt-controller/network"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	fakenetworkclient "kubevirt.io/client-go/generated/network-attachment-definition-client/clientset/versioned/fake"
)

const (
	nadSuffix              = "-net"
	nsSuffix               = "-ns"
	redNetworkLogicalName  = "red"
	redNamespace           = redNetworkLogicalName + nsSuffix
	redNetworkNadName      = redNetworkLogicalName + nadSuffix
	blueNetworkLogicalName = "blue"
	blueNetworkNadName     = blueNetworkLogicalName + nadSuffix
	defaultNamespace       = "default"
	resourceName           = "resource_name"
)

var _ = Describe("GetNetworkAttachmentDefinitionByName", func() {
	var (
		networkClient  *fakenetworkclient.Clientset
		multusNetworks []v1.Network
	)

	BeforeEach(func() {
		networkClient = fakenetworkclient.NewSimpleClientset()

		multusNetworks = []v1.Network{
			*libvmi.MultusNetwork(redNetworkLogicalName, redNetworkNadName),
			*libvmi.MultusNetwork(blueNetworkLogicalName, defaultNamespace+"/"+blueNetworkNadName),
		}
		Expect(createNADs(networkClient, redNamespace, multusNetworks)).To(Succeed())
	})

	It("should return map the expected nads", func() {
		nads, err := network.GetNetworkAttachmentDefinitionByName(networkClient.K8sCniCncfIoV1(), redNamespace, multusNetworks)
		Expect(err).ToNot(HaveOccurred())
		Expect(nads).To(HaveLen(2))
		expectedNamespace := map[string]string{redNetworkLogicalName: redNamespace, blueNetworkLogicalName: defaultNamespace}
		for networkName, nad := range nads {
			Expect(nad.Name).To(Equal(networkName + nadSuffix))
			Expect(nad.Namespace).To(Equal(expectedNamespace[networkName]))
		}
	})
})

var _ = Describe("ExtractNetworkToResourceMap", func() {
	It("should return map the expected networkToResourceMap", func() {
		nadMap := map[string]*networkv1.NetworkAttachmentDefinition{
			redNetworkLogicalName: {
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						network.MULTUS_RESOURCE_NAME_ANNOTATION: resourceName,
					},
				},
			},
			blueNetworkLogicalName: {
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						network.MULTUS_RESOURCE_NAME_ANNOTATION: resourceName,
					},
				},
			},
			"dummy": {},
		}

		networkToResourceMap := network.ExtractNetworkToResourceMap(nadMap)
		Expect(networkToResourceMap).To(Equal(map[string]string{
			"red":   resourceName,
			"blue":  resourceName,
			"dummy": "",
		}))
	})
})

func createNADs(networkClient *fakenetworkclient.Clientset, namespace string, networks []v1.Network) error {
	gvr := schema.GroupVersionResource{
		Group:    "k8s.cni.cncf.io",
		Version:  "v1",
		Resource: "network-attachment-definitions",
	}
	for _, net := range networks {
		ns, networkName := vmispec.GetNamespaceAndNetworkName(namespace, net.Multus.NetworkName)
		nad := &networkv1.NetworkAttachmentDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name:        networkName,
				Namespace:   ns,
				Annotations: map[string]string{network.MULTUS_RESOURCE_NAME_ANNOTATION: resourceName},
			},
		}

		err := networkClient.Tracker().Create(gvr, nad, ns)
		if err != nil {
			return err
		}
	}

	return nil
}
