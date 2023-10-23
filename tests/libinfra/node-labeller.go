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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package libinfra

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/util"
)

func WakeNodeLabellerUp(virtClient kubecli.KubevirtClient) {
	const fakeModel = "fake-model-1423"

	ginkgo.By("Updating Kubevirt CR to wake node-labeller up")
	kvConfig := util.GetCurrentKv(virtClient).Spec.Configuration.DeepCopy()
	if kvConfig.ObsoleteCPUModels == nil {
		kvConfig.ObsoleteCPUModels = make(map[string]bool)
	}
	kvConfig.ObsoleteCPUModels[fakeModel] = true
	tests.UpdateKubeVirtConfigValueAndWait(*kvConfig)
	delete(kvConfig.ObsoleteCPUModels, fakeModel)
	tests.UpdateKubeVirtConfigValueAndWait(*kvConfig)
}

func ExpectStoppingNodeLabellerToSucceed(nodeName string, virtClient kubecli.KubevirtClient) *k8sv1.Node {
	var err error
	var node *k8sv1.Node

	Expect(CurrentSpecReport().IsSerial).To(BeTrue(), "stopping / resuming node-labeller is supported for serial tests only")

	By(fmt.Sprintf("Patching node to %s include %s label", nodeName, v1.LabellerSkipNodeAnnotation))
	p := []patch.PatchOperation{
		{
			Op:    "add",
			Path:  fmt.Sprintf("/metadata/annotations/%s", patch.EscapeJSONPointer(v1.LabellerSkipNodeAnnotation)),
			Value: "true",
		},
	}
	payloadBytes, err := json.Marshal(p)
	Expect(err).ToNot(HaveOccurred())

	_, err = virtClient.ShadowNodeClient().Patch(context.Background(), nodeName, types.JSONPatchType, payloadBytes, metav1.PatchOptions{})
	Expect(err).ToNot(HaveOccurred())

	By(fmt.Sprintf("Expecting node %s to include %s label", nodeName, v1.LabellerSkipNodeAnnotation))
	Eventually(func(g Gomega) bool {
		node, err = virtClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		g.Expect(node.Annotations).To(HaveKeyWithValue(v1.LabellerSkipNodeAnnotation, "true"))

		return true
	}, 30*time.Second, time.Second).Should(BeTrue(), fmt.Sprintf("node %s is expected to have annotation %s", nodeName, v1.LabellerSkipNodeAnnotation))

	return node
}

func ExpectResumingNodeLabellerToSucceed(nodeName string, virtClient kubecli.KubevirtClient) *k8sv1.Node {
	node, err := virtClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	if _, isNodeLabellerStopped := node.Annotations[v1.LabellerSkipNodeAnnotation]; !isNodeLabellerStopped {
		// Nothing left to do
		return node
	}

	By(fmt.Sprintf("Patching node %s to not include %s annotation", nodeName, v1.LabellerSkipNodeAnnotation))
	p := []patch.PatchOperation{
		{
			Op:   "remove",
			Path: fmt.Sprintf("/metadata/annotations/%s", patch.EscapeJSONPointer(v1.LabellerSkipNodeAnnotation)),
		},
	}

	// In order to make sure node-labeller has updated the node, the host-model label (which node-labeller
	// makes sure always resides on any node) will be removed. After node-labeller is enabled again, the
	// host model label would be expected to show up again on the node.
	By(fmt.Sprintf("Removing host model label %s from node %s (so we can later expect it to return)", v1.HostModelCPULabel, nodeName))
	for _, label := range node.Labels {
		if strings.HasPrefix(label, v1.HostModelCPULabel) {
			p = append(p, patch.PatchOperation{
				Op:   "remove",
				Path: fmt.Sprintf("/metadata/labels/%s", patch.EscapeJSONPointer(label)),
			})
		}
	}
	p = append(p, patch.PatchOperation{
		Op:    "add",
		Path:  fmt.Sprintf("/metadata/labels/%s", patch.EscapeJSONPointer(v1.HostModelCPULabel+"by")),
		Value: "true",
	})

	payloadBytes, err := json.Marshal(p)
	Expect(err).ToNot(HaveOccurred())

	// Override shadownode as well
	_, err = virtClient.ShadowNodeClient().Patch(context.Background(), node.Name, types.JSONPatchType, payloadBytes, metav1.PatchOptions{})
	Expect(err).ToNot(HaveOccurred())

	WakeNodeLabellerUp(virtClient)

	By(fmt.Sprintf("Expecting node %s to not include %s annotation", nodeName, v1.LabellerSkipNodeAnnotation))
	Eventually(func(g Gomega) error {
		node, err = virtClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
		Expect(err).ShouldNot(HaveOccurred())

		g.Expect(node.Annotations).To(Not(HaveKey(v1.LabellerSkipNodeAnnotation)))
		g.Expect(node.Labels).To(SatisfyAll(
			Not(HaveKey(v1.HostModelCPULabel+"by")),
			HaveKey(HavePrefix(v1.HostModelCPULabel)),
		))

		return nil
	}, 30*time.Second, time.Second).ShouldNot(HaveOccurred())

	return node
}
