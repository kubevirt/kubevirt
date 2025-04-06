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

package config

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
)

func WithNetBindingPlugin(name string, netBindingPlugin v1.InterfaceBindingPlugin) error {
	return RegisterKubevirtConfigChange(func(c v1.KubeVirtConfiguration) (*patch.PatchSet, error) {
		return registerBindingPugins(c, name, netBindingPlugin)
	})
}

func registerBindingPugins(config v1.KubeVirtConfiguration, name string, binding v1.InterfaceBindingPlugin) (
	*patch.PatchSet, error,
) {
	patchSet := patch.New()

	if config.NetworkConfiguration == nil {
		patchSet.AddOption(patch.WithAdd("/spec/configuration/network", v1.NetworkConfiguration{}))
		patchSet.AddOption(patch.WithAdd("/spec/configuration/network/binding", map[string]v1.InterfaceBindingPlugin{}))
	} else if config.NetworkConfiguration.Binding == nil {
		patchSet.AddOption(patch.WithAdd("/spec/configuration/network/binding", map[string]v1.InterfaceBindingPlugin{}))
	}

	patchSet.AddOption(patch.WithAdd(fmt.Sprintf("/spec/configuration/network/binding/%s", name), binding))

	log.Log.Infof("registering binding plugin: %s, %+v", name, binding)

	return patchSet, nil
}
