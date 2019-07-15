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
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
)

const (
	DefaultStabilizationTimeoutInSeconds = 180
	DefaultPollIntervalInSeconds         = 5
)

var _ = Describe("KubeVirt control plane resilience", func() {

	var err error
	var nodeNames []string

	tests.FlagParse()

	BeforeEach(func() {
		tests.SkipIfNoCmd("kubectl")
		tests.SkipIfSchedulableNodesLessThan(2)
		tests.BeforeTestCleanup()

		virtCli, err := kubecli.GetKubevirtClient()
		tests.PanicOnError(err)

		nodeNames = make([]string, 0)
		nodes := tests.GetAllSchedulableNodes(virtCli).Items
		for _, node := range nodes {
			nodeNames = append(nodeNames, node.ObjectMeta.Name)
		}
	})

	runCommandOnNode := func(command string, node string, args ...string) (err error) {
		cmdName := tests.GetK8sCmdClient()
		newArgs := make([]string, 0)
		if tests.IsOpenShift() {
			// if the cluster is openshift we need to append `adm` for the commands `drain` and `uncordon`
			// as the oc binary is used
			newArgs = append(newArgs, "adm")
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

	allPodsReady := func() bool {
		virtCli, err := kubecli.GetKubevirtClient()
		if err != nil {
			return false
		}
		podList, err := virtCli.CoreV1().Pods("kubevirt").List(metav1.ListOptions{})
		if err != nil {
			return false
		}
		for _, pod := range podList.Items {
			if pod.Status.Phase != v1.PodRunning {
				continue
			}
			if tests.PodReady(&pod) == v1.ConditionFalse {
				return false
			}
		}
		return true
	}

	eventuallyAllPodsShouldBecomeReadyAfterSeconds := func(seconds int) {
		Eventually(allPodsReady(), seconds, DefaultPollIntervalInSeconds).Should(BeTrue())
	}

	eventuallyAllPodsShouldBecomeReady := func() {
		eventuallyAllPodsShouldBecomeReadyAfterSeconds(DefaultStabilizationTimeoutInSeconds)
	}

	AfterEach(func() {
		for _, nodeName := range nodeNames {
			err = uncordonNode(nodeName)
			Expect(err).ToNot(HaveOccurred())
		}

		eventuallyAllPodsShouldBecomeReadyAfterSeconds(DefaultStabilizationTimeoutInSeconds * 3)
	})

	drainNodeWithTimeout := func(node string, podSelector string, timeoutAfterSeconds int) (err error) {
		err = runCommandOnNode("drain", node, fmt.Sprintf("--timeout=%ds", timeoutAfterSeconds), podSelector)
		return
	}

	drainNode := func(node string, podSelector string) (err error) {
		return drainNodeWithTimeout(node, podSelector, 60)
	}

	countReadyPods := func(podName string) int {
		virtCli, err := kubecli.GetKubevirtClient()
		if err != nil {
			return -1
		}
		podList, err := virtCli.CoreV1().Pods("kubevirt").List(metav1.ListOptions{})
		if err != nil {
			return -1
		}
		countPods := 0
		for _, pod := range podList.Items {
			if pod.Status.Phase != v1.PodRunning {
				continue
			}
			if tests.PodReady(&pod) == v1.ConditionFalse {
				continue
			}
			if strings.Contains(pod.Name, podName) {
				countPods++
			}
		}
		return countPods
	}

	When("draining control plane pods a drain of the last node should fail at first", func() {

		table.DescribeTable("for pod", func(podName string) {
			podSelector := fmt.Sprintf("--pod-selector=kubevirt.io=%s", podName)
			lastNode := nodeNames[len(nodeNames)-1]

			By(fmt.Sprintf("draining all nodes except %s", lastNode))
			for _, nodeName := range nodeNames {
				if nodeName != lastNode {
					By(fmt.Sprintf("draining %s", nodeName))
					err = drainNode(nodeName, podSelector)
					Expect(err).ToNot(HaveOccurred())
				}
			}
			eventuallyAllPodsShouldBecomeReady()

			By(fmt.Sprintf(
				"draining last node %s should fail, because target pod is protected from voluntary evictions by pdb",
				lastNode))
			err = drainNode(lastNode, podSelector)
			Expect(err).To(HaveOccurred())
			Expect(countReadyPods(podName)).Should(BeNumerically(">=", 1))

			By(fmt.Sprintf("uncordoning all nodes except %s", lastNode))
			for _, nodeName := range nodeNames {
				if nodeName != lastNode {
					By(fmt.Sprintf("uncordoning %s", nodeName))
					err = uncordonNode(nodeName)
					Expect(err).ToNot(HaveOccurred())
				}
			}
			eventuallyAllPodsShouldBecomeReady()

			By(fmt.Sprintf("draining %s should not fail", lastNode))
			err = drainNode(lastNode, podSelector)
			Expect(err).ToNot(HaveOccurred())
		},
			table.Entry("virt-controller", "virt-controller"),
			table.Entry("virt-api", "virt-api"),
		)

	})

})
