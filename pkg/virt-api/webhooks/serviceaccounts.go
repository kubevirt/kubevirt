/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package webhooks

import (
	"fmt"

	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
)

func KubeVirtServiceAccounts(kubeVirtNamespace string) map[string]struct{} {
	prefix := fmt.Sprintf("system:serviceaccount:%s", kubeVirtNamespace)

	return map[string]struct{}{
		fmt.Sprintf("%s:%s", prefix, components.ApiServiceAccountName):                       {},
		fmt.Sprintf("%s:%s", prefix, components.ControllerServiceAccountName):                {},
		fmt.Sprintf("%s:%s", prefix, components.HandlerServiceAccountName):                   {},
		fmt.Sprintf("%s:%s", prefix, components.SynchronizationControllerServiceAccountName): {},
	}
}
