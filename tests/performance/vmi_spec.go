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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package performance

import (
	"k8s.io/apimachinery/pkg/api/resource"

	k8sv1 "k8s.io/api/core/v1"

	kvv1 "kubevirt.io/client-go/api/v1"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
)

func createVMISpecWithResources(virtClient kubecli.KubevirtClient) *kvv1.VirtualMachineInstance {
	vmImage := cd.ContainerDiskFor("cirros")
	cpuLimit := "100m"
	memLimit := "90Mi"
	cloudInitUserData := "#!/bin/bash\necho 'hello'\n"
	vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(vmImage, cloudInitUserData)
	vmi.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
		k8sv1.ResourceMemory: resource.MustParse(memLimit),
		k8sv1.ResourceCPU:    resource.MustParse(cpuLimit),
	}
	vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
		k8sv1.ResourceMemory: resource.MustParse(memLimit),
		k8sv1.ResourceCPU:    resource.MustParse(cpuLimit),
	}
	return vmi
}
