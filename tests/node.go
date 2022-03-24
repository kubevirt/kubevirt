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
	"encoding/json"
	"strings"

	"github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"

	v13 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"

	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/cleanup"
	"kubevirt.io/kubevirt/tests/util"
)

var SchedulableNode = ""

func CleanNodes() {
	virtCli, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)
	nodes := util.GetAllSchedulableNodes(virtCli).Items

	clusterDrainKey := GetNodeDrainKey()

	for _, node := range nodes {

		old, err := json.Marshal(node)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		new := node.DeepCopy()

		k8sClient := clientcmd.GetK8sCmdClient()
		if k8sClient == "oc" {
			clientcmd.RunCommandWithNS("", k8sClient, "adm", "uncordon", node.Name)
		} else {
			clientcmd.RunCommandWithNS("", k8sClient, "uncordon", node.Name)
		}

		found := false
		taints := []v1.Taint{}
		for _, taint := range node.Spec.Taints {

			if taint.Key == clusterDrainKey && taint.Effect == v1.TaintEffectNoSchedule {
				found = true
			} else if taint.Key == "kubevirt.io/drain" && taint.Effect == v1.TaintEffectNoSchedule {
				// this key is used as a fallback if the original drain key is built-in
				found = true
			} else if taint.Key == "kubevirt.io/alt-drain" && taint.Effect == v1.TaintEffectNoSchedule {
				// this key is used in testing as a custom alternate drain key
				found = true
			} else {
				taints = append(taints, taint)
			}

		}
		new.Spec.Taints = taints

		for k := range node.Labels {
			if strings.HasPrefix(k, cleanup.KubeVirtTestLabelPrefix) {
				found = true
				delete(new.Labels, k)
			}
		}

		if node.Spec.Unschedulable {
			new.Spec.Unschedulable = false
		}

		if !found {
			continue
		}
		newJson, err := json.Marshal(new)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		patch, err := strategicpatch.CreateTwoWayMergePatch(old, newJson, node)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		_, err = virtCli.CoreV1().Nodes().Patch(context.Background(), node.Name, types.StrategicMergePatchType, patch, v12.PatchOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	}
}

func GetNodeDrainKey() string {
	virtClient, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)

	kv := util.GetCurrentKv(virtClient)
	if kv.Spec.Configuration.MigrationConfiguration != nil && kv.Spec.Configuration.MigrationConfiguration.NodeDrainTaintKey != nil {
		return *kv.Spec.Configuration.MigrationConfiguration.NodeDrainTaintKey
	}

	return virtconfig.NodeDrainTaintDefaultKey
}

func AddLabelToNode(nodeName string, key string, value string) {
	virtCli, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)
	node, err := virtCli.CoreV1().Nodes().Get(context.Background(), nodeName, v12.GetOptions{})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	old, err := json.Marshal(node)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	new := node.DeepCopy()
	new.Labels[key] = value

	newJson, err := json.Marshal(new)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	patch, err := strategicpatch.CreateTwoWayMergePatch(old, newJson, node)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	_, err = virtCli.CoreV1().Nodes().Patch(context.Background(), node.Name, types.StrategicMergePatchType, patch, v12.PatchOptions{})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
}

func RemoveLabelFromNode(nodeName string, key string) {
	virtCli, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)
	node, err := virtCli.CoreV1().Nodes().Get(context.Background(), nodeName, v12.GetOptions{})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	if _, exists := node.Labels[key]; !exists {
		return
	}

	old, err := json.Marshal(node)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	new := node.DeepCopy()
	delete(new.Labels, key)

	newJson, err := json.Marshal(new)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	patch, err := strategicpatch.CreateTwoWayMergePatch(old, newJson, node)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	_, err = virtCli.CoreV1().Nodes().Patch(context.Background(), node.Name, types.StrategicMergePatchType, patch, v12.PatchOptions{})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
}

func Taint(nodeName string, key string, effect v1.TaintEffect) {
	virtCli, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)
	node, err := virtCli.CoreV1().Nodes().Get(context.Background(), nodeName, v12.GetOptions{})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	old, err := json.Marshal(node)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	new := node.DeepCopy()
	new.Spec.Taints = append(new.Spec.Taints, v1.Taint{
		Key:    key,
		Effect: effect,
	})

	newJson, err := json.Marshal(new)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	patch, err := strategicpatch.CreateTwoWayMergePatch(old, newJson, node)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	_, err = virtCli.CoreV1().Nodes().Patch(context.Background(), node.Name, types.StrategicMergePatchType, patch, v12.PatchOptions{})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
}

func GetNodesWithKVM() []*v1.Node {
	virtClient, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)
	listOptions := v12.ListOptions{LabelSelector: v13.AppLabel + "=virt-handler"}
	virtHandlerPods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), listOptions)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	nodes := make([]*v1.Node, 0)
	// cluster is not ready until all nodes are ready.
	for _, pod := range virtHandlerPods.Items {
		virtHandlerNode, err := virtClient.CoreV1().Nodes().Get(context.Background(), pod.Spec.NodeName, v12.GetOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		_, ok := virtHandlerNode.Status.Allocatable[services.KvmDevice]
		if ok {
			nodes = append(nodes, virtHandlerNode)
		}
	}
	return nodes
}
