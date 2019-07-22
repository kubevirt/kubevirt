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

	v1 "k8s.io/api/core/v1"
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

	var err error
	var nodeNames []string
	var selectedNodeName string

	controlPlanePodNames := []string{"virt-api", "virt-controller"}

	tests.FlagParse()

	countPodsInStatus := func(podList *v1.PodList, expectedPhase v1.PodPhase, expectedConditionStatus v1.ConditionStatus, podNames []string, nodeNames ...string) int {
		countPods := 0
		for _, pod := range podList.Items {
			if pod.Status.Phase != expectedPhase {
				continue
			}
			podReady := tests.PodReady(&pod)
			if podReady != expectedConditionStatus {
				continue
			}
			for _, podName := range podNames {
				if strings.Contains(pod.Name, podName) {
					if len(nodeNames) > 0 {
						for _, nodeName := range nodeNames {
							if pod.Spec.NodeName == nodeName {
								countPods++
							}
						}
					} else {
						countPods++
					}
				}
			}
		}
		return countPods
	}

	getPodList := func() (*v1.PodList, error) {
		virtCli, err := kubecli.GetKubevirtClient()
		if err != nil {
			return nil, err
		}
		podList, err := virtCli.CoreV1().Pods(tests.KubeVirtInstallNamespace).List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		return podList, nil
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

	runCommandOnNode := func(command string, node string, args ...string) (err error) {
		cmdName := tests.GetK8sCmdClient()
		newArgs := make([]string, 0)

		// if the cluster is openshift we need to append `adm` for the commands `drain` and `uncordon`
		// as the oc binary is used
		if tests.IsOpenShift() {
			if command == "drain" || command == "uncordon" {
				newArgs = append(newArgs, "adm")
			}
		}

		newArgs = append(newArgs, command)
		newArgs = append(newArgs, node)
		newArgs = append(newArgs, args...)
		_, _, err = tests.RunCommandWithNS("", cmdName, newArgs...)
		return
	}

	uncordonNode := func(node string) (err error) {
		err = runCommandOnNode("uncordon", node)
		return
	}

	uncordonNodes := func(nodeNames ...string) {
		for _, nodeName := range nodeNames {
			err = uncordonNode(nodeName)
			tests.PanicOnError(err)
		}
	}

	drainNodeWithTimeout := func(node string, podSelector string, timeoutAfterSeconds int) (err error) {
		err = runCommandOnNode("drain", node, fmt.Sprintf("--timeout=%ds", timeoutAfterSeconds), podSelector)
		return
	}

	waitForNodesToStabilize := func(expectedNumberOfReadyPods int, podNames []string, nodeNames ...string) {
		Eventually(func() int { return countPendingPods(podNames) },
			DefaultStabilizationTimeoutInSeconds, DefaultPollIntervalInSeconds,
		).Should(Equal(0))
		Eventually(func() int { return countReadyPodsOnNodes(podNames, nodeNames...) },
			DefaultStabilizationTimeoutInSeconds, DefaultPollIntervalInSeconds,
		).Should(BeNumerically(">=", expectedNumberOfReadyPods))
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

		uncordonNodes(nodeNames...)

		waitForNodesToStabilize(4, controlPlanePodNames, nodeNames...)

		selectedNodeName = nodes[0].Name
		selectedNode, err := kubecli.KubevirtClient.CoreV1(virtCli).Nodes().Get(selectedNodeName, metav1.GetOptions{})
		tests.PanicOnError(err)

		// Add label to node that was selected for test
		if selectedNode.Labels == nil {
			selectedNode.Labels = make(map[string]string)
		}
		selectedNode.Labels[labelKey] = labelValue
		_, err = kubecli.KubevirtClient.CoreV1(virtCli).Nodes().Update(selectedNode)
		tests.PanicOnError(err)

		// Add nodeSelector to deployments so that they migrate to selectedNode
		deploymentsClient := kubecli.KubevirtClient.AppsV1(virtCli).Deployments(tests.KubeVirtInstallNamespace)
		deployments, err := deploymentsClient.List(metav1.ListOptions{})
		tests.PanicOnError(err)
		for _, deployment := range deployments.Items {
			if deployment.Name != "virt-api" && deployment.Name != "virt-controller" {
				continue
			}

			labelMap := make(map[string]string)
			labelMap[labelKey] = labelValue
			if deployment.Spec.Template.Spec.NodeSelector == nil {
				deployment.Spec.Template.Spec.NodeSelector = make(map[string]string)
			}
			deployment.Spec.Template.Spec.NodeSelector[labelKey] = labelValue
			_, err = deploymentsClient.Update(&deployment)
			if err != nil {
				tests.PanicOnError(fmt.Errorf("unable to update deployment %+v: %v", deployment, err))
			}
		}

		waitForNodesToStabilize(4, controlPlanePodNames, selectedNodeName)
	})

	AfterEach(func() {
		uncordonNodes(nodeNames...)

		virtCli, err := kubecli.GetKubevirtClient()
		tests.PanicOnError(err)

		// Remove nodeSelector from deployments
		deploymentsClient := kubecli.KubevirtClient.AppsV1(virtCli).Deployments(tests.KubeVirtInstallNamespace)
		deployments, err := deploymentsClient.List(metav1.ListOptions{})
		tests.PanicOnError(err)

		for _, deployment := range deployments.Items {
			if deployment.Name != "virt-api" && deployment.Name != "virt-controller" {
				continue
			}
			delete(deployment.Spec.Template.Spec.NodeSelector, labelKey)
			_, err = deploymentsClient.Update(&deployment)
			tests.PanicOnError(err)
		}

		// Remove label from selectedNode
		selectedNode, err := kubecli.KubevirtClient.CoreV1(virtCli).Nodes().Get(selectedNodeName, metav1.GetOptions{})
		tests.PanicOnError(err)
		delete(selectedNode.Labels, labelKey)
		_, err = kubecli.KubevirtClient.CoreV1(virtCli).Nodes().Update(selectedNode)
		tests.PanicOnError(err)

		waitForNodesToStabilize(4, controlPlanePodNames, nodeNames...)
	})

	When("draining pods of control plane, drain should fail", func() {

		test := func(podName string) {
			By(fmt.Sprintf("Check whether %s has enough %s pods", selectedNodeName, podName))

			Eventually(func() int { return countReadyPodsOnNodes([]string{podName}, selectedNodeName) },
				DefaultStabilizationTimeoutInSeconds, DefaultPollIntervalInSeconds,
			).Should(BeNumerically(">=", 2))

			podSelector := fmt.Sprintf("--pod-selector=kubevirt.io=%s", podName)

			By(fmt.Sprintf("drain node %v\n", selectedNodeName))
			err = drainNodeWithTimeout(selectedNodeName, podSelector, 60)

			Eventually(func() int { return countReadyPodsOnNodes([]string{podName}, selectedNodeName) },
				DefaultStabilizationTimeoutInSeconds, DefaultPollIntervalInSeconds,
			).Should(BeNumerically(">=", 1))
			Eventually(func() int { return countPendingPods([]string{podName}) },
				DefaultStabilizationTimeoutInSeconds, DefaultPollIntervalInSeconds,
			).Should(BeNumerically(">=", 1))

			Expect(err).To(HaveOccurred(), "no error occurred on drain")
		}

		It("for virt-controller pods", func() { test("virt-controller") })
		It("for virt-api pods", func() { test("virt-api") })

	})

})
