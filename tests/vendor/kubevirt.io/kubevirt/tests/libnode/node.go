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

package libnode

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"kubevirt.io/kubevirt/pkg/util/nodes"

	utiltype "kubevirt.io/kubevirt/pkg/util/types"

	"k8s.io/apimachinery/pkg/api/equality"

	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"

	v1 "kubevirt.io/api/core/v1"
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
	nodes := GetAllSchedulableNodes(virtCli).Items

	clusterDrainKey := GetNodeDrainKey()

	for _, node := range nodes {

		old, err := json.Marshal(node)
		Expect(err).ToNot(HaveOccurred())
		new := node.DeepCopy()

		k8sClient := clientcmd.GetK8sCmdClient()
		if k8sClient == "oc" {
			clientcmd.RunCommandWithNS("", k8sClient, "adm", "uncordon", node.Name)
		} else {
			clientcmd.RunCommandWithNS("", k8sClient, "uncordon", node.Name)
		}

		found := false
		taints := []k8sv1.Taint{}
		for _, taint := range node.Spec.Taints {

			if taint.Key == clusterDrainKey && taint.Effect == k8sv1.TaintEffectNoSchedule {
				found = true
			} else if taint.Key == "kubevirt.io/drain" && taint.Effect == k8sv1.TaintEffectNoSchedule {
				// this key is used as a fallback if the original drain key is built-in
				found = true
			} else if taint.Key == "kubevirt.io/alt-drain" && taint.Effect == k8sv1.TaintEffectNoSchedule {
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
		Expect(err).ToNot(HaveOccurred())

		patch, err := strategicpatch.CreateTwoWayMergePatch(old, newJson, node)
		Expect(err).ToNot(HaveOccurred())

		_, err = virtCli.CoreV1().Nodes().Patch(context.Background(), node.Name, types.StrategicMergePatchType, patch, k8smetav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())
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

type mapType string

const (
	label      mapType = "label"
	annotation mapType = "annotation"
)

type mapAction string

const (
	add    mapAction = "add"
	remove mapAction = "remove"
)

func patchLabelAnnotationHelper(virtCli kubecli.KubevirtClient, nodeName string, newMap, oldMap map[string]string, mapType mapType) (*k8sv1.Node, error) {
	p := []utiltype.PatchOperation{
		{
			Op:    "test",
			Path:  "/metadata/" + string(mapType) + "s",
			Value: oldMap,
		},
		{
			Op:    "replace",
			Path:  "/metadata/" + string(mapType) + "s",
			Value: newMap,
		},
	}

	patchBytes, err := json.Marshal(p)
	Expect(err).ToNot(HaveOccurred())

	patchedNode, err := virtCli.CoreV1().Nodes().Patch(context.Background(), nodeName, types.JSONPatchType, patchBytes, k8smetav1.PatchOptions{})
	return patchedNode, err
}

// Adds or removes a label or annotation from a node. When removing a label/annotation, the "value" parameter
// is ignored.
func addRemoveLabelAnnotationHelper(nodeName, key, value string, mapType mapType, mapAction mapAction) *k8sv1.Node {
	var fetchMap func(node *k8sv1.Node) map[string]string
	var mutateMap func(key, val string, m map[string]string) map[string]string

	switch mapType {
	case label:
		fetchMap = func(node *k8sv1.Node) map[string]string { return node.Labels }
	case annotation:
		fetchMap = func(node *k8sv1.Node) map[string]string { return node.Annotations }
	}

	switch mapAction {
	case add:
		mutateMap = func(key, val string, m map[string]string) map[string]string {
			m[key] = val
			return m
		}
	case remove:
		mutateMap = func(key, val string, m map[string]string) map[string]string {
			delete(m, key)
			return m
		}
	}

	Expect(fetchMap).ToNot(BeNil())
	Expect(mutateMap).ToNot(BeNil())

	virtCli, err := kubecli.GetKubevirtClient()
	Expect(err).ToNot(HaveOccurred())

	var nodeToReturn *k8sv1.Node
	EventuallyWithOffset(2, func() error {
		origNode, err := virtCli.CoreV1().Nodes().Get(context.Background(), nodeName, k8smetav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		originalMap := fetchMap(origNode)
		expectedMap := make(map[string]string, len(originalMap))
		for k, v := range originalMap {
			expectedMap[k] = v
		}

		expectedMap = mutateMap(key, value, expectedMap)

		if equality.Semantic.DeepEqual(originalMap, expectedMap) {
			// key and value already exist in node
			nodeToReturn = origNode
			return nil
		}

		patchedNode, err := patchLabelAnnotationHelper(virtCli, nodeName, expectedMap, originalMap, mapType)
		if err != nil {
			return err
		}

		resultMap := fetchMap(patchedNode)

		const errPattern = "adding %s (key: %s. value: %s) to node %s failed. Expected %ss: %v, actual: %v"
		if !equality.Semantic.DeepEqual(resultMap, expectedMap) {
			return fmt.Errorf(errPattern, string(mapType), key, value, nodeName, string(mapType), expectedMap, resultMap)
		}

		nodeToReturn = patchedNode
		return nil
	}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

	return nodeToReturn
}

func AddLabelToNode(nodeName, key, value string) *k8sv1.Node {
	return addRemoveLabelAnnotationHelper(nodeName, key, value, label, add)
}

func AddAnnotationToNode(nodeName, key, value string) *k8sv1.Node {
	return addRemoveLabelAnnotationHelper(nodeName, key, value, annotation, add)
}

func RemoveLabelFromNode(nodeName string, key string) *k8sv1.Node {
	return addRemoveLabelAnnotationHelper(nodeName, key, "", label, remove)
}

func RemoveAnnotationFromNode(nodeName string, key string) *k8sv1.Node {
	return addRemoveLabelAnnotationHelper(nodeName, key, "", annotation, remove)
}

func Taint(nodeName string, key string, effect k8sv1.TaintEffect) {
	virtCli, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)
	node, err := virtCli.CoreV1().Nodes().Get(context.Background(), nodeName, k8smetav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	old, err := json.Marshal(node)
	Expect(err).ToNot(HaveOccurred())
	new := node.DeepCopy()
	new.Spec.Taints = append(new.Spec.Taints, k8sv1.Taint{
		Key:    key,
		Effect: effect,
	})

	newJson, err := json.Marshal(new)
	Expect(err).ToNot(HaveOccurred())

	patch, err := strategicpatch.CreateTwoWayMergePatch(old, newJson, node)
	Expect(err).ToNot(HaveOccurred())

	_, err = virtCli.CoreV1().Nodes().Patch(context.Background(), node.Name, types.StrategicMergePatchType, patch, k8smetav1.PatchOptions{})
	Expect(err).ToNot(HaveOccurred())
}

func GetNodesWithKVM() []*k8sv1.Node {
	virtClient, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)
	listOptions := k8smetav1.ListOptions{LabelSelector: v1.AppLabel + "=virt-handler"}
	virtHandlerPods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), listOptions)
	Expect(err).ToNot(HaveOccurred())

	nodes := make([]*k8sv1.Node, 0)
	// cluster is not ready until all nodes are ready.
	for _, pod := range virtHandlerPods.Items {
		virtHandlerNode, err := virtClient.CoreV1().Nodes().Get(context.Background(), pod.Spec.NodeName, k8smetav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		_, ok := virtHandlerNode.Status.Allocatable[services.KvmDevice]
		if ok {
			nodes = append(nodes, virtHandlerNode)
		}
	}
	return nodes
}

func GetAllSchedulableNodes(virtClient kubecli.KubevirtClient) *k8sv1.NodeList {
	nodes, err := virtClient.CoreV1().Nodes().List(context.Background(), k8smetav1.ListOptions{LabelSelector: v1.NodeSchedulable + "=" + "true"})
	Expect(err).ToNot(HaveOccurred(), "Should list compute nodes")
	return nodes
}

func GetHighestCPUNumberAmongNodes(virtClient kubecli.KubevirtClient) int {
	var cpus int64

	nodes, err := virtClient.CoreV1().Nodes().List(context.Background(), k8smetav1.ListOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	for _, node := range nodes.Items {
		if v, ok := node.Status.Capacity[k8sv1.ResourceCPU]; ok && v.Value() > cpus {
			cpus = v.Value()
		}
	}

	return int(cpus)
}

func GetNodeWithHugepages(virtClient kubecli.KubevirtClient, hugepages k8sv1.ResourceName) *k8sv1.Node {
	nodes, err := virtClient.CoreV1().Nodes().List(context.Background(), k8smetav1.ListOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	for _, node := range nodes.Items {
		if v, ok := node.Status.Capacity[hugepages]; ok && !v.IsZero() {
			return &node
		}
	}
	return nil
}

func GetArch() string {
	virtCli, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)
	nodes := GetAllSchedulableNodes(virtCli).Items
	Expect(nodes).ToNot(BeEmpty(), "There should be some node")
	return nodes[0].Status.NodeInfo.Architecture
}

func setNodeSchedualability(nodeName string, virtCli kubecli.KubevirtClient, setSchedulable bool) {
	origNode, err := virtCli.CoreV1().Nodes().Get(context.Background(), nodeName, k8smetav1.GetOptions{})
	Expect(err).ShouldNot(HaveOccurred())

	nodeCopy := origNode.DeepCopy()
	nodeCopy.Spec.Unschedulable = !setSchedulable

	err = nodes.PatchNode(virtCli, origNode, nodeCopy)
	Expect(err).ShouldNot(HaveOccurred())

	Eventually(func() bool {
		patchedNode, err := virtCli.CoreV1().Nodes().Get(context.Background(), nodeName, k8smetav1.GetOptions{})
		Expect(err).ShouldNot(HaveOccurred())

		return patchedNode.Spec.Unschedulable
	}, 30*time.Second, time.Second).Should(Equal(!setSchedulable), fmt.Sprintf("node %s is expected to set to Unschedulable=%t, but it's set to %t", nodeName, !setSchedulable, setSchedulable))
}

func SetNodeUnschedulable(nodeName string, virtCli kubecli.KubevirtClient) {
	setNodeSchedualability(nodeName, virtCli, false)
}

func SetNodeSchedulable(nodeName string, virtCli kubecli.KubevirtClient) {
	setNodeSchedualability(nodeName, virtCli, true)
}
