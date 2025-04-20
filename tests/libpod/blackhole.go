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

package libpod

import (
	"context"

	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
)

func AddKubernetesAPIBlackhole(pods *v1.PodList, containerName string) {
	kubernetesAPIServiceBlackhole(pods, containerName, true)
}

func DeleteKubernetesAPIBlackhole(pods *v1.PodList, containerName string) {
	kubernetesAPIServiceBlackhole(pods, containerName, false)
}

func kubernetesAPIServiceBlackhole(pods *v1.PodList, containerName string, present bool) {
	serviceIP := getKubernetesAPIServiceIP()

	var addOrDel string
	if present {
		addOrDel = "add"
	} else {
		addOrDel = "del"
	}

	for idx := range pods.Items {
		_, err := exec.ExecuteCommandOnPod(&pods.Items[idx], containerName, []string{"ip", "route", addOrDel, "blackhole", serviceIP})
		Expect(err).NotTo(HaveOccurred())
	}
}

func getKubernetesAPIServiceIP() string {
	const serviceName = "kubernetes"
	const serviceNamespace = "default"

	kubernetesService, err := kubevirt.Client().CoreV1().Services(serviceNamespace).Get(context.Background(), serviceName, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())

	return kubernetesService.Spec.ClusterIP
}
