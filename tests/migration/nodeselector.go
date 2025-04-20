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

package migration

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/cleanup"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("Live Migration with addedNodeSelector", decorators.RequiresThreeSchedulableNodes, Serial, func() {
	var virtClient kubecli.KubevirtClient
	var nodes *k8sv1.NodeList

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		Eventually(func() int {
			nodes = libnode.GetAllSchedulableNodes(virtClient)
			return len(nodes.Items)
		}, 60*time.Second, 1*time.Second).Should(BeNumerically(">=", 3), "There should be at lest three compute nodes")
	})

	It("Should successfully migrate a VM to a labelled node", func() {
		zoneLabelKey := fmt.Sprintf("%s/%s", cleanup.KubeVirtTestLabelPrefix, "zone")
		vmiLabelValue := "vmi"
		brokenLabelValue := "broken"

		By("labelling all the nodes")
		for _, node := range nodes.Items {
			libnode.AddLabelToNode(node.Name, zoneLabelKey, vmiLabelValue)
		}

		By("starting a VM (with a node selector) on the source node")
		vmi := libvmifact.NewFedora(
			libnet.WithMasqueradeNetworking(),
		)
		vmi.Spec.NodeSelector = map[string]string{zoneLabelKey: vmiLabelValue}

		vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		vmi = libwait.WaitForSuccessfulVMIStart(vmi, libwait.WithTimeout(180))

		sourceNodeName := vmi.Status.NodeName
		var targetNodeName string

		By("labeling a target node")
		for _, node := range nodes.Items {
			if node.Name != sourceNodeName {
				targetNodeName = node.Name
				libnode.AddLabelToNode(node.Name, cleanup.TestLabelForNamespace(vmi.Namespace), "target")
				break
			}
		}
		Expect(targetNodeName).ToNot(BeEmpty(), "There should be a labeled target node")

		By("Checking nodeSelector on the VMI")
		Expect(vmi.Spec.NodeSelector).ToNot(HaveKeyWithValue(cleanup.TestLabelForNamespace(vmi.Namespace), "target"))

		By("Checking nodeSelector on virt-launcher pod")
		virtLauncherPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
		Expect(err).NotTo(HaveOccurred())
		Expect(virtLauncherPod.Spec.NodeSelector).ToNot(HaveKeyWithValue(cleanup.TestLabelForNamespace(vmi.Namespace), "target"))

		By("Starting the migration to the labeled node")
		migration := libmigration.New(vmi.Name, vmi.Namespace)
		migration.Spec.AddedNodeSelector = map[string]string{cleanup.TestLabelForNamespace(vmi.Namespace): "target"}
		By("by trying to override a selector set on the VMI")
		migration.Spec.AddedNodeSelector[zoneLabelKey] = brokenLabelValue
		libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

		By("Checking that the VMI landed on the target node")
		vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.GetName(), metav1.GetOptions{})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(vmi.Status.NodeName).To(Equal(targetNodeName))

		By("Checking nodeSelector on the VMI")
		Expect(vmi.Spec.NodeSelector).ToNot(HaveKeyWithValue(cleanup.TestLabelForNamespace(vmi.Namespace), "target"))
		Expect(vmi.Spec.NodeSelector).To(HaveKeyWithValue(zoneLabelKey, vmiLabelValue))
		Expect(vmi.Spec.NodeSelector).ToNot(HaveKeyWithValue(zoneLabelKey, brokenLabelValue))

		By("Checking nodeSelector on virt-launcher pod")
		virtLauncherPod, err = libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
		Expect(err).NotTo(HaveOccurred())
		Expect(virtLauncherPod.Spec.NodeSelector).To(HaveKeyWithValue(cleanup.TestLabelForNamespace(vmi.Namespace), "target"))
		Expect(virtLauncherPod.Spec.NodeSelector).To(HaveKeyWithValue(zoneLabelKey, vmiLabelValue))
		Expect(virtLauncherPod.Spec.NodeSelector).ToNot(HaveKeyWithValue(zoneLabelKey, brokenLabelValue))

		By("Migrating again the VM without configuring a nodeselector")
		migration = libmigration.New(vmi.Name, vmi.Namespace)
		libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

		By("Checking nodeSelector on the VMI")
		vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.GetName(), metav1.GetOptions{})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(vmi.Spec.NodeSelector).ToNot(HaveKeyWithValue(cleanup.TestLabelForNamespace(vmi.Namespace), "target"))
		Expect(virtLauncherPod.Spec.NodeSelector).To(HaveKeyWithValue(zoneLabelKey, vmiLabelValue))

		By("Checking nodeSelector on virt-launcher pod")
		virtLauncherPod, err = libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
		Expect(err).NotTo(HaveOccurred())
		Expect(virtLauncherPod.Spec.NodeSelector).ToNot(HaveKeyWithValue(cleanup.TestLabelForNamespace(vmi.Namespace), "target"))
		Expect(virtLauncherPod.Spec.NodeSelector).To(HaveKeyWithValue(zoneLabelKey, vmiLabelValue))
	})

}))
