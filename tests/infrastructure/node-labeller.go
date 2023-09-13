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
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests/libinfra"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	"kubevirt.io/kubevirt/tests/events"
	"kubevirt.io/kubevirt/tests/testsuite"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/util"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/libvmi"

	nodelabellerutil "kubevirt.io/kubevirt/pkg/virt-handler/node-labeller/util"
	"kubevirt.io/kubevirt/tests"
)

var _ = DescribeInfra("Node-labeller", func() {

	var (
		virtClient               kubecli.KubevirtClient
		err                      error
		nodesWithKVM             []*k8sv1.Node
		nonExistingCPUModelLabel = v1.CPUModelLabel + "someNonExistingCPUModel"
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		nodesWithKVM = libnode.GetNodesWithKVM()
		if len(nodesWithKVM) == 0 {
			Skip("Skip testing with node-labeller, because there are no nodes with kvm")
		}
	})

	AfterEach(func() {
		nodesWithKVM = libnode.GetNodesWithKVM()

		for _, node := range nodesWithKVM {
			libnode.RemoveLabelFromNode(node.Name, nonExistingCPUModelLabel)
			libnode.RemoveAnnotationFromNode(node.Name, v1.LabellerSkipNodeAnnotation)
		}
		libinfra.WakeNodeLabellerUp(virtClient)

		for _, node := range nodesWithKVM {
			Eventually(func() error {
				node, err = virtClient.CoreV1().Nodes().Get(context.Background(), node.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				if _, exists := node.Labels[nonExistingCPUModelLabel]; exists {
					return fmt.Errorf("node %s is expected to not have label key %s", node.Name, nonExistingCPUModelLabel)
				}

				if _, exists := node.Annotations[v1.LabellerSkipNodeAnnotation]; exists {
					return fmt.Errorf("node %s is expected to not have annotation key %s", node.Name, v1.LabellerSkipNodeAnnotation)
				}

				return nil
			}, 30*time.Second, 2*time.Second).ShouldNot(HaveOccurred())
		}
	})

	expectNodeLabels := func(nodeName string, labelValidation func(map[string]string) (valid bool, errorMsg string)) {
		var errorMsg string

		EventuallyWithOffset(1, func() (isValid bool) {
			node, err := virtClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			isValid, errorMsg = labelValidation(node.Labels)

			return isValid
		}, 30*time.Second, 2*time.Second).Should(BeTrue(), errorMsg)
	}

	Context("basic labelling", func() {

		type patch struct {
			Op    string            `json:"op"`
			Path  string            `json:"path"`
			Value map[string]string `json:"value"`
		}

		It("skip node reconciliation when node has skip annotation", func() {

			for i, node := range nodesWithKVM {
				node.Labels[nonExistingCPUModelLabel] = "true"
				p := []patch{
					{
						Op:    "add",
						Path:  "/metadata/labels",
						Value: node.Labels,
					},
				}
				if i == 0 {
					node.Annotations[v1.LabellerSkipNodeAnnotation] = "true"

					p = append(p, patch{
						Op:    "add",
						Path:  "/metadata/annotations",
						Value: node.Annotations,
					})
				}
				payloadBytes, err := json.Marshal(p)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.CoreV1().Nodes().Patch(context.Background(), node.Name, types.JSONPatchType, payloadBytes, metav1.PatchOptions{})
				Expect(err).ToNot(HaveOccurred())
			}
			kvConfig := v1.KubeVirtConfiguration{ObsoleteCPUModels: map[string]bool{}}
			// trigger reconciliation
			tests.UpdateKubeVirtConfigValueAndWait(kvConfig)

			Eventually(func() bool {
				nodesWithKVM = libnode.GetNodesWithKVM()

				for _, node := range nodesWithKVM {
					_, skipAnnotationFound := node.Annotations[v1.LabellerSkipNodeAnnotation]
					_, customLabelFound := node.Labels[nonExistingCPUModelLabel]
					if customLabelFound && !skipAnnotationFound {
						return false
					}
				}
				return true
			}, 15*time.Second, 1*time.Second).Should(BeTrue())
		})

		It("[test_id:6246] label nodes with cpu model, cpu features and host cpu model", func() {
			for _, node := range nodesWithKVM {
				Expect(err).ToNot(HaveOccurred())
				cpuModelLabelPresent := false
				cpuFeatureLabelPresent := false
				hyperVLabelPresent := false
				hostCpuModelPresent := false
				hostCpuRequiredFeaturesPresent := false
				for key := range node.Labels {
					if strings.Contains(key, v1.CPUModelLabel) {
						cpuModelLabelPresent = true
					}
					if strings.Contains(key, v1.CPUFeatureLabel) {
						cpuFeatureLabelPresent = true
					}
					if strings.Contains(key, v1.HypervLabel) {
						hyperVLabelPresent = true
					}
					if strings.Contains(key, v1.HostModelCPULabel) {
						hostCpuModelPresent = true
					}
					if strings.Contains(key, v1.HostModelRequiredFeaturesLabel) {
						hostCpuRequiredFeaturesPresent = true
					}

					if cpuModelLabelPresent && cpuFeatureLabelPresent && hyperVLabelPresent && hostCpuModelPresent &&
						hostCpuRequiredFeaturesPresent {
						break
					}
				}

				errorMessageTemplate := "node " + node.Name + " does not contain %s label"
				Expect(cpuModelLabelPresent).To(BeTrue(), fmt.Sprintf(errorMessageTemplate, "cpu"))
				Expect(cpuFeatureLabelPresent).To(BeTrue(), fmt.Sprintf(errorMessageTemplate, "feature"))
				Expect(hyperVLabelPresent).To(BeTrue(), fmt.Sprintf(errorMessageTemplate, "hyperV"))
				Expect(hostCpuModelPresent).To(BeTrue(), fmt.Sprintf(errorMessageTemplate, "host cpu model"))
				Expect(hostCpuRequiredFeaturesPresent).To(BeTrue(), fmt.Sprintf(errorMessageTemplate, "host cpu required featuers"))
			}
		})

		It("[test_id:6247] should set default obsolete cpu models filter when obsolete-cpus-models is not set in kubevirt config", func() {
			node := nodesWithKVM[0]

			for key := range node.Labels {
				if strings.Contains(key, v1.CPUModelLabel) {
					model := strings.TrimPrefix(key, v1.CPUModelLabel)
					Expect(nodelabellerutil.DefaultObsoleteCPUModels).ToNot(HaveKey(model),
						"Node can't contain label with cpu model, which is in default obsolete filter")
				}
			}
		})

		It("[test_id:6995]should expose tsc frequency and tsc scalability", func() {
			node := nodesWithKVM[0]
			Expect(node.Labels).To(HaveKey("cpu-timer.node.kubevirt.io/tsc-frequency"))
			Expect(node.Labels).To(HaveKey("cpu-timer.node.kubevirt.io/tsc-scalable"))
			Expect(node.Labels["cpu-timer.node.kubevirt.io/tsc-scalable"]).To(Or(Equal("true"), Equal("false")))
			val, err := strconv.ParseInt(node.Labels["cpu-timer.node.kubevirt.io/tsc-frequency"], 10, 64)
			Expect(err).ToNot(HaveOccurred())
			Expect(val).To(BeNumerically(">", 0))
		})
	})

	Context("advanced labelling", func() {
		var originalKubeVirt *v1.KubeVirt

		BeforeEach(func() {
			originalKubeVirt = util.GetCurrentKv(virtClient)
		})

		AfterEach(func() {
			tests.UpdateKubeVirtConfigValueAndWait(originalKubeVirt.Spec.Configuration)
		})

		It("[test_id:6249] should update node with new cpu model label set", func() {
			obsoleteModel := ""
			node := nodesWithKVM[0]

			kvConfig := originalKubeVirt.Spec.Configuration.DeepCopy()
			kvConfig.ObsoleteCPUModels = make(map[string]bool)

			for key := range node.Labels {
				if strings.Contains(key, v1.CPUModelLabel) {
					obsoleteModel = strings.TrimPrefix(key, v1.CPUModelLabel)
					kvConfig.ObsoleteCPUModels[obsoleteModel] = true
					break
				}
			}

			tests.UpdateKubeVirtConfigValueAndWait(*kvConfig)

			labelKeyExpectedToBeMissing := v1.CPUModelLabel + obsoleteModel
			expectNodeLabels(node.Name, func(m map[string]string) (valid bool, errorMsg string) {
				_, exists := m[labelKeyExpectedToBeMissing]
				return !exists, fmt.Sprintf("node %s is expected to not have label key %s", node.Name, labelKeyExpectedToBeMissing)
			})
		})

		It("[test_id:6250] should update node with new cpu model vendor label", func() {
			nodes, err := virtClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			for _, node := range nodes.Items {
				for key := range node.Labels {
					if strings.HasPrefix(key, v1.CPUModelVendorLabel) {
						return
					}
				}
			}

			Fail("No node contains label " + v1.CPUModelVendorLabel)
		})

		It("[test_id:6252] should remove all cpu model labels (all cpu model are in obsolete list)", func() {
			node := nodesWithKVM[0]

			obsoleteModels := map[string]bool{}
			for k, v := range nodelabellerutil.DefaultObsoleteCPUModels {
				obsoleteModels[k] = v
			}

			for key := range node.Labels {
				if strings.Contains(key, v1.CPUModelLabel) {
					obsoleteModels[strings.TrimPrefix(key, v1.CPUModelLabel)] = true
				}
				if strings.Contains(key, v1.SupportedHostModelMigrationCPU) {
					obsoleteModels[strings.TrimPrefix(key, v1.SupportedHostModelMigrationCPU)] = true
				}
			}

			kvConfig := originalKubeVirt.Spec.Configuration.DeepCopy()
			kvConfig.ObsoleteCPUModels = obsoleteModels
			tests.UpdateKubeVirtConfigValueAndWait(*kvConfig)

			expectNodeLabels(node.Name, func(m map[string]string) (valid bool, errorMsg string) {
				found := false
				label := ""
				for key := range m {
					if strings.Contains(key, v1.CPUModelLabel) || strings.Contains(key, v1.SupportedHostModelMigrationCPU) {
						found = true
						label = key
						break
					}
				}

				return !found, fmt.Sprintf("node %s should not contain any cpu model label, but contains %s", node.Name, label)
			})
		})
	})

	Context("[Serial]node with obsolete host-model cpuModel", Serial, func() {

		var node *k8sv1.Node
		var obsoleteModel string
		var kvConfig *v1.KubeVirtConfiguration

		BeforeEach(func() {
			node = &(libnode.GetAllSchedulableNodes(virtClient).Items[0])
			obsoleteModel = tests.GetNodeHostModel(node)

			By("Updating Kubevirt CR , this should wake node-labeller ")
			kvConfig = util.GetCurrentKv(virtClient).Spec.Configuration.DeepCopy()
			if kvConfig.ObsoleteCPUModels == nil {
				kvConfig.ObsoleteCPUModels = make(map[string]bool)
			}
			kvConfig.ObsoleteCPUModels[obsoleteModel] = true
			tests.UpdateKubeVirtConfigValueAndWait(*kvConfig)

			Eventually(func() error {
				node, err = virtClient.CoreV1().Nodes().Get(context.Background(), node.Name, metav1.GetOptions{})
				Expect(err).ShouldNot(HaveOccurred())

				_, exists := node.Annotations[v1.LabellerSkipNodeAnnotation]
				if exists {
					return fmt.Errorf("node %s is expected to not have annotation %s", node.Name, v1.LabellerSkipNodeAnnotation)
				}

				obsoleteModelLabelFound := false
				for labelKey := range node.Labels {
					if strings.Contains(labelKey, v1.NodeHostModelIsObsoleteLabel) {
						obsoleteModelLabelFound = true
						break
					}
				}
				if !obsoleteModelLabelFound {
					return fmt.Errorf("node %s is expected to have a label with %s substring. this means node-labeller is not enabled for the node", node.Name, v1.NodeHostModelIsObsoleteLabel)
				}

				return nil
			}, 30*time.Second, time.Second).ShouldNot(HaveOccurred())
		})

		AfterEach(func() {
			delete(kvConfig.ObsoleteCPUModels, obsoleteModel)
			tests.UpdateKubeVirtConfigValueAndWait(*kvConfig)

			Eventually(func() error {
				node, err = virtClient.CoreV1().Nodes().Get(context.Background(), node.Name, metav1.GetOptions{})
				Expect(err).ShouldNot(HaveOccurred())

				obsoleteHostModelLabel := false
				for labelKey := range node.Labels {
					if strings.HasPrefix(labelKey, v1.NodeHostModelIsObsoleteLabel) {
						obsoleteHostModelLabel = true
						break
					}
				}
				if obsoleteHostModelLabel {
					return fmt.Errorf("node %s is expected to have a label with %s prefix. this means node-labeller is not enabled for the node", node.Name, v1.HostModelCPULabel)
				}

				return nil
			}, 30*time.Second, time.Second).ShouldNot(HaveOccurred())
		})

		It("[Serial]should not schedule vmi with host-model cpuModel to node with obsolete host-model cpuModel", func() {
			vmi := libvmi.NewFedora(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)
			By("Making sure the vmi start running on the source node and will be able to run only in source/target nodes")
			vmi.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": node.Name}

			By("Starting the VirtualMachineInstance")
			_, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred())

			By("Checking that the VMI failed")
			Eventually(func() bool {
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				for _, condition := range vmi.Status.Conditions {
					if condition.Type == v1.VirtualMachineInstanceConditionType(k8sv1.PodScheduled) && condition.Status == k8sv1.ConditionFalse {
						return strings.Contains(condition.Message, "didn't match Pod's node affinity/selector")
					}
				}
				return false
			}, 3*time.Minute, 2*time.Second).Should(BeTrue())

			events.ExpectEvent(node, k8sv1.EventTypeWarning, "HostModelIsObsolete")
			// Remove as Node is persistent
			events.DeleteEvents(node, k8sv1.EventTypeWarning, "HostModelIsObsolete")
		})

	})

	Context("Clean up after old labeller", func() {
		nfdLabel := "feature.node.kubernetes.io/some-fancy-feature-which-should-not-be-deleted"
		var originalKubeVirt *v1.KubeVirt

		BeforeEach(func() {
			originalKubeVirt = util.GetCurrentKv(virtClient)

		})

		AfterEach(func() {
			originalNode, err := virtClient.CoreV1().Nodes().Get(context.Background(), nodesWithKVM[0].Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			node := originalNode.DeepCopy()

			for key := range node.Labels {
				if strings.Contains(key, nfdLabel) {
					delete(node.Labels, nfdLabel)
				}
			}
			originalLabelsBytes, err := json.Marshal(originalNode.Labels)
			Expect(err).ToNot(HaveOccurred())

			labelsBytes, err := json.Marshal(node.Labels)
			Expect(err).ToNot(HaveOccurred())

			patchTestLabels := fmt.Sprintf(`{ "op": "test", "path": "/metadata/labels", "value": %s}`, string(originalLabelsBytes))
			patchLabels := fmt.Sprintf(`{ "op": "replace", "path": "/metadata/labels", "value": %s}`, string(labelsBytes))

			data := []byte(fmt.Sprintf("[ %s, %s ]", patchTestLabels, patchLabels))

			_, err = virtClient.CoreV1().Nodes().Patch(context.Background(), nodesWithKVM[0].Name, types.JSONPatchType, data, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:6253] should remove old labeller labels and annotations", func() {
			originalNode, err := virtClient.CoreV1().Nodes().Get(context.Background(), nodesWithKVM[0].Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			node := originalNode.DeepCopy()

			node.Labels[nodelabellerutil.DeprecatedLabelNamespace+nodelabellerutil.DeprecatedcpuModelPrefix+"Penryn"] = "true"
			node.Labels[nodelabellerutil.DeprecatedLabelNamespace+nodelabellerutil.DeprecatedcpuFeaturePrefix+"mmx"] = "true"
			node.Labels[nodelabellerutil.DeprecatedLabelNamespace+nodelabellerutil.DeprecatedHyperPrefix+"synic"] = "true"
			node.Labels[nfdLabel] = "true"
			node.Annotations[nodelabellerutil.DeprecatedLabellerNamespaceAnnotation+nodelabellerutil.DeprecatedcpuModelPrefix+"Penryn"] = "true"
			node.Annotations[nodelabellerutil.DeprecatedLabellerNamespaceAnnotation+nodelabellerutil.DeprecatedcpuFeaturePrefix+"mmx"] = "true"
			node.Annotations[nodelabellerutil.DeprecatedLabellerNamespaceAnnotation+nodelabellerutil.DeprecatedHyperPrefix+"synic"] = "true"

			originalLabelsBytes, err := json.Marshal(originalNode.Labels)
			Expect(err).ToNot(HaveOccurred())

			originalAnnotationsBytes, err := json.Marshal(originalNode.Annotations)
			Expect(err).ToNot(HaveOccurred())

			labelsBytes, err := json.Marshal(node.Labels)
			Expect(err).ToNot(HaveOccurred())

			annotationsBytes, err := json.Marshal(node.Annotations)
			Expect(err).ToNot(HaveOccurred())

			patchTestLabels := fmt.Sprintf(`{ "op": "test", "path": "/metadata/labels", "value": %s}`, string(originalLabelsBytes))
			patchTestAnnotations := fmt.Sprintf(`{ "op": "test", "path": "/metadata/annotations", "value": %s}`, string(originalAnnotationsBytes))
			patchLabels := fmt.Sprintf(`{ "op": "replace", "path": "/metadata/labels", "value": %s}`, string(labelsBytes))
			patchAnnotations := fmt.Sprintf(`{ "op": "replace", "path": "/metadata/annotations", "value": %s}`, string(annotationsBytes))

			data := []byte(fmt.Sprintf("[ %s, %s, %s, %s ]", patchTestLabels, patchLabels, patchTestAnnotations, patchAnnotations))

			_, err = virtClient.CoreV1().Nodes().Patch(context.Background(), nodesWithKVM[0].Name, types.JSONPatchType, data, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())
			kvConfig := originalKubeVirt.Spec.Configuration.DeepCopy()
			kvConfig.ObsoleteCPUModels = map[string]bool{"486": true}
			tests.UpdateKubeVirtConfigValueAndWait(*kvConfig)

			expectNodeLabels(node.Name, func(m map[string]string) (valid bool, errorMsg string) {
				foundSpecialLabel := false

				for key := range m {
					for _, deprecatedPrefix := range []string{nodelabellerutil.DeprecatedcpuModelPrefix, nodelabellerutil.DeprecatedcpuFeaturePrefix, nodelabellerutil.DeprecatedHyperPrefix} {
						fullDeprecationLabel := nodelabellerutil.DeprecatedLabelNamespace + deprecatedPrefix
						if strings.Contains(key, fullDeprecationLabel) {
							return false, fmt.Sprintf("node %s should not contain any label with prefix %s", node.Name, fullDeprecationLabel)
						}
					}

					if key == nfdLabel {
						foundSpecialLabel = true
					}
				}

				if !foundSpecialLabel {
					return false, "labeller should not delete NFD labels"
				}

				return true, ""
			})

			Eventually(func() error {
				node, err = virtClient.CoreV1().Nodes().Get(context.Background(), nodesWithKVM[0].Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				for key := range node.Annotations {
					if strings.Contains(key, nodelabellerutil.DeprecatedLabellerNamespaceAnnotation) {
						return fmt.Errorf("node %s shouldn't contain any annotations with prefix %s, but found annotation key %s", node.Name, nodelabellerutil.DeprecatedLabellerNamespaceAnnotation, key)
					}
				}

				return nil
			}, 30*time.Second, 2*time.Second).ShouldNot(HaveOccurred())
		})

	})
})
