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
 * Copyright 2020 Red Hat, Inc.
 *
 */

package libvmi

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	kutil "kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/tests/libpod"
)

func GetRunningVirtualMachineInstanceDomainXML(virtClient kubecli.KubevirtClient,
	vmi *v1.VirtualMachineInstance, namespace string) (string, error) {
	vmiPod, err := libpod.GetRunningPodByVirtualMachineInstance(vmi, namespace)
	if err != nil {
		return "", err
	}

	found := false
	containerIdx := 0
	for idx, container := range vmiPod.Spec.Containers {
		if container.Name == "compute" {
			containerIdx = idx
			found = true
		}
	}
	if !found {
		return "", fmt.Errorf("%s", libpod.CouldNotFindComputeContainer)
	}

	// get current vmi
	freshVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get vmi, %s", err)
	}

	command := []string{"virsh"}
	if kutil.IsNonRootVMI(freshVMI) {
		command = append(command, "-c", "qemu+unix:///session?socket=/var/run/libvirt/libvirt-sock")
	}
	command = append(command, []string{"dumpxml", vmi.Namespace + "_" + vmi.Name}...)

	stdout, stderr, err := libpod.RunCommandV2(
		virtClient,
		vmiPod,
		vmiPod.Spec.Containers[containerIdx].Name,
		command,
	)
	if err != nil {
		return "", fmt.Errorf("could not dump libvirt domxml (remotely on pod): %v: %s", err, stderr)
	}
	return stdout, err
}
