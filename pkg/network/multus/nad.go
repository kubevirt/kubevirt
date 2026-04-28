/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package multus

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/precond"
)

func NetAttachDefNamespacedName(namespace, fullNetworkName string) types.NamespacedName {
	if strings.Contains(fullNetworkName, "/") {
		const twoParts = 2
		res := strings.SplitN(fullNetworkName, "/", twoParts)
		return types.NamespacedName{
			Namespace: res[0],
			Name:      res[1],
		}
	}

	return types.NamespacedName{
		Namespace: precond.MustNotBeEmpty(namespace),
		Name:      fullNetworkName,
	}
}

func NetworkToResource(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) (map[string]string, error) {
	networkToResourceMap := map[string]string{}

	for _, network := range vmi.Spec.Networks {
		if network.Multus == nil {
			continue
		}

		nadNamespacedName := NetAttachDefNamespacedName(vmi.Namespace, network.Multus.NetworkName)
		netAttachDef, err := virtClient.NetworkClient().
			K8sCniCncfIoV1().
			NetworkAttachmentDefinitions(nadNamespacedName.Namespace).
			Get(context.Background(), nadNamespacedName.Name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to locate network attachment definition %s", nadNamespacedName.String())
		}

		networkToResourceMap[network.Name] = netAttachDef.Annotations[ResourceNameAnnotation]
	}

	return networkToResourceMap, nil
}
