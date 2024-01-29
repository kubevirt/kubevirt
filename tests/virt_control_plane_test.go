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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k6sv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/testsuite"
	"kubevirt.io/kubevirt/tests/util"
)

const (
	DefaultStabilizationTimeoutInSeconds = 300
	DefaultPollIntervalInSeconds         = 3
)

const (
	multiReplica  = true
	singleReplica = false
)

var _ = Describe("[Serial][ref_id:2717][sig-compute]KubeVirt control plane resilience", Serial, decorators.SigCompute, func() {

	var virtCli kubecli.KubevirtClient

	RegisterFailHandler(Fail)

	controlPlaneDeploymentNames := []string{"virt-api", "virt-controller"}

	BeforeEach(func() {
		virtCli = kubevirt.Client()
	})

	Context("pod eviction", func() {
		var nodeList []k8sv1.Node

		getRunningReadyPods := func(podList *k8sv1.PodList, podNames []string, nodeNames ...string) (pods []*k8sv1.Pod) {
			pods = make([]*k8sv1.Pod, 0)
			for _, pod := range podList.Items {
				if pod.Status.Phase != k8sv1.PodRunning {
					continue
				}

				if success, err := matcher.HaveConditionTrue(k8sv1.PodReady).Match(pod); !success {
					Expect(err).ToNot(HaveOccurred())
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

		BeforeEach(func() {
			nodeList = libnode.GetAllSchedulableNodes(virtCli).Items
			for _, node := range nodeList {
				libnode.SetNodeUnschedulable(node.Name, virtCli)
			}
			eventuallyWithTimeout(waitForDeploymentsToStabilize)
		})

		AfterEach(func() {
			for _, node := range nodeList {
				libnode.SetNodeSchedulable(node.Name, virtCli)
			}
			eventuallyWithTimeout(waitForDeploymentsToStabilize)
		})

		DescribeTable("evicting pods of control plane", func(podName string, isMultiReplica bool, msg string) {
			if isMultiReplica {
				checks.SkipIfSingleReplica(virtCli)
			} else {
				checks.SkipIfMultiReplica(virtCli)
			}
			By(fmt.Sprintf("Try to evict all pods %s\n", podName))
			podList, err := getPodList()
			Expect(err).ToNot(HaveOccurred())
			runningPods := getRunningReadyPods(podList, []string{podName})
			Expect(runningPods).ToNot(BeEmpty())
			for index, pod := range runningPods {
				err = virtCli.CoreV1().Pods(flags.KubeVirtInstallNamespace).EvictV1beta1(context.Background(), &v1beta1.Eviction{ObjectMeta: metav1.ObjectMeta{Name: pod.Name}})
				if index == len(runningPods)-1 {
					if isMultiReplica {
						Expect(err).To(HaveOccurred(), msg)
					} else {
						Expect(err).ToNot(HaveOccurred(), msg)
					}
				} else {
					Expect(err).ToNot(HaveOccurred())
				}
			}
		},
			Entry("[test_id:2830]last eviction should fail for multi-replica virt-controller pods",
				"virt-controller", multiReplica, "no error occurred on evict of last virt-controller pod"),
			Entry("[test_id:2799]last eviction should fail for multi-replica virt-api pods",
				"virt-api", multiReplica, "no error occurred on evict of last virt-api pod"),
			Entry("eviction of single-replica virt-controller pod should succeed",
				"virt-controller", singleReplica, "error occurred on eviction of single-replica virt-controller pod"),
			Entry("eviction of multi-replica virt-api pod should succeed",
				"virt-api", singleReplica, "error occurred on eviction of single-replica virt-api pod"),
		)
	})

	Context("control plane components check", func() {

		When("control plane pods are running", func() {

			It("[test_id:2806]virt-controller and virt-api pods have a pod disruption budget", func() {
				// Single replica deployments do not create PDBs
				checks.SkipIfSingleReplica(virtCli)

				deploymentsClient := virtCli.AppsV1().Deployments(flags.KubeVirtInstallNamespace)
				By("check deployments")
				deployments, err := deploymentsClient.List(context.Background(), metav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				for _, expectedDeployment := range controlPlaneDeploymentNames {
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
				podDisruptionBudgetList, err := virtCli.PolicyV1().PodDisruptionBudgets(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{})
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

			getHandlerPods := func() *k8sv1.PodList {
				pods, err := virtCli.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: fmt.Sprintf("kubevirt.io=%s", componentName)})
				Expect(err).NotTo(HaveOccurred())
				return pods
			}

			It("should fail health checks when connectivity is lost, and recover when connectivity is regained", func() {
				desiredDeamonsSetCount := getVirtHandler().Status.DesiredNumberScheduled

				By("ensuring we have ready pods")
				Eventually(readyFunc, 30*time.Second, time.Second).Should(BeNumerically(">", 0))

				By("blocking connection to API on pods")
				libpod.AddKubernetesApiBlackhole(getHandlerPods(), componentName)

				By("ensuring we no longer have a ready pod")
				Eventually(readyFunc, 120*time.Second, time.Second).Should(BeNumerically("==", 0))

				By("removing blockage to API")
				libpod.DeleteKubernetesApiBlackhole(getHandlerPods(), componentName)

				By("ensuring we now have a ready virt-handler daemonset")
				Eventually(readyFunc, 30*time.Second, time.Second).Should(BeNumerically("==", desiredDeamonsSetCount))

				By("changing a setting and ensuring that the config update watcher eventually resumes and picks it up")
				migrationBandwidth := resource.MustParse("1Mi")
				kv := util.GetCurrentKv(virtCli)
				kv.Spec.Configuration.MigrationConfiguration = &k6sv1.MigrationConfiguration{
					BandwidthPerMigration: &migrationBandwidth,
				}
				kv = testsuite.UpdateKubeVirtConfigValue(kv.Spec.Configuration)
				tests.WaitForConfigToBePropagatedToComponent("kubevirt.io=virt-handler", kv.ResourceVersion, tests.ExpectResourceVersionToBeLessEqualThanConfigVersion, 60*time.Second)
			})
		})

	})

})
