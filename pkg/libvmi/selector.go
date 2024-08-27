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
 * Copyright the KubeVirt Authors.
 *
 */

package libvmi

import (
	k8sv1 "k8s.io/api/core/v1"
	v1 "kubevirt.io/api/core/v1"
)

// WithNodeSelectorFor ensures that the VMI gets scheduled on the specified node
func WithNodeSelectorFor(nodeName string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.NodeSelector == nil {
			vmi.Spec.NodeSelector = map[string]string{}
		}
		vmi.Spec.NodeSelector[k8sv1.LabelHostname] = nodeName
	}
}

func WithNodeAffinityFor(nodeName string) Option {
	return WithNodeAffinityForLabel(k8sv1.LabelHostname, nodeName)
}

func WithNodeAffinityForLabel(nodeLabelKey, nodeLabelValue string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		nodeSelectorTerm := k8sv1.NodeSelectorTerm{
			MatchExpressions: []k8sv1.NodeSelectorRequirement{
				{Key: nodeLabelKey, Operator: k8sv1.NodeSelectorOpIn, Values: []string{nodeLabelValue}},
			},
		}

		if vmi.Spec.Affinity == nil {
			vmi.Spec.Affinity = &k8sv1.Affinity{}
		}

		if vmi.Spec.Affinity.NodeAffinity == nil {
			vmi.Spec.Affinity.NodeAffinity = &k8sv1.NodeAffinity{}
		}

		if vmi.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
			vmi.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &k8sv1.NodeSelector{}
		}

		if vmi.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms == nil {
			vmi.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms = []k8sv1.NodeSelectorTerm{}
		}

		vmi.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms = append(
			vmi.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms,
			nodeSelectorTerm,
		)
	}
}

func WithPreferredPodAffinity(term k8sv1.WeightedPodAffinityTerm) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Affinity == nil {
			vmi.Spec.Affinity = &k8sv1.Affinity{}
		}

		if vmi.Spec.Affinity.PodAffinity == nil {
			vmi.Spec.Affinity.PodAffinity = &k8sv1.PodAffinity{}
		}

		vmi.Spec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(
			vmi.Spec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution, term,
		)
	}
}

func WithPreferredNodeAffinity(term k8sv1.PreferredSchedulingTerm) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Affinity == nil {
			vmi.Spec.Affinity = &k8sv1.Affinity{}
		}

		if vmi.Spec.Affinity.NodeAffinity == nil {
			vmi.Spec.Affinity.NodeAffinity = &k8sv1.NodeAffinity{}
		}

		vmi.Spec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(
			vmi.Spec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution,
			term,
		)
	}
}
