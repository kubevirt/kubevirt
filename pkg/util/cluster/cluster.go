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
 */

package cluster

import (
	secv1 "github.com/openshift/api/security/v1"
	"k8s.io/client-go/discovery"

	"kubevirt.io/client-go/kubecli"
)

func IsOnOpenShift(clientset kubecli.KubevirtClient) (bool, error) {
	_, apis, err := clientset.DiscoveryClient().ServerGroupsAndResources()
	if err != nil && !discovery.IsGroupDiscoveryFailedError(err) {
		return false, err
	}

	// In case of an error, check if security.openshift.io is the reason (unlikely).
	// If it is, we are obviously on an openshift cluster.
	// Otherwise we can do a positive check.
	if discovery.IsGroupDiscoveryFailedError(err) {
		e := err.(*discovery.ErrGroupDiscoveryFailed)
		if _, exists := e.Groups[secv1.GroupVersion]; exists {
			return true, nil
		}
	}

	for _, api := range apis {
		if api.GroupVersion == secv1.GroupVersion.String() {
			for _, resource := range api.APIResources {
				if resource.Name == "securitycontextconstraints" {
					return true, nil
				}
			}
		}
	}

	return false, nil
}
