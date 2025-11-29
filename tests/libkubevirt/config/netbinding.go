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
 * Copyright The KubeVirt Authors.
 *
 */

package config

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
)

// WithNetBindingPluginIfNotPresent registers a network binding plugin in KubeVirt CR, only if a plugin of that name
// is not already registered
func WithNetBindingPluginIfNotPresent(name string, netBindingPlugin v1.InterfaceBindingPlugin) KvChangeOption {
	return func(kv *v1.KubeVirt) *patch.PatchSet {
		patchSet := patch.New()
		config := kv.Spec.Configuration

		if config.NetworkConfiguration == nil {
			patchSet.AddOption(patch.WithAdd("/spec/configuration/network", v1.NetworkConfiguration{}))
			patchSet.AddOption(patch.WithAdd("/spec/configuration/network/binding", map[string]v1.InterfaceBindingPlugin{}))
		} else if config.NetworkConfiguration.Binding == nil {
			patchSet.AddOption(patch.WithAdd("/spec/configuration/network/binding", map[string]v1.InterfaceBindingPlugin{}))
		} else if _, exists := config.NetworkConfiguration.Binding[name]; exists {
			return &patch.PatchSet{}
		}

		patchSet.AddOption(patch.WithAdd(fmt.Sprintf("/spec/configuration/network/binding/%s", name), netBindingPlugin))

		log.Log.Infof("registering binding plugin: %s, %+v", name, netBindingPlugin)

		return patchSet
	}
}
