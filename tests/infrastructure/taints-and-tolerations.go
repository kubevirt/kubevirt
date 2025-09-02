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

package infrastructure

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libnode"
)

var _ = Describe(SIGSerial("[rfe_id:4126][crit:medium][vendor:cnv-qe@redhat.com][level:component]Taints and toleration", func() {
	var virtClient kubecli.KubevirtClient
	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("CriticalAddonsOnly taint set on a node", func() {
		var (
			possiblyTaintedNodeName string
			kubevirtPodsOnNode      []string
			deploymentsOnNode       []types.NamespacedName
		)

		BeforeEach(func() {
			possiblyTaintedNodeName = ""
			kubevirtPodsOnNode = nil
			deploymentsOnNode = nil

			pods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{})
			Expect(err).ShouldNot(HaveOccurred(), "failed listing kubevirt pods")
			Expect(pods.Items).ToNot(BeEmpty(), "no kubevirt pods found")

			nodeName := getNodeWithOneOfPods(virtClient, pods.Items)

			// It is possible to run this test on a cluster that simply does not have worker nodes.
			// Since KubeVirt can't control that, the only correct action is to halt the test.
			if nodeName == "" {
				Fail("Could not determine a node to safely taint")
			}

			podsOnNode, err := virtClient.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{
				FieldSelector: fields.OneTermEqualSelector("spec.nodeName", nodeName).String(),
			})
			Expect(err).NotTo(HaveOccurred())

			kubevirtPodsOnNode = filterKubevirtPods(podsOnNode.Items)
			deploymentsOnNode = getDeploymentsForPods(virtClient, podsOnNode.Items)

			By("tainting the selected node")
			selectedNode, err := virtClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			possiblyTaintedNodeName = nodeName

			taints := append(selectedNode.Spec.Taints, k8sv1.Taint{
				Key:    "CriticalAddonsOnly",
				Value:  "",
				Effect: k8sv1.TaintEffectNoExecute,
			})

			patchData, err := patch.GenerateTestReplacePatch("/spec/taints", selectedNode.Spec.Taints, taints)
			Expect(err).ToNot(HaveOccurred())
			_, err = virtClient.CoreV1().Nodes().Patch(
				context.Background(), selectedNode.Name,
				types.JSONPatchType, patchData, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			if possiblyTaintedNodeName == "" {
				return
			}

			By("removing the taint from the tainted node")
			selectedNode, err := virtClient.CoreV1().Nodes().Get(context.Background(), possiblyTaintedNodeName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			var hasTaint bool
			var otherTaints []k8sv1.Taint
			for _, taint := range selectedNode.Spec.Taints {
				if taint.Key == "CriticalAddonsOnly" {
					hasTaint = true
				} else {
					otherTaints = append(otherTaints, taint)
				}
			}

			if !hasTaint {
				return
			}

			patchData, err := patch.GenerateTestReplacePatch("/spec/taints", selectedNode.Spec.Taints, otherTaints)
			Expect(err).NotTo(HaveOccurred())
			_, err = virtClient.CoreV1().Nodes().Patch(
				context.Background(), selectedNode.Name,
				types.JSONPatchType, patchData, metav1.PatchOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Waiting until all affected deployments have at least 1 ready replica
			checkedDeployments := map[types.NamespacedName]struct{}{}
			Eventually(func(g Gomega) {
				for _, namespacedName := range deploymentsOnNode {
					if _, ok := checkedDeployments[namespacedName]; ok {
						continue
					}

					deployment, err := virtClient.AppsV1().Deployments(namespacedName.Namespace).
						Get(context.Background(), namespacedName.Name, metav1.GetOptions{})
					if errors.IsNotFound(err) {
						checkedDeployments[namespacedName] = struct{}{}
						continue
					}
					g.Expect(err).NotTo(HaveOccurred())

					if deployment.DeletionTimestamp != nil || *deployment.Spec.Replicas == 0 {
						checkedDeployments[namespacedName] = struct{}{}
						continue
					}
					g.Expect(deployment.Status.ReadyReplicas).To(
						BeNumerically(">=", 1),
						fmt.Sprintf("Deployment %s is not ready", namespacedName.String()),
					)
					checkedDeployments[namespacedName] = struct{}{}
				}
			}, time.Minute, time.Second).Should(Succeed())
		})

		It("[test_id:4134] kubevirt components on that node should not evict", func() {
			timeout := 10 * time.Second
			Consistently(func(g Gomega) {
				for _, podName := range kubevirtPodsOnNode {
					pod, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).Get(context.Background(), podName, metav1.GetOptions{})
					g.Expect(err).NotTo(HaveOccurred(),
						fmt.Sprintf("error getting pod %s/%s",
							flags.KubeVirtInstallNamespace, podName))

					g.Expect(pod.DeletionTimestamp).To(BeNil(), fmt.Sprintf("pod %s/%s is being deleted", flags.KubeVirtInstallNamespace, podName))

					g.Expect(pod.Spec.NodeName).To(Equal(possiblyTaintedNodeName),
						fmt.Sprintf("pod %s/%s does not run on tainted node",
							flags.KubeVirtInstallNamespace, podName))
				}
			}, timeout, time.Second).Should(Succeed())
		})
	})
}))

func getNodeWithOneOfPods(virtClient kubecli.KubevirtClient, pods []k8sv1.Pod) string {
	schedulableNodesList := libnode.GetAllSchedulableNodes(virtClient)
	schedulableNodes := map[string]*k8sv1.Node{}
	for _, node := range schedulableNodesList.Items {
		schedulableNodes[node.Name] = node.DeepCopy()
	}

	// control-plane nodes should never have the CriticalAddonsOnly taint because core components might not
	// tolerate this taint because it is meant to be used on compute nodes only. If we set this taint
	// on a control-plane node, we risk in breaking the test cluster.
	for i := range pods {
		node, ok := schedulableNodes[pods[i].Spec.NodeName]
		if !ok {
			// Pod is running on a non-schedulable node?
			continue
		}

		if _, isControlPlane := node.Labels["node-role.kubernetes.io/control-plane"]; isControlPlane {
			continue
		}

		return node.Name
	}
	return ""
}

func filterKubevirtPods(pods []k8sv1.Pod) []string {
	kubevirtPodPrefixes := []string{
		"virt-handler",
		"virt-controller",
		"virt-api",
		"virt-operator",
	}

	var result []string
	for i := range pods {
		if pods[i].Namespace != flags.KubeVirtInstallNamespace {
			continue
		}
		for _, prefix := range kubevirtPodPrefixes {
			if strings.HasPrefix(pods[i].Name, prefix) {
				result = append(result, pods[i].Name)
				break
			}
		}
	}
	return result
}

func getDeploymentsForPods(virtClient kubecli.KubevirtClient, pods []k8sv1.Pod) []types.NamespacedName {
	// Listing all deployments to find which ones belong to the pods.
	allDeployments, err := virtClient.AppsV1().Deployments("").List(context.Background(), metav1.ListOptions{})
	Expect(err).NotTo(HaveOccurred())

	var result []types.NamespacedName
	for i := range allDeployments.Items {
		deployment := allDeployments.Items[i]
		selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
		Expect(err).NotTo(HaveOccurred())

		for k := range pods {
			if selector.Matches(labels.Set(pods[k].Labels)) {
				result = append(result, types.NamespacedName{
					Namespace: deployment.Namespace,
					Name:      deployment.Name,
				})
				break
			}
		}
	}
	return result
}
