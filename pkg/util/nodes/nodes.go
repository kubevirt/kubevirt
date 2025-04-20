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
 */

package nodes

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"

	"kubevirt.io/client-go/kubecli"
)

func PatchNode(client kubecli.KubevirtClient, original, modified *corev1.Node) error {
	originalBytes, err := json.Marshal(original)
	if err != nil {
		return fmt.Errorf("could not serialize original object: %v", err)
	}
	modifiedBytes, err := json.Marshal(modified)
	if err != nil {
		return fmt.Errorf("could not serialize modified object: %v", err)
	}
	patch, err := strategicpatch.CreateTwoWayMergePatch(originalBytes, modifiedBytes, corev1.Node{})
	if err != nil {
		return fmt.Errorf("could not create merge patch: %v", err)
	}
	if _, err := client.CoreV1().Nodes().Patch(context.Background(), original.Name, types.StrategicMergePatchType, patch, metav1.PatchOptions{}); err != nil {
		return fmt.Errorf("could not patch the node: %v", err)
	}
	return nil
}
