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
package placement

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestInjectKubeVirtControlPlanePlacement_FullAffinityChain(t *testing.T) {
	podSpec := &corev1.PodSpec{
		Affinity: &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{Key: "existing-key", Operator: corev1.NodeSelectorOpExists},
							},
						},
					},
				},
			},
		},
	}

	InjectKubeVirtControlPlanePlacement(podSpec)

	terms := podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
	if len(terms) != 2 {
		t.Fatalf("expected 2 NodeSelectorTerms, got %d", len(terms))
	}
	if terms[1].MatchExpressions[0].Key != KubeVirtControlPlaneLabel {
		t.Errorf("expected key %q, got %q", KubeVirtControlPlaneLabel, terms[1].MatchExpressions[0].Key)
	}
	if len(podSpec.Tolerations) != 1 || podSpec.Tolerations[0].Key != KubeVirtControlPlaneLabel {
		t.Errorf("expected toleration with key %q", KubeVirtControlPlaneLabel)
	}
}

func TestInjectKubeVirtControlPlanePlacement_NilAffinity(t *testing.T) {
	podSpec := &corev1.PodSpec{}

	InjectKubeVirtControlPlanePlacement(podSpec)

	if podSpec.Affinity == nil || podSpec.Affinity.NodeAffinity == nil ||
		podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
		t.Fatal("expected affinity chain to be initialized")
	}
	terms := podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
	if len(terms) != 1 {
		t.Fatalf("expected 1 NodeSelectorTerm, got %d", len(terms))
	}
	if terms[0].MatchExpressions[0].Key != KubeVirtControlPlaneLabel {
		t.Errorf("expected key %q, got %q", KubeVirtControlPlaneLabel, terms[0].MatchExpressions[0].Key)
	}
	if len(podSpec.Tolerations) != 1 {
		t.Fatalf("expected 1 toleration, got %d", len(podSpec.Tolerations))
	}
}

func TestInjectKubeVirtControlPlanePlacement_NilNodeAffinity(t *testing.T) {
	podSpec := &corev1.PodSpec{
		Affinity: &corev1.Affinity{
			PodAntiAffinity: &corev1.PodAntiAffinity{},
		},
	}

	InjectKubeVirtControlPlanePlacement(podSpec)

	if podSpec.Affinity.NodeAffinity == nil {
		t.Fatal("expected NodeAffinity to be initialized")
	}
	terms := podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
	if len(terms) != 1 || terms[0].MatchExpressions[0].Key != KubeVirtControlPlaneLabel {
		t.Error("expected NodeSelectorTerm with kubevirt control-plane label")
	}
	if podSpec.Affinity.PodAntiAffinity == nil {
		t.Error("expected existing PodAntiAffinity to be preserved")
	}
	if len(podSpec.Tolerations) != 1 {
		t.Fatalf("expected 1 toleration, got %d", len(podSpec.Tolerations))
	}
}

func TestInjectKubeVirtControlPlanePlacement_NilRequiredDuring(t *testing.T) {
	podSpec := &corev1.PodSpec{
		Affinity: &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{
					{Weight: 1},
				},
			},
		},
	}

	InjectKubeVirtControlPlanePlacement(podSpec)

	if podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
		t.Fatal("expected RequiredDuringSchedulingIgnoredDuringExecution to be initialized")
	}
	terms := podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
	if len(terms) != 1 || terms[0].MatchExpressions[0].Key != KubeVirtControlPlaneLabel {
		t.Error("expected NodeSelectorTerm with kubevirt control-plane label")
	}
	if len(podSpec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution) != 1 {
		t.Error("expected existing PreferredDuring terms to be preserved")
	}
}

func TestInjectKubeVirtControlPlanePlacement_PreservesExistingTolerations(t *testing.T) {
	podSpec := &corev1.PodSpec{
		Tolerations: []corev1.Toleration{
			{Key: "existing-taint", Operator: corev1.TolerationOpExists},
		},
	}

	InjectKubeVirtControlPlanePlacement(podSpec)

	if len(podSpec.Tolerations) != 2 {
		t.Fatalf("expected 2 tolerations, got %d", len(podSpec.Tolerations))
	}
	if podSpec.Tolerations[0].Key != "existing-taint" {
		t.Error("expected existing toleration to be preserved")
	}
	if podSpec.Tolerations[1].Key != KubeVirtControlPlaneLabel {
		t.Error("expected kubevirt control-plane toleration to be appended")
	}
}
