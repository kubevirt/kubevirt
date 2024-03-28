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
	"fmt"
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-controller/network"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	fakenetworkclient "kubevirt.io/client-go/generated/network-attachment-definition-client/clientset/versioned/fake"

	"kubevirt.io/client-go/testutils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	nadSuffix              = "-net"
	redNetworkLogicalName  = "red"
	redNetworkNadName      = redNetworkLogicalName + nadSuffix
	blueNetworkLogicalName = "blue"
	blueNetworkNadName     = blueNetworkLogicalName + nadSuffix
	namespace              = "test-ns"
	resourceName           = "resource_name"
)

func TestNetwork(t *testing.T) {
	testutils.KubeVirtTestSuiteSetup(t)
}

func logicalSecondaryNetwork(logicalName string, nadName string) v1.Network {
	return v1.Network{
		Name:          logicalName,
		NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: nadName}},
	}
}

func createNADs(networkClient *fakenetworkclient.Clientset, namespace string, networks []v1.Network, persistentIPs map[string]struct{}) error {
	gvr := schema.GroupVersionResource{
		Group:    "k8s.cni.cncf.io",
		Version:  "v1",
		Resource: "network-attachment-definitions",
	}
	for _, net := range networks {
		if net.Multus == nil {
			continue
		}

		nad := &networkv1.NetworkAttachmentDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name:        net.NetworkSource.Multus.NetworkName,
				Namespace:   namespace,
				Annotations: map[string]string{network.MULTUS_RESOURCE_NAME_ANNOTATION: resourceName},
			},
		}

		if _, exists := persistentIPs[net.NetworkSource.Multus.NetworkName]; exists {
			nad.Spec.Config = fmt.Sprintf(`{"allowPersistentIPs": true, "name": "%s"}`, nadNetworkName)
		}

		err := networkClient.Tracker().Create(gvr, nad, namespace)
		if err != nil {
			return err
		}
	}

	return nil
}
