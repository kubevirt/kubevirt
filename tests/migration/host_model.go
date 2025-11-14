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
 * Copyright The KubeVirt Authors
 *
 */

package migration

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/events"
	"kubevirt.io/kubevirt/tests/framework/k8s"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libinfra"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
)

var _ = Describe(SIG("VM Live Migration", decorators.RequiresTwoSchedulableNodes, func() {
	Context("with a host-model cpu", func() {
		It("[test_id:6981]should migrate only to nodes supporting right cpu model", func() {
			sourceNode, targetNode, err := libmigration.GetValidSourceNodeAndTargetNodeForHostModelMigration(k8s.Client())
			if err != nil {
				Skip(err.Error())
			}

			By("Creating a VMI with default CPU mode to land in source node")
			vmi := libvmifact.NewAlpine(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithEvictionStrategy(v1.EvictionStrategyLiveMigrate),
				libvmi.WithCPUModel(v1.CPUModeHostModel),
			)
			By("Making sure the vmi start running on the source node and will be able to run only in source/target nodes")
			nodeAffinityRule, err := libmigration.CreateNodeAffinityRuleToMigrateFromSourceToTargetAndBack(sourceNode, targetNode)
			Expect(err).ToNot(HaveOccurred())
			vmi.Spec.Affinity = &k8sv1.Affinity{
				NodeAffinity: nodeAffinityRule,
			}

			By("Starting the VirtualMachineInstance")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsHuge)
			Expect(vmi.Spec.Domain.CPU.Model).To(Equal(v1.CPUModeHostModel))

			By("Fetching original host CPU model & supported CPU features")
			originalNode, err := k8s.Client().CoreV1().Nodes().Get(context.Background(), vmi.Status.NodeName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			hostModel := getNodeHostModel(originalNode)
			requiredFeatures := getNodeHostRequiredFeatures(originalNode)

			By("Starting the migration and expecting it to end successfully")
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			_ = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(kubevirt.Client(), migration)

			By("Ensuring that target pod has correct nodeSelector label")
			vmi, err = kubevirt.Client().VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			vmiPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(err).NotTo(HaveOccurred())

			Expect(vmiPod.Spec.NodeSelector).To(HaveKey(v1.SupportedHostModelMigrationCPU+hostModel),
				"target pod is expected to have correct nodeSelector label defined")

			By("Ensuring that target node has correct CPU mode & features")
			newNode, err := k8s.Client().CoreV1().Nodes().Get(context.Background(), vmi.Status.NodeName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(isModelSupportedOnNode(newNode, hostModel)).To(BeTrue(), "original host model should be supported on new node")
			expectFeatureToBeSupportedOnNode(newNode, requiredFeatures)
		})

		Context("Should trigger event if vmi with host-model start on source node with uniq host-model", Serial, func() {
			const fakeHostModelLabel = v1.HostModelCPULabel + "fake-model"
			var (
				vmi  *v1.VirtualMachineInstance
				node *k8sv1.Node
			)

			BeforeEach(func() {
				var err error
				By("Creating a VMI with default CPU mode")
				vmi = alpineVMIWithEvictionStrategy()
				vmi.Spec.Domain.CPU = &v1.CPU{Model: v1.CPUModeHostModel}

				By("Starting the VirtualMachineInstance")
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsHuge)

				By("Saving the original node's state")
				node, err = k8s.Client().CoreV1().Nodes().Get(context.Background(), vmi.Status.NodeName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				node = libinfra.ExpectStoppingNodeLabellerToSucceed(node.Name, k8s.Client())
			})

			AfterEach(func() {
				By("Resuming node labeller")
				node = libinfra.ExpectResumingNodeLabellerToSucceed(node.Name, kubevirt.Client(), k8s.Client())
				_, doesFakeHostLabelExists := node.Labels[fakeHostModelLabel]
				Expect(doesFakeHostLabelExists).To(BeFalse(), fmt.Sprintf("label %s is expected to disappear from node %s", fakeHostModelLabel, node.Name))
			})

			It("[test_id:7505]when no node is suited for host model", func() {
				By("Changing node labels to support fake host model")
				// Remove all supported host models
				for key := range node.Labels {
					if strings.HasPrefix(key, v1.HostModelCPULabel) {
						libnode.RemoveLabelFromNode(node.Name, key)
					}
				}
				node = libnode.AddLabelToNode(node.Name, fakeHostModelLabel, "true")

				Eventually(func() bool {
					var err error
					node, err = k8s.Client().CoreV1().Nodes().Get(context.Background(), node.Name, metav1.GetOptions{})
					Expect(err).ShouldNot(HaveOccurred())

					labelValue, ok := node.Labels[v1.HostModelCPULabel+"fake-model"]
					return ok && labelValue == "true"
				}, 10*time.Second, 1*time.Second).Should(BeTrue(), "Node should have fake host model")

				By("Starting the migration")
				migration := libmigration.New(vmi.Name, vmi.Namespace)
				_ = libmigration.RunMigration(kubevirt.Client(), migration)

				events.ExpectEvent(vmi, k8sv1.EventTypeWarning, controller.NoSuitableNodesForHostModelMigration)
			})
		})

		Context("Should trigger event if the nodes doesn't contain MigrationSelectorLabel for the vmi host-model type", Serial, func() {
			var (
				vmi   *v1.VirtualMachineInstance
				nodes []k8sv1.Node
			)

			BeforeEach(func() {
				nodes = libnode.GetAllSchedulableNodes(k8s.Client()).Items
				if len(nodes) == 1 || len(nodes) > 10 {
					Skip("This test can't run with single node and it's too slow to run with more than 10 nodes")
				}

				By("Creating a VMI with default CPU mode")
				vmi = alpineVMIWithEvictionStrategy()
				vmi.Spec.Domain.CPU = &v1.CPU{Model: v1.CPUModeHostModel}

				By("Starting the VirtualMachineInstance")
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsHuge)

				for index, node := range nodes {
					patchedNode := libinfra.ExpectStoppingNodeLabellerToSucceed(node.Name, k8s.Client())
					Expect(patchedNode).ToNot(BeNil())
					nodes[index] = *patchedNode
				}
			})

			AfterEach(func() {
				By("Restore node to its original state")
				for _, node := range nodes {
					updatedNode := libinfra.ExpectResumingNodeLabellerToSucceed(node.Name, kubevirt.Client(), k8s.Client())

					supportedHostModelLabelExists := false
					for labelKey := range updatedNode.Labels {
						if strings.HasPrefix(labelKey, v1.SupportedHostModelMigrationCPU) {
							supportedHostModelLabelExists = true
							break
						}
					}
					Expect(supportedHostModelLabelExists).To(BeTrue(), fmt.Sprintf("label with %s prefix is supposed to exist for node %s", v1.SupportedHostModelMigrationCPU, updatedNode.Name))
				}
			})

			It("no node contain suited SupportedHostModelMigrationCPU label", func() {
				By("Changing node labels to support fake host model")
				// Remove all supported host models
				for _, node := range nodes {
					currNode, err := k8s.Client().CoreV1().Nodes().Get(context.Background(), node.Name, metav1.GetOptions{})
					Expect(err).ShouldNot(HaveOccurred())
					for key := range currNode.Labels {
						if strings.HasPrefix(key, v1.SupportedHostModelMigrationCPU) {
							libnode.RemoveLabelFromNode(currNode.Name, key)
						}
					}
				}

				By("Starting the migration")
				migration := libmigration.New(vmi.Name, vmi.Namespace)
				_ = libmigration.RunMigration(kubevirt.Client(), migration)

				events.ExpectEvent(vmi, k8sv1.EventTypeWarning, controller.NoSuitableNodesForHostModelMigration)
			})
		})
	})

	Context("Testing host-model cpuModel edge cases in the cluster if the cluster is host-model migratable", Serial, func() {
		const (
			fakeRequiredFeature = v1.HostModelRequiredFeaturesLabel + "fakeFeature"
			fakeHostModel       = v1.HostModelCPULabel + "fakeHostModel"
		)
		var (
			sourceNode *k8sv1.Node
			targetNode *k8sv1.Node
		)

		BeforeEach(func() {
			var err error
			sourceNode, targetNode, err = libmigration.GetValidSourceNodeAndTargetNodeForHostModelMigration(k8s.Client())
			if err != nil {
				Skip(err.Error())
			}
			targetNode = libinfra.ExpectStoppingNodeLabellerToSucceed(targetNode.Name, k8s.Client())
		})

		AfterEach(func() {
			By("Resuming node labeller")
			targetNode = libinfra.ExpectResumingNodeLabellerToSucceed(targetNode.Name, kubevirt.Client(), k8s.Client())

			By("Validating that fake labels are being removed")
			for _, labelKey := range []string{fakeRequiredFeature, fakeHostModel} {
				_, fakeLabelExists := targetNode.Labels[labelKey]
				Expect(fakeLabelExists).To(BeFalse(), fmt.Sprintf("fake feature %s is expected to disappear form node %s", labelKey, targetNode.Name))
			}
		})

		It("Should be able to migrate back to the initial node from target node with host-model even if target is newer than source", func() {
			libnode.AddLabelToNode(targetNode.Name, fakeRequiredFeature, "true")

			vmiToMigrate := libvmifact.NewFedora(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)
			By("Creating a VMI with default CPU mode to land in source node")
			vmiToMigrate.Spec.Domain.CPU = &v1.CPU{Model: v1.CPUModeHostModel}
			By("Making sure the vmi start running on the source node and will be able to run only in source/target nodes")
			nodeAffinityRule, err := libmigration.CreateNodeAffinityRuleToMigrateFromSourceToTargetAndBack(sourceNode, targetNode)
			Expect(err).ToNot(HaveOccurred())
			vmiToMigrate.Spec.Affinity = &k8sv1.Affinity{
				NodeAffinity: nodeAffinityRule,
			}
			By("Starting the VirtualMachineInstance")
			vmiToMigrate = libvmops.RunVMIAndExpectLaunch(vmiToMigrate, libvmops.StartupTimeoutSecondsHuge)
			Expect(vmiToMigrate.Status.NodeName).To(Equal(sourceNode.Name))
			Expect(console.LoginToFedora(vmiToMigrate)).To(Succeed())

			// execute a migration, wait for finalized state
			By("Starting the Migration to target node(with the amazing feature")
			migration := libmigration.New(vmiToMigrate.Name, vmiToMigrate.Namespace)
			libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(kubevirt.Client(), migration)

			vmiToMigrate, err = kubevirt.Client().VirtualMachineInstance(vmiToMigrate.Namespace).Get(context.Background(), vmiToMigrate.GetName(), metav1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(vmiToMigrate.Status.NodeName).To(Equal(targetNode.Name))

			labelsBeforeMigration := make(map[string]string)
			labelsAfterMigration := make(map[string]string)
			By("Fetching virt-launcher pod")
			virtLauncherPod, err := libpod.GetPodByVirtualMachineInstance(vmiToMigrate, vmiToMigrate.Namespace)
			Expect(err).NotTo(HaveOccurred())
			for key, value := range virtLauncherPod.Spec.NodeSelector {
				if strings.HasPrefix(key, v1.CPUFeatureLabel) {
					labelsBeforeMigration[key] = value
				}
			}

			By("Starting the Migration to return to the source node")
			libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(kubevirt.Client(), migration)
			Expect(console.LoginToFedora(vmiToMigrate)).To(Succeed())

			vmiToMigrate, err = kubevirt.Client().VirtualMachineInstance(vmiToMigrate.Namespace).Get(context.Background(), vmiToMigrate.GetName(), metav1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(vmiToMigrate.Status.NodeName).To(Equal(sourceNode.Name))
			By("Fetching virt-launcher pod")
			virtLauncherPod, err = libpod.GetPodByVirtualMachineInstance(vmiToMigrate, vmiToMigrate.Namespace)
			Expect(err).NotTo(HaveOccurred())
			for key, value := range virtLauncherPod.Spec.NodeSelector {
				if strings.HasPrefix(key, v1.CPUFeatureLabel) {
					labelsAfterMigration[key] = value
				}
			}
			Expect(labelsAfterMigration).To(BeEquivalentTo(labelsBeforeMigration))
		})

		It("vmi with host-model should be able to migrate to node that support the initial node's host-model even if this model isn't the target's host-model", func() {
			var err error
			targetNode, err = k8s.Client().CoreV1().Nodes().Get(context.Background(), targetNode.Name, metav1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())

			targetHostModel := libnode.GetNodeHostModel(targetNode)
			targetNode = libnode.RemoveLabelFromNode(targetNode.Name, v1.HostModelCPULabel+targetHostModel)
			targetNode = libnode.AddLabelToNode(targetNode.Name, fakeHostModel, "true")

			vmiToMigrate := libvmifact.NewFedora(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)
			By("Creating a VMI with default CPU mode to land in source node")
			vmiToMigrate.Spec.Domain.CPU = &v1.CPU{Model: v1.CPUModeHostModel}
			By("Making sure the vmi start running on the source node and will be able to run only in source/target nodes")
			nodeAffinityRule, err := libmigration.CreateNodeAffinityRuleToMigrateFromSourceToTargetAndBack(sourceNode, targetNode)
			Expect(err).ToNot(HaveOccurred())
			vmiToMigrate.Spec.Affinity = &k8sv1.Affinity{
				NodeAffinity: nodeAffinityRule,
			}
			By("Starting the VirtualMachineInstance")
			vmiToMigrate = libvmops.RunVMIAndExpectLaunch(vmiToMigrate, libvmops.StartupTimeoutSecondsHuge)
			Expect(vmiToMigrate.Status.NodeName).To(Equal(sourceNode.Name))
			Expect(console.LoginToFedora(vmiToMigrate)).To(Succeed())

			// execute a migration, wait for finalized state
			By("Starting the Migration to target node(with the amazing feature")
			migration := libmigration.New(vmiToMigrate.Name, vmiToMigrate.Namespace)
			libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(kubevirt.Client(), migration)

			vmiToMigrate, err = kubevirt.Client().VirtualMachineInstance(vmiToMigrate.Namespace).Get(context.Background(), vmiToMigrate.GetName(), metav1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(vmiToMigrate.Status.NodeName).To(Equal(targetNode.Name))
			Expect(console.LoginToFedora(vmiToMigrate)).To(Succeed())
		})
	})
}))

func getNodeHostModel(node *k8sv1.Node) (hostModel string) {
	for key := range node.Labels {
		if strings.HasPrefix(key, v1.HostModelCPULabel) {
			hostModel = strings.TrimPrefix(key, v1.HostModelCPULabel)
			break
		}
	}
	Expect(hostModel).ToNot(BeEmpty(), "must find node's host model")
	return hostModel
}

func getNodeHostRequiredFeatures(node *k8sv1.Node) (features []string) {
	for key := range node.Labels {
		if strings.HasPrefix(key, v1.HostModelRequiredFeaturesLabel) {
			features = append(features, strings.TrimPrefix(key, v1.HostModelRequiredFeaturesLabel))
		}
	}
	return features
}

func isModelSupportedOnNode(node *k8sv1.Node, model string) bool {
	for key := range node.Labels {
		if strings.HasPrefix(key, v1.HostModelCPULabel) && strings.Contains(key, model) {
			return true
		}
	}
	return false
}

func isFeatureSupported(node *k8sv1.Node, feature string) bool {
	for key := range node.Labels {
		if strings.HasPrefix(key, v1.CPUFeatureLabel) && strings.Contains(key, feature) {
			return true
		}
	}
	return false
}

func expectFeatureToBeSupportedOnNode(node *k8sv1.Node, features []string) {
	supportedFeatures := make(map[string]bool)
	for _, feature := range features {
		supportedFeatures[feature] = isFeatureSupported(node, feature)
	}

	Expect(supportedFeatures).Should(Not(ContainElement(false)),
		"copy features must be supported on node")
}
