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

type hookApplier interface {
	Apply(vmi *v1.VirtualMachineInstance, domain *libvirtxml.Domain, invocationContext string) (*libvirtxml.Domain, error)
}

type celHookApplier struct {
	evaluator  *celutil.Evaluator
	expression string
}

func (c *celHookApplier) Apply(vmi *v1.VirtualMachineInstance, domain *libvirtxml.Domain, _ string) (*libvirtxml.Domain, error) {
	return c.evaluator.EvaluateMutation(c.expression, vmi, domain)
}

type sidecarHookApplier struct {
	socketPath string
	pluginName string
	timeout    time.Duration
	deadline   time.Time
}

func (s *sidecarHookApplier) Apply(vmi *v1.VirtualMachineInstance, domain *libvirtxml.Domain, invocationContext string) (*libvirtxml.Domain, error) {
	if err := waitForSidecarSocket(s.socketPath, s.deadline); err != nil {
		return nil, err
	}

	domainXML, err := domain.Marshal()
	if err != nil {
		return nil, fmt.Errorf("marshal domain: %w", err)
	}
	vmiJSON, err := json.Marshal(vmi)
	if err != nil {
		return nil, fmt.Errorf("marshal VMI: %w", err)
	}

	resultXML, err := callSidecarHook(s.socketPath, s.pluginName, []byte(domainXML), vmiJSON, invocationContext, s.timeout)
	if err != nil {
		return nil, err
	}

	mutated := &libvirtxml.Domain{}
	if err := mutated.Unmarshal(string(resultXML)); err != nil {
		return nil, fmt.Errorf("unmarshal sidecar response: %w", err)
	}
	return mutated, nil
}

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
	log.Log.Infof("Evaluating domain hooks from plugins: [%s]", strings.Join(pluginNames, ", "))

	for _, plugin := range sortedPlugins {
		if plugin.Spec.Condition != "" {
			matched, err := evaluator.EvaluateCondition(plugin.Spec.Condition, vmi, domain)
			if err != nil {
				return nil, "", fmt.Errorf("plugin %s condition evaluation failed: %w", plugin.Name, err)
			}
			if !matched {
				log.Log.Infof("Skipping plugin %s: condition not met", plugin.Name)
				continue
			}
		}

		for hookIdx, hook := range plugin.Spec.DomainHooks {
			if hook.Condition != "" {
				matched, err := evaluator.EvaluateCondition(hook.Condition, vmi, domain)
				if err != nil {
					return nil, "", fmt.Errorf("plugin %s hook %d condition evaluation failed: %w", plugin.Name, hookIdx, err)
				}
				if !matched {
					continue
				}
			}

			failureStrategy := cmp.Or(hook.FailureStrategy, plugin.Spec.FailureStrategy, pluginv1alpha1.FailureStrategyFail)

			var applier hookApplier
			switch {
			case hook.CEL != nil:
				applier = &celHookApplier{evaluator: evaluator, expression: hook.CEL.Expression}
			case hook.Sidecar != nil:
				deadline := time.Now().Add(sidecarReadinessTimeout)
				timeout := defaultSidecarCallTimeout
				if hook.Timeout != nil {
					timeout = hook.Timeout.Duration
				}
				applier = &sidecarHookApplier{socketPath: hook.Sidecar.SocketPath, pluginName: plugin.Name, timeout: timeout, deadline: deadline}
			default:
				continue
			}

			mutated, err := applier.Apply(vmi, domain, string(invocationContext))
			if err != nil {
				if failureStrategy == pluginv1alpha1.FailureStrategyIgnore {
					log.Log.Warningf("Plugin %s hook %d failed (ignored): %v", plugin.Name, hookIdx, err)
					continue
				}
				return nil, "", fmt.Errorf("plugin %s hook %d failed: %w", plugin.Name, hookIdx, err)
			}
			domain = mutated
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
