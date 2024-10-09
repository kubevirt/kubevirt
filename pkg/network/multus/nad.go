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

package multus

import (
	"strings"

	"k8s.io/apimachinery/pkg/types"

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
