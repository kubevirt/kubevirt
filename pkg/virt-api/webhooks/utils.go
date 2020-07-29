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
	"sync"

	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/virt-operator/creation/rbac"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	clientutil "kubevirt.io/client-go/util"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/util/openapi"
	"kubevirt.io/kubevirt/pkg/virt-api/rest"
)

var webhookInformers *Informers
var once sync.Once

var Validator = openapi.CreateOpenAPIValidator(rest.ComposeAPIDefinitions())

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

var MigrationGroupVersionResource = metav1.GroupVersionResource{
	Group:    v1.VirtualMachineInstanceMigrationGroupVersionKind.Group,
	Version:  v1.VirtualMachineInstanceMigrationGroupVersionKind.Version,
	Resource: "virtualmachineinstancemigrations",
}

type Informers struct {
	VMIPresetInformer       cache.SharedIndexInformer
	NamespaceLimitsInformer cache.SharedIndexInformer
	VMIInformer             cache.SharedIndexInformer
}

// XXX fix this, this is a huge mess. Move informers to Admitter and Mutator structs.
var mutex sync.Mutex

func GetInformers() *Informers {
	mutex.Lock()
	defer mutex.Unlock()
	if webhookInformers == nil {
		webhookInformers = newInformers()
	}
	return webhookInformers
}

// SetInformers created for unittest usage only
func SetInformers(informers *Informers) {
	mutex.Lock()
	defer mutex.Unlock()
	webhookInformers = informers
}

func newInformers() *Informers {
	kubeClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		panic(err)
	}
	namespace, err := clientutil.GetNamespace()
	if err != nil {
		glog.Fatalf("Error searching for namespace: %v", err)
	}
	kubeInformerFactory := controller.NewKubeInformerFactory(kubeClient.RestClient(), kubeClient, nil, namespace)
	return &Informers{
		VMIInformer:             kubeInformerFactory.VMI(),
		VMIPresetInformer:       kubeInformerFactory.VirtualMachinePreset(),
		NamespaceLimitsInformer: kubeInformerFactory.LimitRanges(),
	}
}

func GetAllowedServiceAccounts() map[string]struct{} {
	ns, err := clientutil.GetNamespace()
	logger := log.DefaultLogger()

	if err != nil {
		logger.Info("Failed to get namespace. Fallback to default: 'kubevirt'")
		ns = "kubevirt"
	}

	// system:serviceaccount:{namespace}:{kubevirt-component}
	prefix := fmt.Sprintf("%s:%s:%s", "system", "serviceaccount", ns)
	return map[string]struct{}{
		fmt.Sprintf("%s:%s", prefix, rbac.ApiServiceAccountName):        {},
		fmt.Sprintf("%s:%s", prefix, rbac.HandlerServiceAccountName):    {},
		fmt.Sprintf("%s:%s", prefix, rbac.ControllerServiceAccountName): {},
	}
}
