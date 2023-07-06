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

package tests_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests/libinfra"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	"kubevirt.io/kubevirt/tests/events"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/testsuite"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"

	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/util"

	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	nodelabellerutil "kubevirt.io/kubevirt/pkg/virt-handler/node-labeller/util"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/flags"
)

const (
	remoteCmdErrPattern = "failed running `%s` with stdout:\n %v \n stderr:\n %v \n err: \n %v \n"
)

var _ = Describe("[Serial][sig-compute]Infrastructure", Serial, decorators.SigCompute, func() {
	var (
		virtClient       kubecli.KubevirtClient
		aggregatorClient *aggregatorclient.Clientset
		err              error
	)
	BeforeEach(func() {
		virtClient = kubevirt.Client()

		if aggregatorClient == nil {
			config, err := kubecli.GetKubevirtClientConfig()
			if err != nil {
				panic(err)
			}

			aggregatorClient = aggregatorclient.NewForConfigOrDie(config)
		}
	})

	Describe("Start a VirtualMachineInstance", func() {
		Context("when the controller pod is not running and an election happens", func() {
			It("[test_id:4642]should succeed afterwards", func() {
				// This test needs at least 2 controller pods. Skip on single-replica.
				checks.SkipIfSingleReplica(virtClient)

				newLeaderPod := getNewLeaderPod(virtClient)
				Expect(newLeaderPod).NotTo(BeNil())

				// TODO: It can be race condition when newly deployed pod receive leadership, in this case we will need
				// to reduce Deployment replica before destroying the pod and to restore it after the test
				By("Destroying the leading controller pod")
				Eventually(func() string {
					leaderPodName := libinfra.GetLeader()

					Expect(virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).Delete(context.Background(), leaderPodName, metav1.DeleteOptions{})).To(Succeed())

					Eventually(libinfra.GetLeader, 30*time.Second, 5*time.Second).ShouldNot(Equal(leaderPodName))

					leaderPod, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).Get(context.Background(), libinfra.GetLeader(), metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					return leaderPod.Name
				}, 90*time.Second, 5*time.Second).Should(Equal(newLeaderPod.Name))

				Expect(matcher.ThisPod(newLeaderPod)()).To(matcher.HaveConditionTrue(k8sv1.PodReady))

				vmi := tests.NewRandomVMI()

				By("Starting a new VirtualMachineInstance")
				obj, err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(testsuite.GetTestNamespace(vmi)).Body(vmi).Do(context.Background()).Get()
				Expect(err).ToNot(HaveOccurred())
				vmiObj, ok := obj.(*v1.VirtualMachineInstance)
				Expect(ok).To(BeTrue(), "Object is not of type *v1.VirtualMachineInstance")
				libwait.WaitForSuccessfulVMIStart(vmiObj)
			})
		})

	})

	Describe("Node-labeller", func() {
		var nodesWithKVM []*k8sv1.Node
		var nonExistingCPUModelLabel = v1.CPUModelLabel + "someNonExistingCPUModel"
		type patch struct {
			Op    string            `json:"op"`
			Path  string            `json:"path"`
			Value map[string]string `json:"value"`
		}

		BeforeEach(func() {
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
			wakeNodeLabellerUp(virtClient)

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

				obsoleteModels := nodelabellerutil.DefaultObsoleteCPUModels

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
					for labelKey, _ := range node.Labels {
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
					for labelKey, _ := range node.Labels {
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

	Describe("virt-handler", func() {
		var (
			originalKubeVirt *v1.KubeVirt
			nodesToEnableKSM []*k8sv1.Node
		)
		type ksmTestFunc func() (*v1.KSMConfiguration, []*k8sv1.Node)

		getNodesWithKSMAvailable := func(virtCli kubecli.KubevirtClient) []*k8sv1.Node {
			nodes := libnode.GetAllSchedulableNodes(virtCli)

			nodesWithKSM := make([]*k8sv1.Node, 0)
			for _, node := range nodes.Items {
				command := []string{"cat", "/sys/kernel/mm/ksm/run"}
				_, err := tests.ExecuteCommandInVirtHandlerPod(node.Name, command)
				if err == nil {
					nodesWithKSM = append(nodesWithKSM, &node)
				}
			}
			return nodesWithKSM
		}

		BeforeEach(func() {
			nodesToEnableKSM = getNodesWithKSMAvailable(virtClient)
			if len(nodesToEnableKSM) == 0 {
				Fail("There isn't any node with KSM available")
			}
			originalKubeVirt = util.GetCurrentKv(virtClient)
		})

		AfterEach(func() {
			tests.UpdateKubeVirtConfigValueAndWait(originalKubeVirt.Spec.Configuration)
		})

		DescribeTable("should enable/disable ksm and add/remove annotation", decorators.KSMRequired, func(ksmConfigFun ksmTestFunc) {
			kvConfig := originalKubeVirt.Spec.Configuration.DeepCopy()
			ksmConfig, expectedEnabledNodes := ksmConfigFun()
			kvConfig.KSMConfiguration = ksmConfig
			tests.UpdateKubeVirtConfigValueAndWait(*kvConfig)
			By("Ensure ksm is enabled and annotation is added in the expected nodes")
			for _, node := range expectedEnabledNodes {
				Eventually(func() (string, error) {
					command := []string{"cat", "/sys/kernel/mm/ksm/run"}
					ksmValue, err := tests.ExecuteCommandInVirtHandlerPod(node.Name, command)
					if err != nil {
						return "", err
					}

					return ksmValue, nil
				}, 30*time.Second, 2*time.Second).Should(BeEquivalentTo("1\n"), fmt.Sprintf("KSM should be enabled in node %s", node.Name))

				Eventually(func() (bool, error) {
					node, err := virtClient.CoreV1().Nodes().Get(context.Background(), node.Name, metav1.GetOptions{})
					if err != nil {
						return false, err
					}
					_, found := node.GetAnnotations()[v1.KSMHandlerManagedAnnotation]
					return found, nil
				}, 30*time.Second, 2*time.Second).Should(BeTrue(), fmt.Sprintf("Node %s should have %s annotation", node.Name, v1.KSMHandlerManagedAnnotation))
			}

			tests.UpdateKubeVirtConfigValueAndWait(originalKubeVirt.Spec.Configuration)

			By("Ensure ksm is disabled and annotation is removed in the expected nodes")
			for _, node := range expectedEnabledNodes {
				Eventually(func() (string, error) {
					command := []string{"cat", "/sys/kernel/mm/ksm/run"}
					ksmValue, err := tests.ExecuteCommandInVirtHandlerPod(node.Name, command)
					if err != nil {
						return "", err
					}

					return ksmValue, nil
				}, 30*time.Second, 2*time.Second).Should(BeEquivalentTo("0\n"), fmt.Sprintf("KSM should be disabled in node %s", node.Name))

				Eventually(func() (bool, error) {
					node, err := virtClient.CoreV1().Nodes().Get(context.Background(), node.Name, metav1.GetOptions{})
					if err != nil {
						return false, err
					}
					_, found := node.GetAnnotations()[v1.KSMHandlerManagedAnnotation]
					return found, nil
				}, 30*time.Second, 2*time.Second).Should(BeFalse(), fmt.Sprintf("Annotation %s should be removed from the node %s", v1.KSMHandlerManagedAnnotation, node.Name))
			}
		},
			Entry("in specific nodes when the selector with MatchLabels matches the node label", func() (*v1.KSMConfiguration, []*k8sv1.Node) {
				return &v1.KSMConfiguration{
					NodeLabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"kubernetes.io/hostname": nodesToEnableKSM[0].Name,
						},
					},
				}, []*k8sv1.Node{nodesToEnableKSM[0]}
			}),
			Entry("in specific nodes when the selector with MatchExpressions matches the node label", func() (*v1.KSMConfiguration, []*k8sv1.Node) {
				return &v1.KSMConfiguration{
					NodeLabelSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "kubernetes.io/hostname",
								Operator: metav1.LabelSelectorOpIn,
								Values:   []string{nodesToEnableKSM[0].Name},
							},
						},
					},
				}, []*k8sv1.Node{nodesToEnableKSM[0]}
			}),
			Entry("in all the nodes when the selector is empty", func() (*v1.KSMConfiguration, []*k8sv1.Node) {
				return &v1.KSMConfiguration{
					NodeLabelSelector: &metav1.LabelSelector{},
				}, nodesToEnableKSM
			}),
		)
	})

	Describe("cluster profiler for pprof data aggregation", func() {
		Context("when ClusterProfiler feature gate", func() {
			It("is disabled it should prevent subresource access", func() {
				tests.DisableFeatureGate("ClusterProfiler")

				err := virtClient.ClusterProfiler().Start()
				Expect(err).To(HaveOccurred())

				err = virtClient.ClusterProfiler().Stop()
				Expect(err).To(HaveOccurred())

				_, err = virtClient.ClusterProfiler().Dump(&v1.ClusterProfilerRequest{})
				Expect(err).To(HaveOccurred())
			})
			It("is enabled it should allow subresource access", func() {
				tests.EnableFeatureGate("ClusterProfiler")

				err := virtClient.ClusterProfiler().Start()
				Expect(err).ToNot(HaveOccurred())

				err = virtClient.ClusterProfiler().Stop()
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.ClusterProfiler().Dump(&v1.ClusterProfilerRequest{})
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})

func getNewLeaderPod(virtClient kubecli.KubevirtClient) *k8sv1.Pod {
	labelSelector, err := labels.Parse(fmt.Sprint(v1.AppLabel + "=virt-controller"))
	util.PanicOnError(err)
	fieldSelector := fields.ParseSelectorOrDie("status.phase=" + string(k8sv1.PodRunning))
	controllerPods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(),
		metav1.ListOptions{LabelSelector: labelSelector.String(), FieldSelector: fieldSelector.String()})
	util.PanicOnError(err)
	leaderPodName := libinfra.GetLeader()
	for _, pod := range controllerPods.Items {
		if pod.Name != leaderPodName {
			return &pod
		}
	}
	return nil
}
