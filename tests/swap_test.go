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

package tests_test

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/libwait"
)

const (
	//20% of the default size which is 2G in kubevirt-ci
	maxSwapSizeToUseKib = 415948
	swapPartToUse       = 0.2
	gigbytesInkib       = 1048576
	bytesInKib          = 1024
)

var _ = Describe("[sig-compute]SwapTest", decorators.RequiresTwoSchedulableNodes, Serial, decorators.SigCompute, decorators.Swap, func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()

		nodes := libnode.GetAllSchedulableNodes(virtClient)
		Expect(len(nodes.Items)).To(BeNumerically(">", 1),
			"should have at least two schedulable nodes in the cluster")

		failIfSwapOff(fmt.Sprintf("swap should be enabled through env var: KUBEVIRT_SWAP_ON=true "+
			"and contain at least %dMi in the nodes when running these tests", maxSwapSizeToUseKib/bytesInKib))

	})

	Context("Migration to/from memory overcommitted nodes", decorators.SigComputeMigrations, func() {
		It("Postcopy Migration of vmi that is dirtying(stress-ng) more memory than the source node's memory", func() {
			sourceNode, targetNode, err := libmigration.GetValidSourceNodeAndTargetNodeForHostModelMigration(virtClient)
			Expect(err).ToNot(HaveOccurred(), "should be able to get valid source and target nodes for migartion")
			totalMemKib := getTotalMemSizeInKib(*sourceNode)
			availableMemSizeKib := getAvailableMemSizeInKib(*sourceNode)
			availableSwapSizeKib := getSwapFreeSizeInKib(*sourceNode)
			swapSizeKib := getSwapSizeInKib(*sourceNode)
			Expect(availableSwapSizeKib).Should(BeNumerically(">", maxSwapSizeToUseKib), "not enough available swap space")

			swapSizeToUseKib := int(math.Min(maxSwapSizeToUseKib, swapPartToUse*float64(availableSwapSizeKib)))
			//use more memory than what node can handle without swap memory
			memToUseInTheVmKib := availableMemSizeKib + swapSizeToUseKib

			By("Allowing post-copy")
			kv := libkubevirt.GetCurrentKv(virtClient)
			kv.Spec.Configuration.MigrationConfiguration = &virtv1.MigrationConfiguration{
				AllowPostCopy:           pointer.P(true),
				CompletionTimeoutPerGiB: pointer.P(int64(1)),
			}
			config.UpdateKubeVirtConfigValueAndWait(kv.Spec.Configuration)

			//The vmi should have more memory than memToUseInTheVmKib
			vmiMemSizeMi := resource.MustParse(fmt.Sprintf("%dMi", int((float64(memToUseInTheVmKib)+float64(gigbytesInkib*2))/bytesInKib)))

			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())
			nodeAffinityRule, err := libmigration.CreateNodeAffinityRuleToMigrateFromSourceToTargetAndBack(sourceNode, targetNode)
			Expect(err).ToNot(HaveOccurred())
			vmi.Spec.Affinity = &v1.Affinity{
				NodeAffinity: nodeAffinityRule,
			}
			vmi.Spec.Domain.Resources.OvercommitGuestOverhead = true
			vmi.Spec.Domain.Memory = &virtv1.Memory{Guest: &vmiMemSizeMi}

			By("Starting the VirtualMachineInstance")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsHuge)
			Expect(console.LoginToFedora(vmi)).To(Succeed())
			By("Consume more memory than the node's memory")
			err = fillMemWithStressFedoraVMI(vmi, memToUseInTheVmKib)
			ExpectWithOffset(1, err).ToNot(HaveOccurred())

			By("The workloads in the node should consume more memory than the memory size eventually.")
			Eventually(func() int {
				usedMemoryWithOutSwap := totalMemKib - getAvailableMemSizeInKib(*sourceNode)
				usedSwapMemory := swapSizeKib - getSwapFreeSizeInKib(*sourceNode)
				return usedMemoryWithOutSwap + usedSwapMemory
			}, 240*time.Second, 1*time.Second).Should(BeNumerically(">", totalMemKib))

			// execute a migration, wait for finalized state
			By("Starting the Migration")
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			migration = libmigration.RunMigrationAndExpectToComplete(virtClient, migration, libmigration.MigrationWaitTime*2)

			// check VMI, confirm migration state
			libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
			libmigration.ConfirmMigrationMode(virtClient, vmi, virtv1.MigrationPostCopy)
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			By("Deleting the VMI")
			Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, metav1.DeleteOptions{})).To(Succeed())

			By("Waiting for VMI to disappear")
			libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, 240)
		})

		It("Migration of vmi to memory overcommited node", func() {
			sourceNode, targetNode, err := libmigration.GetValidSourceNodeAndTargetNodeForHostModelMigration(virtClient)
			Expect(err).ToNot(HaveOccurred(), "should be able to get valid source and target nodes for migartion")
			vmMemoryRequestkib := 512000
			availableMemSizeKib := getAvailableMemSizeInKib(*targetNode)
			availableSwapSizeKib := getSwapFreeSizeInKib(*targetNode)
			swapSizeToUsekib := int(math.Min(maxSwapSizeToUseKib, swapPartToUse*float64(availableSwapSizeKib)))

			//make sure that the vm migrate data to swap space (leave enough space for the vm that we will migrate)
			memToUseInTargetNodeVmKib := availableMemSizeKib + swapSizeToUsekib - vmMemoryRequestkib

			//The vmi should have more memory than memToUseInTheVm
			vmiMemSize := resource.MustParse(fmt.Sprintf("%dMi", int((float64(memToUseInTargetNodeVmKib)+float64(gigbytesInkib*2))/bytesInKib)))
			vmiMemReq := resource.MustParse(fmt.Sprintf("%dMi", vmMemoryRequestkib/bytesInKib))
			vmiToFillTargetNodeMem := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())
			//we want vmiToFillTargetNodeMem to land on the target node to achieve memory-overcommitment in target
			affinityRuleForVmiToFill, err := getAffinityForTargetNode(targetNode)
			Expect(err).ToNot(HaveOccurred())
			vmiToFillTargetNodeMem.Spec.Affinity = affinityRuleForVmiToFill
			vmiToFillTargetNodeMem.Spec.Domain.Resources.OvercommitGuestOverhead = true
			vmiToFillTargetNodeMem.Spec.Domain.Memory = &virtv1.Memory{Guest: &vmiMemSize}
			vmiToFillTargetNodeMem.Spec.Domain.Resources.Requests["memory"] = vmiMemReq

			By("Starting the VirtualMachineInstance")
			vmiToFillTargetNodeMem = libvmops.RunVMIAndExpectLaunch(vmiToFillTargetNodeMem, libvmops.StartupTimeoutSecondsHuge)
			Expect(console.LoginToFedora(vmiToFillTargetNodeMem)).To(Succeed())

			By("reaching memory overcommitment in the target node")
			err = fillMemWithStressFedoraVMI(vmiToFillTargetNodeMem, memToUseInTargetNodeVmKib)
			ExpectWithOffset(1, err).ToNot(HaveOccurred())

			vmiToMigrate := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())
			nodeAffinityRule, err := libmigration.CreateNodeAffinityRuleToMigrateFromSourceToTargetAndBack(sourceNode, targetNode)
			Expect(err).ToNot(HaveOccurred())
			vmiToMigrate.Spec.Affinity = &v1.Affinity{
				NodeAffinity: nodeAffinityRule,
			}
			vmiToMigrate.Spec.Domain.Resources.Requests["memory"] = vmiMemReq
			//add label the source node to make sure that the vm we want to migrate will be scheduled to the source node

			By("Starting the VirtualMachineInstance that we should migrate to the target node")
			vmiToMigrate = libvmops.RunVMIAndExpectLaunch(vmiToMigrate, libvmops.StartupTimeoutSecondsHuge)
			Expect(console.LoginToFedora(vmiToMigrate)).To(Succeed())

			// execute a migration, wait for finalized state
			By("Starting the Migration")
			migration := libmigration.New(vmiToMigrate.Name, vmiToMigrate.Namespace)
			libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

			By("The workloads in the node should consume more memory than the memory size eventually.")
			swapSizeKib := getSwapSizeInKib(*targetNode)
			totalMemKib := getTotalMemSizeInKib(*targetNode)
			Eventually(func() int {
				usedMemoryWithOutSwap := totalMemKib - getAvailableMemSizeInKib(*targetNode)
				usedSwapMemory := swapSizeKib - getSwapFreeSizeInKib(*targetNode)
				return usedMemoryWithOutSwap + usedSwapMemory
			}, 240*time.Second, 1*time.Second).Should(BeNumerically(">", totalMemKib),
				"at this point the node should has use more memory than totalMemKib "+
					"we check this to insure we migrated into memory overcommited node")
			By("Deleting the VMIs")
			Expect(virtClient.VirtualMachineInstance(vmiToFillTargetNodeMem.Namespace).Delete(context.Background(), vmiToFillTargetNodeMem.Name, metav1.DeleteOptions{})).To(Succeed())
			Expect(virtClient.VirtualMachineInstance(vmiToMigrate.Namespace).Delete(context.Background(), vmiToMigrate.Name, metav1.DeleteOptions{})).To(Succeed())

			By("Waiting for VMIs to disappear")
			libwait.WaitForVirtualMachineToDisappearWithTimeout(vmiToFillTargetNodeMem, 240)
			libwait.WaitForVirtualMachineToDisappearWithTimeout(vmiToMigrate, 240)
		})

	})
})

