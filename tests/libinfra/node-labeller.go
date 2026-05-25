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

package libinfra

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libnode"
)

func WakeNodeLabellerUp(virtClient kubecli.KubevirtClient) {
	const fakeModel = "fake-model-1423"

	ginkgo.By("Updating Kubevirt CR to wake node-labeller up")
	kvConfig := libkubevirt.GetCurrentKv(virtClient).Spec.Configuration.DeepCopy()
	if kvConfig.ObsoleteCPUModels == nil {
		kvConfig.ObsoleteCPUModels = make(map[string]bool)
	}
	kvConfig.ObsoleteCPUModels[fakeModel] = true
	config.UpdateKubeVirtConfigValueAndWait(*kvConfig)
	delete(kvConfig.ObsoleteCPUModels, fakeModel)
	config.UpdateKubeVirtConfigValueAndWait(*kvConfig)
}

func ExpectStoppingNodeLabellerToSucceed(nodeName string, virtClient kubecli.KubevirtClient) *k8sv1.Node {
	var err error
	var node *k8sv1.Node

	Expect(CurrentSpecReport().IsSerial).To(BeTrue(), "stopping / resuming node-labeller is supported for serial tests only")

	By(fmt.Sprintf("Patching node to %s include %s label", nodeName, v1.LabellerSkipNodeAnnotation))
	key, value := v1.LabellerSkipNodeAnnotation, "true"
	libnode.AddAnnotationToNode(nodeName, key, value)

	By(fmt.Sprintf("Expecting node %s to include %s label", nodeName, v1.LabellerSkipNodeAnnotation))
	Eventually(func() bool {
		node, err = virtClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		value, exists := node.Annotations[v1.LabellerSkipNodeAnnotation]
		return exists && value == "true"
	}, 30*time.Second, time.Second).Should(BeTrue(), fmt.Sprintf("node %s is expected to have annotation %s", nodeName, v1.LabellerSkipNodeAnnotation))

	return node
}

func ExpectResumingNodeLabellerToSucceed(nodeName string, virtClient kubecli.KubevirtClient) *k8sv1.Node {
	var err error
	var node *k8sv1.Node

	node, err = virtClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	if _, isNodeLabellerStopped := node.Annotations[v1.LabellerSkipNodeAnnotation]; !isNodeLabellerStopped {
		// Nothing left to do
		return node
	}

	By(fmt.Sprintf("Patching node %s to not include %s annotation", nodeName, v1.LabellerSkipNodeAnnotation))
	libnode.RemoveAnnotationFromNode(nodeName, v1.LabellerSkipNodeAnnotation)

	By(fmt.Sprintf("Removing machine type labels from node %s (so we can later expect them to return)", nodeName))
	for label := range node.Labels {
		if strings.HasPrefix(label, v1.SupportedMachineTypeLabel) {
			libnode.RemoveLabelFromNode(nodeName, label)
		}
	}

	WakeNodeLabellerUp(virtClient)

	By(fmt.Sprintf("Expecting node %s to not include %s annotation", nodeName, v1.LabellerSkipNodeAnnotation))
	Eventually(func() error {
		node, err = virtClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
		Expect(err).ShouldNot(HaveOccurred())

		_, exists := node.Annotations[v1.LabellerSkipNodeAnnotation]
		if exists {
			return fmt.Errorf("node %s is expected to not have annotation %s", node.Name, v1.LabellerSkipNodeAnnotation)
		}

		foundMachineTypeLabel := false
		for labelKey := range node.Labels {
			if strings.HasPrefix(labelKey, v1.SupportedMachineTypeLabel) {
				foundMachineTypeLabel = true
				break
			}
		}
		if !foundMachineTypeLabel {
			return fmt.Errorf("node %s is expected to have a label with %s prefix. this means node-labeller is not enabled for the node", nodeName, v1.SupportedMachineTypeLabel)
		}

		return nil
	}, 30*time.Second, time.Second).ShouldNot(HaveOccurred())

	return node
}
