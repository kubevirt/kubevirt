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
	"context"
	"fmt"
	"strings"
	"time"

	v1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/tests/util"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/flags"
)

const (
	DefaultStabilizationTimeoutInSeconds = 300
	DefaultPollIntervalInSeconds         = 3
)

var _ = Describe("[Serial][ref_id:2717][sig-compute]KubeVirt control plane resilience", func() {

	var err error
	var virtCli kubecli.KubevirtClient

	RegisterFailHandler(Fail)

	controlPlaneDeploymentNames := []string{"virt-api", "virt-controller"}

	BeforeEach(func() {
		virtCli, err = kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())
	})

	Context("pod eviction", func() {
		var nodeList []k8sv1.Node

		getRunningReadyPods := func(podList *k8sv1.PodList, podNames []string, nodeNames ...string) (pods []*k8sv1.Pod) {
			pods = make([]*k8sv1.Pod, 0)
			for _, pod := range podList.Items {
				if pod.Status.Phase != k8sv1.PodRunning {
					continue
				}
				podReady := tests.PodReady(&pod)
				if podReady != k8sv1.ConditionTrue {
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

		getPodList := func() (podList *k8sv1.PodList, err error) {
			podList, err = virtCli.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{})
			return
		}

		waitForDeploymentsToStabilize := func() (bool, error) {
			deploymentsClient := virtCli.AppsV1().Deployments(flags.KubeVirtInstallNamespace)
			for _, deploymentName := range controlPlaneDeploymentNames {
				deployment, err := deploymentsClient.Get(context.Background(), deploymentName, metav1.GetOptions{})
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

		eventuallyWithTimeout := func(f func() (bool, error)) {
			Eventually(f,
				DefaultStabilizationTimeoutInSeconds, DefaultPollIntervalInSeconds,
			).Should(BeTrue())
		}

		setNodeUnschedulable := func(nodeName string) {
			Eventually(func() error {
				selectedNode, err := virtCli.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				selectedNode.Spec.Unschedulable = true
				if _, err = virtCli.CoreV1().Nodes().Update(context.Background(), selectedNode, metav1.UpdateOptions{}); err != nil {
					return err
				}
				return nil
			}, 30*time.Second, time.Second).ShouldNot(HaveOccurred())
		}

		setNodeSchedulable := func(nodeName string) {
			Eventually(func() error {
				selectedNode, err := virtCli.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				selectedNode.Spec.Unschedulable = false
				if _, err = virtCli.CoreV1().Nodes().Update(context.Background(), selectedNode, metav1.UpdateOptions{}); err != nil {
					return err
				}
				return nil
			}, 30*time.Second, time.Second).ShouldNot(HaveOccurred())
		}

		BeforeEach(func() {
			tests.BeforeTestCleanup()

			nodeList = util.GetAllSchedulableNodes(virtCli).Items
			for _, node := range nodeList {
				setNodeUnschedulable(node.Name)
			}
			eventuallyWithTimeout(waitForDeploymentsToStabilize)
		})

		AfterEach(func() {
			for _, node := range nodeList {
				setNodeSchedulable(node.Name)
			}
			eventuallyWithTimeout(waitForDeploymentsToStabilize)
		})

		When("evicting pods of control plane", func() {
			test := func(podName string) {
				By(fmt.Sprintf("Try to evict all pods %s\n", podName))
				podList, err := getPodList()
				Expect(err).ToNot(HaveOccurred())
				runningPods := getRunningReadyPods(podList, []string{podName})
				Expect(len(runningPods)).ToNot(Equal(0))
				for index, pod := range runningPods {
					err = virtCli.CoreV1().Pods(flags.KubeVirtInstallNamespace).Evict(context.Background(), &v1beta1.Eviction{ObjectMeta: metav1.ObjectMeta{Name: pod.Name}})
					// trying to evict the last running pod in this list should fail
					if index == len(runningPods)-1 {
						Expect(err).To(HaveOccurred(), "no error occurred on evict of last pod")
					} else {
						Expect(err).ToNot(HaveOccurred())
					}
				}
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
				deployments, err := deploymentsClient.List(context.Background(), metav1.ListOptions{})
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
				podDisruptionBudgetList, err := virtCli.PolicyV1beta1().PodDisruptionBudgets(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{})
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

		When("Control plane pods temporarily lose connection to Kubernetes API", func() {
			// virt-handler is the only component that has the tools to add blackhole routes for testing healthz. Ideally we would test all component healthz endpoints.
			componentName := "virt-handler"

			getVirtHandler := func() *v1.DaemonSet {
				daemonSet, err := virtCli.AppsV1().DaemonSets(flags.KubeVirtInstallNamespace).Get(context.Background(), componentName, metav1.GetOptions{})
				ExpectWithOffset(1, err).NotTo(HaveOccurred())
				return daemonSet
			}

			readyFunc := func() int32 {
				return getVirtHandler().Status.NumberReady
			}

			blackHolePodFunc := func(addOrDel string) {
				pods, err := virtCli.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: fmt.Sprintf("kubevirt.io=%s", componentName)})
				Expect(err).NotTo(HaveOccurred())

				serviceIp, err := tests.GetKubernetesApiServiceIp(virtCli)
				Expect(err).NotTo(HaveOccurred())

				for _, pod := range pods.Items {
					_, err = tests.ExecuteCommandOnPod(virtCli, &pod, componentName, []string{"ip", "route", addOrDel, "blackhole", serviceIp})
					Expect(err).NotTo(HaveOccurred())
				}
			}

			It("should fail health checks when connectivity is lost, and recover when connectivity is regained", func() {
				desiredDeamonsSetCount := getVirtHandler().Status.DesiredNumberScheduled

				By("ensuring we have ready pods")
				Eventually(readyFunc, 30*time.Second, time.Second).Should(BeNumerically(">", 0))

				By("blocking connection to API on pods")
				blackHolePodFunc("add")

				By("ensuring we no longer have a ready pod")
				Eventually(readyFunc, 120*time.Second, time.Second).Should(BeNumerically("==", 0))

				By("removing blockage to API")
				blackHolePodFunc("del")

				By("ensuring we now have a ready virt-handler daemonset")
				Eventually(readyFunc, 30*time.Second, time.Second).Should(BeNumerically("==", desiredDeamonsSetCount))
			})
		})

	})

})