func getMemInfoByString(node v1.Node, field string) int {
	stdout, err := libnode.ExecuteCommandInVirtHandlerPod(node.Name, []string{"grep", field, "/proc/meminfo"})
	ExpectWithOffset(2, err).ToNot(HaveOccurred())
	fields := strings.Fields(stdout)
	size, err := strconv.Atoi(fields[1])
	ExpectWithOffset(2, err).ToNot(HaveOccurred())
	return size
}

func getAvailableMemSizeInKib(node v1.Node) int {
	return getMemInfoByString(node, "MemAvailable")
}

func getTotalMemSizeInKib(node v1.Node) int {
	return getMemInfoByString(node, "MemTotal")
}

func getSwapFreeSizeInKib(node v1.Node) int {
	return getMemInfoByString(node, "SwapFree")
}

func getSwapSizeInKib(node v1.Node) int {
	return getMemInfoByString(node, "SwapTotal")
}

func fillMemWithStressFedoraVMI(vmi *virtv1.VirtualMachineInstance, memToUseInTheVmKib int) error {
	_, err := console.SafeExpectBatchWithResponse(vmi, []expect.Batcher{
		&expect.BSnd{S: fmt.Sprintf("stress-ng --vm-bytes %db --vm-keep -m 1 &\n", memToUseInTheVmKib*bytesInKib)},
		&expect.BExp{R: ""},
	}, 15)
	return err
}

func failIfSwapOff(message string) {
	nodes := libnode.GetAllSchedulableNodes(kubevirt.Client())
	for _, node := range nodes.Items {
		swapSizeKib := getSwapSizeInKib(node)
		if swapSizeKib < maxSwapSizeToUseKib {
			Fail(message)
		}
	}
}

func getAffinityForTargetNode(targetNode *v1.Node) (nodeAffinity *v1.Affinity, err error) {
	nodeAffinityRuleForVmiToFill, err := libmigration.CreateNodeAffinityRuleToMigrateFromSourceToTargetAndBack(targetNode, targetNode)
	if err != nil {
		return nil, err
	}

	return &v1.Affinity{
		NodeAffinity: nodeAffinityRuleForVmiToFill,
	}, nil
}
