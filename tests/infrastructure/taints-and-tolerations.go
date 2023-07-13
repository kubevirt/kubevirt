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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package infrastructure

import (
	"context"
	"fmt"
	"time"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/tests/libnode"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/tests/flags"
)

var _ = DescribeInfra("[rfe_id:4126][crit:medium][vendor:cnv-qe@redhat.com][level:component]Taints and toleration", func() {

	var (
		virtClient kubecli.KubevirtClient
	)
	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("CriticalAddonsOnly taint set on a node", func() {

		var selectedNodeName string

		BeforeEach(func() {
			selectedNodeName = ""
		})

		AfterEach(func() {
			if selectedNodeName != "" {
				By("removing the taint from the tainted node")
				selectedNode, err := virtClient.CoreV1().Nodes().Get(context.Background(), selectedNodeName, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				var taints []k8sv1.Taint
				for _, taint := range selectedNode.Spec.Taints {
					if taint.Key != "CriticalAddonsOnly" {
						taints = append(taints, taint)
					}
				}
				patchData, err := patch.GenerateTestReplacePatch("/spec/taints", selectedNode.Spec.Taints, taints)
				Expect(err).NotTo(HaveOccurred())
				selectedNode, err = virtClient.CoreV1().Nodes().Patch(context.Background(), selectedNode.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("[test_id:4134] kubevirt components on that node should not evict", func() {

			By("finding all kubevirt pods")
			pods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{})
			Expect(err).ShouldNot(HaveOccurred(), "failed listing kubevirt pods")
			Expect(pods.Items).ToNot(BeEmpty(), "no kubevirt pods found")

			By("finding all schedulable nodes")
			schedulableNodesList := libnode.GetAllSchedulableNodes(virtClient)
			schedulableNodes := map[string]*k8sv1.Node{}
			for _, node := range schedulableNodesList.Items {
				schedulableNodes[node.Name] = node.DeepCopy()
			}

			By("selecting one compute only node that runs kubevirt components")
			// control-plane nodes should never have the CriticalAddonsOnly taint because core components might not
			// tolerate this taint because it is meant to be used on compute nodes only. If we set this taint
			// on a control-plane node, we risk in breaking the test cluster.
			for _, pod := range pods.Items {
				node, ok := schedulableNodes[pod.Spec.NodeName]
				if !ok {
					// Pod is running on a non-schedulable node?
					continue
				}

				if _, isControlPlane := node.Labels["node-role.kubernetes.io/control-plane"]; isControlPlane {
					continue
				}

				selectedNodeName = node.Name
				break
			}

			// It is possible to run this test on a cluster that simply does not have worker nodes.
			// Since KubeVirt can't control that, the only correct action is to halt the test.
			if selectedNodeName == "" {
				Skip("Could nould determine a node to safely taint")
			}

			By("setting up a watch for terminated pods")
			lw, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).Watch(context.Background(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			// in the test env, we also deploy non core-kubevirt apps
			kvCoreApps := map[string]string{
				"virt-handler":    "",
				"virt-controller": "",
				"virt-api":        "",
				"virt-operator":   "",
			}

			signalTerminatedPods := func(stopCn <-chan bool, eventsCn <-chan watch.Event, terminatedPodsCn chan<- bool) {
				for {
					select {
					case <-stopCn:
						return
					case e := <-eventsCn:
						pod, ok := e.Object.(*k8sv1.Pod)
						Expect(ok).To(BeTrue())
						if _, isCoreApp := kvCoreApps[pod.Name]; !isCoreApp {
							continue
						}
						if pod.DeletionTimestamp != nil {
							By(fmt.Sprintf("%s terminated", pod.Name))
							terminatedPodsCn <- true
							return
						}
					}
				}
			}
			stopCn := make(chan bool, 1)
			terminatedPodsCn := make(chan bool, 1)
			go signalTerminatedPods(stopCn, lw.ResultChan(), terminatedPodsCn)

			By("tainting the selected node")
			selectedNode, err := virtClient.CoreV1().Nodes().Get(context.Background(), selectedNodeName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			taints := append(selectedNode.Spec.Taints, k8sv1.Taint{
				Key:    "CriticalAddonsOnly",
				Value:  "",
				Effect: k8sv1.TaintEffectNoExecute,
			})

			patchData, err := patch.GenerateTestReplacePatch("/spec/taints", selectedNode.Spec.Taints, taints)
			Expect(err).ToNot(HaveOccurred())
			selectedNode, err = virtClient.CoreV1().Nodes().Patch(context.Background(), selectedNode.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			Consistently(terminatedPodsCn, 5*time.Second).ShouldNot(Receive(), "pods should not terminate")
			stopCn <- true
		})

	})
})
