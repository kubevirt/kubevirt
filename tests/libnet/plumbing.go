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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package libnet

import (
	"context"
	"fmt"

	nadv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"
)

func CreateNAD(namespace, nadName string) error {
	nadSpec := newNetworkAttachmentDefinitionUsingBridgeCNI(nadName)
	kubevirtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		return err
	}

	_, err = kubevirtClient.NetworkClient().K8sCniCncfIoV1().NetworkAttachmentDefinitions(namespace).Create(
		context.Background(),
		nadSpec,
		metav1.CreateOptions{},
	)
	return err
}

func newNetworkAttachmentDefinitionUsingBridgeCNI(networkName string) *nadv1.NetworkAttachmentDefinition {
	config := fmt.Sprintf(`{"cniVersion": "0.3.1", "name": %q, "type": "cnv-bridge", "bridge": %q}`, networkName, networkName)
	return &nadv1.NetworkAttachmentDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: networkName,
		},
		Spec: nadv1.NetworkAttachmentDefinitionSpec{Config: config},
	}
}
