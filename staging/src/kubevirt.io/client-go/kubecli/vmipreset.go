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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package kubecli

import (
	"context"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"

	v1 "kubevirt.io/api/core/v1"
	kvcorev1 "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/core/v1"
)

func (k *kubevirt) VirtualMachineInstancePreset(namespace string) VirtualMachineInstancePresetInterface {
	return &vmiPresets{
		k.GeneratedKubeVirtClient().KubevirtV1().VirtualMachineInstancePresets(namespace),
		k.restClient,
		namespace,
		"virtualmachineinstancepresets",
	}
}

type vmiPresets struct {
	kvcorev1.VirtualMachineInstancePresetInterface
	restClient *rest.RESTClient
	namespace  string
	resource   string
}

func (v *vmiPresets) Get(name string, options k8smetav1.GetOptions) (vmi *v1.VirtualMachineInstancePreset, err error) {
	vmi, err = v.VirtualMachineInstancePresetInterface.Get(context.Background(), name, options)
	vmi.SetGroupVersionKind(v1.VirtualMachineInstancePresetGroupVersionKind)
	return
}

func (v *vmiPresets) List(options k8smetav1.ListOptions) (vmiPresetList *v1.VirtualMachineInstancePresetList, err error) {
	vmiPresetList, err = v.VirtualMachineInstancePresetInterface.List(context.Background(), options)
	for i := range vmiPresetList.Items {
		vmiPresetList.Items[i].SetGroupVersionKind(v1.VirtualMachineInstancePresetGroupVersionKind)
	}

	return
}

func (v *vmiPresets) Create(vmi *v1.VirtualMachineInstancePreset) (result *v1.VirtualMachineInstancePreset, err error) {
	result, err = v.VirtualMachineInstancePresetInterface.Create(context.Background(), vmi, k8smetav1.CreateOptions{})
	result.SetGroupVersionKind(v1.VirtualMachineInstancePresetGroupVersionKind)
	return
}

func (v *vmiPresets) Update(vmi *v1.VirtualMachineInstancePreset) (result *v1.VirtualMachineInstancePreset, err error) {
	result, err = v.VirtualMachineInstancePresetInterface.Update(context.Background(), vmi, k8smetav1.UpdateOptions{})
	result.SetGroupVersionKind(v1.VirtualMachineInstancePresetGroupVersionKind)
	return
}

func (v *vmiPresets) Delete(name string, options *k8smetav1.DeleteOptions) error {
	opts := k8smetav1.DeleteOptions{}
	if options != nil {
		opts = *options
	}
	return v.VirtualMachineInstancePresetInterface.Delete(context.Background(), name, opts)
}

func (v *vmiPresets) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.VirtualMachineInstancePreset, err error) {
	return v.VirtualMachineInstancePresetInterface.Patch(context.Background(), name, pt, data, k8smetav1.PatchOptions{}, subresources...)
}
