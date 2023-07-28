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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package webhooks

import (
	"fmt"
	"runtime"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	poolv1 "kubevirt.io/api/pool/v1alpha1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"

	v1 "kubevirt.io/api/core/v1"
	clientutil "kubevirt.io/client-go/util"
)

var Arch = runtime.GOARCH

var VirtualMachineInstanceGroupVersionResource = metav1.GroupVersionResource{
	Group:    v1.VirtualMachineInstanceGroupVersionKind.Group,
	Version:  v1.VirtualMachineInstanceGroupVersionKind.Version,
	Resource: "virtualmachineinstances",
}

var VirtualMachineGroupVersionResource = metav1.GroupVersionResource{
	Group:    v1.VirtualMachineGroupVersionKind.Group,
	Version:  v1.VirtualMachineGroupVersionKind.Version,
	Resource: "virtualmachines",
}

var VirtualMachineInstancePresetGroupVersionResource = metav1.GroupVersionResource{
	Group:    v1.VirtualMachineInstancePresetGroupVersionKind.Group,
	Version:  v1.VirtualMachineInstancePresetGroupVersionKind.Version,
	Resource: "virtualmachineinstancepresets",
}

var VirtualMachineInstanceReplicaSetGroupVersionResource = metav1.GroupVersionResource{
	Group:    v1.VirtualMachineInstanceReplicaSetGroupVersionKind.Group,
	Version:  v1.VirtualMachineInstanceReplicaSetGroupVersionKind.Version,
	Resource: "virtualmachineinstancereplicasets",
}

var VirtualMachinePoolGroupVersionResource = metav1.GroupVersionResource{
	Group:    poolv1.SchemeGroupVersion.Group,
	Version:  poolv1.SchemeGroupVersion.Version,
	Resource: "virtualmachinepools",
}

var MigrationGroupVersionResource = metav1.GroupVersionResource{
	Group:    v1.VirtualMachineInstanceMigrationGroupVersionKind.Group,
	Version:  v1.VirtualMachineInstanceMigrationGroupVersionKind.Version,
	Resource: "virtualmachineinstancemigrations",
}

type Informers struct {
	VMIPresetInformer  cache.SharedIndexInformer
	VMRestoreInformer  cache.SharedIndexInformer
	DataSourceInformer cache.SharedIndexInformer
	NamespaceInformer  cache.SharedIndexInformer
}

func IsKubeVirtServiceAccount(serviceAccount string) bool {
	ns, err := clientutil.GetNamespace()
	logger := log.DefaultLogger()

	if err != nil {
		logger.Info("Failed to get namespace. Fallback to default: 'kubevirt'")
		ns = "kubevirt"
	}

	prefix := fmt.Sprintf("system:serviceaccount:%s", ns)
	return serviceAccount == fmt.Sprintf("%s:%s", prefix, components.ApiServiceAccountName) ||
		serviceAccount == fmt.Sprintf("%s:%s", prefix, components.HandlerServiceAccountName) ||
		serviceAccount == fmt.Sprintf("%s:%s", prefix, components.ControllerServiceAccountName)
}

func IsARM64(vmiSpec *v1.VirtualMachineInstanceSpec) bool {
	if vmiSpec.Architecture == "arm64" {
		return true
	}
	return false
}

func IsPPC64(vmiSpec *v1.VirtualMachineInstanceSpec) bool {
	if vmiSpec.Architecture == "ppc64le" {
		return true
	}
	return false
}
