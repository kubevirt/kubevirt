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
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/tests/decorators"

	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/util"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
)

var _ = DescribeInfra("virt-handler", func() {

	var (
		virtClient       kubecli.KubevirtClient
		originalKubeVirt *v1.KubeVirt
		nodesToEnableKSM []string
	)

	getNodesWithKSMAvailable := func(virtCli kubecli.KubevirtClient) []string {
		nodes := libnode.GetAllSchedulableNodes(virtCli)

		nodesWithKSM := make([]string, 0)
		for _, node := range nodes.Items {
			command := []string{"cat", "/sys/kernel/mm/ksm/run"}
			_, err := tests.ExecuteCommandInVirtHandlerPod(node.Name, command)
			if err == nil {
				nodesWithKSM = append(nodesWithKSM, node.Name)
			}
		}
		return nodesWithKSM
	}

	forceMemoryPressureOnNodes := func(nodes []string) {
		for _, node := range nodes {
			data := []byte(fmt.Sprintf(`{"metadata": { "annotations": {"%s": "%s", "%s": "%s"}}}`,
				v1.KSMFreePercentOverride, "1.0",
				v1.KSMPagesDecayOverride, "-300",
			))
			_, err := virtClient.CoreV1().Nodes().Patch(context.Background(), node, types.StrategicMergePatchType, data, metav1.PatchOptions{})
			Expect(err).NotTo(HaveOccurred())
		}
	}

	restoreNodes := func(nodes []string) {
		for _, node := range nodes {
			patchBytes := []byte(fmt.Sprintf(`[{"op": "remove", "path": "/metadata/annotations/%s"}, {"op": "remove", "path": "/metadata/annotations/%s"}]`,
				strings.ReplaceAll(v1.KSMFreePercentOverride, "/", "~1"), strings.ReplaceAll(v1.KSMPagesDecayOverride, "/", "~1")))
			_, err := virtClient.CoreV1().Nodes().Patch(context.Background(), node, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
			if err != nil {
				node2, err2 := virtClient.CoreV1().Nodes().Get(context.Background(), node, metav1.GetOptions{})
				Expect(err2).NotTo(HaveOccurred())
				Expect(err).NotTo(HaveOccurred(), `patch:"%s" annotations:%#v`, string(patchBytes), node2.GetAnnotations())
			}
		}
	}

	BeforeEach(func() {
		virtClient = kubevirt.Client()

		nodesToEnableKSM = getNodesWithKSMAvailable(virtClient)
		if len(nodesToEnableKSM) == 0 {
			Fail("There isn't any node with KSM available")
		}

		forceMemoryPressureOnNodes(nodesToEnableKSM)

		originalKubeVirt = util.GetCurrentKv(virtClient)
	})

	AfterEach(func() {
		restoreNodes(nodesToEnableKSM)
		tests.UpdateKubeVirtConfigValueAndWait(originalKubeVirt.Spec.Configuration)
	})

	It("should enable/disable ksm and add/remove annotation on all the nodes when the selector is empty", decorators.KSMRequired, func() {
		kvConfig := originalKubeVirt.Spec.Configuration.DeepCopy()
		ksmConfig := &v1.KSMConfiguration{NodeLabelSelector: &metav1.LabelSelector{}}
		kvConfig.KSMConfiguration = ksmConfig
		tests.UpdateKubeVirtConfigValueAndWait(*kvConfig)
		By("Ensure ksm is enabled and annotation is added in the expected nodes")
		for _, node := range nodesToEnableKSM {
			Eventually(func() (string, error) {
				command := []string{"cat", "/sys/kernel/mm/ksm/run"}
				ksmValue, err := tests.ExecuteCommandInVirtHandlerPod(node, command)
				if err != nil {
					return "", err
				}

				return ksmValue, nil
			}, 3*time.Minute, 2*time.Second).Should(BeEquivalentTo("1\n"), fmt.Sprintf("KSM should be enabled in node %s", node))

			Eventually(func() (bool, error) {
				node, err := virtClient.CoreV1().Nodes().Get(context.Background(), node, metav1.GetOptions{})
				if err != nil {
					return false, err
				}
				value, found := node.GetAnnotations()[v1.KSMHandlerManagedAnnotation]
				return found && value == "true", nil
			}, 3*time.Minute, 2*time.Second).Should(BeTrue(), fmt.Sprintf("Node %s should have %s annotation", node, v1.KSMHandlerManagedAnnotation))
		}

		tests.UpdateKubeVirtConfigValueAndWait(originalKubeVirt.Spec.Configuration)

		By("Ensure ksm is disabled and annotation is set to false in the expected nodes")
		for _, node := range nodesToEnableKSM {
			Eventually(func() (string, error) {
				command := []string{"cat", "/sys/kernel/mm/ksm/run"}
				ksmValue, err := tests.ExecuteCommandInVirtHandlerPod(node, command)
				if err != nil {
					return "", err
				}

				return ksmValue, nil
			}, 3*time.Minute, 2*time.Second).Should(BeEquivalentTo("0\n"), fmt.Sprintf("KSM should be disabled in node %s", node))

			Eventually(func() (bool, error) {
				node, err := virtClient.CoreV1().Nodes().Get(context.Background(), node, metav1.GetOptions{})
				if err != nil {
					return false, err
				}
				value, found := node.GetAnnotations()[v1.KSMHandlerManagedAnnotation]
				return found && value == "true", nil
			}, 3*time.Minute, 2*time.Second).Should(BeFalse(), fmt.Sprintf("Annotation %s should be removed from the node %s", v1.KSMHandlerManagedAnnotation, node))
		}
	})
})
