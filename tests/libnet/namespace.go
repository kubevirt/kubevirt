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

package libnet

import (
	"context"
	"encoding/json"
	"reflect"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"

	"kubevirt.io/client-go/kubecli"
)

func AddLabelToNamespace(client kubecli.KubevirtClient, namespace, key, value string) error {
	return PatchNamespace(client, namespace, func(ns *v1.Namespace) {
		if ns.Labels == nil {
			ns.Labels = map[string]string{}
		}
		ns.Labels[key] = value
	})
}

func RemoveLabelFromNamespace(client kubecli.KubevirtClient, namespace, key string) error {
	return PatchNamespace(client, namespace, func(ns *v1.Namespace) {
		if ns.Labels == nil {
			return
		}
		delete(ns.Labels, key)
	})
}

func PatchNamespace(client kubecli.KubevirtClient, namespace string, patchFunc func(*v1.Namespace)) error {
	ns, err := client.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
	if err != nil {
		return err
	}

	newNS := ns.DeepCopy()
	patchFunc(newNS)
	if reflect.DeepEqual(ns, newNS) {
		return nil
	}

	oldJSON, err := json.Marshal(ns)
	if err != nil {
		return err
	}

	newJSON, err := json.Marshal(newNS)
	if err != nil {
		return err
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(oldJSON, newJSON, ns)
	if err != nil {
		return err
	}

	_, err = client.CoreV1().Namespaces().Patch(context.Background(), ns.Name, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		return err
	}
	return nil
}
