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
	"kubevirt.io/kubevirt/tests/flags"
)

const (
	WaitSecondsBeforeDeploymentCheck     = 2 * time.Second
	DefaultStabilizationTimeoutInSeconds = 300
	DefaultPollIntervalInSeconds         = 3
	labelKey                             = "control-plane-test"
	labelValue                           = "selected"
)

var _ = Describe("[ref_id:2717]KubeVirt control plane resilience", func() {

	var err error
	var virtCli kubecli.KubevirtClient

	RegisterFailHandler(Fail)

	controlPlaneDeploymentNames := []string{"virt-api", "virt-controller"}

	BeforeEach(func() {
		virtCli, err = kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())
	})

	Context("pod eviction", func() {

		var nodeNames []string
		var selectedNodeName string

		getRunningReadyPods := func(podList *v1.PodList, podNames []string, nodeNames ...string) (pods []*v1.Pod) {
			pods = make([]*v1.Pod, 0)
			for _, pod := range podList.Items {
				if pod.Status.Phase != v1.PodRunning {
					continue
				}
				podReady := tests.PodReady(&pod)
				if podReady != v1.ConditionTrue {
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

		getPodList := func() (podList *v1.PodList, err error) {
			podList, err = virtCli.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(metav1.ListOptions{})
			return
		}

		getSelectedNode := func() (selectedNode *v1.Node, err error) {
			selectedNode, err = virtCli.CoreV1().Nodes().Get(selectedNodeName, metav1.GetOptions{})
			return
		}

		waitForDeploymentsToStabilize := func() (bool, error) {
			deploymentsClient := virtCli.AppsV1().Deployments(flags.KubeVirtInstallNamespace)
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
			return true, nil
		}

		addLabelToSelectedNode := func() (bool, error) {
			selectedNode, err := getSelectedNode()
			if err != nil {
				return false, err
			}
			if selectedNode.Labels == nil {
				selectedNode.Labels = make(map[string]string)
			}
			selectedNode.Labels[labelKey] = labelValue
			_, err = virtCli.CoreV1().Nodes().Update(selectedNode)
			if err != nil {
				return false, fmt.Errorf("failed to update node: %v", err)
			}
			return true, nil
		}

		// Add nodeSelector to deployments so that they get scheduled to selectedNode
		addNodeSelectorToDeployments := func() (bool, error) {
			deploymentsClient := virtCli.AppsV1().Deployments(flags.KubeVirtInstallNamespace)
			for _, deploymentName := range controlPlaneDeploymentNames {
				deployment, err := deploymentsClient.Get(deploymentName, metav1.GetOptions{})
				if err != nil {
					return false, err
				}

				labelMap := make(map[string]string)
				labelMap[labelKey] = labelValue
				if deployment.Spec.Template.Spec.NodeSelector == nil {
					deployment.Spec.Template.Spec.NodeSelector = make(map[string]string)
				}
				deployment.Spec.Template.Spec.NodeSelector[labelKey] = labelValue
				_, err = deploymentsClient.Update(deployment)
				if err != nil {
					return false, err
				}
			}
			return true, nil
		}

		checkControlPlanePodsHaveNodeSelector := func() (bool, error) {
			podList, err := getPodList()
			if err != nil {
				return false, err
			}
			runningControlPlanePods := getRunningReadyPods(podList, controlPlaneDeploymentNames)
			for _, pod := range runningControlPlanePods {
				if actualLabelValue, ok := pod.Spec.NodeSelector[labelKey]; ok {
					if actualLabelValue != labelValue {
						return false, fmt.Errorf("pod %s has node selector %s with value %s, expected was %s", pod.Name, labelKey, actualLabelValue, labelValue)
					}
				} else {
					return false, fmt.Errorf("pod %s has no node selector %s", pod.Name, labelKey)
				}
			}
			return true, nil
		}

		eventuallyWithTimeout := func(f func() (bool, error)) {
			Eventually(f,
				DefaultStabilizationTimeoutInSeconds, DefaultPollIntervalInSeconds,
			).Should(BeTrue())
		}

		setSelectedNodeUnschedulable := func() {
			Eventually(func() error {
				selectedNode, err := getSelectedNode()
				if err != nil {
					return err
				}
				selectedNode.Spec.Unschedulable = true
				if _, err = virtCli.CoreV1().Nodes().Update(selectedNode); err != nil {
					return err
				}
				return nil
			}, 30*time.Second, time.Second).ShouldNot(HaveOccurred())
		}

		BeforeEach(func() {
			tests.BeforeTestCleanup()

			nodes := tests.GetAllSchedulableNodes(virtCli).Items
			nodeNames = make([]string, len(nodes))
			for index, node := range nodes {
				nodeNames[index] = node.Name
			}

			// select one node from result for test, first node will do
			selectedNodeName = nodes[0].Name

			eventuallyWithTimeout(addLabelToSelectedNode)
			eventuallyWithTimeout(addNodeSelectorToDeployments)

			time.Sleep(WaitSecondsBeforeDeploymentCheck)

			eventuallyWithTimeout(checkControlPlanePodsHaveNodeSelector)
			eventuallyWithTimeout(waitForDeploymentsToStabilize)

			setSelectedNodeUnschedulable()
		})

		removeNodeSelectorFromDeployments := func() (bool, error) {
			deploymentsClient := virtCli.AppsV1().Deployments(flags.KubeVirtInstallNamespace)
			for _, deploymentName := range controlPlaneDeploymentNames {
				deployment, err := deploymentsClient.Get(deploymentName, metav1.GetOptions{})
				if err != nil {
					return false, err
				}
				delete(deployment.Spec.Template.Spec.NodeSelector, labelKey)
				_, err = deploymentsClient.Update(deployment)
				if err != nil {
					return false, err
				}
			}
			return true, nil
		}

		// Clean up selectedNode: Remove label and make schedulable again
		cleanUpSelectedNode := func() (bool, error) {
			selectedNode, err := getSelectedNode()
			if err != nil {
				return false, err
			}
			selectedNode.Spec.Unschedulable = false
			delete(selectedNode.Labels, labelKey)
			_, err = virtCli.CoreV1().Nodes().Update(selectedNode)
			if err != nil {
				return false, err
			}
			return true, nil
		}

		checkControlPlanePodsDontHaveNodeSelector := func() (bool, error) {
			podList, err := getPodList()
			if err != nil {
				return false, err
			}
			runningControlPlanePods := getRunningReadyPods(podList, controlPlaneDeploymentNames)
			for _, pod := range runningControlPlanePods {
				if _, ok := pod.Spec.NodeSelector[labelKey]; ok {
					return false, fmt.Errorf("pod %s has still node selector %s", pod.Name, labelKey)
				}
			}
			return true, nil
		}

		AfterEach(func() {
			eventuallyWithTimeout(removeNodeSelectorFromDeployments)
			eventuallyWithTimeout(cleanUpSelectedNode)

			time.Sleep(WaitSecondsBeforeDeploymentCheck)

			eventuallyWithTimeout(checkControlPlanePodsDontHaveNodeSelector)
			eventuallyWithTimeout(waitForDeploymentsToStabilize)
		})

		When("evicting pods of control plane", func() {

			test := func(podName string) {
				By(fmt.Sprintf("Try to evict all pods %s from node %s\n", podName, selectedNodeName))
				podList, err := getPodList()
				Expect(err).ToNot(HaveOccurred())
				runningPods := getRunningReadyPods(podList, []string{podName})
				for index, pod := range runningPods {
					err = virtCli.CoreV1().Pods(flags.KubeVirtInstallNamespace).Evict(&v1beta1.Eviction{ObjectMeta: metav1.ObjectMeta{Name: pod.Name}})
					if index < len(runningPods)-1 {
						Expect(err).ToNot(HaveOccurred())
					}
				}
				Expect(err).To(HaveOccurred(), "no error occurred on evict of last pod")
			}

			It("[test_id:2830]last eviction should fail for virt-controller pods", func() { test("virt-controller") })
			It("[test_id:2799]last eviction should fail for virt-api pods", func() { test("virt-api") })

		})

	})

	Context("control plane components check", func() {

		When("control plane pods are running", func() {

			It("[test_id:2806]virt-controller and virt-api pods have a pod disruption budget", func() {
				deploymentsClient := virtCli.AppsV1().Deployments(flags.KubeVirtInstallNamespace)
				By("check deployments")
				deployments, err := deploymentsClient.List(metav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				expectedDeployments := []string{"virt-api", "virt-controller"}
				for _, expectedDeployment := range expectedDeployments {
					found := false
					for _, deployment := range deployments.Items {
						if deployment.Name != expectedDeployment {
							continue
						}
						found = true
						break
					}
					if !found {
						Fail(fmt.Sprintf("deployment %s not found", expectedDeployment))
					}
				}

				By("check pod disruption budgets exist")
				podDisruptionBudgetList, err := virtCli.PolicyV1beta1().PodDisruptionBudgets(flags.KubeVirtInstallNamespace).List(metav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				for _, controlPlaneDeploymentName := range controlPlaneDeploymentNames {
					pdbName := controlPlaneDeploymentName + "-pdb"
					found := false
					for _, pdb := range podDisruptionBudgetList.Items {
						if pdb.Name != pdbName {
							continue
						}
						found = true
						break
					}
					if !found {
						Fail(fmt.Sprintf("pod disruption budget %s not found for control plane pod %s", pdbName, controlPlaneDeploymentName))
					}
				}
			})

		})

	})

})
