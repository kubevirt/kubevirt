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
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2022 Red Hat, Inc.
 *
 */

package watch

import (
	"context"
	"fmt"

	k8sv1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"kubevirt.io/client-go/kubecli"
)

const PSALabel = "pod-security.kubernetes.io/enforce"
const OpenshiftPSAsync = "security.openshift.io/scc.podSecurityLabelSync"

func escalateNamespace(namespaceStore cache.Store, client kubecli.KubevirtClient, namespace string, onOpenshift bool) error {
	obj, exists, err := namespaceStore.GetByKey(namespace)
	if err != nil {
		return fmt.Errorf("Failed to get namespace, %w", err)
	}
	if !exists {
		return fmt.Errorf("Namespace %s not observed, %w", namespace, err)
	}
	namespaceObj := obj.(*k8sv1.Namespace)
	enforceLevel, labelExist := namespaceObj.Labels[PSALabel]
	if !labelExist || enforceLevel != "privileged" {
		labels := ""
		if !onOpenshift {
			labels = fmt.Sprintf(`{"%s": "privileged"}`, PSALabel)
		} else {
			labels = fmt.Sprintf(`{"%s": "privileged", "%s": "false"}`, PSALabel, OpenshiftPSAsync)
		}
		data := []byte(fmt.Sprintf(`{"metadata": { "labels": %s}}`, labels))
		_, err := client.CoreV1().Namespaces().Patch(context.TODO(), namespace, types.StrategicMergePatchType, data, v1.PatchOptions{})
		if err != nil {
			return &syncErrorImpl{err, fmt.Sprintf("Failed to apply enforce label on namespace %s", namespace)}
		}
	}
	return nil
}
