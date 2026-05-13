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
	"fmt"
	"slices"
	"sort"
	"time"

	"k8s.io/client-go/tools/cache"

	pluginv1alpha1 "kubevirt.io/api/plugin/v1alpha1"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	hooksclient "kubevirt.io/kubevirt/pkg/hooks/plugins/v1alpha1"
	plugincel "kubevirt.io/kubevirt/pkg/plugins/cel"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_manager.go

type NodeHookExecutor interface {
	CallNodeHooks(hookPoint pluginv1alpha1.NodeHookPoint, vmi *virtv1.VirtualMachineInstance, nodeName string) error
}

type nodeHookManager struct {
	pluginStore   cache.Store
	clusterConfig *virtconfig.ClusterConfig
	celEvaluator  *plugincel.Evaluator
}

func NewNodeHookManager(pluginStore cache.Store, clusterConfig *virtconfig.ClusterConfig) (NodeHookExecutor, error) {
	celEvaluator, err := plugincel.NewEvaluator()
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL evaluator: %w", err)
	}
	return &nodeHookManager{
		pluginStore:   pluginStore,
		clusterConfig: clusterConfig,
		celEvaluator:  celEvaluator,
	}, nil
}

type matchedHook struct {
	pluginName string
	nodeHook   pluginv1alpha1.NodeHook
}

func (m *nodeHookManager) CallNodeHooks(hookPoint pluginv1alpha1.NodeHookPoint, vmi *virtv1.VirtualMachineInstance, nodeName string) error {
	if !m.clusterConfig.PluginsEnabled() {
		return nil
	}

	items := m.pluginStore.List()
	if len(items) == 0 {
		return nil
	}

	var matches []matchedHook
	for _, obj := range items {
		plugin, ok := obj.(*pluginv1alpha1.Plugin)
		if !ok {
			log.Log.Warningf("Unexpected object type in plugin store: %T", obj)
			continue
		}

		for _, nh := range plugin.Spec.NodeHooks {
			if !slices.Contains(nh.PermittedHooks, hookPoint) {
				log.Log.Object(vmi).V(4).Infof("Hook point %s not permitted for plugin %s, skipping", hookPoint, plugin.Name)
				continue
			}

			match, err := m.celEvaluator.EvaluateCondition(nh.Condition, map[string]any{"vmi": vmi})
			if err != nil {
				return fmt.Errorf("CEL evaluation failed for plugin %s: %w", plugin.Name, err)
			}
			if !match {
				log.Log.Object(vmi).V(3).Infof("CEL condition for plugin %s did not match, skipping hook", plugin.Name)
				continue
			}

			matches = append(matches, matchedHook{
				pluginName: plugin.Name,
				nodeHook:   nh,
			})
		}
	}

	sort.SliceStable(matches, func(i, j int) bool {
		return matches[i].pluginName < matches[j].pluginName
	})

	for _, match := range matches {
		timeout := hooksclient.DefaultTimeout
		if match.nodeHook.Timeout != nil {
			timeout = match.nodeHook.Timeout.Duration
		}

		if err := executeHook(match.nodeHook.Socket, string(hookPoint), vmi, match.pluginName, nodeName, timeout); err != nil {
			switch match.nodeHook.FailureStrategy {
			case pluginv1alpha1.FailureStrategyIgnore:
				log.Log.Object(vmi).Warningf("Node hook %s from plugin %s failed (ignored): %v", hookPoint, match.pluginName, err)
				continue
			default: // FailureStrategyFail
				return fmt.Errorf("node hook %s from plugin %s failed: %w", hookPoint, match.pluginName, err)
			}
		}
	}

	return nil
}

func executeHook(socketPath, hookPoint string, vmi *virtv1.VirtualMachineInstance, pluginName string, nodeName string, timeout time.Duration) error {
	conn, err := hooksclient.DialSocket(socketPath)
	if err != nil {
		return fmt.Errorf("failed to dial plugin %s at %s: %w", pluginName, socketPath, err)
	}
	defer conn.Close()

	client := hooksclient.NewNodeHookServiceClient(conn)
	return hooksclient.ExecuteNodeHook(client, hookPoint, vmi, nodeName, timeout)
}
