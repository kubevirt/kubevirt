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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package tests

import (
	"context"
	"fmt"
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests/util"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k6tv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests/libnode"
)

var _ = Describe("[Serial]Node Labeller", func() {
	var virtClient kubecli.KubevirtClient
	var err error

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())

		BeforeTestCleanup()
	})

	Context("basic node-labeller operations", func() {

		Context("ensure node-labeller wakes up", func() {

			It("When node updates its labels", func() {
				By("Fetching one of the nodes")
				nodeList := libnode.GetAllSchedulableNodes(virtClient)
				Expect(nodeList.Items).ToNot(BeEmpty())
				node := nodeList.Items[0].DeepCopy()

				By("Finding host model label key")
				var hostModelLabelKey string
				for key, _ := range node.Labels {
					if strings.HasPrefix(key, k6tv1.SupportedHostModelMigrationCPU) {
						hostModelLabelKey = key
						break
					}
				}
				Expect(hostModelLabelKey).ToNot(BeEmpty(), "cannot find host model for node %s", node.Name)

				By("Removing host model label key")
				libnode.RemoveLabelFromNode(node.Name, hostModelLabelKey)

				By("Expecting host model label key to re-appear")
				Eventually(func() bool {
					node, err = virtClient.CoreV1().Nodes().Get(context.Background(), node.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					_, exists := node.Labels[hostModelLabelKey]
					return exists
				}, 30*time.Second, 3*time.Second).Should(BeTrue(), "host model label is expected to re-appear")
			})

			It("When Kubevirt CR configurations change", func() {
				By("Fetching one of the nodes")
				nodeList := libnode.GetAllSchedulableNodes(virtClient)
				Expect(nodeList.Items).ToNot(BeEmpty())
				node := nodeList.Items[0].DeepCopy()

				By("Finding one supported CPU model")
				var supportedCpuModel string
				for key, _ := range node.Labels {
					if strings.HasPrefix(key, k6tv1.CPUModelLabel) {
						supportedCpuModel = strings.TrimPrefix(key, k6tv1.CPUModelLabel)
						break
					}
				}
				Expect(supportedCpuModel).ToNot(BeEmpty(), "cannot find a supported CPU model for node %s", node.Name)

				By(fmt.Sprintf("Found supported CPU: %s. Marking it as an obsolete CPU model", supportedCpuModel))
				kv := util.GetCurrentKv(virtClient)
				config := kv.Spec.Configuration
				if config.ObsoleteCPUModels == nil {
					config.ObsoleteCPUModels = make(map[string]bool)
				}
				config.ObsoleteCPUModels[supportedCpuModel] = true
				UpdateKubeVirtConfigValueAndWait(config)

				By("Expecting CPU model label to disappear from all nodes")
				Eventually(func() error {
					nodeList = libnode.GetAllSchedulableNodes(virtClient)

					for _, node := range nodeList.Items {
						_, exists := node.Labels[k6tv1.CPUModelLabel+supportedCpuModel]
						if exists {
							return fmt.Errorf("CPU %s did not dissapear from node %s", supportedCpuModel, node.Name)
						}
					}

					return nil
				}, 30*time.Second, 3*time.Second).ShouldNot(HaveOccurred())
			})

		})

	})
})
