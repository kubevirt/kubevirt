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
	"fmt"
	"slices"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/log"

	pluginv1alpha1 "kubevirt.io/api/plugin/v1alpha1"
)

// resolvedHook is a launcher hook that has been selected for a given hook point, with its
// failure strategy already resolved.
type resolvedHook struct {
	pluginName      string
	index           int
	hook            pluginv1alpha1.LauncherHook
	failureStrategy pluginv1alpha1.FailureStrategy
	timeout         *metav1.Duration
}

// selectHooks resolves the ordered list of launcher hooks that apply to hookPoint, evaluating
// plugin- and hook-level conditions via evalCondition. Plugins are sorted alphabetically by
// name for deterministic ordering across plugins; hooks within a plugin preserve their
// declaration order.
func selectHooks(plugins []pluginv1alpha1.Plugin, hookPoint pluginv1alpha1.LauncherHookPoint, evalCondition func(expr string) (bool, error)) ([]resolvedHook, error) {
	sortedPlugins := slices.Clone(plugins)
	slices.SortStableFunc(sortedPlugins, func(a, b pluginv1alpha1.Plugin) int {
		return cmp.Compare(a.Name, b.Name)
	})

	pluginNames := make([]string, len(sortedPlugins))
	for i, plugin := range sortedPlugins {
		pluginNames[i] = plugin.Name
	}
	log.Log.Infof("Evaluating %s hooks from plugins: [%s]", hookPoint, strings.Join(pluginNames, ", "))

	var resolvedHooks []resolvedHook
	for _, plugin := range sortedPlugins {
		if plugin.Spec.Condition != "" {
			matched, err := evalCondition(plugin.Spec.Condition)
			if err != nil {
				return nil, fmt.Errorf("plugin %s condition evaluation failed: %w", plugin.Name, err)
			}
			if !matched {
				log.Log.Infof("Skipping plugin %s: condition not met", plugin.Name)
				continue
			}
		}

		for hookIdx, hook := range plugin.Spec.LauncherHooks {
			switch {
			case hook.CEL != nil && hook.CEL.HookPoint == hookPoint:
			case hook.Sidecar != nil && slices.Contains(hook.Sidecar.PermittedHooks, hookPoint):
			default:
				continue
			}

			if hook.Condition != "" {
				matched, err := evalCondition(hook.Condition)
				if err != nil {
					return nil, fmt.Errorf("plugin %s hook %d condition evaluation failed: %w", plugin.Name, hookIdx, err)
				}
				if !matched {
					continue
				}
			}

			failureStrategy := cmp.Or(hook.FailureStrategy, plugin.Spec.FailureStrategy, pluginv1alpha1.FailureStrategyFail)

			resolvedHooks = append(resolvedHooks, resolvedHook{
				pluginName:      plugin.Name,
				index:           hookIdx,
				hook:            hook,
				failureStrategy: failureStrategy,
				timeout:         hook.Timeout,
			})
		}
	}

	return resolvedHooks, nil
}
