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
