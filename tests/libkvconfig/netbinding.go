/*
 * This file is part of the kubevirt project
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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package libkvconfig

import (
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
)

func WithNetBindingPlugin(name string, netBindingPlugin v1.InterfaceBindingPlugin) error {
	return RegisterKubevirtConfigChange(func(c v1.KubeVirtConfiguration) ([]patch.PatchOperation, error) {
		return registerBindingPugins(c, name, netBindingPlugin)
	})
}

func registerBindingPugins(config v1.KubeVirtConfiguration, name string, binding v1.InterfaceBindingPlugin) (
	[]patch.PatchOperation, error) {
	var changePatchOperations []patch.PatchOperation

	if config.NetworkConfiguration == nil {
		changePatchOperations = append(changePatchOperations, networkInitPatch())
	}
	if config.NetworkConfiguration.Binding == nil {
		changePatchOperations = append(changePatchOperations, networkBindingInitPatch())
	}

	changePatchOperations = append(changePatchOperations, networkBindingPatch(name, binding))

	log.Log.Infof("registering binding plugin: %s, %+v", name, binding)

	return changePatchOperations, nil
}

func networkInitPatch() patch.PatchOperation {
	return patch.PatchOperation{
		Op:    patch.PatchAddOp,
		Path:  "/spec/configuration/network",
		Value: v1.NetworkConfiguration{},
	}
}

func networkBindingInitPatch() patch.PatchOperation {
	return patch.PatchOperation{
		Op:    patch.PatchAddOp,
		Path:  "/spec/configuration/network/binding",
		Value: map[string]v1.InterfaceBindingPlugin{},
	}
}

func networkBindingPatch(name string, binding v1.InterfaceBindingPlugin) patch.PatchOperation {
	return patch.PatchOperation{
		Op:    patch.PatchAddOp,
		Path:  "/spec/configuration/network/binding/" + name,
		Value: binding,
	}
}
