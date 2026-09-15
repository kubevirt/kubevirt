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

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/controller"
	virtutil "kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
)

// Simulates virt-handler version skew by faking the
// kubevirt.io/virt-handler-image-hash node label, instead of a real upgrade.
var _ = Describe(SIG("Live migration and virt-handler version skew", Serial, func() {
	// never a real virt-handler image, so its hash can't match the real one
	const fakeOutdatedVirtHandlerImage = "registry.example.com/kubevirt-functest/fake-outdated-virt-handler"

	// pkg/virt-handler/vm.go: heartBeatInterval. Not configurable.
	const virtHandlerHeartbeatInterval = time.Minute

	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	newTestVMI := func() *v1.VirtualMachineInstance {
		return libvmifact.NewGuestless(libnet.WithMasqueradeNetworking())
	}

	realVirtHandlerHash := func(nodeName string) string {
		node, err := virtClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		hash := node.Labels[v1.VirtHandlerImageHashLabel]
		Expect(hash).ToNot(BeEmpty(),
			"node %s has no %s label; this cluster's virt-handler build may predate the version-skew guard",
			nodeName, v1.VirtHandlerImageHashLabel)
		return hash
	}

	// pinFakeOutdatedVirtHandlerLabel fakes nodeName's image-hash label right
	// after a heartbeat tick, and returns the deadline before which
	// virt-handler's next heartbeat could revert it (see
	// requireSafetyMarginBefore).
	pinFakeOutdatedVirtHandlerLabel := func(nodeName string) time.Time {
		realHash := realVirtHandlerHash(nodeName)
		fakeHash := virtutil.ImageHashLabelValue(fakeOutdatedVirtHandlerImage)
		Expect(fakeHash).ToNot(Equal(realHash), "fake and real virt-handler hashes unexpectedly collided")

		node, err := virtClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		priorBeat := node.Annotations[v1.VirtHandlerHeartbeat]

		By(fmt.Sprintf("waiting for a fresh virt-handler heartbeat on node %s", nodeName))
		Eventually(func() string {
			node, err := virtClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return node.Annotations[v1.VirtHandlerHeartbeat]
		}, virtHandlerHeartbeatInterval+90*time.Second, 2*time.Second).ShouldNot(Equal(priorBeat),
			"node %s never recorded a new virt-handler heartbeat", nodeName)
		deadline := time.Now().Add(virtHandlerHeartbeatInterval)

		By(fmt.Sprintf("faking node %s's virt-handler image hash label to simulate an outdated/pre-upgrade node", nodeName))
		libnode.AddLabelToNode(nodeName, v1.VirtHandlerImageHashLabel, fakeHash)
		DeferCleanup(func() {
			By(fmt.Sprintf("restoring node %s's real virt-handler image hash label", nodeName))
			libnode.AddLabelToNode(nodeName, v1.VirtHandlerImageHashLabel, realHash)
		})

		return deadline
	}

	// requireSafetyMarginBefore fails loudly if less than margin remains
	// before deadline, instead of silently racing virt-handler's heartbeat.
	requireSafetyMarginBefore := func(deadline time.Time, margin time.Duration, what string) {
		remaining := time.Until(deadline)
		ExpectWithOffset(1, remaining).To(BeNumerically(">", margin),
			"only %s left before virt-handler's next heartbeat could revert the fake label; not enough safety "+
				"margin to reliably %s. This means the test took longer than expected, not that the migration "+
				"guard is broken - investigate before re-running.", remaining, what)
	}

	targetPodHasVersionSkewAffinity := func(migration *v1.VirtualMachineInstanceMigration) bool {
		var targetPod *k8sv1.Pod
		EventuallyWithOffset(1, func() (*k8sv1.Pod, error) {
			var err error
			targetPod, err = libpod.GetTargetPodForMigration(migration)
			return targetPod, err
		}, 15*time.Second, 1*time.Second).ShouldNot(BeNil())

		if targetPod.Spec.Affinity == nil || targetPod.Spec.Affinity.NodeAffinity == nil ||
			targetPod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
			return false
		}
		for _, term := range targetPod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms {
			for _, expr := range term.MatchExpressions {
				if expr.Key == v1.VirtHandlerImageHashLabel {
					return true
				}
			}
		}
		return false
	}

	setNodesSchedulable := func(nodes []k8sv1.Node, schedulable bool) {
		for _, n := range nodes {
			if schedulable {
				libnode.SetNodeSchedulable(n.Name, virtClient)
			} else {
				libnode.SetNodeUnschedulable(n.Name, virtClient)
			}
		}
	}

	It("allows migrating a VMI off an outdated node, and between updated nodes, during a simulated rolling upgrade",
		decorators.RequiresThreeSchedulableNodes, func() {
			allNodes := libnode.GetAllSchedulableNodes(virtClient).Items
			Expect(len(allNodes)).To(BeNumerically(">=", 3), "this test requires at least 3 schedulable nodes")

			// use exactly 3 nodes; cordon any extras for the whole test
			nodes := allNodes[:3]
			extraNodes := allNodes[3:]
			outdatedNode := nodes[0].Name
			newNodeA := nodes[1].Name
			newNodeB := nodes[2].Name
			otherNodes := nodes[1:]

			if len(extraNodes) > 0 {
				By("cordoning nodes beyond the 3 this test uses")
				setNodesSchedulable(extraNodes, false)
				DeferCleanup(func() { setNodesSchedulable(extraNodes, true) })
			}

			By(fmt.Sprintf("cordoning every node except the simulated outdated node %s", outdatedNode))
			setNodesSchedulable(otherNodes, false)
			DeferCleanup(func() { setNodesSchedulable(otherNodes, true) })

			vmi := newTestVMI()
			By("starting the VMI; it can only land on the simulated outdated node")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, flags.StartupTimeoutSecondsSmall())
			Expect(vmi.Status.NodeName).To(Equal(outdatedNode))

			By("uncordoning the remaining nodes so migrations have real freedom of choice")
			setNodesSchedulable(otherNodes, true)

			deadline := pinFakeOutdatedVirtHandlerLabel(outdatedNode)
			requireSafetyMarginBefore(deadline, 20*time.Second, "create the first migration and inspect its target pod")

			By("migrating off the simulated outdated node; this must be allowed")
			migration1 := libmigration.New(vmi.Name, vmi.Namespace)
			createdMigration1 := libmigration.RunMigration(virtClient, migration1)

			By("verifying the guard recognized the simulated outdated source and did not restrict the target")
			Expect(targetPodHasVersionSkewAffinity(createdMigration1)).To(BeFalse(),
				"migrating off an outdated node must not be restricted by the version-skew guard")

			createdMigration1 = libmigration.ExpectMigrationToSucceed(virtClient, createdMigration1, flags.MigrationTimeout())
			vmi = libmigration.ConfirmVMIPostMigration(virtClient, vmi, createdMigration1)
			Expect(vmi.Status.NodeName).To(BeElementOf(newNodeA, newNodeB), "VMI should have landed on one of the updated nodes")

			firstNewNode := vmi.Status.NodeName
			secondNewNode := newNodeA
			if firstNewNode == newNodeA {
				secondNewNode = newNodeB
			}

			By("migrating again between two updated nodes; this must also be allowed")
			migration2 := libmigration.New(vmi.Name, vmi.Namespace)
			createdMigration2 := libmigration.RunMigration(virtClient, migration2)

			By("verifying the guard recognized the updated source and did restrict the target to updated nodes")
			Expect(targetPodHasVersionSkewAffinity(createdMigration2)).To(BeTrue(),
				"migrating off an updated node must be restricted by the version-skew guard to other updated nodes")

			createdMigration2 = libmigration.ExpectMigrationToSucceed(virtClient, createdMigration2, flags.MigrationTimeout())
			vmi = libmigration.ConfirmVMIPostMigration(virtClient, vmi, createdMigration2)
			Expect(vmi.Status.NodeName).To(Equal(secondNewNode),
				"with only one other updated node available, the VMI must have migrated there")
		})

	It("refuses to migrate a VMI from an updated node onto an outdated node",
		decorators.RequiresTwoSchedulableNodes, func() {
			allNodes := libnode.GetAllSchedulableNodes(virtClient).Items
			Expect(len(allNodes)).To(BeNumerically(">=", 2), "this test requires at least 2 schedulable nodes")

			// use exactly 2 nodes; cordon any extras for the whole test
			nodes := allNodes[:2]
			extraNodes := allNodes[2:]
			updatedNode := nodes[0].Name
			outdatedNode := nodes[1].Name

			if len(extraNodes) > 0 {
				By("cordoning nodes beyond the 2 this test uses")
				setNodesSchedulable(extraNodes, false)
				DeferCleanup(func() { setNodesSchedulable(extraNodes, true) })
			}

			By(fmt.Sprintf("cordoning the simulated outdated node %s so the VMI is forced to start on the updated node", outdatedNode))
			libnode.SetNodeUnschedulable(outdatedNode, virtClient)
			DeferCleanup(func() { libnode.SetNodeSchedulable(outdatedNode, virtClient) })

			vmi := newTestVMI()
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, flags.StartupTimeoutSecondsSmall())
			Expect(vmi.Status.NodeName).To(Equal(updatedNode))

			deadline := pinFakeOutdatedVirtHandlerLabel(outdatedNode)

			By("uncordoning the simulated outdated node so it is a schedulable, but version-mismatched, migration target")
			libnode.SetNodeSchedulable(outdatedNode, virtClient)

			requireSafetyMarginBefore(deadline, 45*time.Second, "observe the migration get stuck in Scheduling")

			By("attempting to migrate from the updated node; with only the outdated node available, this must not be allowed to complete")
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			createdMigration := libmigration.RunMigration(virtClient, migration)

			Eventually(matcher.ThisMigration(createdMigration), 15*time.Second, 1*time.Second).Should(matcher.BeInPhase(v1.MigrationScheduling))

			By("verifying the target pod is Unschedulable because of the version-skew guard's node affinity")
			var scheduledCond *k8sv1.PodCondition
			Eventually(func() *k8sv1.PodCondition {
				targetPod, err := libpod.GetTargetPodForMigration(createdMigration)
				Expect(err).ToNot(HaveOccurred())
				scheduledCond = controller.NewPodConditionManager().GetCondition(targetPod, k8sv1.PodScheduled)
				return scheduledCond
			}, 10*time.Second, 1*time.Second).ShouldNot(BeNil(), "PodScheduled condition should not be nil")
			Expect(scheduledCond.Status).To(BeEquivalentTo(k8sv1.ConditionFalse), "PodScheduled status should be False")
			Expect(scheduledCond.Reason).To(BeEquivalentTo(k8sv1.PodReasonUnschedulable), "PodScheduled reason should be Unschedulable")
			Expect(scheduledCond.Message).To(ContainSubstring("node(s) didn't match Pod's node affinity/selector"), "PodScheduled message mismatch")

			requireSafetyMarginBefore(deadline, 25*time.Second, "hold the final Consistently check")

			By("confirming the migration never makes progress while the target remains version-mismatched")
			Consistently(matcher.ThisMigration(createdMigration), 25*time.Second, 2*time.Second).Should(matcher.BeInPhase(v1.MigrationScheduling))

			vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Status.NodeName).To(Equal(updatedNode), "the VMI must never have actually migrated")
		})
}))
