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
 * Copyright 2020 Red Hat, Inc.
 *
 */
package components

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//NewKubevirtConfigMap returns base config map
func NewKubevirtConfigMap(namespace string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
	}

}

//NewNodeLabellerConfigMap returns configmap with configuration of obsolete cpus and minCPU for node labeller
func NewNodeLabellerConfigMap(namespace string) *corev1.ConfigMap {
	cm := NewKubevirtConfigMap(namespace)
	cm.Name = "kubevirt-cpu-plugin-configmap"
	cm.Data = map[string]string{
		"cpu-plugin-configmap.yaml": `obsoleteCPUs:
- "486"
- "pentium"
- "pentium2"
- "pentium3"
- "pentiumpro"
- "coreduo"
- "n270"
- "core2duo"
- "Conroe"
- "athlon"
- "phenom"
minCPU: "Penryn"`,
	}
	return cm
}
