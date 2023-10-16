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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package libvmi

import (
	"context"
	"fmt"

	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

func GetVirtLauncherLogs(virtCli kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) string {
	namespace := vmi.GetObjectMeta().GetNamespace()
	uid := vmi.GetObjectMeta().GetUID()

	labelSelector := fmt.Sprintf(v1.CreatedByLabel + "=" + string(uid))

	pods, err := virtCli.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
	Expect(err).ToNot(HaveOccurred(), "Should list pods")

	podName := ""
	for _, pod := range pods.Items {
		if pod.ObjectMeta.DeletionTimestamp == nil {
			podName = pod.ObjectMeta.Name
			break
		}
	}
	Expect(podName).ToNot(BeEmpty(), "Should find pod not scheduled for deletion")

	logsRaw, err := virtCli.CoreV1().
		Pods(namespace).
		GetLogs(podName, &k8sv1.PodLogOptions{
			Container: "compute",
		}).
		DoRaw(context.Background())
	Expect(err).ToNot(HaveOccurred(), "Should get virt-launcher pod logs")

	return string(logsRaw)
}
