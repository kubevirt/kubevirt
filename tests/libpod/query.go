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

package libpod

import (
	"context"
	"fmt"
	"sort"
	"strings"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/tests/framework/k8s"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetRunningPodByLabel(label, labelType, namespace, node string) (*k8sv1.Pod, error) {
	labelSelector := fmt.Sprintf("%s=%s", labelType, label)
	var fieldSelector string
	if node != "" {
		fieldSelector = fmt.Sprintf("status.phase==%s,spec.nodeName==%s", k8sv1.PodRunning, node)
	} else {
		fieldSelector = fmt.Sprintf("status.phase==%s", k8sv1.PodRunning)
	}
	pod, err := lookupPodBySelector(namespace, labelSelector, fieldSelector)
	if err != nil {
		return nil, fmt.Errorf("failed to find pod with the label %s", label)
	}
	return pod, nil
}

func GetPodByVirtualMachineInstance(vmi *v1.VirtualMachineInstance, namespace string) (*k8sv1.Pod, error) {
	pod, err := lookupPodBySelector(namespace, vmiLabelSelector(vmi), vmiFieldSelector(vmi))
	if err != nil {
		return nil, fmt.Errorf("failed to find pod for VMI %s (%s)", vmi.Name, string(vmi.GetUID()))
	}
	return pod, nil
}

func GetTargetPodForMigration(migration *v1.VirtualMachineInstanceMigration) (*k8sv1.Pod, error) {
	migrationLabelSelector := fmt.Sprintf("%s=%s", v1.MigrationJobLabel, migration.UID)
	pod, err := lookupPodBySelector(migration.Namespace, migrationLabelSelector, "")
	if err != nil {
		return nil, fmt.Errorf("failed to find pod for migration %s (%s)", migration.Name, migration.UID)
	}
	return pod, nil
}

func lookupPodBySelector(namespace, labelSelector, fieldSelector string) (*k8sv1.Pod, error) {
	pods, err := k8s.Client().CoreV1().Pods(namespace).List(context.Background(),
		metav1.ListOptions{LabelSelector: labelSelector, FieldSelector: fieldSelector},
	)
	if err != nil {
		return nil, err
	}

	if len(pods.Items) == 0 {
		return nil, fmt.Errorf("failed to lookup pod")
	}

	// There can be more than one running pod in case of migration
	// therefore 	return the latest running pod
	sort.Sort(sort.Reverse(PodsByCreationTimestamp(pods.Items)))
	return &pods.Items[0], nil
}

func vmiLabelSelector(vmi *v1.VirtualMachineInstance) string {
	return fmt.Sprintf("%s=%s", v1.CreatedByLabel, string(vmi.GetUID()))
}

func vmiFieldSelector(vmi *v1.VirtualMachineInstance) string {
	var fieldSelectors []string
	if vmi.Status.Phase == v1.Running {
		fieldSelectors = append(fieldSelectors, fmt.Sprintf("status.phase==%s", k8sv1.PodRunning))
	}
	if node := vmi.Status.NodeName; node != "" {
		fieldSelectors = append(fieldSelectors, fmt.Sprintf("spec.nodeName==%s", node))
	}
	return strings.Join(fieldSelectors, ",")
}
