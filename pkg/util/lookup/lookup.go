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

package lookup

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

func VirtualMachinesOnNode(cli kubecli.KubevirtClient, nodeName string) ([]*virtv1.VirtualMachineInstance, error) {
	labelSelector, err := labels.Parse(fmt.Sprintf("%s in (%s)", virtv1.NodeNameLabel, nodeName))
	if err != nil {
		return nil, err
	}
	list, err := cli.VirtualMachineInstance(v1.NamespaceAll).List(context.Background(), metav1.ListOptions{
		LabelSelector: labelSelector.String(),
	})

	if err != nil {
		return nil, err
	}

	vmis := []*virtv1.VirtualMachineInstance{}

	for i := range list.Items {
		vmis = append(vmis, &list.Items[i])
	}
	return vmis, nil
}

func ActiveVirtualMachinesOnNode(cli kubecli.KubevirtClient, nodeName string) ([]*virtv1.VirtualMachineInstance, error) {
	vmis, err := VirtualMachinesOnNode(cli, nodeName)
	if err != nil {
		return nil, err
	}

	activeVMIs := []*virtv1.VirtualMachineInstance{}

	for _, vmi := range vmis {
		if !vmi.IsRunning() && !vmi.IsScheduled() {
			continue
		}

		activeVMIs = append(activeVMIs, vmi)
	}

	return activeVMIs, nil
}
