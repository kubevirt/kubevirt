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

	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	waitForPodRunningTimeoutSeconds   = 180
	waitForPodSucceededTimeoutSeconds = 120
)

func RunPodInNamespace(pod *k8sv1.Pod, namespace string) *k8sv1.Pod {
	virtClient := kubevirt.Client()

	var err error
	pod, err = virtClient.CoreV1().Pods(namespace).Create(context.Background(), pod, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	Eventually(matcher.ThisPod(pod), waitForPodRunningTimeoutSeconds).Should(matcher.BeInPhase(k8sv1.PodRunning))

	pod, err = matcher.ThisPod(pod)()
	Expect(err).ToNot(HaveOccurred())
	return pod
}

func RunPod(pod *k8sv1.Pod) *k8sv1.Pod {
	return RunPodInNamespace(pod, testsuite.GetTestNamespace(pod))
}

func RunPodAndExpectCompletion(pod *k8sv1.Pod) *k8sv1.Pod {
	virtClient := kubevirt.Client()

	var err error
	pod, err = virtClient.CoreV1().Pods(testsuite.GetTestNamespace(pod)).Create(context.Background(), pod, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	Eventually(matcher.ThisPod(pod), waitForPodSucceededTimeoutSeconds).Should(matcher.BeInPhase(k8sv1.PodSucceeded))

	pod, err = matcher.ThisPod(pod)()
	Expect(err).ToNot(HaveOccurred())
	return pod
}
