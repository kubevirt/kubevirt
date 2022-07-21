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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package libpod

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"
)

func GetRunningPodByLabel(label string, labelType string, namespace string, node string) (*corev1.Pod, error) {
	virtCli, err := kubecli.GetKubevirtClient()
	if err != nil {
		return nil, err
	}

	labelSelector := fmt.Sprintf("%s=%s", labelType, label)
	var fieldSelector string
	if node != "" {
		fieldSelector = fmt.Sprintf("status.phase==%s,spec.nodeName==%s", corev1.PodRunning, node)
	} else {
		fieldSelector = fmt.Sprintf("status.phase==%s", corev1.PodRunning)
	}
	pods, err := virtCli.CoreV1().Pods(namespace).List(context.Background(),
		metav1.ListOptions{LabelSelector: labelSelector, FieldSelector: fieldSelector},
	)
	if err != nil {
		return nil, err
	}

	if len(pods.Items) == 0 {
		return nil, fmt.Errorf("failed to find pod with the label %s", label)
	}

	var readyPod *corev1.Pod
	for _, pod := range pods.Items {
		ready := true
		for _, status := range pod.Status.ContainerStatuses {
			if status.Name == "kubevirt-infra" {
				ready = status.Ready
				break
			}
		}
		if ready {
			readyPod = &pod
			break
		}
	}
	if readyPod == nil {
		return nil, fmt.Errorf("no ready pods with the label %s", label)
	}

	return readyPod, nil
}
