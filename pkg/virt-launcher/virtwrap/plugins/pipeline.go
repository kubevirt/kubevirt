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

package plugins

import (
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	libvirtxml "libvirt.org/go/libvirtxml"

	"kubevirt.io/client-go/log"

	v1 "kubevirt.io/api/core/v1"
	pluginv1alpha1 "kubevirt.io/api/plugin/v1alpha1"

	virtwrapApi "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/translate"
	celutil "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/plugins/cel"
)

func ApplyDomainHooks(plugins []pluginv1alpha1.Plugin, vmi *v1.VirtualMachineInstance, spec *virtwrapApi.DomainSpec, invocationContext pluginv1alpha1.InvocationContext) (*virtwrapApi.DomainSpec, string, error) {
	if len(plugins) == 0 {
		return spec, "", nil
	}

	domain, err := translate.ToLibvirtDomain(spec)
	if err != nil {
		return nil, "", fmt.Errorf("converting DomainSpec to libvirtxml: %w", err)
	}

	evaluator := celutil.GetEvaluator()

	// Sort plugins alphabetically by name for deterministic ordering across plugins.
	// Hooks within each plugin preserve their declaration order.
	sortedPlugins := slices.Clone(plugins)
	slices.SortStableFunc(sortedPlugins, func(a, b pluginv1alpha1.Plugin) int {
		return cmp.Compare(a.Name, b.Name)
	})

	pluginNames := make([]string, len(sortedPlugins))
	for i, plugin := range sortedPlugins {
		pluginNames[i] = plugin.Name
	}
	log.Log.Infof("Applying domain hooks from plugins: [%s]", strings.Join(pluginNames, ", "))

	deadline := time.Now().Add(sidecarReadinessTimeout)
	for _, plugin := range sortedPlugins {
		for hookIdx, hook := range plugin.Spec.DomainHooks {
			failureStrategy := hook.FailureStrategy
			if failureStrategy == "" {
				failureStrategy = pluginv1alpha1.FailureStrategyFail
			}

			if hook.Condition != "" {
				matched, err := evaluator.EvaluateCondition(hook.Condition, vmi, domain)
				if err != nil {
					if failureStrategy == pluginv1alpha1.FailureStrategyIgnore {
						log.Log.Warningf("Plugin %s hook %d condition evaluation failed (ignored): %v", plugin.Name, hookIdx, err)
						continue
					}
					return nil, "", fmt.Errorf("plugin %s hook %d condition evaluation failed: %w", plugin.Name, hookIdx, err)
				}
				if !matched {
					continue
				}
			}

			if hook.CEL != nil {
				mutated, err := evaluator.EvaluateMutation(hook.CEL.Expression, vmi, domain)
				if err != nil {
					if failureStrategy == pluginv1alpha1.FailureStrategyIgnore {
						log.Log.Warningf("Plugin %s hook %d mutation failed (ignored): %v", plugin.Name, hookIdx, err)
						continue
					}
					return nil, "", fmt.Errorf("plugin %s hook %d mutation failed: %w", plugin.Name, hookIdx, err)
				}
				domain = mutated
			} else if hook.Sidecar != nil {
				if err := waitForSidecarSocket(hook.Sidecar.SocketPath, deadline); err != nil {
					if failureStrategy == pluginv1alpha1.FailureStrategyIgnore {
						log.Log.Warningf("Plugin %s hook %d: %v (ignored)", plugin.Name, hookIdx, err)
						continue
					}
					return nil, "", fmt.Errorf("plugin %s hook %d: %w", plugin.Name, hookIdx, err)
				}

				domainXML, err := domain.Marshal()
				if err != nil {
					return nil, "", fmt.Errorf("plugin %s hook %d: marshal domain: %w", plugin.Name, hookIdx, err)
				}
				vmiJSON, err := json.Marshal(vmi)
				if err != nil {
					return nil, "", fmt.Errorf("plugin %s hook %d: marshal VMI: %w", plugin.Name, hookIdx, err)
				}

				timeout := defaultSidecarCallTimeout
				if hook.Timeout != nil {
					timeout = hook.Timeout.Duration
				}

				resultXML, err := callSidecarHook(hook.Sidecar.SocketPath, plugin.Name, []byte(domainXML), vmiJSON, string(invocationContext), timeout)
				if err != nil {
					if failureStrategy == pluginv1alpha1.FailureStrategyIgnore {
						log.Log.Warningf("Plugin %s hook %d: sidecar call failed (ignored): %v", plugin.Name, hookIdx, err)
						continue
					}
					return nil, "", fmt.Errorf("plugin %s hook %d sidecar failed: %w", plugin.Name, hookIdx, err)
				}

				mutated := &libvirtxml.Domain{}
				if err := mutated.Unmarshal(string(resultXML)); err != nil {
					if failureStrategy == pluginv1alpha1.FailureStrategyIgnore {
						log.Log.Warningf("Plugin %s hook %d: unmarshal sidecar response failed (ignored): %v", plugin.Name, hookIdx, err)
						continue
					}
					return nil, "", fmt.Errorf("plugin %s hook %d: unmarshal sidecar response: %w", plugin.Name, hookIdx, err)
				}
				domain = mutated
			}
		}
	}

	xmlStr, err := domain.Marshal()
	if err != nil {
		return nil, "", fmt.Errorf("marshaling domain to XML: %w", err)
	}

	updatedSpec, err := translate.FromLibvirtDomain(domain)
	if err != nil {
		return nil, "", fmt.Errorf("converting domain back to DomainSpec: %w", err)
	}

	log.Log.Infof("Successfully applied domain hooks from %d plugin(s)", len(sortedPlugins))
	return updatedSpec, xmlStr, nil
}
