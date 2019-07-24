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
 * Copyright 2019 Red Hat, Inc.
 *
 */

package tests_test

import (
	"fmt"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
)

const (
	DefaultStabilizationTimeoutInSeconds = 300
	DefaultPollIntervalInSeconds         = 3
	labelKey                             = "control-plane-test"
	labelValue                           = "selected"
)

var _ = Describe("KubeVirt control plane resilience", func() {

	var nodeNames []string
	var selectedNodeName string

	controlPlaneDeploymentNames := []string{"virt-api", "virt-controller"}

	tests.FlagParse()

	getPodsInStatus := func(podList *v1.PodList, expectedPhase v1.PodPhase, expectedConditionStatus v1.ConditionStatus, podNames []string, nodeNames ...string) (pods []*v1.Pod) {
		pods = make([]*v1.Pod, 0)
		for _, pod := range podList.Items {
			if pod.Status.Phase != expectedPhase {
				continue
			}
			podReady := tests.PodReady(&pod)
			if podReady != expectedConditionStatus {
				continue
			}
			for _, podName := range podNames {
				if strings.HasPrefix(pod.Name, podName) {
					if len(nodeNames) > 0 {
						for _, nodeName := range nodeNames {
							if pod.Spec.NodeName == nodeName {
								deepCopy := pod.DeepCopy()
								pods = append(pods, deepCopy)
							}
						}
					} else {
						deepCopy := pod.DeepCopy()
						pods = append(pods, deepCopy)
					}
				}
			}
		}
		return
	}

	countPodsInStatus := func(podList *v1.PodList, expectedPhase v1.PodPhase, expectedConditionStatus v1.ConditionStatus, podNames []string, nodeNames ...string) int {
		pods := getPodsInStatus(podList, expectedPhase, expectedConditionStatus, podNames, nodeNames...)
		return len(pods)
	}

	getPodList := func() (podList *v1.PodList, err error) {
		virtCli, err := kubecli.GetKubevirtClient()
		if err != nil {
			return
		}
		podList, err = virtCli.CoreV1().Pods(tests.KubeVirtInstallNamespace).List(metav1.ListOptions{})
		return
	}

	countReadyPodsOnNodes := func(podNames []string, nodeNames ...string) int {
		podList, err := getPodList()
		if err != nil {
			return -1
		}
		return countPodsInStatus(podList, v1.PodRunning, v1.ConditionTrue, podNames, nodeNames...)
	}

	countPendingPods := func(podNames []string) int {
		podList, err := getPodList()
		if err != nil {
			return -1
		}
		return countPodsInStatus(podList, v1.PodPending, v1.ConditionFalse, podNames)
	}

	getSelectedNode := func() (selectedNode *v1.Node) {
		virtCli, err := kubecli.GetKubevirtClient()
		tests.PanicOnError(err)
		selectedNode, err = kubecli.KubevirtClient.CoreV1(virtCli).Nodes().Get(selectedNodeName, metav1.GetOptions{})
		tests.PanicOnError(err)
		return
	}

	waitForDeploymentsToStabilize := func() (bool, error) {
		virtCli, err := kubecli.GetKubevirtClient()
		tests.PanicOnError(err)
		deploymentsClient := kubecli.KubevirtClient.AppsV1(virtCli).Deployments(tests.KubeVirtInstallNamespace)
		for numberOfSuccessfulChecks := 0; numberOfSuccessfulChecks < 3; numberOfSuccessfulChecks++ {
			for _, deploymentName := range controlPlaneDeploymentNames {
				deployment, err := deploymentsClient.Get(deploymentName, metav1.GetOptions{})
				if err != nil {
					return false, err
				}

				if !(deployment.Status.UpdatedReplicas == *(deployment.Spec.Replicas) &&
					deployment.Status.Replicas == *(deployment.Spec.Replicas) &&
					deployment.Status.AvailableReplicas == *(deployment.Spec.Replicas)) {
					return false, err
				}
			}
			time.Sleep(time.Second)
		}
		return true, nil
	}

	BeforeEach(func() {
		tests.SkipIfNoCmd("kubectl")
		tests.BeforeTestCleanup()

		virtCli, err := kubecli.GetKubevirtClient()
		tests.PanicOnError(err)

		nodes := tests.GetAllSchedulableNodes(virtCli).Items
		nodeNames = make([]string, len(nodes))
		for index, node := range nodes {
			nodeNames[index] = node.Name
		}

		Eventually(func() int { return countReadyPodsOnNodes(controlPlaneDeploymentNames, nodeNames...) },
			DefaultStabilizationTimeoutInSeconds, DefaultPollIntervalInSeconds,
		).Should(Equal(4))
		Eventually(func() int { return countPendingPods(controlPlaneDeploymentNames) },
			DefaultStabilizationTimeoutInSeconds, DefaultPollIntervalInSeconds,
		).Should(Equal(0))

		// select node for test
		selectedNodeName = nodes[0].Name
		selectedNode, err := kubecli.KubevirtClient.CoreV1(virtCli).Nodes().Get(selectedNodeName, metav1.GetOptions{})
		tests.PanicOnError(err)

		// Add label to node that was selected for test
		for {
			if selectedNode.Labels == nil {
				selectedNode.Labels = make(map[string]string)
			}
			selectedNode.Labels[labelKey] = labelValue
			_, err = kubecli.KubevirtClient.CoreV1(virtCli).Nodes().Update(selectedNode)
			if err == nil {
				break
			}
		}

		// Add nodeSelector to deployments so that they get scheduled to selectedNode
		deploymentsClient := kubecli.KubevirtClient.AppsV1(virtCli).Deployments(tests.KubeVirtInstallNamespace)
		tests.PanicOnError(err)
		for _, deploymentName := range controlPlaneDeploymentNames {
			for {
				deployment, err := deploymentsClient.Get(deploymentName, metav1.GetOptions{})
				tests.PanicOnError(err)

				labelMap := make(map[string]string)
				labelMap[labelKey] = labelValue
				if deployment.Spec.Template.Spec.NodeSelector == nil {
					deployment.Spec.Template.Spec.NodeSelector = make(map[string]string)
				}
				deployment.Spec.Template.Spec.NodeSelector[labelKey] = labelValue
				_, err = deploymentsClient.Update(deployment)
				if err == nil {
					break
				}
			}
		}

		Eventually(waitForDeploymentsToStabilize,
			DefaultStabilizationTimeoutInSeconds, DefaultPollIntervalInSeconds,
		).Should(BeTrue())
	})

	AfterEach(func() {
		virtCli, err := kubecli.GetKubevirtClient()
		tests.PanicOnError(err)

		// Remove nodeSelector from deployments
		deploymentsClient := kubecli.KubevirtClient.AppsV1(virtCli).Deployments(tests.KubeVirtInstallNamespace)
		for _, deploymentName := range controlPlaneDeploymentNames {
			for {
				deployment, err := deploymentsClient.Get(deploymentName, metav1.GetOptions{})
				if err != nil {
					continue
				}
				delete(deployment.Spec.Template.Spec.NodeSelector, labelKey)
				_, err = deploymentsClient.Update(deployment)
				if err == nil {
					break
				}
			}
		}

		// Clean up selectedNode: Remove label and make schedulable again
		for {
			selectedNode := getSelectedNode()
			selectedNode.Spec.Unschedulable = false
			delete(selectedNode.Labels, labelKey)
			_, err = kubecli.KubevirtClient.CoreV1(virtCli).Nodes().Update(selectedNode)
			if err == nil {
				break
			}
		}

		Eventually(waitForDeploymentsToStabilize,
			DefaultStabilizationTimeoutInSeconds, DefaultPollIntervalInSeconds,
		).Should(BeTrue())
	})

	When("evicting pods of control plane, last eviction should fail", func() {

		test := func(podName string) {
			virtCli, err := kubecli.GetKubevirtClient()
			tests.PanicOnError(err)

			By(fmt.Sprintf("set node %s unschedulable\n", selectedNodeName))
			selectedNode := getSelectedNode()
			selectedNode.Spec.Unschedulable = true
			_, err = kubecli.KubevirtClient.CoreV1(virtCli).Nodes().Update(selectedNode)

			By(fmt.Sprintf("Try to evict all pods %s from node %s\n", podName, selectedNodeName))
			podList, err := getPodList()
			Expect(err).ToNot(HaveOccurred())
			runningPods := getPodsInStatus(podList, v1.PodRunning, v1.ConditionTrue, []string{podName})
			for index, pod := range runningPods {
				err = virtCli.CoreV1().Pods(tests.KubeVirtInstallNamespace).Evict(&v1beta1.Eviction{ObjectMeta: metav1.ObjectMeta{Name: pod.Name}})
				if index < len(runningPods)-1 {
					Expect(err).ToNot(HaveOccurred())
				}
			}
			Expect(err).To(HaveOccurred(), "no error occurred on evict of last pod")
		}

		It("for virt-controller pods", func() { test("virt-controller") })
		It("for virt-api pods", func() { test("virt-api") })

	})

})
