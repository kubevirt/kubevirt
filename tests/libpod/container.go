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

package libpod

import (
	"fmt"

	virtv1 "kubevirt.io/api/core/v1"

	v1 "k8s.io/api/core/v1"
)

const (
	computeContainerName = "compute"
)

func LookupComputeContainer(pod *v1.Pod) *v1.Container {
	return LookupContainer(pod, computeContainerName)
}

func LookupContainer(pod *v1.Pod, containerName string) *v1.Container {
	for i := range pod.Spec.Containers {
		if pod.Spec.Containers[i].Name == containerName {
			return &pod.Spec.Containers[i]
		}
	}
	panic(fmt.Errorf("could not find the %s container", containerName))
}

func LookupComputeContainerFromVmi(vmi *virtv1.VirtualMachineInstance) (*v1.Container, error) {
	if vmi.Namespace == "" {
		return nil, fmt.Errorf("vmi namespace is empty")
	}

	pod, err := GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
	if err != nil {
		return nil, err
	}

	return LookupComputeContainer(pod), nil
}
