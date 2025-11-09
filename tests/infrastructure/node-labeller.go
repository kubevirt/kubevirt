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
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	nodelabellerutil "kubevirt.io/kubevirt/pkg/virt-handler/node-labeller/util"
	"kubevirt.io/kubevirt/tests/events"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libinfra"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIGSerial("Node-labeller", func() {
	const trueStr = "true"

	var (
		virtClient               kubecli.KubevirtClient
		nodesWithKVM             []*k8sv1.Node
		nonExistingCPUModelLabel = v1.CPUModelLabel + "someNonExistingCPUModel"
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		nodesWithKVM = libnode.GetNodesWithKVM()
		if len(nodesWithKVM) == 0 {
			Fail("No nodes with kvm")
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
				nodeObj, err := virtClient.CoreV1().Nodes().Get(context.Background(), node.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				if _, exists := nodeObj.Labels[nonExistingCPUModelLabel]; exists {
					return fmt.Errorf("node %s is expected to not have label key %s", node.Name, nonExistingCPUModelLabel)
				}

				if _, exists := nodeObj.Annotations[v1.LabellerSkipNodeAnnotation]; exists {
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

	Context("basic labeling", func() {
		type patch struct {
			Op    string            `json:"op"`
			Path  string            `json:"path"`
			Value map[string]string `json:"value"`
		}

		It("skip node reconciliation when node has skip annotation", func() {
			for i, node := range nodesWithKVM {
				node.Labels[nonExistingCPUModelLabel] = trueStr
				p := []patch{
					{
						Op:    "add",
						Path:  "/metadata/labels",
						Value: node.Labels,
					},
				}
				if i == 0 {
					node.Annotations[v1.LabellerSkipNodeAnnotation] = trueStr

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
			config.UpdateKubeVirtConfigValueAndWait(kvConfig)

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
				cpuModelLabelPresent := false
				cpuFeatureLabelPresent := false
				hyperVLabelPresent := false
				hostCPUModelPresent := false
				hostCPURequiredFeaturesPresent := false
				vendorPresent := false
				for key := range node.Labels {
					if strings.Contains(key, v1.CPUModelVendorLabel) {
						vendorPresent = true
					}
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
						hostCPUModelPresent = true
					}
					if strings.Contains(key, v1.HostModelRequiredFeaturesLabel) {
						hostCPURequiredFeaturesPresent = true
					}

					if cpuModelLabelPresent && cpuFeatureLabelPresent && hyperVLabelPresent && hostCPUModelPresent &&
						hostCPURequiredFeaturesPresent && vendorPresent {
						break
					}
				}

				errorMessageTemplate := "node " + node.Name + " does not contain %s label"
				Expect(cpuModelLabelPresent).To(BeTrue(), fmt.Sprintf(errorMessageTemplate, "cpu"))
				Expect(cpuFeatureLabelPresent).To(BeTrue(), fmt.Sprintf(errorMessageTemplate, "feature"))
				Expect(hyperVLabelPresent).To(BeTrue(), fmt.Sprintf(errorMessageTemplate, "hyperV"))
				Expect(hostCPUModelPresent).To(BeTrue(), fmt.Sprintf(errorMessageTemplate, "host cpu model"))
				Expect(hostCPURequiredFeaturesPresent).To(BeTrue(), fmt.Sprintf(errorMessageTemplate, "host cpu required features"))
				Expect(vendorPresent).To(BeTrue(), fmt.Sprintf(errorMessageTemplate, "vendor"))
			}
		})

		It("[test_id:6247] should set default obsolete cpu models filter when obsolete-cpus-models is not set in kubevirt config", func() {
			kvConfig := libkubevirt.GetCurrentKv(virtClient)
			kvConfig.Spec.Configuration.ObsoleteCPUModels = nil
			config.UpdateKubeVirtConfigValueAndWait(kvConfig.Spec.Configuration)
			node := nodesWithKVM[0]
			timeout := 30 * time.Second
			Eventually(func() error {
				nodeObj, err := virtClient.CoreV1().Nodes().Get(context.Background(), node.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				for key := range nodeObj.Labels {
					if strings.Contains(key, v1.CPUModelLabel) {
						model := strings.TrimPrefix(key, v1.CPUModelLabel)
						if _, ok := nodelabellerutil.DefaultObsoleteCPUModels[model]; ok {
							return fmt.Errorf("node can't contain label with cpu model, which is in default obsolete filter")
						}
					}
				}
				return nil
			}).WithTimeout(timeout).WithPolling(1 * time.Second).ShouldNot(HaveOccurred())
		})

		It("[test_id:6995]should expose tsc frequency and tsc scalability", func() {
			node := nodesWithKVM[0]
			Expect(node.Labels).To(HaveKey("cpu-timer.node.kubevirt.io/tsc-frequency"))
			Expect(node.Labels).To(HaveKey("cpu-timer.node.kubevirt.io/tsc-scalable"))
			Expect(node.Labels["cpu-timer.node.kubevirt.io/tsc-scalable"]).To(Or(Equal(trueStr), Equal("false")))
			val, err := strconv.ParseInt(node.Labels["cpu-timer.node.kubevirt.io/tsc-frequency"], 10, 64)
			Expect(err).ToNot(HaveOccurred())
			Expect(val).To(BeNumerically(">", 0))
		})
	})

	Context("advanced labeling", func() {
		var originalKubeVirt *v1.KubeVirt

		BeforeEach(func() {
			originalKubeVirt = libkubevirt.GetCurrentKv(virtClient)
		})

		AfterEach(func() {
			config.UpdateKubeVirtConfigValueAndWait(originalKubeVirt.Spec.Configuration)
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

			config.UpdateKubeVirtConfigValueAndWait(*kvConfig)

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
			config.UpdateKubeVirtConfigValueAndWait(*kvConfig)

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

	Context("node with obsolete host-model cpuModel", Serial, func() {
		var node *k8sv1.Node
		var obsoleteModel string
		var kvConfig *v1.KubeVirtConfiguration

		BeforeEach(func() {
			node = &(libnode.GetAllSchedulableNodes(virtClient).Items[0])
			obsoleteModel = libnode.GetNodeHostModel(node)

			By("Updating Kubevirt CR , this should wake node-labeller ")
			kvConfig = libkubevirt.GetCurrentKv(virtClient).Spec.Configuration.DeepCopy()
			if kvConfig.ObsoleteCPUModels == nil {
				kvConfig.ObsoleteCPUModels = make(map[string]bool)
			}
			kvConfig.ObsoleteCPUModels[obsoleteModel] = true
			config.UpdateKubeVirtConfigValueAndWait(*kvConfig)

			Eventually(func() error {
				nodeObj, err := virtClient.CoreV1().Nodes().Get(context.Background(), node.Name, metav1.GetOptions{})
				Expect(err).ShouldNot(HaveOccurred())

				_, exists := nodeObj.Annotations[v1.LabellerSkipNodeAnnotation]
				if exists {
					return fmt.Errorf("node %s is expected to not have annotation %s", node.Name, v1.LabellerSkipNodeAnnotation)
				}

				obsoleteModelLabelFound := false
				for labelKey := range nodeObj.Labels {
					if strings.Contains(labelKey, v1.NodeHostModelIsObsoleteLabel) {
						obsoleteModelLabelFound = true
						break
					}
				}
				if !obsoleteModelLabelFound {
					return fmt.Errorf(
						"node %s is expected to have a label with %s substring. node-labeller is not enabled for the node",
						node.Name, v1.NodeHostModelIsObsoleteLabel)
				}

				return nil
			}, 30*time.Second, time.Second).ShouldNot(HaveOccurred())
		})

		AfterEach(func() {
			delete(kvConfig.ObsoleteCPUModels, obsoleteModel)
			config.UpdateKubeVirtConfigValueAndWait(*kvConfig)

			Eventually(func() error {
				nodeObj, err := virtClient.CoreV1().Nodes().Get(context.Background(), node.Name, metav1.GetOptions{})
				Expect(err).ShouldNot(HaveOccurred())

				obsoleteHostModelLabel := false
				for labelKey := range nodeObj.Labels {
					if strings.HasPrefix(labelKey, v1.NodeHostModelIsObsoleteLabel) {
						obsoleteHostModelLabel = true
						break
					}
				}
				if obsoleteHostModelLabel {
					return fmt.Errorf(
						"node %s is expected to have a label with %s prefix. node-labeller is not enabled for the node",
						node.Name, v1.HostModelCPULabel)
				}

				return nil
			}, 30*time.Second, time.Second).ShouldNot(HaveOccurred())
		})

		It("should not schedule vmi with host-model cpuModel to node with obsolete host-model cpuModel", func() {
			vmi := libvmifact.NewFedora(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)
			By("Making sure the vmi start running on the source node and will be able to run only in source/target nodes")
			vmi.Spec.NodeSelector = map[string]string{k8sv1.LabelHostname: node.Name}

			By("Starting the VirtualMachineInstance")
			_, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Checking that the VMI failed")
			Eventually(func() bool {
				vmiObj, err := virtClient.VirtualMachineInstance(
					testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				for _, condition := range vmiObj.Status.Conditions {
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
}))
